#!/bin/bash

# Script to update onboarding_key in TOML files
# Usage: ./update_onboarding_keys.sh file1.toml file2.toml ...

set -e

generate_onboarding_key() {
    openssl rand -base64 64 | tr -d '=\n'
}

is_onboarding_key_empty() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo "Error: File '$file' does not exist" >&2
        return 1
    fi
    
    if grep -q "^[[:space:]]*onboarding_key[[:space:]]*=[[:space:]]*[\"']*[\"']*[[:space:]]*$" "$file" 2>/dev/null; then
        return 0
    elif ! grep -q "^[[:space:]]*onboarding_key[[:space:]]*=" "$file" 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

update_onboarding_key() {
    local file="$1"
    local new_key="$2"
    
    cp "$file" "$file.backup"
    
    if grep -q "^[[:space:]]*onboarding_key[[:space:]]*=" "$file" 2>/dev/null; then
        if [[ "$OSTYPE" == "darwin"* ]]; then
            # macOS uses BSD sed
            sed -i '' "s|^[[:space:]]*onboarding_key[[:space:]]*=[[:space:]]*.*|onboarding_key = \"$new_key\"|" "$file"
        else
            # Linux uses GNU sed
            sed -i "s|^[[:space:]]*onboarding_key[[:space:]]*=[[:space:]]*.*|onboarding_key = \"$new_key\"|" "$file"
        fi
    else
        echo "" >> "$file"
        echo "onboarding_key = \"$new_key\"" >> "$file"
    fi
    
    echo "Updated onboarding_key in '$file'"
}

main() {
    if [[ $# -eq 0 ]]; then
        echo "Usage: $0 file1.toml file2.toml ..."
        echo "This script updates onboarding_key in TOML files to ensure they all have the same value."
        exit 1
    fi
    
    if ! command -v openssl &> /dev/null; then
        echo "Error: openssl is required but not installed" >&2
        exit 1
    fi
    
    local files=("$@")
    local needs_update=false
    local generated_key=""
    
    echo "Checking TOML files for empty onboarding_key..."
    
    for file in "${files[@]}"; do
        if is_onboarding_key_empty "$file"; then
            needs_update=true
            echo "Found empty onboarding_key in '$file'"
        fi
    done
    
    if [[ "$needs_update" == "false" ]]; then
        echo "All files already have non-empty onboarding_key values."
        exit 0
    fi
    
    generated_key=$(generate_onboarding_key)
    echo "Generated new onboarding_key: $generated_key"
    echo ""
    
    for file in "${files[@]}"; do
        if is_onboarding_key_empty "$file"; then
            update_onboarding_key "$file" "$generated_key"
        else
            update_onboarding_key "$file" "$generated_key"
        fi
    done
    
    echo ""
    echo "All TOML files have been updated with the same onboarding_key."
    echo "Backup files have been created with .backup extension."
}

main "$@" 