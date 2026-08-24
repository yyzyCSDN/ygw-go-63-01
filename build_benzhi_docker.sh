#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

docker build -f benzhi.Dockerfile -t modelrouter:local .
echo "built modelrouter:local"
