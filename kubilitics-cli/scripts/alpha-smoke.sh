#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="${ROOT_DIR}/bin/kcli"

mkdir -p "${ROOT_DIR}/bin"
cd "${ROOT_DIR}"
go build -o "${BIN}" ./cmd/kcli

pass_count=0
fail_count=0

run_check() {
  local name="$1"
  shift
  if "$@" >/tmp/kcli-alpha.out 2>/tmp/kcli-alpha.err; then
    echo "PASS  ${name}"
    pass_count=$((pass_count+1))
  else
    echo "FAIL  ${name}"
    sed -n '1,10p' /tmp/kcli-alpha.err
    fail_count=$((fail_count+1))
  fi
}

run_check "version" "${BIN}" version
run_check "help" "${BIN}" --help
run_check "completion bash" "${BIN}" completion bash
run_check "completion zsh" "${BIN}" completion zsh
run_check "completion fish" "${BIN}" completion fish
run_check "completion powershell" "${BIN}" completion powershell

if kubectl config current-context >/dev/null 2>&1; then
  current_ctx=$(kubectl config current-context)
  run_check "ctx list" "${BIN}" ctx
  run_check "ctx switch same" "${BIN}" ctx "${current_ctx}"
  run_check "ns list" "${BIN}" ns
  run_check "get pods -A" "${BIN}" get pods -A
  run_check "auth check" "${BIN}" auth check
else
  echo "SKIP  cluster checks (no kubectl context)"
fi

run_check "perf gate" "${ROOT_DIR}/scripts/perf-check.sh"

echo
echo "alpha smoke summary: pass=${pass_count} fail=${fail_count}"
rm -f /tmp/kcli-alpha.out /tmp/kcli-alpha.err

if [[ "${fail_count}" -ne 0 ]]; then
  exit 1
fi
