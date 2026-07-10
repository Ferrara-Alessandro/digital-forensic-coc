#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/../lib/bash_ui.sh"

UPLOAD_BIN="$ROOT_DIR/bin/upload"

log() {
  printf "\n[%s] %s\n" "$(date +"%H:%M:%S")" "$1"
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Errore: comando '$1' non trovato" >&2
    exit 1
  fi
}

build_upload_if_missing() {
  if [[ -x "$UPLOAD_BIN" ]]; then
    return 0
  fi
  require_cmd go
  log "Compilo cmd/upload"
  (cd "$ROOT_DIR/cmd/upload" && go build -o "$UPLOAD_BIN" .)
}

cd "$ROOT_DIR" || exit 1

require_cmd docker
build_upload_if_missing

read_nonempty FILE_INPUT "Percorso file evidenza: "
[[ -f "$FILE_INPUT" ]] || { echo "File non trovato: $FILE_INPUT" >&2; exit 1; }

read_nonempty ID_REPERTO "ID reperto: "
read_nonempty ID_CASO "ID caso: "
read_nonempty ID_AGENTE "ID agente: "
read_nonempty ID_DISTRETTO "ID distretto: "
read_nonempty DESCRIZIONE "Descrizione bene: "

read -r -p "IPFS API [http://127.0.0.1:5001]: " IPFS_API
IPFS_API="${IPFS_API:-http://127.0.0.1:5001}"

ID_EVIDENZA="EVI-${ID_REPERTO}"

log "CreaReperto (scheda custodia)"
"$UPLOAD_BIN" \
  -mode reperto \
  -id-reperto "$ID_REPERTO" \
  -id-caso "$ID_CASO" \
  -id-agente "$ID_AGENTE" \
  -id-distretto "$ID_DISTRETTO" \
  -descrizione-bene "$DESCRIZIONE" \
  -pki "$ROOT_DIR/infrastruttura_blockchain/certificati_pki" \
  -channel canale-coc \
  -chaincode reperto

log "RegistraEvidenza (file cifrato su IPFS)"
"$UPLOAD_BIN" \
  -mode evidenza \
  -file "$FILE_INPUT" \
  -id-evidenza "$ID_EVIDENZA" \
  -id-caso "$ID_CASO" \
  -id-reperto-evidenza "$ID_REPERTO" \
  -descrizione-evidenza "Evidenza reperto" \
  -ipfs-api "$IPFS_API" \
  -pki "$ROOT_DIR/infrastruttura_blockchain/certificati_pki" \
  -channel canale-coc \
  -chaincode reperto

echo
log "Completato: reperto=$ID_REPERTO evidenza=$ID_EVIDENZA"
