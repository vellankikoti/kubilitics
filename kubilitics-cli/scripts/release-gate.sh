#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

run() {
  local name="$1"
  shift
  echo "==> ${name}"
  "$@"
  echo "PASS ${name}"
  echo
}

run "unit+integration tests" go test ./...
run "alpha smoke" "${ROOT_DIR}/scripts/alpha-smoke.sh"
run "performance gate" "${ROOT_DIR}/scripts/perf-check.sh"

echo "RELEASE GATE PASSED"
