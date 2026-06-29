#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

DOWNLOAD_BIN="$ROOT_DIR/bin/download"

log() {
  printf "\n[%s] %s\n" "$(date +"%H:%M:%S")" "$1"
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || { echo "Manca: $1" >&2; exit 1; }
}

build_download_if_missing() {
  if [[ -x "$DOWNLOAD_BIN" ]]; then
    return 0
  fi
  require_cmd go
  log "Compilo cmd/download"
  (cd "$ROOT_DIR/cmd/download" && go build -o "$DOWNLOAD_BIN" .)
}

cd "$ROOT_DIR" || exit 1
build_download_if_missing

read_nonempty ID_CASO "ID caso: "
read_nonempty ID_EVIDENZA "ID evidenza: "

log "Download evidenza $ID_EVIDENZA"
"$DOWNLOAD_BIN" \
  -mode evidenza \
  -id-caso "$ID_CASO" \
  -id-evidenza "$ID_EVIDENZA" \
  -ipfs-api "http://127.0.0.1:5001" \
  -pki "$ROOT_DIR/infrastruttura_blockchain/certificati_pki" \
  -channel canale-coc \
  -chaincode reperto \
  -out-dir "$ROOT_DIR/downloads"

log "File in downloads/"
