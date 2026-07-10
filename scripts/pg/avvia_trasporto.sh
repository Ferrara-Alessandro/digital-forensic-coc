#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/../lib/bash_ui.sh"

cd "$ROOT_DIR" || {
  echo "Errore: cartella progetto non accessibile: $ROOT_DIR" >&2
  exit 1
}

CHANNEL_NAME="${CHANNEL_NAME:-canale-coc}"
CC_NAME="${CC_NAME:-reperto}"
ORDERER_ADDR="orderer.example.com:7050"
ORDERER_TLS_CA="/etc/hyperledger/coc-pki/certificati_pki/ordererOrganizations/example.com/orderers/orderer.example.com/tls/ca.crt"
PG_ADMIN_MSP="/etc/hyperledger/coc-pki/certificati_pki/peerOrganizations/pg.it/users/Admin@pg.it/msp"
PG_TLS_CA="/etc/hyperledger/coc-pki/certificati_pki/peerOrganizations/pg.it/peers/peer0.pg.it/tls/ca.crt"


read_nonempty ID_REPERTO "Inserisci l'ID del reperto: "
read_nonempty ID_AGENTE "Inserisci l'ID dell'agente PG (custode in transito): "

echo "Operazione avviata per il reperto: $ID_REPERTO (agente $ID_AGENTE)..."

docker exec -e CORE_PEER_MSPCONFIGPATH="$PG_ADMIN_MSP" peer0.pg.it \
  peer chaincode invoke \
  -o "$ORDERER_ADDR" \
  --ordererTLSHostnameOverride orderer.example.com \
  --tls \
  --cafile "$ORDERER_TLS_CA" \
  -C "$CHANNEL_NAME" \
  -n "$CC_NAME" \
  --peerAddresses peer0.pg.it:7051 \
  --tlsRootCertFiles "$PG_TLS_CA" \
  -c "{\"function\":\"AvviaTrasporto\",\"Args\":[\"$ID_REPERTO\",\"$ID_AGENTE\"]}"

echo "Operazione completata per il reperto: $ID_REPERTO"
