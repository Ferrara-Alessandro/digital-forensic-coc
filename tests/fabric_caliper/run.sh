#!/usr/bin/env bash
# Benchmark prestazioni chaincode Fabric con Hyperledger Caliper.
# Uso: bash run.sh [read|query-mix|create|all]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CALIPER_DIR="$SCRIPT_DIR/caliper"
# shellcheck source=../lib/common.sh
source "$SCRIPT_DIR/../lib/common.sh"

SCENARIO="${1:-read}"

require_docker_peer
require_fabric_peers_sync

section "Benchmark Fabric (Caliper) — $SCENARIO"
case "$SCENARIO" in
  read)
    bash "$CALIPER_DIR/scripts/run_readReperto.sh"
    ;;
  query-mix)
    bash "$CALIPER_DIR/scripts/run_queryMix.sh"
    ;;
  create)
    bash "$CALIPER_DIR/scripts/run_createReperto.sh"
    ;;
  all)
    bash "$CALIPER_DIR/scripts/run_readReperto.sh"
    bash "$CALIPER_DIR/scripts/run_queryMix.sh"
    bash "$CALIPER_DIR/scripts/run_createReperto.sh"
    ;;
  *)
    echo "Uso: $0 [read|query-mix|create|all]" >&2
    exit 1
    ;;
esac
