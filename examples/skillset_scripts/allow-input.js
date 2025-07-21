#!/usr/bin/env node

const process = require("process");

function main() {
  if (process.argv.length !== 3) {
    console.error("Usage: node allow-input.js '<SkillInputArgs JSON>'");
    process.exit(1);
  }

  let input;
  try {
    input = JSON.parse(process.argv[2]);
  } catch (err) {
    console.error("Invalid JSON input:", err.message);
    process.exit(2);
  }

  const output = {
    allowed: true,
    input: input,
  };

  console.log(JSON.stringify(output, null, 2));
}

main();
