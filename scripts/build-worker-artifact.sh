#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${OUT_DIR:-${ROOT_DIR}/build/deploy}"
ARTIFACT_NAME="${ARTIFACT_NAME:-capstone-worker-linux-amd64.tar.gz}"

mkdir -p "${OUT_DIR}/worker"

cd "${ROOT_DIR}"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o "${OUT_DIR}/worker/worker" ./cmd/worker

tar -C "${OUT_DIR}/worker" -czf "${OUT_DIR}/${ARTIFACT_NAME}" worker

printf '%s\n' "${OUT_DIR}/${ARTIFACT_NAME}"
