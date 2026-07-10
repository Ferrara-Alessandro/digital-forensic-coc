#!/usr/bin/env bash
# Libreria condivisa dagli script in tests/
set -euo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ROOT_DIR="$(cd "$TESTS_DIR/.." && pwd)"
BIN_DIR="$ROOT_DIR/bin"
UPLOAD_BIN="$BIN_DIR/upload"
DOWNLOAD_BIN="$BIN_DIR/download"
RESULTS_DIR="${TEST_RESULTS_DIR:-$TESTS_DIR/results}"

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Manca comando richiesto: $1" >&2
    exit 1
  }
}

require_docker_peer() {
  if ! docker ps --format '{{.Names}}' | grep -qx 'peer0.pg.it'; then
    echo "Rete Fabric non attiva. Avvia prima:" >&2
    echo "  bash scripts/network/bootstrap_rete.sh" >&2
    echo "  bash scripts/network/deploy_lifecycle_reperto.sh" >&2
    exit 1
  fi
}

require_ipfs() {
  if ! curl -sf --max-time 3 -X POST "http://127.0.0.1:5001/api/v0/id" >/dev/null 2>&1; then
    echo "IPFS API non raggiungibile su http://127.0.0.1:5001" >&2
    echo "Avvia il nodo IPFS (es. ipfs daemon) in un terminale dedicato." >&2
    exit 1
  fi
}

build_go_bin() {
  local pkg="$1"
  local out="$2"
  mkdir -p "$BIN_DIR"
  (cd "$ROOT_DIR/$pkg" && go build -o "$out" .)
}

ensure_upload_bin() {
  require_cmd go
  if [[ ! -x "$UPLOAD_BIN" ]]; then
    build_go_bin "cmd/upload" "$UPLOAD_BIN"
  fi
}

ensure_download_bin() {
  require_cmd go
  if [[ ! -x "$DOWNLOAD_BIN" ]]; then
    build_go_bin "cmd/download" "$DOWNLOAD_BIN"
  fi
}

ensure_client_bins() {
  ensure_upload_bin
  ensure_download_bin
}

mkdir_results() {
  mkdir -p "$RESULTS_DIR"
}

section() {
  printf '\n========== %s ==========\n' "$1"
}

fabric_channel_height() {
  local peer_container="$1"
  docker exec "$peer_container" peer channel getinfo -c canale-coc 2>/dev/null \
    | grep -o '"height":[0-9]*' | head -1 | cut -d: -f2
}

require_fabric_peers_sync() {
  local max="${FABRIC_SYNC_TIMEOUT_SEC:-120}"
  local i=0
  local pg_h pm_h lab_h

  while (( i < max )); do
    pg_h="$(fabric_channel_height peer0.pg.it)"
    pm_h="$(fabric_channel_height peer0.pm.it)"
    lab_h="$(fabric_channel_height peer0.lab.it)"
    if [[ -n "$pg_h" && "$pg_h" == "$pm_h" && "$pg_h" == "$lab_h" ]]; then
      return 0
    fi
    sleep 2
    i=$((i + 2))
  done

  echo "Peer Fabric non allineati sul canale canale-coc (PG=${pg_h:-?} PM=${pm_h:-?} LAB=${lab_h:-?})." >&2
  echo "Riavvia la rete e riprova:" >&2
  echo "  bash scripts/network/bootstrap_rete.sh" >&2
  echo "  bash scripts/network/deploy_lifecycle_reperto.sh" >&2
  exit 1
}

# Timeout HTTP IPFS (secondi) proporzionale alla dimensione del blob.
# Il default del client Go (45s) basta solo per file piccoli; evidenze forensi
# (backup telefono, dump disco) richiedono minuti o ore.
bench_ipfs_timeout_for_mb() {
  local size_mb="$1"
  local sec=$((60 + size_mb * 3))
  if (( sec < 120 )); then
    sec=120
  elif (( sec > 7200 )); then
    sec=7200
  fi
  echo "$sec"
}
