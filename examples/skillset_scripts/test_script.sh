#!/bin/bash

# Print raw input
echo "Raw input: $1"
sdfds
# Parse JSON using jq if available, otherwise use basic parsing
if command -v jq >/dev/null 2>&1; then
    echo "Parsed JSON:"
    echo "$1" | jq -r "to_entries | .[] | \"\(.key): \(.value)\""
else
    echo "Parsed JSON (basic):"
    # Basic JSON parsing for testing
    echo "$1" | sed -E "s/[{}]//g" | tr "," "\n" | sed -E "s/\"([^\"]+)\":\"([^\"]+)\"/\1: \2/"
fi

# Check if should fail
if echo "$1" | grep -q "should_fail"; then
    exit 1
fi

# Check if should display environment variables
if echo "$1" | grep -q "check_env"; then
    echo "Environment variables:"
    env | sort
fi

# Check if should display home directory contents
if echo "$1" | grep -q "check_home"; then
    echo "Home directory contents:"
    ls -la ~
fi

# Print all arguments
echo "All arguments:"
for arg in "$@"; do
    echo "arg: $arg"
done
