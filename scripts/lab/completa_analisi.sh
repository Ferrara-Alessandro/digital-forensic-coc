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
LAB_ADMIN_MSP="/etc/hyperledger/coc-pki/certificati_pki/peerOrganizations/lab.it/users/Admin@lab.it/msp"
LAB_TLS_CA="/etc/hyperledger/coc-pki/certificati_pki/peerOrganizations/lab.it/peers/peer0.lab.it/tls/ca.crt"
PG_TLS_CA="/etc/hyperledger/coc-pki/certificati_pki/peerOrganizations/pg.it/peers/peer0.pg.it/tls/ca.crt"


read_nonempty ID_REPERTO "Inserisci l'ID del reperto: "
read_nonempty CID_RELAZIONE "Inserisci il CID della relazione tecnica: "
read_nonempty CHIAVE_RELAZIONE "Inserisci la chiave di decifratura della relazione (base64): "

echo "Operazione avviata per il reperto: $ID_REPERTO (relazione tecnica LAB+PG)..."

docker exec -e CORE_PEER_MSPCONFIGPATH="$LAB_ADMIN_MSP" peer0.lab.it \
  peer chaincode invoke \
  -o "$ORDERER_ADDR" \
  --ordererTLSHostnameOverride orderer.example.com \
  --tls \
  --cafile "$ORDERER_TLS_CA" \
  -C "$CHANNEL_NAME" \
  -n "$CC_NAME" \
  --peerAddresses peer0.lab.it:9051 \
  --tlsRootCertFiles "$LAB_TLS_CA" \
  --peerAddresses peer0.pg.it:7051 \
  --tlsRootCertFiles "$PG_TLS_CA" \
  --waitForEvent \
  --waitForEventTimeout 60s \
  -c "{\"function\":\"CompletaAnalisi\",\"Args\":[\"$ID_REPERTO\",\"$CID_RELAZIONE\",\"$CHIAVE_RELAZIONE\"]}"

echo "Operazione completata per il reperto: $ID_REPERTO"
