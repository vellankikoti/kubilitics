#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="${ROOT_DIR}/bin/kcli"
mkdir -p "${ROOT_DIR}/bin"

cd "${ROOT_DIR}"
go build -o "${BIN}" ./cmd/kcli

measure_ms() {
  local cmd="$1"
  local start end
  start=$(perl -MTime::HiRes=time -e 'printf "%.9f", time')
  eval "${cmd}" >/dev/null 2>&1
  end=$(perl -MTime::HiRes=time -e 'printf "%.9f", time')
  perl -e "printf \"%.3f\", ((${end})-(${start}))*1000"
}

p95_file() {
  local f="$1"
  sort -n "$f" | awk '{a[NR]=$1} END { if (NR==0) { print 0; exit } idx=int((NR*95+99)/100); if (idx<1) idx=1; if (idx>NR) idx=NR; printf "%.3f", a[idx] }'
}

avg_file() {
  local f="$1"
  awk '{s+=$1; n++} END { if (n==0) { print 0; exit } printf "%.3f", s/n }' "$f"
}

run_bench() {
  local label="$1"
  local iterations="$2"
  local cmd="$3"
  local f
  f=$(mktemp)
  for _ in $(seq 1 "$iterations"); do
    measure_ms "$cmd" >> "$f"
    echo >> "$f"
  done
  local p95 avg
  p95=$(p95_file "$f")
  avg=$(avg_file "$f")
  rm -f "$f"
  printf "%s avg=%sms p95=%sms\n" "$label" "$avg" "$p95" >&2
  echo "$p95"
}

current_ctx=$(kubectl config current-context 2>/dev/null || true)
if [[ -z "${current_ctx}" ]]; then
  echo "No current kube context; cannot run ctx/get benchmarks" >&2
  exit 1
fi

echo "Running performance checks with ${BIN}"
startup_p95=$(run_bench "startup(version)" 40 "'${BIN}' version")
get_p95=$(run_bench "get pods -A" 20 "'${BIN}' get pods -A")
ctx_p95=$(run_bench "ctx switch (same context)" 20 "'${BIN}' ctx '${current_ctx}'")

mem_mb=0
if /usr/bin/time -l "${BIN}" version >/dev/null 2>"${ROOT_DIR}/.perf_mem"; then
  if grep -q "peak memory footprint" "${ROOT_DIR}/.perf_mem"; then
    mem_bytes=$(awk '/peak memory footprint/{print $1}' "${ROOT_DIR}/.perf_mem")
    mem_mb=$(awk -v b="$mem_bytes" 'BEGIN { printf "%.2f", b/1024/1024 }')
  fi
fi
if [[ "$mem_mb" == "0" ]]; then
  /usr/bin/time -v "${BIN}" version >/dev/null 2>"${ROOT_DIR}/.perf_mem" || true
  if grep -q "Maximum resident set size" "${ROOT_DIR}/.perf_mem"; then
    mem_kb=$(awk -F': *' '/Maximum resident set size/{print $2}' "${ROOT_DIR}/.perf_mem")
    mem_mb=$(awk -v kb="$mem_kb" 'BEGIN { printf "%.2f", kb/1024 }')
  fi
fi
rm -f "${ROOT_DIR}/.perf_mem"

echo "memory(version): ${mem_mb} MB"

fail=0
awk -v v="$startup_p95" 'BEGIN { exit !(v > 200.0) }' && { echo "FAIL: startup p95 > 200ms"; fail=1; } || true
awk -v v="$get_p95" 'BEGIN { exit !(v > 500.0) }' && { echo "FAIL: get pods p95 > 500ms"; fail=1; } || true
awk -v v="$ctx_p95" 'BEGIN { exit !(v > 100.0) }' && { echo "FAIL: ctx switch p95 > 100ms"; fail=1; } || true
awk -v v="$mem_mb" 'BEGIN { exit !(v > 50.0) }' && { echo "FAIL: memory > 50MB"; fail=1; } || true

if [[ "$fail" -ne 0 ]]; then
  exit 1
fi

echo "PASS: all TASK-KCLI-018 performance gates met"
