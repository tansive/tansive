#!/bin/bash

# my-tools-script

set -e

# Read entire input
INPUT=$1

# Extract fields from JSON
SKILL_NAME=$(echo "$INPUT" | jq -r '.skillName')
INPUT_ARGS=$(echo "$INPUT" | jq -c '.inputArgs')

# Respond based on skill name
case "$SKILL_NAME" in
  list_pods)
    LABEL_SELECTOR=$(echo "$INPUT_ARGS" | jq -r '.labelSelector')
    echo "NAME                                READY   STATUS    RESTARTS   AGE"
    echo "api-server-5f5b7f77b7-zx9qs          1/1     Running   0          2d"
    echo "web-frontend-6f6f9d7b7b-xv2mn        1/1     Running   1          5h"
    echo "cache-worker-7d7d9d9b7b-pv9lk        1/1     Running   0          1d"
    echo "orders-api-7ff9d44db7-abcde          0/1     CrashLoopBackOff   12         3h"
    echo "# Filter applied: $LABEL_SELECTOR"
    ;;
  restart_deployment)
    DEPLOYMENT=$(echo "$INPUT_ARGS" | jq -r '.deployment')
    echo "deployment.apps/$DEPLOYMENT restarted"
    ;;
  *)
    echo "Unknown skillName: $SKILL_NAME" >&2
    exit 1
    ;;
esac
