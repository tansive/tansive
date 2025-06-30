#!/usr/bin/env python3
import sys
import json


def process(skill_input_json: str) -> dict:
    try:
        data = json.loads(skill_input_json)
    except json.JSONDecodeError as e:
        raise ValueError(f"Invalid JSON input: {e}")

    input_args = data.get("inputArgs", {})
    patient_id = input_args.get("patient_id")
    if not patient_id:
        raise ValueError("Missing required field: inputArgs.patient_id")

    # Return canned bloodwork
    return {
        "patient_id": patient_id,
        "bloodwork": {
            "hemoglobin": 13.5,
            "white_cell_count": 6.2,
            "platelets": 250,
            "glucose": 98,
            "cholesterol": {"total": 180, "ldl": 100, "hdl": 55},
        },
    }


def main():
    if len(sys.argv) != 2:
        print(
            "Usage: python3 patient_bloodwork.py '<SkillInputArgs JSON>'",
            file=sys.stderr,
        )
        sys.exit(1)

    skill_input = sys.argv[1]

    try:
        result = process(skill_input)
        print(json.dumps(result))
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(2)


if __name__ == "__main__":
    main()
