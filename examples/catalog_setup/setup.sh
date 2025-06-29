#!/bin/bash

# Exit on any error
set -e

# Function to run commands with color
run_cmd() {
    echo -e "\033[34m+ $*\033[0m"
    "$@"
}

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Create catalog
run_cmd tansive apply -f "$SCRIPT_DIR/catalog.yaml"

# Adopt catalog view
run_cmd tansive set-catalog demo-catalog

# Create Namespaces and Variants
run_cmd tansive apply -f "$SCRIPT_DIR/catalog-setup.yaml"

# Create kubernetes troubleshooter skillset in dev and prod
run_cmd tansive apply -f "$SCRIPT_DIR/skillset-k8s.yaml" --variant dev
run_cmd tansive apply -f "$SCRIPT_DIR/skillset-k8s.yaml" --variant prod

# Create the health demo skillset in dev
run_cmd tansive apply -f "$SCRIPT_DIR/skillset-patient.yaml" --variant dev