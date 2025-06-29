#!/usr/bin/env node

const process = require("process");

function resolvePatientId(firstName) {
  switch (firstName.toLowerCase()) {
    case "john":
      return "H12345";
    case "sheila":
      return "H23456";
    case "anand":
      return "H56789";
    default:
      return null;
  }
}

function main() {
  if (process.argv.length !== 3) {
    console.error("Usage: node resolve_patient_id.js '<SkillInputArgs JSON>'");
    process.exit(1);
  }

  let input;
  try {
    input = JSON.parse(process.argv[2]);
  } catch (err) {
    console.error("Invalid JSON input:", err.message);
    process.exit(2);
  }

  const name = input?.inputArgs?.name;
  if (!name) {
    console.error("Missing inputArgs.name");
    process.exit(3);
  }

  const firstName = name.trim().split(/\s+/)[0];
  const patientId = resolvePatientId(firstName);

  if (!patientId) {
    console.error(`Unknown patient name: ${firstName}`);
    process.exit(4);
  }

  console.log(JSON.stringify({ patient_id: patientId }, null, 2));
}

main();
