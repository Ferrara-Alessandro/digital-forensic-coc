#!/usr/bin/env bash
# Test end-to-end del workflow del reperto (PG → PM → LAB).
# Strumento: bash + peer CLI in Docker.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../lib/common.sh
source "$SCRIPT_DIR/../lib/common.sh"

require_docker_peer
require_fabric_peers_sync

section "Workflow ciclo vita reperto"
bash "$SCRIPT_DIR/test_ciclo_vita_reperto.sh"
