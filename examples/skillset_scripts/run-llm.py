import sys
import json
import openai
from tansive.skillset_sdk import SkillSetClient

LLM_BLOCKED_BY_POLICY_PROMPT = """
All tools with tag [TansivePolicy: true] are governed by Tansive policy.
If any tool call with such tag returns an error containing "This operation is blocked by Tansive policy", you must respond to the user with:
"I tried to use Skill: <tool-name> for <reason> but it was blocked by Tansive policy. Please contact the administrator of your Tansive system to obtain access." Do not attempt to bypass, hallucinate, or reroute the request. Respect the policy boundaries.
"""


def get_skills(client: SkillSetClient, session_id: str) -> list[dict]:
    try:
        skills = client.get_skills(session_id)
    except Exception as e:
        print(f"failed to get skills: {e}", file=sys.stderr)
        sys.exit(1)

    result = []
    for skill in skills:
        try:
            schema = skill["inputSchema"]
        except KeyError as e:
            print(f"missing required field in skill: {e}", file=sys.stderr)
            sys.exit(1)
        except Exception as e:
            print(f"failed to process input schema: {e}", file=sys.stderr)
            sys.exit(1)

        result.append(
            {
                "type": "function",
                "function": {
                    "name": skill["name"],
                    "description": skill.get("description", ""),
                    "parameters": schema,
                },
            }
        )
    return result


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

    model_name = input_args.get("model")
    if not isinstance(model_name, str) or not model_name:
        print("model not provided in input args", file=sys.stderr)
        sys.exit(1)

    client = SkillSetClient(
        socket_path, dial_timeout=10.0, max_retries=3, retry_delay=0.1
    )

    try:
        model_info = client.get_context(session_id, invocation_id, model_name)
    except Exception as e:
        print(f"failed to get model info: {e}", file=sys.stderr)
        sys.exit(1)

    if not isinstance(model_info, dict):
        print("model info is not a dict", file=sys.stderr)
        sys.exit(1)

    api_key = model_info.get("apiKey")
    model = model_info.get("model")
    if not isinstance(api_key, str) or not isinstance(model, str):
        print("invalid apiKey or model in model info", file=sys.stderr)
        sys.exit(1)

    question = input_args.get("prompt")
    if not isinstance(question, str) or not question:
        print("prompt not provided in input args", file=sys.stderr)
        sys.exit(1)

    openai_args = {"api_key": api_key}
    if model.startswith("claude"):
        openai_args["base_url"] = "https://api.anthropic.com/v1"

    openai_client = openai.OpenAI(**openai_args)

    messages = [
        {"role": "system", "content": LLM_BLOCKED_BY_POLICY_PROMPT},
        {"role": "user", "content": question},
    ]
    skills = get_skills(client, session_id)

    while True:
        try:
            response = openai_client.chat.completions.create(
                model=model,
                messages=messages,
                tools=skills,
                seed=0,
            )
        except Exception as e:
            print(f"OpenAI call failed: {e}", file=sys.stderr)
            sys.exit(1)

        choice = response.choices[0]
        message = choice.message
        finish_reason = choice.finish_reason

        if finish_reason and finish_reason != "tool_calls":
            print(f"✅ Final response: {message.content}")
            break
        else:
            print(f"🤔 Thinking: {message.content}")

        messages.append(message.model_dump(exclude_unset=True))

        if not message.tool_calls:
            break

        for tool_call in message.tool_calls:
            try:
                tool_args = json.loads(tool_call.function.arguments)
                result = client.invoke_skill(
                    session_id, invocation_id, tool_call.function.name, tool_args
                )
                tool_response = json.dumps(result.output)

            except Exception as e:
                print(f"Tool call failed: {e}", file=sys.stderr)
                sys.exit(1)

            messages.append(
                {
                    "role": "tool",
                    "tool_call_id": tool_call.id,
                    "content": tool_response,
                }
            )


if __name__ == "__main__":
    main()
