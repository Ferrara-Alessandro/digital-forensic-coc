#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
INFRA_DIR="$ROOT_DIR/infrastruttura_blockchain"

CC_NAME="${CC_NAME:-reperto}"
CC_VERSION="${CC_VERSION:-5.5}"
CC_SEQUENCE="${CC_SEQUENCE:-1}"
CHANNEL_ID="${CHANNEL_ID:-canale-coc}"
TOOLS_IMAGE="${TOOLS_IMAGE:-hyperledger/fabric-tools:2.5.15}"
# Almeno un endorser tra PG, PM, LAB (necessario per transazioni invocabili dal solo LAB o da una singola org).
ENDORSER_POLICY="${ENDORSER_POLICY:-"OR('PGMSP.peer','PMMSP.peer','LABMSP.peer')"}"

TAR_NAME="reperto_${CC_VERSION}.tar.gz"
PACKAGE_HOST="$INFRA_DIR/$TAR_NAME"

# Percorsi nei container (volume: infrastruttura_blockchain -> /etc/hyperledger/coc-pki)
PIKI="/etc/hyperledger/coc-pki"
ORDERER_ADDR="orderer.example.com:7050"
ORDERER_TLS_CA="${PIKI}/certificati_pki/ordererOrganizations/example.com/orderers/orderer.example.com/tls/ca.crt"
COLLECT_JSON="${PIKI}/collections_config.json"
PACKAGE_IN_CONTAINER="${PIKI}/$TAR_NAME"

log() { printf "\n[%s] %s\n" "$(date +"%H:%M:%S")" "$1"; }
require_cmd() { command -v "$1" >/dev/null 2>&1 || { echo "Manca: $1" >&2; exit 1; }; }

extract_pkg_id() {
  local out="$1" id
  out="$(printf '%s' "$out" | sed 's/\x1b\[[0-9;]*m//g')"
  id="$(printf '%s' "$out" | grep -oE 'reperto_[^:[:space:]]+:[a-f0-9]{64}' | head -1)"
  if [[ -n "$id" ]]; then
    printf '%s' "$id"
    return 0
  fi
  id="$(printf '%s' "$out" | sed -n "s/.*package ID '\([^']*\)'.*/\1/p" | head -1)"
  if [[ -n "$id" ]]; then
    printf '%s' "$id"
    return 0
  fi
  id="$(printf '%s' "$out" | sed -n 's/^Chaincode code package identifier:[[:space:]]*//p' | tr -d '\r' | head -1)"
  [[ -n "$id" ]] && printf '%s' "$id" && return 0
  return 1
}

# PACKAGE_ID dallo strumento queryinstalled (dopo gia' install o tar identico)
query_pkg_id() {
  local j
  j="$(docker exec -e CORE_PEER_LOCALMSPID=PGMSP \
    -e CORE_PEER_MSPCONFIGPATH="${PIKI}/certificati_pki/peerOrganizations/pg.it/users/Admin@pg.it/msp" \
    -e CORE_PEER_ADDRESS=peer0.pg.it:7051 \
    peer0.pg.it peer lifecycle chaincode queryinstalled --output json 2>/dev/null)" || return 1
  if command -v jq >/dev/null 2>&1; then
    jq -r --arg l "${CC_NAME}_${CC_VERSION}" '.installed_chaincodes[]? | select(.label==$l) | .package_id' <<<"$j" 2>/dev/null | head -1
  else
    # fallback grezzo: prima etichetta che contiene "reperto" e 64 char hex
    echo "$j" | grep -oE 'reperto_[^"[:space:]]+:[a-f0-9]{64}' | head -1
  fi
}

cc_install() {
  local c="$1" mspid="$2" msp_path="$3" addr="$4"
  docker exec \
    -e CORE_PEER_LOCALMSPID="$mspid" \
    -e CORE_PEER_MSPCONFIGPATH="$msp_path" \
    -e CORE_PEER_ADDRESS="$addr" \
    -e CORE_PEER_TLS_ENABLED=true \
    "$c" \
    peer lifecycle chaincode install "$PACKAGE_IN_CONTAINER"
}

cc_approve() {
  local c="$1" mspid="$2" msp_path="$3" addr="$4" peer_tls_ca="$5"
  docker exec \
    -e CORE_PEER_LOCALMSPID="$mspid" \
    -e CORE_PEER_MSPCONFIGPATH="$msp_path" \
    -e CORE_PEER_ADDRESS="$addr" \
    -e CORE_PEER_TLS_ENABLED=true \
    -e CORE_PEER_TLS_ROOTCERT_FILE="$peer_tls_ca" \
    "$c" \
    peer lifecycle chaincode approveformyorg \
    -o "$ORDERER_ADDR" \
    --ordererTLSHostnameOverride orderer.example.com \
    --tls \
    --cafile "$ORDERER_TLS_CA" \
    --channelID "$CHANNEL_ID" \
    --name "$CC_NAME" \
    --version "$CC_VERSION" \
    --package-id "$PACKAGE_ID" \
    --sequence "$CC_SEQUENCE" \
    --signature-policy "$ENDORSER_POLICY" \
    --collections-config "$COLLECT_JSON"
}

cc_commit() {
  docker exec \
    -e CORE_PEER_LOCALMSPID=PGMSP \
    -e CORE_PEER_MSPCONFIGPATH="${PIKI}/certificati_pki/peerOrganizations/pg.it/users/Admin@pg.it/msp" \
    -e CORE_PEER_ADDRESS=peer0.pg.it:7051 \
    -e CORE_PEER_TLS_ENABLED=true \
    -e CORE_PEER_TLS_ROOTCERT_FILE="${PIKI}/certificati_pki/peerOrganizations/pg.it/peers/peer0.pg.it/tls/ca.crt" \
    peer0.pg.it \
    peer lifecycle chaincode commit \
    -o "$ORDERER_ADDR" \
    --ordererTLSHostnameOverride orderer.example.com \
    --tls \
    --cafile "$ORDERER_TLS_CA" \
    --channelID "$CHANNEL_ID" \
    --name "$CC_NAME" \
    --version "$CC_VERSION" \
    --sequence "$CC_SEQUENCE" \
    --peerAddresses "peer0.pg.it:7051" \
    --tlsRootCertFiles "${PIKI}/certificati_pki/peerOrganizations/pg.it/peers/peer0.pg.it/tls/ca.crt" \
    --peerAddresses "peer0.pm.it:8051" \
    --tlsRootCertFiles "${PIKI}/certificati_pki/peerOrganizations/pm.it/peers/peer0.pm.it/tls/ca.crt" \
    --peerAddresses "peer0.lab.it:9051" \
    --tlsRootCertFiles "${PIKI}/certificati_pki/peerOrganizations/lab.it/peers/peer0.lab.it/tls/ca.crt" \
    --signature-policy "$ENDORSER_POLICY" \
    --collections-config "$COLLECT_JSON"
}

main() {
  require_cmd docker
  cd "$ROOT_DIR" || {
    echo "Errore: cartella progetto non accessibile: $ROOT_DIR" >&2
    exit 1
  }
  if ! docker network inspect rete-coc >/dev/null 2>&1; then
    echo "Crea/usa rete rete-coc (es. bootstrap) prima del deploy." >&2
    exit 1
  fi
  for c in orderer.example.com peer0.pg.it peer0.pm.it peer0.lab.it; do
    if ! docker ps --format '{{.Names}}' | grep -qx "$c"; then
      echo "Container $c assente. Avviare i compose in infrastruttura_blockchain." >&2
      exit 1
    fi
  done
  [[ -f "$ROOT_DIR/chaincode/go.mod" ]] || { echo "Manca $ROOT_DIR/chaincode" >&2; exit 1; }
  [[ -f "$INFRA_DIR/collections_config.json" ]] || { echo "Manca $INFRA_DIR/collections_config.json" >&2; exit 1; }

  log "Verifica: orderer ordina su $CHANNEL_ID (evita 'channel does not exist' in lifecycle)"
  set +e
  _pf_err="$(docker exec -e CORE_PEER_LOCALMSPID=PGMSP \
    -e CORE_PEER_MSPCONFIGPATH="${PIKI}/certificati_pki/peerOrganizations/pg.it/users/Admin@pg.it/msp" \
    -e CORE_PEER_ADDRESS=peer0.pg.it:7051 \
    peer0.pg.it peer channel fetch config /tmp/preflight_channel.block \
    -c "$CHANNEL_ID" \
    -o orderer.example.com:7050 \
    --ordererTLSHostnameOverride orderer.example.com \
    --tls --cafile "$ORDERER_TLS_CA" 2>&1)"
  _pf_rc=$?
  set -e
  if [[ $_pf_rc -ne 0 ]]; then
    echo "Impossibile leggere l'ultimo blocco di configurazione del canale da orderer+canale $CHANNEL_ID." >&2
    echo "Causa tipica: dati orderer azzerati o can mai creato, mentre i peer hanno ancora $CHANNEL_ID in locale (incoerenza)." >&2
    echo "Esempio: docker compose + volumi, poi rifare 'peer channel create' o ripristinare genesis+canale coerenti." >&2
    echo "$_pf_err" >&2
    exit 1
  fi

  log "Package -> $TAR_NAME (image $TOOLS_IMAGE)"
  docker run --rm \
    --network rete-coc \
    -v "$ROOT_DIR:/work" \
    -w /work/chaincode \
    "$TOOLS_IMAGE" \
    peer lifecycle chaincode package \
    "/work/infrastruttura_blockchain/$TAR_NAME" \
    --path /work/chaincode \
    --lang golang \
    --label "${CC_NAME}_${CC_VERSION}"

  log "Rimuovo container chaincode dev (dev-peer*) per ridurre carico su Docker durante la build"
  docker ps -q --filter "name=dev-peer" | xargs -r docker rm -f || true
  sleep 2

  log "Install PG"
  set +e
  INSTALL_OUT="$(cc_install peer0.pg.it PGMSP "${PIKI}/certificati_pki/peerOrganizations/pg.it/users/Admin@pg.it/msp" peer0.pg.it:7051 2>&1)"
  INSTALL_RC=$?
  set -e
  echo "$INSTALL_OUT"
  PACKAGE_ID="$(extract_pkg_id "$INSTALL_OUT" || true)"
  if [[ -z "$PACKAGE_ID" ]]; then
    PACKAGE_ID="$(query_pkg_id || true)"
  fi
  if [[ -z "$PACKAGE_ID" ]]; then
    if [[ $INSTALL_RC -ne 0 ]]; then
      echo "Install fallita e impossibile ricavare PACKAGE_ID." >&2
      exit 1
    fi
    echo "Install riuscita ma nessun PACKAGE_ID riconosciuto." >&2
    exit 1
  fi
  if [[ $INSTALL_RC -ne 0 ]] && echo "$INSTALL_OUT" | grep -qF "chaincode already successfully installed"; then
    log "Install PG: pacchetto gia' presente sul peer (ok)"
  fi
  log "PACKAGE_ID=$PACKAGE_ID"

  log "Install PM + LAB (stesso pacchetto)"
  set +e
  _out="$(cc_install peer0.pm.it PMMSP "${PIKI}/certificati_pki/peerOrganizations/pm.it/users/Admin@pm.it/msp" peer0.pm.it:8051 2>&1)"; _rc=$?; set -e
  echo "$_out"
  if [[ $_rc -ne 0 ]] && ! echo "$_out" | grep -qF "chaincode already successfully installed"; then exit 1; fi
  set +e
  _out="$(cc_install peer0.lab.it LABMSP "${PIKI}/certificati_pki/peerOrganizations/lab.it/users/Admin@lab.it/msp" peer0.lab.it:9051 2>&1)"; _rc=$?; set -e
  echo "$_out"
  if [[ $_rc -ne 0 ]] && ! echo "$_out" | grep -qF "chaincode already successfully installed"; then exit 1; fi

  log "Verifica definizione gia' committata (deploy ripetuto)"
  set +e
  _qc="$(docker exec -e CORE_PEER_LOCALMSPID=PGMSP \
    -e CORE_PEER_MSPCONFIGPATH="${PIKI}/certificati_pki/peerOrganizations/pg.it/users/Admin@pg.it/msp" \
    -e CORE_PEER_ADDRESS=peer0.pg.it:7051 \
    -e CORE_PEER_TLS_ENABLED=true \
    -e CORE_PEER_TLS_ROOTCERT_FILE="${PIKI}/certificati_pki/peerOrganizations/pg.it/peers/peer0.pg.it/tls/ca.crt" \
    peer0.pg.it peer lifecycle chaincode querycommitted -C "$CHANNEL_ID" --name "$CC_NAME" 2>&1)"
  _qc_rc=$?
  set -e
  if [[ $_qc_rc -eq 0 ]] && echo "$_qc" | grep -qF "chaincode '${CC_NAME}'" && \
     echo "$_qc" | grep -qF "Version: ${CC_VERSION}," && \
     echo "$_qc" | grep -qF "Sequence: ${CC_SEQUENCE},"; then
    log "Definizione lifecycle gia' presente: $CC_NAME $CC_VERSION seq=$CC_SEQUENCE su $CHANNEL_ID. Uscita senza approve/commit."
    exit 0
  fi

  log "Approveformyorg: PG, PM, LAB"
  cc_approve peer0.pg.it PGMSP "${PIKI}/certificati_pki/peerOrganizations/pg.it/users/Admin@pg.it/msp" \
    peer0.pg.it:7051 "${PIKI}/certificati_pki/peerOrganizations/pg.it/peers/peer0.pg.it/tls/ca.crt"
  cc_approve peer0.pm.it PMMSP "${PIKI}/certificati_pki/peerOrganizations/pm.it/users/Admin@pm.it/msp" \
    peer0.pm.it:8051 "${PIKI}/certificati_pki/peerOrganizations/pm.it/peers/peer0.pm.it/tls/ca.crt"
  cc_approve peer0.lab.it LABMSP "${PIKI}/certificati_pki/peerOrganizations/lab.it/users/Admin@lab.it/msp" \
    peer0.lab.it:9051 "${PIKI}/certificati_pki/peerOrganizations/lab.it/peers/peer0.lab.it/tls/ca.crt"

  log "Commit (PG cli)"
  cc_commit

  log "Verifica querycommitted (PG Admin)"
  docker exec -e CORE_PEER_LOCALMSPID=PGMSP \
    -e CORE_PEER_MSPCONFIGPATH="${PIKI}/certificati_pki/peerOrganizations/pg.it/users/Admin@pg.it/msp" \
    -e CORE_PEER_ADDRESS=peer0.pg.it:7051 \
    -e CORE_PEER_TLS_ENABLED=true \
    -e CORE_PEER_TLS_ROOTCERT_FILE="${PIKI}/certificati_pki/peerOrganizations/pg.it/peers/peer0.pg.it/tls/ca.crt" \
    peer0.pg.it peer lifecycle chaincode querycommitted -C "$CHANNEL_ID"

  log "Deploy completato: $CC_NAME@$CC_VERSION seq=$CC_SEQUENCE"
}

main "$@"
