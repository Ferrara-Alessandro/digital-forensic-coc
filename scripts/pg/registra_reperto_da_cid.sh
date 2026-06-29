#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/../lib/bash_ui.sh"

CHANNEL_NAME="canale-coc"
CC_NAME="reperto"
ORDERER_ADDR="orderer.example.com:7050"
ORDERER_TLS_CA="/etc/hyperledger/coc-pki/certificati_pki/ordererOrganizations/example.com/orderers/orderer.example.com/tls/ca.crt"

PG_ADMIN_MSP="/etc/hyperledger/coc-pki/certificati_pki/peerOrganizations/pg.it/users/Admin@pg.it/msp"
PG_TLS_CA="/etc/hyperledger/coc-pki/certificati_pki/peerOrganizations/pg.it/peers/peer0.pg.it/tls/ca.crt"
PM_TLS_CA="/etc/hyperledger/coc-pki/certificati_pki/peerOrganizations/pm.it/peers/peer0.pm.it/tls/ca.crt"

log() {
  printf "\n[%s] %s\n" "$(date +"%H:%M:%S")" "$1"
}

invoke_pg_pm() {
  docker exec -e CORE_PEER_MSPCONFIGPATH="$PG_ADMIN_MSP" peer0.pg.it \
    peer chaincode invoke \
    -o "$ORDERER_ADDR" --ordererTLSHostnameOverride orderer.example.com \
    --tls --cafile "$ORDERER_TLS_CA" \
    -C "$CHANNEL_NAME" -n "$CC_NAME" \
    --peerAddresses peer0.pg.it:7051 --tlsRootCertFiles "$PG_TLS_CA" \
    --peerAddresses peer0.pm.it:8051 --tlsRootCertFiles "$PM_TLS_CA" \
    "$@"
}

invoke_pg() {
  docker exec -e CORE_PEER_MSPCONFIGPATH="$PG_ADMIN_MSP" peer0.pg.it \
    peer chaincode invoke \
    -o "$ORDERER_ADDR" --ordererTLSHostnameOverride orderer.example.com \
    --tls --cafile "$ORDERER_TLS_CA" \
    -C "$CHANNEL_NAME" -n "$CC_NAME" \
    --peerAddresses peer0.pg.it:7051 --tlsRootCertFiles "$PG_TLS_CA" \
    "$@"
}

cd "$ROOT_DIR" || {
  echo "Errore: cartella progetto non accessibile: $ROOT_DIR" >&2
  exit 1
}

read_nonempty ID_REPERTO "Inserisci l'ID reperto (chiave univoca): "
read_nonempty ID_CASO "Inserisci l'ID caso: "
read_nonempty ID_AGENTE "Inserisci l'ID agente: "
read_nonempty ID_DISTRETTO "Inserisci l'ID distretto / ufficio (custodia SEQUESTRATO): "
read_nonempty DESCRIZIONE "Inserisci la descrizione sintetica del bene: "

read -r -p "CID IPFS verbale di sequestro [opzionale, invio vuoto = salta]: " CID_VERBALE
read -r -p "CID IPFS evidenza digitale [opzionale, invio vuoto = salta]: " CID_EVIDENZA

read -r -p "Chiave cifrata base64 [opzionale]: " CHIAVE_CIFRATA

read -r -p "Data/ora prelievo UTC ISO [invio = ora UTC]: " DATA_ORA_PRELIEVO
DATA_ORA_PRELIEVO="${DATA_ORA_PRELIEVO:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"

if ! command -v jq >/dev/null 2>&1; then
  echo "Errore: serve jq." >&2
  exit 1
fi

log "Riepilogo (verifica prima di proseguire)"
echo "ID_REPERTO=$ID_REPERTO"
echo "ID_CASO=$ID_CASO"
echo "DATA_ORA_PRELIEVO=$DATA_ORA_PRELIEVO"

log "1) CreaReperto (solo scheda custodia, endorsement PG+PM)"
PRIVATE_JSON="$(jq -cn \
  --arg idCaso "$ID_CASO" \
  --arg idAgente "$ID_AGENTE" \
  --arg idDistretto "$ID_DISTRETTO" \
  --arg dataOraPrelievo "$DATA_ORA_PRELIEVO" \
  --arg descrizioneBene "$DESCRIZIONE" \
  '{idCaso:$idCaso,idAgente:$idAgente,idDistretto:$idDistretto,dataOraPrelievo:$dataOraPrelievo,descrizioneBene:$descrizioneBene}')"
PRIVATE_B64="$(printf '%s' "$PRIVATE_JSON" | base64 | tr -d '\n')"
TRANSIENT_REP="{\"reperto_privato\":\"$PRIVATE_B64\"}"
invoke_pg_pm --transient "$TRANSIENT_REP" \
  -c "{\"function\":\"CreaReperto\",\"Args\":[\"$ID_REPERTO\"]}"

if [[ -n "${CID_VERBALE// }" ]]; then
  log "2) RegistraDocumentoConTransient VERBALE_SEQUESTRO"
  DOC_JSON="$(jq -cn --arg c "$CID_VERBALE" --arg k "$CHIAVE_CIFRATA" '{cid:$c,chiaveCifrata:$k}')"
  DOC_B64="$(printf '%s' "$DOC_JSON" | base64 | tr -d '\n')"
  invoke_pg --transient "{\"documento\":\"$DOC_B64\"}" \
    -c "{\"function\":\"RegistraDocumentoConTransient\",\"Args\":[\"DOC-SEQUESTRO-${ID_REPERTO}\",\"$ID_CASO\",\"VERBALE_SEQUESTRO\",\"$ID_REPERTO\",\"Verbale di sequestro\",\"$ID_DISTRETTO\"]}"
fi

if [[ -n "${CID_EVIDENZA// }" ]]; then
  log "3) RegistraEvidenzaConTransient"
  EVI_JSON="$(jq -cn --arg c "$CID_EVIDENZA" --arg k "$CHIAVE_CIFRATA" '{cid:$c,chiaveCifrata:$k}')"
  EVI_B64="$(printf '%s' "$EVI_JSON" | base64 | tr -d '\n')"
  invoke_pg --transient "{\"evidenza\":\"$EVI_B64\"}" \
    -c "{\"function\":\"RegistraEvidenzaConTransient\",\"Args\":[\"EVI-${ID_REPERTO}\",\"$ID_CASO\",\"$ID_REPERTO\",\"Evidenza digitale del reperto\",\"\"]}"
fi

sleep 2

log "Verifica lettura reperto: $ID_REPERTO"
docker exec -e CORE_PEER_MSPCONFIGPATH="$PG_ADMIN_MSP" peer0.pg.it \
  peer chaincode query -C "$CHANNEL_NAME" -n "$CC_NAME" \
  -c "{\"function\":\"ReadReperto\",\"Args\":[\"$ID_REPERTO\"]}"

echo
log "Operazione completata per il reperto: $ID_REPERTO"
