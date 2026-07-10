#!/usr/bin/env bash
# Campagna Caliper unificata per la tesi: read + query-mix + create, 4 worker ovunque.
# Scrive un CSV fresco in tests/results/caliper/caliper_summary.csv
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CALIPER_DIR="$SCRIPT_DIR/caliper"
# shellcheck source=../lib/common.sh
source "$SCRIPT_DIR/../lib/common.sh"

RESULTS="${TEST_RESULTS_DIR:-$TESTS_DIR/results}/caliper"
TS="$(date +%Y%m%d-%H%M%S)"
ARCHIVE="$RESULTS/archive-pre-campaign-$TS"

require_docker_peer
require_fabric_peers_sync

section "Campagna Caliper unificata (tesi)"

if [[ -f "$RESULTS/caliper_summary.csv" ]]; then
  mkdir -p "$ARCHIVE"
  cp -a "$RESULTS"/caliper_summary.csv "$ARCHIVE/" 2>/dev/null || true
  cp -a "$RESULTS"/report-*.html "$ARCHIVE/" 2>/dev/null || true
  echo "Backup risultati precedenti: $ARCHIVE"
fi

mkdir -p "$RESULTS"
echo "timestamp,benchmark,round,succ,fail,send_tps,max_lat_s,min_lat_s,avg_lat_s,throughput_tps" \
  >"$RESULTS/caliper_summary.csv"

run_bench() {
  local yaml="$1"
  local tag="$2"
  bash "$CALIPER_DIR/scripts/run_benchmark.sh" "$yaml" "$tag"
}

run_bench readReperto_full read
run_bench queryMix query-mix
run_bench createReperto_full create

section "Campagna completata"
echo "CSV: $RESULTS/caliper_summary.csv"
wc -l "$RESULTS/caliper_summary.csv"
