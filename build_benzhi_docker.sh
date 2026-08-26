#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
docker build -f benzhi.Dockerfile -t ygw-go-104-01:latest .
