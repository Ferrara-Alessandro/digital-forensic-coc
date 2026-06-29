#!/usr/bin/env bash
set -euo pipefail
CALIPER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
bash "$CALIPER_DIR/scripts/run_benchmark.sh" createReperto create
