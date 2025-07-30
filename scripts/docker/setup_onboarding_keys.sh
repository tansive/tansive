#!/bin/sh

set -e

echo "Setting up onboarding keys..."

# Generate a new key
NEW_KEY=`openssl rand -base64 64 | tr -d '=\n'`
echo "Generated new onboarding_key: $NEW_KEY"

# Process first file
echo "Processing tansivesrv.docker.conf..."
cp tansivesrv.docker.conf tansivesrv.docker.conf.updated
sed -i "s|^[[:space:]]*onboarding_key[[:space:]]*=[[:space:]]*.*|onboarding_key = \"$NEW_KEY\"|" tansivesrv.docker.conf.updated
echo "Updated onboarding_key in 'tansivesrv.docker.conf.updated'"

# Process second file
echo "Processing tangent.docker.conf..."
cp tangent.docker.conf tangent.docker.conf.updated
sed -i "s|^[[:space:]]*onboarding_key[[:space:]]*=[[:space:]]*.*|onboarding_key = \"$NEW_KEY\"|" tangent.docker.conf.updated
echo "Updated onboarding_key in 'tangent.docker.conf.updated'"

echo ""
echo "All TOML files have been updated with the same onboarding_key."
echo "Updated files have been created with .updated extension."
echo "Original files remain unchanged." 