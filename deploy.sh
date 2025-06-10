#!/bin/bash

set -e

SCRIPT_DIR="$(CDPATH= command cd -- "$(dirname -- "$0")" && pwd -P)"

cd "$SCRIPT_DIR"

MODE="$1"
ENV_FILE="$MODE.env"

if [ ! -f "$ENV_FILE" ]; then
    echo "Env file $ENV_FILE does not exist"
    exit 1
fi

mkdir -p .kamal
go run utils/kamal.go $1

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

read -r -p "Show secrets? (y/N): " input && [[ "$input" == "y" ]] && \
  kamal secrets print -d "$MODE"
read -r -p "Ready to deploy? (y/N): " input && [[ "$input" == "y" ]] && \
  kamal deploy -d "$MODE"
