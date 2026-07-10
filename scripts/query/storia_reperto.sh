#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/../lib/bash_ui.sh"

CHANNEL_NAME="canale-coc"
CC_NAME="reperto"
PIKI="/etc/hyperledger/coc-pki"
PG_ADMIN_MSP="${PIKI}/certificati_pki/peerOrganizations/pg.it/users/Admin@pg.it/msp"


cd "$ROOT_DIR" || {
  echo "Errore: cartella progetto non accessibile: $ROOT_DIR" >&2
  exit 1
}

read_nonempty ID_REPERTO "Inserisci l'ID del reperto: "

while true; do
  read -r -p "Peer per la query (pg / pm / lab) [pg]: " ORG
  ORG="${ORG,,}"
  ORG="${ORG:-pg}"
  case "$ORG" in
    pg|pm|lab) break ;;
    *) echo "Errore: usa pg, pm o lab." >&2 ;;
  esac
done

echo "Query storia avviata per il reperto: $ID_REPERTO (peer $ORG)..."

_OUT=""
case "$ORG" in
  pg)
    _OUT="$(docker exec -e CORE_PEER_MSPCONFIGPATH="$PG_ADMIN_MSP" peer0.pg.it \
      peer chaincode query -C "$CHANNEL_NAME" -n "$CC_NAME" \
      -c "{\"function\":\"OttieniStoriaReperto\",\"Args\":[\"$ID_REPERTO\"]}")"
    ;;
  pm)
    PM_ADMIN="${PIKI}/certificati_pki/peerOrganizations/pm.it/users/Admin@pm.it/msp"
    _OUT="$(docker exec -e CORE_PEER_MSPCONFIGPATH="$PM_ADMIN" peer0.pm.it \
      peer chaincode query -C "$CHANNEL_NAME" -n "$CC_NAME" \
      -c "{\"function\":\"OttieniStoriaReperto\",\"Args\":[\"$ID_REPERTO\"]}")"
    ;;
  lab)
    LAB_ADMIN="${PIKI}/certificati_pki/peerOrganizations/lab.it/users/Admin@lab.it/msp"
    _OUT="$(docker exec -e CORE_PEER_MSPCONFIGPATH="$LAB_ADMIN" peer0.lab.it \
      peer chaincode query -C "$CHANNEL_NAME" -n "$CC_NAME" \
      -c "{\"function\":\"OttieniStoriaReperto\",\"Args\":[\"$ID_REPERTO\"]}")"
    ;;
esac

if command -v jq >/dev/null 2>&1; then
  if printf '%s' "$_OUT" | jq . >/dev/null 2>&1; then
    printf '%s\n' "$_OUT" | jq .
  elif raw="$(printf '%s' "$_OUT" | jq -r . 2>/dev/null)" && printf '%s' "$raw" | jq . >/dev/null 2>&1; then
    printf '%s\n' "$raw" | jq .
  else
    printf '%s\n' "$_OUT"
  fi
else
  printf '%s\n' "$_OUT"
fi
