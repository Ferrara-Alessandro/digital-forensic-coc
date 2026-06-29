#!/usr/bin/env bash
# Esegue tutti i test, oppure una singola categoria.
#
# Uso:
#   bash tests/run_all.sh                    # tutto
#   bash tests/run_all.sh ciclo_vita         # workflow reperto (PG→PM→LAB)
#   bash tests/run_all.sh fabric_caliper     # benchmark chaincode (Caliper, read)
#   bash tests/run_all.sh cifratura_ipfs     # upload cifrato + download (client Go)
set -euo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "$TESTS_DIR/lib/common.sh"

TARGET="${1:-all}"

run_ciclo_vita()     { bash "$TESTS_DIR/ciclo_vita/run.sh"; }
run_fabric_caliper() { bash "$TESTS_DIR/fabric_caliper/run.sh" read; }
run_cifratura_ipfs() { bash "$TESTS_DIR/cifratura_ipfs/run.sh" all; }

case "$TARGET" in
  all)
    run_ciclo_vita
    run_fabric_caliper
    run_cifratura_ipfs
    ;;
  ciclo_vita)     run_ciclo_vita ;;
  fabric_caliper) run_fabric_caliper ;;
  cifratura_ipfs) run_cifratura_ipfs ;;
  *)
    echo "Target sconosciuto: $TARGET" >&2
    echo "Valori: all | ciclo_vita | fabric_caliper | cifratura_ipfs" >&2
    exit 1
    ;;
esac

section "Fine suite ($TARGET)"
echo "Risultati CSV (se generati): $RESULTS_DIR"
