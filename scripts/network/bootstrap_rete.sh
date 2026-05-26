#!/usr/bin/env bash
set -euo pipefail

# Directory dove si trova questo script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Parametri principali del bootstrap
NETWORK_NAME="rete-coc"
CHANNEL_NAME="canale-coc"
CHANNEL_BLOCK="/etc/hyperledger/coc-pki/canale-coc.block"
TOOLS_IMAGE="${TOOLS_IMAGE:-hyperledger/fabric-tools:2.5.15}"

# File compose usati per orderer e peer
ORDERER_COMPOSE="$ROOT_DIR/infrastruttura_blockchain/avvio_nodi.yaml"
PEER_COMPOSE="$ROOT_DIR/infrastruttura_blockchain/peer_nodi.yaml"
INFRA="$ROOT_DIR/infrastruttura_blockchain"

# Mappa peer gestiti dallo script nel formato:
# org|container|admin_msp_path
# admin_msp_path serve per eseguire il join con identita' amministrativa.
PEERS=(
  "pg|peer0.pg.it|/etc/hyperledger/coc-pki/certificati_pki/peerOrganizations/pg.it/users/Admin@pg.it/msp"
  "pm|peer0.pm.it|/etc/hyperledger/coc-pki/certificati_pki/peerOrganizations/pm.it/users/Admin@pm.it/msp"
  "lab|peer0.lab.it|/etc/hyperledger/coc-pki/certificati_pki/peerOrganizations/lab.it/users/Admin@lab.it/msp"
)

# Log uniforme con timestamp per seguire il flusso a colpo d'occhio.
log() {
  printf "\n[%s] %s\n" "$(date +"%H:%M:%S")" "$1"
}

# Verifica prerequisiti minimi prima di iniziare.
require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Errore: comando '$1' non trovato" >&2
    exit 1
  fi
}

# Ritorna 0 (true) se il peer e' gia' iscritto al canale.
is_peer_joined() {
  local container="$1"
  docker exec "$container" peer channel list 2>/dev/null | awk 'NR>1 {print $1}' | grep -qx "$CHANNEL_NAME"
}

# Esegue il join solo se necessario (idempotente).
join_peer_if_needed() {
  local org="$1"
  local container="$2"
  local admin_msp="$3"

  if is_peer_joined "$container"; then
    log "Peer $container ($org) gia' iscritto a $CHANNEL_NAME, skip join"
    return 0
  fi

  log "Join del peer $container ($org) al canale $CHANNEL_NAME"
  # Join con MSP admin: senza questo contesto Fabric rifiuta l'operazione.
  docker exec \
    -e CORE_PEER_MSPCONFIGPATH="$admin_msp" \
    "$container" \
    peer channel join -b "$CHANNEL_BLOCK"
}

# Flusso principale:
# 1) PKI e genesis (se assenti)
# 2) validazione compose
# 3) rete docker
# 4) avvio nodi
# 5) join canale
# 6) verifiche finali
main() {
  require_cmd docker
  cd "$ROOT_DIR" || {
    echo "Errore: cartella progetto non accessibile: $ROOT_DIR" >&2
    exit 1
  }

  # Genero PKI solo se non esiste ancora (cryptogen via Docker).
  if [[ ! -d "$INFRA/certificati_pki" ]]; then
    log "Genero PKI (cryptogen)"
    docker run --rm \
      -v "$INFRA:/work" -w /work \
      "$TOOLS_IMAGE" \
      cryptogen generate \
        --config=/work/definizione_organizzazioni.yaml \
        --output=/work/certificati_pki
    # Le chiavi generate da Docker appartengono a root: le riassegno all'utente corrente.
    find "$INFRA/certificati_pki" -name "priv_sk" -exec chown "$(id -u):$(id -g)" {} \; 2>/dev/null || true
  else
    log "PKI presente, skip cryptogen"
  fi

  # Genero genesis.block solo se assente.
  if [[ ! -f "$INFRA/genesis.block" ]]; then
    log "Genero genesis.block (configtxgen, profilo CocGenesis)"
    docker run --rm \
      -v "$INFRA:/work" -w /work \
      -e FABRIC_CFG_PATH=/work \
      "$TOOLS_IMAGE" \
      configtxgen -profile CocGenesis -channelID system-channel -outputBlock /work/genesis.block
  else
    log "genesis.block presente, skip configtxgen"
  fi

  log "Validazione file compose"
  docker compose -f "$ORDERER_COMPOSE" config >/dev/null
  docker compose -f "$PEER_COMPOSE" config >/dev/null

  if ! docker network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
    log "Rete Docker $NETWORK_NAME assente: la creo"
    docker network create "$NETWORK_NAME" >/dev/null
  else
    log "Rete Docker $NETWORK_NAME presente"
  fi

  log "Avvio orderer + peer (stessi -f insieme: se avvii solo peer_nodi.yaml, Compose rimuove l'orderer dallo stack)"
  (
    cd "$ROOT_DIR/infrastruttura_blockchain" || exit 1
    docker compose -f "$ORDERER_COMPOSE" -f "$PEER_COMPOSE" up -d
  )

  log "Attendo orderer pronto"
  sleep 3
  # Un container orderer creato prima di aggiornamenti al compose (es. FABRIC_CFG_PATH) puo'
  # essere solo riavviato senza recreate: l'orderer fallisce con "open : no such file".
  local orderer_status
  orderer_status="$(docker inspect orderer.example.com --format '{{.State.Status}}' 2>/dev/null || echo missing)"
  if [[ "$orderer_status" != "running" ]]; then
    log "Orderer in stato '$orderer_status' dopo up: applico --force-recreate per allineare env/volumi al compose attuale"
    (
      cd "$ROOT_DIR/infrastruttura_blockchain" || exit 1
      docker compose -f "$ORDERER_COMPOSE" -f "$PEER_COMPOSE" up -d --force-recreate
    )
    sleep 2
  fi

  log "Attendo stabilizzazione orderer"
  sleep 2

  # Crea il canale applicativo sull'orderer e aggiorna canale-coc.block (idempotente).
  log "Verifica/creazione canale $CHANNEL_NAME su orderer (configtxgen + channel create)"
  bash "$ROOT_DIR/scripts/network/ensure_canale_coc_orderer.sh"

  log "Peer gia' avviati con l'up combinato sopra; attendo stabilizzazione"

  # Il join usa canale-coc.block sotto infrastruttura_blockchain (montato nei container).
  if [[ ! -f "$ROOT_DIR/infrastruttura_blockchain/canale-coc.block" ]]; then
    echo "Errore: blocco canale assente dopo ensure. Controlla ensure_canale_coc_orderer.sh" >&2
    exit 1
  fi

  # Piccola attesa per dare tempo ai servizi di entrare in stato operativo.
  log "Attendo bootstrap servizi"
  sleep 3

  for p in "${PEERS[@]}"; do
    IFS='|' read -r org container admin_msp <<< "$p"
    join_peer_if_needed "$org" "$container" "$admin_msp"
  done

  log "Verifica finale: canali su peer"
  for p in "${PEERS[@]}"; do
    IFS='|' read -r _ container _ <<< "$p"
    echo "--- $container ---"
    docker exec "$container" peer channel list
  done

  # Verifica rapida ledger da un peer di riferimento.
  log "Verifica ledger su peer0.pg.it"
  docker exec peer0.pg.it peer channel getinfo -c "$CHANNEL_NAME"

  log "Bootstrap completato"
}

main "$@"
