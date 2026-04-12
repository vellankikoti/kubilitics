#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_HOME="$(mktemp -d)"
trap 'rm -rf "$TMP_HOME"' EXIT

KCLI_BIN="${ROOT_DIR}/bin/kcli"
(cd "$ROOT_DIR" && go build -o "$KCLI_BIN" ./cmd/kcli)

for p in cert-manager argocd istio backup; do
  KCLI_HOME_DIR="$TMP_HOME" "$KCLI_BIN" plugin install "${ROOT_DIR}/official-plugins/${p}" >/dev/null
  KCLI_HOME_DIR="$TMP_HOME" "$KCLI_BIN" plugin inspect "$p" >/dev/null
  KCLI_HOME_DIR="$TMP_HOME" "$KCLI_BIN" plugin allow "$p" >/dev/null
  echo "PASS ${p} install/inspect/allow"
done

echo "PASS official plugin smoke"
