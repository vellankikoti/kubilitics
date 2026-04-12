#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
KCLI_BIN="${ROOT_DIR}/bin/kcli"

(cd "$ROOT_DIR" && go build -o "$KCLI_BIN" ./cmd/kcli)

for p in cert-manager argocd istio backup; do
  "$KCLI_BIN" plugin install "${ROOT_DIR}/official-plugins/${p}"
  "$KCLI_BIN" plugin allow "$p"
done

echo "Installed official plugins: cert-manager, argocd, istio, backup"
