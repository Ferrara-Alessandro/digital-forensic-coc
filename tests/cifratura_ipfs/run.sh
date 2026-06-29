#!/usr/bin/env bash
# Benchmark cifratura, IPFS, upload e download — tutto via client Go (non Caliper).
# Uso: bash run.sh [upload|download|all]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODE="${1:-all}"

case "$MODE" in
  upload)   bash "$SCRIPT_DIR/benchmark_upload.sh" ;;
  download) bash "$SCRIPT_DIR/benchmark_download.sh" ;;
  all)
    bash "$SCRIPT_DIR/benchmark_upload.sh"
    bash "$SCRIPT_DIR/benchmark_download.sh"
    ;;
  *)
    echo "Uso: $0 [upload|download|all]" >&2
    exit 1
    ;;
esac
