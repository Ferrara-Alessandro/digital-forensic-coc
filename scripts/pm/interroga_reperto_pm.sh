#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/../lib/bash_ui.sh"

CHANNEL_NAME="canale-coc"
CC_NAME="reperto"
PM_ADMIN_MSP="/etc/hyperledger/coc-pki/certificati_pki/peerOrganizations/pm.it/users/Admin@pm.it/msp"


cd "$ROOT_DIR" || {
  echo "Errore: cartella progetto non accessibile: $ROOT_DIR" >&2
  exit 1
}

read_nonempty ID_REPERTO "Inserisci l'ID del reperto da interrogare: "

echo "Query avviata per il reperto: $ID_REPERTO (MSP PMMSP)..."

QUERY_OUTPUT="$(docker exec \
  -e CORE_PEER_MSPCONFIGPATH="$PM_ADMIN_MSP" \
  peer0.pm.it \
  peer chaincode query \
  -C "$CHANNEL_NAME" \
  -n "$CC_NAME" \
  -c "{\"function\":\"ReadReperto\",\"Args\":[\"$ID_REPERTO\"]}")"

echo
echo "Risultato per il reperto $ID_REPERTO:"

if command -v jq >/dev/null 2>&1; then
  if echo "$QUERY_OUTPUT" | jq . >/dev/null 2>&1; then
    echo "$QUERY_OUTPUT" | jq .
    exit 0
  fi
fi

if command -v python3 >/dev/null 2>&1; then
  if echo "$QUERY_OUTPUT" | python3 -m json.tool >/dev/null 2>&1; then
    echo "$QUERY_OUTPUT" | python3 -m json.tool
    exit 0
  fi
fi

echo "$QUERY_OUTPUT"
