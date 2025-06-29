#!/bin/bash

# Exit on error
set -e

echo "Creating test resources..."

# First create the catalog
echo "Creating catalog..."
tansive create -f test/catalog.yaml

# Then create the variant (it will use the catalog context)
echo "Creating variant..."
tansive create -f test/variant.yaml -c example-catalog

# Create the namespace (it will use both catalog and variant context)
echo "Creating namespace..."
tansive create -f test/namespace.yaml -c example-catalog -v example-variant

# Create the workspace (it will use all contexts)
echo "Creating workspace..."
tansive create -f test/workspace.yaml -c example-catalog -v example-variant -n example-namespace

# Create the schemas
echo "Creating collection schema..."
tansive create -f test/collection-schema.yaml -c example-catalog -v example-variant -n example-namespace

echo "Creating parameter schema..."
tansive create -f test/parameter-schema.yaml -c example-catalog -v example-variant -n example-namespace

# Finally create the collection
echo "Creating collection..."
tansive create -f test/collection.yaml -c example-catalog -v example-variant -n example-namespace -w example-workspace

echo "All resources created successfully!" 