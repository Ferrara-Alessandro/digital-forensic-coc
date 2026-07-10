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
PM_ADMIN_MSP="/etc/hyperledger/coc-pki/certificati_pki/peerOrganizations/pm.it/users/Admin@pm.it/msp"
PM_TLS_CA="/etc/hyperledger/coc-pki/certificati_pki/peerOrganizations/pm.it/peers/peer0.pm.it/tls/ca.crt"


read_nonempty ID_REPERTO "Inserisci l'ID del reperto: "
read_nonempty TIPO_ANALISI "Tipo analisi richiesta: "
read_nonempty CID_DECRETO "Inserisci il CID del decreto: "
read_nonempty CHIAVE_DECRETO "Inserisci la chiave di decifratura del decreto (base64): "
read_nonempty ID_LAB "Inserisci l'ID del laboratorio di destinazione: "

echo "Operazione avviata per il reperto: $ID_REPERTO (decreto → lab $ID_LAB)..."

docker exec -e CORE_PEER_MSPCONFIGPATH="$PM_ADMIN_MSP" peer0.pm.it \
  peer chaincode invoke \
  -o "$ORDERER_ADDR" \
  --ordererTLSHostnameOverride orderer.example.com \
  --tls \
  --cafile "$ORDERER_TLS_CA" \
  -C "$CHANNEL_NAME" \
  -n "$CC_NAME" \
  --peerAddresses peer0.pm.it:8051 \
  --tlsRootCertFiles "$PM_TLS_CA" \
  -c "{\"function\":\"RichiediAnalisi\",\"Args\":[\"$ID_REPERTO\",\"$ID_LAB\",\"$TIPO_ANALISI\",\"$CID_DECRETO\",\"$CHIAVE_DECRETO\"]}"

echo "Operazione completata per il reperto: $ID_REPERTO"
