#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
INFRA="$ROOT_DIR/infrastruttura_blockchain"
TOOLS_IMAGE="${TOOLS_IMAGE:-hyperledger/fabric-tools:2.5.15}"
CHANNEL_ID="${CHANNEL_ID:-canale-coc}"
CREATE_TX_NAME="canal_crea.pb"
PIKI="/etc/hyperledger/coc-pki"

ORDERER_TLS_CA="${PIKI}/certificati_pki/ordererOrganizations/example.com/orderers/orderer.example.com/tls/ca.crt"

log() { printf "\n[%s] %s\n" "$(date +"%H:%M:%S")" "$1"; }

main() {
  cd "$ROOT_DIR" || {
    echo "Errore: cartella progetto non accessibile: $ROOT_DIR" >&2
    exit 1
  }
  log "configtxgen ($CHANNEL_ID, profilo CocChannel)"
  docker run --rm \
    -v "$INFRA:/work" -w /work \
    -e FABRIC_CFG_PATH=/work \
    "$TOOLS_IMAGE" \
    configtxgen -profile CocChannel -outputCreateChannelTx "/work/$CREATE_TX_NAME" -channelID "$CHANNEL_ID"

  log "peer channel create -> canale-coc.block (orderer)"
  set +e
  _out="$(docker run --rm --network rete-coc \
    -v "$INFRA:$PIKI" \
    -e CORE_PEER_LOCALMSPID=PGMSP \
    -e CORE_PEER_MSPCONFIGPATH="$PIKI/certificati_pki/peerOrganizations/pg.it/users/Admin@pg.it/msp" \
    -e CORE_PEER_TLS_ENABLED=true \
    -e CORE_PEER_TLS_ROOTCERT_FILE="$PIKI/certificati_pki/peerOrganizations/pg.it/peers/peer0.pg.it/tls/ca.crt" \
    "$TOOLS_IMAGE" \
    peer channel create \
    -o orderer.example.com:7050 \
    --ordererTLSHostnameOverride orderer.example.com \
    -c "$CHANNEL_ID" \
    -f "$PIKI/$CREATE_TX_NAME" \
    --outputBlock "$PIKI/canale-coc.block" \
    --tls \
    --cafile "$ORDERER_TLS_CA" 2>&1)"
  _rc=$?
  set -e
  echo "$_out"
  if [[ $_rc -eq 0 ]]; then
    log "Canale $CHANNEL_ID creato; canale-coc.block scritto."
    return 0
  fi
  if echo "$_out" | grep -qE 'error applying config update to existing channel|BAD_REQUEST'; then
    log "Canale $CHANNEL_ID gia' presente su orderer; scarico blocco 0 se serve"
    if [[ ! -f "$INFRA/canale-coc.block" ]]; then
      docker run --rm --network rete-coc \
        -v "$INFRA:$PIKI" \
        -e CORE_PEER_LOCALMSPID=PGMSP \
        -e CORE_PEER_MSPCONFIGPATH="$PIKI/certificati_pki/peerOrganizations/pg.it/users/Admin@pg.it/msp" \
        -e CORE_PEER_TLS_ENABLED=true \
        -e CORE_PEER_TLS_ROOTCERT_FILE="$PIKI/certificati_pki/peerOrganizations/pg.it/peers/peer0.pg.it/tls/ca.crt" \
        "$TOOLS_IMAGE" \
        peer channel fetch oldest "$PIKI/canale-coc.block" \
        -c "$CHANNEL_ID" \
        -o orderer.example.com:7050 \
        --ordererTLSHostnameOverride orderer.example.com \
        --tls \
        --cafile "$ORDERER_TLS_CA"
      log "canale-coc.block ripristinato da orderer"
    fi
    return 0
  fi
  return 1
}

main "$@"
