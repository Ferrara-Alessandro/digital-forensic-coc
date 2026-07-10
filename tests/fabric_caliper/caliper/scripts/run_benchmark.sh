#!/usr/bin/env bash
# Esegue un file benchmarks/*.yaml e archivia report + CSV.
# Uso: run_benchmark.sh readReperto read
set -euo pipefail

BENCH_FILE="${1:?usage: run_benchmark.sh <yaml senza path> <tag>}"
BENCH_TAG="${2:-${BENCH_FILE%.yaml}}"

CALIPER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

bash "$CALIPER_DIR/scripts/prepara_dati.sh"

cd "$CALIPER_DIR"
if [[ ! -d node_modules ]]; then
  npm install --omit=dev
fi
npx caliper bind --caliper-bind-sut fabric:2.2
npx caliper launch manager \
  --caliper-workspace . \
  --caliper-networkconfig networks/network.yaml \
  --caliper-benchconfig "benchmarks/${BENCH_FILE}.yaml" \
  --caliper-flow-only-test

bash "$CALIPER_DIR/scripts/archive_report.sh" "$BENCH_TAG"
