#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

ORDERER_COMPOSE="$ROOT_DIR/infrastruttura_blockchain/avvio_nodi.yaml"
PEER_COMPOSE="$ROOT_DIR/infrastruttura_blockchain/peer_nodi.yaml"

log() {
  printf "\n[%s] %s\n" "$(date +"%H:%M:%S")" "$1"
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Errore: comando '$1' non trovato" >&2
    exit 1
  fi
}

main() {
  require_cmd docker
  cd "$ROOT_DIR" || {
    echo "Errore: cartella progetto non accessibile: $ROOT_DIR" >&2
    exit 1
  }

  log "Stop peer"
  (cd "$ROOT_DIR/infrastruttura_blockchain" && docker compose -f "$PEER_COMPOSE" down)

  log "Stop orderer"
  (cd "$ROOT_DIR/infrastruttura_blockchain" && docker compose -f "$ORDERER_COMPOSE" down)

  log "Verifica finale (nessun container Fabric attivo)"
  docker ps --format '{{.Names}}' | grep -E 'orderer.example.com|peer0.pg.it|peer0.pm.it|peer0.lab.it' || true

  log "Rete spenta"
}

main "$@"
