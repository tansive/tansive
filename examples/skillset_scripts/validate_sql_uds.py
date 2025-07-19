#!/usr/bin/env python3
import asyncio
import os
import sys
import time
from aiohttp import web

from sqlglot import parse, expressions
from tansive.skillset_sdk import SkillSetClient


def extract_sql_info_multi(sql: str) -> list:
    try:
        statements = parse(sql)
        result = []

        for stmt in statements:
            stmt_type = stmt.key.upper()
            # Find CTE names in this statement
            cte_names = set()
            if hasattr(stmt, "ctes") and stmt.ctes:
                for cte in stmt.ctes:
                    if hasattr(cte, "alias_or_name"):
                        cte_names.add(str(cte.alias_or_name))
                    elif hasattr(cte, "this") and hasattr(cte.this, "name"):
                        cte_names.add(str(cte.this.name))
            tables = {table.name for table in stmt.find_all(expressions.Table)}
            result.append(
                {"type": stmt_type, "tables": sorted(tables), "ctes": sorted(cte_names)}
            )

        return result

    except Exception as e:
        return [{"type": None, "tables": [], "ctes": [], "error": str(e)}]


def normalize_table_name(table):
    # Remove schema and lower-case, strip quotes
    return table.lower().split(".")[-1].strip('"')


def process(input_args):
    sql = input_args.get("sql")
    if not sql:
        return {"allowed": False, "reason": "No SQL provided"}

    # Get sql-permissions from the context (injected in main)
    sql_permissions = input_args.get("sql_permissions")
    if sql_permissions is None:
        return {"allowed": False, "reason": "No sql-permissions context provided"}

    allow = sql_permissions.get("allow", {})
    deny = sql_permissions.get("deny", {})

    stmts = extract_sql_info_multi(sql)
    details = []
    denied = False
    deny_reason = None

    for stmt in stmts:
        stmt_type = stmt.get("type")
        tables = stmt.get("tables", [])
        ctes = set(stmt.get("ctes", []))
        if stmt_type is None:
            # Handle parse error for this statement
            stmt_detail = {
                "type": None,
                "tables": tables,
                "ctes": list(ctes),
                "allowed": False,
                "denied": True,
                "reason": stmt.get("error", "SQL parse error"),
            }
            details.append(stmt_detail)
            denied = True
            deny_reason = stmt_detail["reason"]
            continue
        stmt_detail = {
            "type": stmt_type,
            "tables": tables,
            "ctes": list(ctes),
            "allowed": True,
            "denied": False,
            "reason": None,
        }

        # Only check real tables (not CTEs)
        real_tables = [t for t in tables if t not in ctes]

        # Check deny first (deny takes precedence)
        for deny_action, deny_tables in deny.items():
            if deny_action == "all" or deny_action.lower() == (stmt_type or "").lower():
                deny_tables_normalized = {normalize_table_name(t) for t in deny_tables}
                for table in real_tables:
                    table_normalized = normalize_table_name(table)
                    if table_normalized in deny_tables_normalized:
                        stmt_detail["allowed"] = False
                        stmt_detail["denied"] = True
                        stmt_detail["reason"] = (
                            f"Denied by deny.{deny_action} for table {table}"
                        )
                        denied = True
                        deny_reason = stmt_detail["reason"]
        # Only check allow if not denied
        if not stmt_detail["denied"]:
            allowed_tables = allow.get(stmt_type.lower(), [])
            allowed_tables_all = allow.get("all", [])
            allowed_tables_normalized = {
                normalize_table_name(t) for t in allowed_tables
            } | {normalize_table_name(t) for t in allowed_tables_all}
            for table in real_tables:
                table_normalized = normalize_table_name(table)
                if table_normalized not in allowed_tables_normalized:
                    stmt_detail["allowed"] = False
                    stmt_detail["reason"] = f"Table {table} not allowed for {stmt_type}"
                    denied = True
                    deny_reason = stmt_detail["reason"]
        details.append(stmt_detail)

    return {
        "allowed": not denied,
        "details": details,
        "reason": deny_reason if denied else "All statements allowed",
    }


# HTTP handler
async def handle_post(request):
    start_time = time.perf_counter()
    try:
        args = await request.json()
        session_id = args.get("sessionID")
        invocation_id = args.get("invocationID")
        socket_path = args.get("serviceEndpoint")
        input_args = args.get("inputArgs", {})

        client = SkillSetClient(
            socket_path, dial_timeout=10.0, max_retries=3, retry_delay=0.1
        )

        t0 = time.perf_counter()
        sql_permissions = client.get_context(
            session_id, invocation_id, "sql-permissions"
        )
        t1 = time.perf_counter()

        print(f"[timing] skill fetch: {(t1 - t0) * 1000:.2f} ms", file=sys.stderr)

        if not isinstance(sql_permissions, dict):
            raise ValueError("sql_permissions context is not a dict")

        input_args["sql_permissions"] = sql_permissions

        result = process(input_args)

        end_time = time.perf_counter()
        print(
            f"[timing] total request: {(end_time - start_time) * 1000:.2f} ms",
            file=sys.stderr,
        )

        return web.json_response(result)

    except Exception as e:
        end_time = time.perf_counter()
        print(
            f"[timing] total request (error): {(end_time - start_time) * 1000:.2f} ms",
            file=sys.stderr,
        )
        return web.json_response({"error": str(e)}, status=500)


# Launch server on UDS
async def main():
    sock_path = "/tmp/sqlchecker.sock"
    if os.path.exists(sock_path):
        os.remove(sock_path)

    app = web.Application()
    app.router.add_post("/", handle_post)

    runner = web.AppRunner(app)
    await runner.setup()
    site = web.UnixSite(runner, sock_path)
    await site.start()
    print(f"Listening on UDS {sock_path}...")
    while True:
        await asyncio.sleep(3600)


if __name__ == "__main__":
    asyncio.run(main())
