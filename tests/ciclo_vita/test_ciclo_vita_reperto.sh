#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

cd "$ROOT_DIR" || {
  echo "Errore: cartella progetto non accessibile: $ROOT_DIR" >&2
  exit 1
}

require_cmd() { command -v "$1" >/dev/null 2>&1 || { echo "Manca: $1" >&2; exit 1; }; }
require_cmd docker
require_cmd jq

CHANNEL_NAME="${CHANNEL_NAME:-canale-coc}"
CC_NAME="${CC_NAME:-reperto}"
ORDERER_ADDR="orderer.example.com:7050"
PIKI="/etc/hyperledger/coc-pki"
ORDERER_TLS_CA="${PIKI}/certificati_pki/ordererOrganizations/example.com/orderers/orderer.example.com/tls/ca.crt"

PG_ADMIN_MSP="${PIKI}/certificati_pki/peerOrganizations/pg.it/users/Admin@pg.it/msp"
PM_ADMIN_MSP="${PIKI}/certificati_pki/peerOrganizations/pm.it/users/Admin@pm.it/msp"
LAB_ADMIN_MSP="${PIKI}/certificati_pki/peerOrganizations/lab.it/users/Admin@lab.it/msp"

PG_TLS_CA="${PIKI}/certificati_pki/peerOrganizations/pg.it/peers/peer0.pg.it/tls/ca.crt"
PM_TLS_CA="${PIKI}/certificati_pki/peerOrganizations/pm.it/peers/peer0.pm.it/tls/ca.crt"
LAB_TLS_CA="${PIKI}/certificati_pki/peerOrganizations/lab.it/peers/peer0.lab.it/tls/ca.crt"

RID="${ID_REPERTO:-REP-WF-$(date +%s)}"
CID_BASE="bafybeigdyrzt3s3fk7fsfpksmwga3rprr7a3x6z7x7a3q6q6q6q6q6q6q6q"
# Chiave AES-256 di test (base64), stessa usata da cmd/upload dopo cifratura file
CHIAVE_TEST_B64="$(printf '%s' '01234567890123456789012345678901' | base64 | tr -d '\n')"

log() { printf "\n[%s] %s\n" "$(date +"%H:%M:%S")" "$1"; }

invoke_pg() {
  docker exec -e CORE_PEER_MSPCONFIGPATH="$PG_ADMIN_MSP" peer0.pg.it \
    peer chaincode invoke \
    -o "$ORDERER_ADDR" --ordererTLSHostnameOverride orderer.example.com \
    --tls --cafile "$ORDERER_TLS_CA" \
    -C "$CHANNEL_NAME" -n "$CC_NAME" \
    --peerAddresses "peer0.pg.it:7051" --tlsRootCertFiles "$PG_TLS_CA" \
    --waitForEvent --waitForEventTimeout 120s \
    "$@"
}

invoke_pm() {
  docker exec -e CORE_PEER_MSPCONFIGPATH="$PM_ADMIN_MSP" peer0.pm.it \
    peer chaincode invoke \
    -o "$ORDERER_ADDR" --ordererTLSHostnameOverride orderer.example.com \
    --tls --cafile "$ORDERER_TLS_CA" \
    -C "$CHANNEL_NAME" -n "$CC_NAME" \
    --peerAddresses "peer0.pm.it:8051" --tlsRootCertFiles "$PM_TLS_CA" \
    --waitForEvent --waitForEventTimeout 120s \
    "$@"
}

invoke_lab() {
  docker exec -e CORE_PEER_MSPCONFIGPATH="$LAB_ADMIN_MSP" peer0.lab.it \
    peer chaincode invoke \
    -o "$ORDERER_ADDR" --ordererTLSHostnameOverride orderer.example.com \
    --tls --cafile "$ORDERER_TLS_CA" \
    -C "$CHANNEL_NAME" -n "$CC_NAME" \
    --peerAddresses "peer0.lab.it:9051" --tlsRootCertFiles "$LAB_TLS_CA" \
    --waitForEvent --waitForEventTimeout 60s \
    "$@"
}

invoke_lab_with_pg() {
  docker exec -e CORE_PEER_MSPCONFIGPATH="$LAB_ADMIN_MSP" peer0.lab.it \
    peer chaincode invoke \
    -o "$ORDERER_ADDR" --ordererTLSHostnameOverride orderer.example.com \
    --tls --cafile "$ORDERER_TLS_CA" \
    -C "$CHANNEL_NAME" -n "$CC_NAME" \
    --peerAddresses "peer0.lab.it:9051" --tlsRootCertFiles "$LAB_TLS_CA" \
    --peerAddresses "peer0.pg.it:7051" --tlsRootCertFiles "$PG_TLS_CA" \
    --waitForEvent --waitForEventTimeout 60s \
    "$@"
}

invoke_pg_pm() {
  docker exec -e CORE_PEER_MSPCONFIGPATH="$PG_ADMIN_MSP" peer0.pg.it \
    peer chaincode invoke \
    -o "$ORDERER_ADDR" --ordererTLSHostnameOverride orderer.example.com \
    --tls --cafile "$ORDERER_TLS_CA" \
    -C "$CHANNEL_NAME" -n "$CC_NAME" \
    --peerAddresses "peer0.pg.it:7051" --tlsRootCertFiles "$PG_TLS_CA" \
    --peerAddresses "peer0.pm.it:8051" --tlsRootCertFiles "$PM_TLS_CA" \
    --waitForEvent --waitForEventTimeout 120s \
    "$@"
}

invoke_create_reperto_transient() {
  invoke_pg_pm "$@"
}

query_pg() {
  docker exec -e CORE_PEER_LOCALMSPID=PGMSP \
    -e CORE_PEER_MSPCONFIGPATH="$PG_ADMIN_MSP" \
    -e CORE_PEER_ADDRESS=peer0.pg.it:7051 \
    peer0.pg.it \
    peer chaincode query -C "$CHANNEL_NAME" -n "$CC_NAME" -c "$1"
}

query_lab() {
  docker exec -e CORE_PEER_LOCALMSPID=LABMSP \
    -e CORE_PEER_MSPCONFIGPATH="$LAB_ADMIN_MSP" \
    -e CORE_PEER_ADDRESS=peer0.lab.it:9051 \
    peer0.lab.it \
    peer chaincode query -C "$CHANNEL_NAME" -n "$CC_NAME" -c "$1"
}

query_pm() {
  docker exec -e CORE_PEER_LOCALMSPID=PMMSP \
    -e CORE_PEER_MSPCONFIGPATH="$PM_ADMIN_MSP" \
    -e CORE_PEER_ADDRESS=peer0.pm.it:8051 \
    peer0.pm.it \
    peer chaincode query -C "$CHANNEL_NAME" -n "$CC_NAME" -c "$1"
}

wait_reperto_su_pm() {
  local i=0
  local max=90
  sleep 3
  log "Sincronizzazione: attesa finche' peer0.pm vede $RID (max ${max}s)..."
  while (( i < max )); do
    if out="$(query_pm "{\"function\":\"ReadReperto\",\"Args\":[\"$RID\"]}" 2>/dev/null)" && \
      echo "$out" | jq -e --arg id "$RID" '.idReperto == $id' >/dev/null 2>&1; then
      log "PM allineato (${i}s)"
      return 0
    fi
    sleep 1
    i=$((i + 1))
  done
  echo "Timeout: PM non vede il reperto $RID" >&2
  echo "Diagnostica: ultima query su peer0.pm (stderr incluso)" >&2
  docker exec -e CORE_PEER_LOCALMSPID=PMMSP \
    -e CORE_PEER_MSPCONFIGPATH="$PM_ADMIN_MSP" \
    -e CORE_PEER_ADDRESS=peer0.pm.it:8051 \
    peer0.pm.it \
    peer chaincode query -C "$CHANNEL_NAME" -n "$CC_NAME" \
    -c "{\"function\":\"ReadReperto\",\"Args\":[\"$RID\"]}" 2>&1 || true
  return 1
}

wait_reperto_su_pg() {
  local i=0
  local max=30
  log "Sincronizzazione: attesa finche' peer0.pg vede $RID (max ${max}s)..."
  while (( i < max )); do
    if out="$(query_pg "{\"function\":\"ReadReperto\",\"Args\":[\"$RID\"]}" 2>/dev/null)" && \
      echo "$out" | jq -e --arg id "$RID" '.idReperto == $id' >/dev/null 2>&1; then
      log "PG allineato (presenza) (${i}s)"
      return 0
    fi
    sleep 1
    i=$((i + 1))
  done
  echo "Timeout: PG non vede il reperto $RID" >&2
  return 1
}

wait_reperto_cond_pm() {
  local jq_filter="$1"
  local i=0
  local max=60
  log "Sincronizzazione PM: attesa condizione (max ${max}s) per $RID..."
  while (( i < max )); do
    if out="$(query_pm "{\"function\":\"ReadReperto\",\"Args\":[\"$RID\"]}" 2>/dev/null)" && \
      echo "$out" | jq -e "$jq_filter" >/dev/null 2>&1; then
      log "PM allineato (condizione) (${i}s)"
      return 0
    fi
    sleep 1
    i=$((i + 1))
  done
  echo "Timeout: condizione non soddisfatta su PM per $RID (jq: $jq_filter)" >&2
  query_pm "{\"function\":\"ReadReperto\",\"Args\":[\"$RID\"]}" 2>&1 || true
  return 1
}

wait_reperto_cond_pg() {
  local jq_filter="$1"
  local i=0
  local max=30
  log "Sincronizzazione PG: attesa condizione (max ${max}s) per $RID..."
  while (( i < max )); do
    if out="$(query_pg "{\"function\":\"ReadReperto\",\"Args\":[\"$RID\"]}" 2>/dev/null)" && \
      echo "$out" | jq -e "$jq_filter" >/dev/null 2>&1; then
      log "PG allineato (stato) (${i}s)"
      return 0
    fi
    sleep 1
    i=$((i + 1))
  done
  echo "Timeout: condizione non soddisfatta su PG per $RID (jq: $jq_filter)" >&2
  echo "--- query ReadReperto su peer0.pg (stderr incluso) ---" >&2
  docker exec -e CORE_PEER_LOCALMSPID=PGMSP \
    -e CORE_PEER_MSPCONFIGPATH="$PG_ADMIN_MSP" \
    -e CORE_PEER_ADDRESS=peer0.pg.it:7051 \
    peer0.pg.it \
    peer chaincode query -C "$CHANNEL_NAME" -n "$CC_NAME" \
    -c "{\"function\":\"ReadReperto\",\"Args\":[\"$RID\"]}" 2>&1 || true
  echo "--- stesso, su peer0.lab (confronto gossip/commit) ---" >&2
  docker exec -e CORE_PEER_LOCALMSPID=LABMSP \
    -e CORE_PEER_MSPCONFIGPATH="$LAB_ADMIN_MSP" \
    -e CORE_PEER_ADDRESS=peer0.lab.it:9051 \
    peer0.lab.it \
    peer chaincode query -C "$CHANNEL_NAME" -n "$CC_NAME" \
    -c "{\"function\":\"ReadReperto\",\"Args\":[\"$RID\"]}" 2>&1 || true
  return 1
}

wait_reperto_cond_lab() {
  local jq_filter="$1"
  local i=0
  local max=60
  log "Sincronizzazione LAB: attesa condizione (max ${max}s) per $RID..."
  while (( i < max )); do
    if out="$(query_lab "{\"function\":\"ReadReperto\",\"Args\":[\"$RID\"]}" 2>/dev/null)" && \
      echo "$out" | jq -e "$jq_filter" >/dev/null 2>&1; then
      log "LAB allineato (${i}s)"
      return 0
    fi
    sleep 1
    i=$((i + 1))
  done
  echo "Timeout: condizione non soddisfatta su LAB per $RID (jq: $jq_filter)" >&2
  query_lab "{\"function\":\"ReadReperto\",\"Args\":[\"$RID\"]}" 2>&1 || true
  return 1
}

# Dopo RichiediAnalisi il decreto e' in PDC PM_LAB: LAB deve ricevere il gossip prima di RiceviInLaboratorio.
wait_decreto_lab() {
  local id_decreto="$1"
  local i=0
  local max=90
  log "Sincronizzazione LAB: metadati decreto $id_decreto (max ${max}s)..."
  while (( i < max )); do
    if out="$(query_lab "{\"function\":\"LeggiDocumento\",\"Args\":[\"CASO-WF-TEST\",\"$id_decreto\"]}" 2>/dev/null)" && \
      [[ "$(echo "$out" | jq -r '.riferimentoEnte // empty')" == "LAB-TEST" ]]; then
      log "LAB: decreto leggibile con riferimentoEnte (${i}s)"
      return 0
    fi
    sleep 1
    i=$((i + 1))
  done
  echo "Timeout: LAB non legge riferimentoEnte sul decreto $id_decreto" >&2
  query_lab "{\"function\":\"LeggiDocumento\",\"Args\":[\"CASO-WF-TEST\",\"$id_decreto\"]}" 2>&1 || true
  return 1
}

log "Preflight canale $CHANNEL_NAME"
set +e
_pf="$(docker exec -e CORE_PEER_LOCALMSPID=PGMSP \
  -e CORE_PEER_MSPCONFIGPATH="$PG_ADMIN_MSP" \
  -e CORE_PEER_ADDRESS=peer0.pg.it:7051 \
  peer0.pg.it peer channel fetch config /tmp/wf_channel.block \
  -c "$CHANNEL_NAME" -o "$ORDERER_ADDR" \
  --ordererTLSHostnameOverride orderer.example.com \
  --tls --cafile "$ORDERER_TLS_CA" 2>&1)"
_prc=$?
set -e
if [[ $_prc -ne 0 ]]; then
  echo "$_pf" >&2
  exit 1
fi

log "0b) RegistraDocumentoConTransient VERBALE_SOPRALLUOGO sul caso CASO-WF-TEST (senza reperto)"
DOC_TRANSIENT_JSON="$(jq -cn --arg c "${CID_BASE}-sop" --arg k "$CHIAVE_TEST_B64" '{cid:$c,chiaveCifrata:$k}')"
DOC_TRANSIENT_B64="$(printf '%s' "$DOC_TRANSIENT_JSON" | base64 | tr -d '\n')"
TRANSIENT_DOC="{\"documento\":\"$DOC_TRANSIENT_B64\"}"
invoke_pg_pm --transient "$TRANSIENT_DOC" \
  -c "{\"function\":\"RegistraDocumentoConTransient\",\"Args\":[\"DOC-SOP-${RID}\",\"CASO-WF-TEST\",\"VERBALE_SOPRALLUOGO\",\"\",\"Verbale sopralluogo luogo del fatto\",\"\"]}"

log "0) CreaReperto id=$RID (solo scheda custodia, endorsement PG+PM)"
PRIV_JSON="$(jq -cn \
  --arg idCaso "CASO-WF-TEST" --arg idAgente "AG-PG-INIZ" --arg idDistretto "DIST-1" \
  --arg dataOraPrelievo "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
  --arg descrizioneBene "Reperto workflow test" \
  '{idCaso:$idCaso,idAgente:$idAgente,idDistretto:$idDistretto,dataOraPrelievo:$dataOraPrelievo,descrizioneBene:$descrizioneBene}')"
B64="$(printf '%s' "$PRIV_JSON" | base64 | tr -d '\n')"
TRANSIENT_REP="{\"reperto_privato\":\"$B64\"}"
invoke_create_reperto_transient --transient "$TRANSIENT_REP" \
  -c "{\"function\":\"CreaReperto\",\"Args\":[\"$RID\"]}"

wait_reperto_su_pm
sleep 1

log "0c) RegistraDocumentoConTransient VERBALE_SEQUESTRO collegato al reperto"
VERB_JSON="$(jq -cn --arg cid "${CID_BASE}-verb-sequestro" --arg k "$CHIAVE_TEST_B64" '{cid:$cid,chiaveCifrata:$k}')"
VERB_B64="$(printf '%s' "$VERB_JSON" | base64 | tr -d '\n')"
TRANSIENT_VERB="{\"documento\":\"$VERB_B64\"}"
invoke_pg_pm --transient "$TRANSIENT_VERB" \
  -c "{\"function\":\"RegistraDocumentoConTransient\",\"Args\":[\"DOC-SEQUESTRO-${RID}\",\"CASO-WF-TEST\",\"VERBALE_SEQUESTRO\",\"$RID\",\"Verbale di sequestro del reperto\",\"DIST-1\"]}"

wait_reperto_cond_pg '.idVerbaleSequestro != "" and .idVerbaleSequestro != null'
wait_reperto_cond_pm '.idVerbaleSequestro != "" and .idVerbaleSequestro != null'

log "1) Read: SEQUESTRATO, custode = id distretto istituzionale (DIST-1)"
readout="$(query_pg "{\"function\":\"ReadReperto\",\"Args\":[\"$RID\"]}")"
echo "$readout" | jq .
_stato="$(echo "$readout" | jq -r '.stato // empty')"
if [[ "$_stato" != "SEQUESTRATO" ]]; then
  echo "Atteso stato SEQUESTRATO, trovato: $_stato" >&2
  exit 1
fi
if [[ "$(echo "$readout" | jq -r '.idVerbaleSequestro // empty')" == "" ]]; then
  echo "Atteso idVerbaleSequestro dopo registrazione verbale di sequestro" >&2
  exit 1
fi
if [[ "$(echo "$readout" | jq -r '.custodeAttuale')" != "DIST-1" ]]; then
  echo "Atteso custodeAttuale DIST-1 (custodia istituzionale), trovato: $(echo "$readout" | jq -r '.custodeAttuale')" >&2
  exit 1
fi

log "2) RichiediAnalisi (PM): tipo analisi + DECRETO_ACCERTAMENTO verso LAB-TEST"
invoke_pm -c "{\"function\":\"RichiediAnalisi\",\"Args\":[\"$RID\",\"LAB-TEST\",\"Perizia DNA\",\"${CID_BASE}-decreto\",\"$CHIAVE_TEST_B64\"]}"

wait_reperto_cond_pg '.stato == "ATTESA_TRASPORTO"'

log "3) Read: ATTESA_TRASPORTO, riferimento decreto on-chain"
readout="$(query_pg "{\"function\":\"ReadReperto\",\"Args\":[\"$RID\"]}")"
echo "$readout" | jq '{stato, custodeAttuale, idDecretoAccertamento}'
if [[ "$(echo "$readout" | jq -r '.stato')" != "ATTESA_TRASPORTO" ]]; then
  echo "Stato inatteso dopo RichiediAnalisi" >&2
  exit 1
fi
log "4) AvviaTrasporto (PG only)"
invoke_pg -c "{\"function\":\"AvviaTrasporto\",\"Args\":[\"$RID\",\"AG-PG-TRASP\"]}"

wait_reperto_cond_pg '.stato == "IN_TRANSITO"'
wait_reperto_cond_lab '.stato == "IN_TRANSITO"'

log "5) Read: IN_TRANSITO"
readout="$(query_pg "{\"function\":\"ReadReperto\",\"Args\":[\"$RID\"]}")"
if [[ "$(echo "$readout" | jq -r '.stato')" != "IN_TRANSITO" ]]; then
  echo "Stato inatteso dopo AvviaTrasporto" >&2
  exit 1
fi

log "6) RiceviInLaboratorio (LAB): idLaboratorio = destinazione decreto"
invoke_lab -c "{\"function\":\"RiceviInLaboratorio\",\"Args\":[\"$RID\",\"LAB-TEST\"]}"

wait_reperto_cond_pg '.stato == "IN_ANALISI"'

log "7) Read: IN_ANALISI, custode = ID laboratorio (non MSP)"
readout="$(query_pg "{\"function\":\"ReadReperto\",\"Args\":[\"$RID\"]}")"
echo "$readout" | jq '{stato, custodeAttuale}'
if [[ "$(echo "$readout" | jq -r '.stato')" != "IN_ANALISI" ]]; then
  echo "Stato inatteso dopo RiceviInLaboratorio" >&2
  exit 1
fi
if [[ "$(echo "$readout" | jq -r '.custodeAttuale')" != "LAB-TEST" ]]; then
  echo "Atteso custode LAB-TEST, trovato: $(echo "$readout" | jq -r '.custodeAttuale')" >&2
  exit 1
fi

log "8) CompletaAnalisi (LAB + endorse PG): RELAZIONE_TECNICA, stato ATTESA_RITIRO"
invoke_lab_with_pg -c "{\"function\":\"CompletaAnalisi\",\"Args\":[\"$RID\",\"${CID_BASE}-relazione\",\"$CHIAVE_TEST_B64\"]}"

wait_reperto_cond_pg '.stato == "ATTESA_RITIRO"'

log "9) Read: ATTESA_RITIRO, riferimento relazione tecnica"
readout="$(query_pg "{\"function\":\"ReadReperto\",\"Args\":[\"$RID\"]}")"
echo "$readout" | jq '{stato, idRelazioneTecnica}'
if [[ "$(echo "$readout" | jq -r '.stato')" != "ATTESA_RITIRO" ]]; then
  echo "Stato atteso ATTESA_RITIRO dopo CompletaAnalisi" >&2
  exit 1
fi

log "9b) PreparaRiconsegna (LAB): conferma formale pronto per ritiro PG"
invoke_lab -c "{\"function\":\"PreparaRiconsegna\",\"Args\":[\"$RID\"]}"

log "10) AvviaTrasporto (PG): ritorno verso sede da ATTESA_RITIRO"
invoke_pg -c "{\"function\":\"AvviaTrasporto\",\"Args\":[\"$RID\",\"AG-PG-RITORNO\"]}"

wait_reperto_cond_pg '.stato == "IN_TRANSITO"'

log "11) DepositaInSede (PG): SEQUESTRATO + custode distretto + VERBALE_RICONSEGNA"
invoke_pg -c "{\"function\":\"DepositaInSede\",\"Args\":[\"$RID\",\"${CID_BASE}-riconsegna\",\"$CHIAVE_TEST_B64\"]}"

wait_reperto_cond_pg '.stato == "SEQUESTRATO"'

log "12) Read finale: SEQUESTRATO, custode DIST-1, tipi documento validati on-chain"
readout="$(query_pg "{\"function\":\"ReadReperto\",\"Args\":[\"$RID\"]}")"
echo "$readout" | jq .
if [[ "$(echo "$readout" | jq -r '.stato')" != "SEQUESTRATO" ]]; then
  echo "Stato finale inatteso" >&2
  exit 1
fi
if [[ "$(echo "$readout" | jq -r '.custodeAttuale')" != "DIST-1" ]]; then
  echo "Atteso custode distretto DIST-1 dopo DepositaInSede" >&2
  exit 1
fi
echo "$readout" | jq '{idDecretoAccertamento, idRelazioneTecnica, idVerbaleRiconsegna}'
for _f in idDecretoAccertamento idRelazioneTecnica idVerbaleRiconsegna; do
  if [[ "$(echo "$readout" | jq -r --arg k "$_f" '.[$k] // empty')" == "" ]]; then
    echo "Campo $_f atteso valorizzato dopo workflow" >&2
    exit 1
  fi
done
_docs_json="$(query_pg "{\"function\":\"ListaDocumentiCaso\",\"Args\":[\"CASO-WF-TEST\"]}")"
_ndocs="$(echo "$_docs_json" | jq 'length')"
if [[ "$_ndocs" -lt 5 ]]; then
  echo "Attesi almeno 5 documenti sul caso (sopralluogo + verbale sequestro + decreto + relazione + riconsegna), trovati: $_ndocs" >&2
  exit 1
fi

log "13) OttieniStoriaReperto (PG): audit su world state"
hist="$(docker exec -e CORE_PEER_MSPCONFIGPATH="$PG_ADMIN_MSP" peer0.pg.it \
  peer chaincode query -C "$CHANNEL_NAME" -n "$CC_NAME" \
  -c "{\"function\":\"OttieniStoriaReperto\",\"Args\":[\"$RID\"]}")"
# La query ritorna una stringa JSON; prova a parsarla
if echo "$hist" | jq . >/dev/null 2>&1; then
  _hist_len="$(echo "$hist" | jq 'length')"
  log "Voci storia: $_hist_len"
  echo "$hist" | jq '.[0] | {txId, timestamp, hasReperto: (.reperto != null)}' 2>/dev/null || echo "$hist" | head -c 500
  echo
else
  echo "$hist" | head -c 1500
  echo
fi

log "OK workflow completo su id=$RID"
printf 'ID_REPERTO_PROVATO=%s\n' "$RID"
