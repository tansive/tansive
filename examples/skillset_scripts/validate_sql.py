#!/usr/bin/env python3
import sys
import json

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


def main():
    if len(sys.argv) < 2:
        print("No args provided", file=sys.stderr)
        sys.exit(1)

    try:
        args = json.loads(sys.argv[1])
    except Exception as e:
        print(f"Failed to parse input args: {e}", file=sys.stderr)
        sys.exit(1)

    session_id = args.get("sessionID")
    invocation_id = args.get("invocationID")
    socket_path = args.get("serviceEndpoint")
    input_args = args.get("inputArgs", {})
    client = SkillSetClient(
        socket_path, dial_timeout=10.0, max_retries=3, retry_delay=0.1
    )

    try:
        sql_permissions = client.get_context(
            session_id, invocation_id, "sql-permissions"
        )
        # Ensure sql_permissions is a dict
        if not isinstance(sql_permissions, dict):
            print("sql_permissions context is not a dict!", file=sys.stderr)
            sys.exit(1)
    except Exception as e:
        print(f"failed to get sql permissions: {e}", file=sys.stderr)
        sys.exit(1)

    # Inject sql_permissions into input_args for process
    input_args["sql_permissions"] = sql_permissions

    try:
        result = process(input_args)
        print(json.dumps(result))
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(2)


if __name__ == "__main__" and "--test" in sys.argv:
    # Simulate context and input
    sql_permissions = {
        "allow": {
            "select": ["support_tickets", "support_messages"],
            "update": ["support_messages"],
        },
        "deny": {"all": ["integration_tokens"]},
    }
    test_cases = [
        {"sql": "SELECT * FROM support_tickets;", "expect": True},
        {"sql": "SELECT * FROM integration_tokens;", "expect": False},
        {"sql": "UPDATE support_messages SET content='hi' WHERE id=1;", "expect": True},
        {
            "sql": "UPDATE support_tickets SET status='closed' WHERE id=1;",
            "expect": False,
        },
        {"sql": "DELETE FROM support_tickets WHERE id=1;", "expect": False},
        {
            "sql": "SELECT * FROM support_tickets; DELETE FROM integration_tokens;",
            "expect": False,
        },
        {
            "sql": "SELECT t.id, m.content FROM support_tickets t JOIN support_messages m ON t.id = m.ticket_id WHERE m.content IS NOT NULL;",
            "expect": True,
        },
        {
            "sql": "SELECT id, (SELECT COUNT(*) FROM support_messages m WHERE m.ticket_id = t.id) as message_count FROM support_tickets t;",
            "expect": True,
        },
        {
            "sql": "WITH recent_tickets AS (SELECT * FROM support_tickets WHERE created_at > NOW() - INTERVAL '7 days') SELECT r.id, m.content FROM recent_tickets r LEFT JOIN support_messages m ON r.id = m.ticket_id;",
            "expect": True,
        },
        {
            "sql": "UPDATE support_tickets SET status = 'closed' WHERE id IN (SELECT ticket_id FROM support_messages WHERE content LIKE '%resolved%');",
            "expect": False,
        },
        {
            "sql": "DELETE FROM support_messages USING support_tickets WHERE support_messages.ticket_id = support_tickets.id AND support_tickets.status = 'closed';",
            "expect": False,
        },
        {
            "sql": "INSERT INTO integration_tokens (token) SELECT content FROM support_messages WHERE content LIKE 'token:%';",
            "expect": False,
        },
        {
            "sql": "UPDATE support_tickets SET status = 'pending' WHERE id = 1; DELETE FROM integration_tokens WHERE token = 'abc123';",
            "expect": False,
        },
        {
            "sql": "WITH valid_tokens AS (SELECT token FROM integration_tokens WHERE token IS NOT NULL) SELECT t.id, m.content, v.token FROM support_tickets t JOIN support_messages m ON t.id = m.ticket_id LEFT JOIN valid_tokens v ON m.content LIKE '%' || v.token || '%';",
            "expect": False,
        },
        {
            "sql": "WITH tokens AS (SELECT * FROM integration_tokens) SELECT * FROM tokens;",
            "expect": False,
        },
        {
            "sql": "WITH updated AS (UPDATE integration_tokens SET token = 'new' WHERE id = 1 RETURNING *) SELECT * FROM updated;",
            "expect": False,
        },
        {
            "sql": "WITH deleted AS (DELETE FROM integration_tokens WHERE id = 1 RETURNING *) SELECT * FROM deleted;",
            "expect": False,
        },
        {
            "sql": "WITH tokens AS (SELECT * FROM integration_tokens) SELECT t.id, m.content FROM tokens t JOIN support_messages m ON t.id = m.token_id;",
            "expect": False,
        },
        {
            "sql": "WITH messages AS (SELECT * FROM support_messages) SELECT * FROM messages;",
            "expect": True,
        },
        {"sql": 'SELECT * FROM public."Integration_Tokens";', "expect": False},
        {
            "sql": "SELECT * FROM (SELECT * FROM integration_tokens) AS it;",
            "expect": False,
        },
    ]
    summary = []
    for idx, case in enumerate(test_cases):
        case["sql_permissions"] = sql_permissions
        print(f"Test {idx + 1}: SQL: {case['sql']}")
        result = process(case)
        actual = result.get("allowed", False)
        expected = case["expect"]
        passed = actual == expected
        print(json.dumps(result, indent=2))
        print(
            f"Expected: {'PASS' if expected else 'FAIL'} | Actual: {'PASS' if actual else 'FAIL'} | {'✅' if passed else '❌'}"
        )
        print("-" * 40)
        summary.append((idx + 1, passed, expected, actual))
    # Print summary
    print("\nTest Summary:")
    for idx, passed, expected, actual in summary:
        print(
            f"Test {idx}: {'PASS' if actual else 'FAIL'} (Expected: {'PASS' if expected else 'FAIL'}) {'✅' if passed else '❌'}"
        )
    print(
        f"\nTotal: {len(summary)} | Passed: {sum(1 for _, p, _, _ in summary if p)} | Failed: {sum(1 for _, p, _, _ in summary if not p)}"
    )
    sys.exit(0)


if __name__ == "__main__":
    main()
