#!/usr/bin/env bash
# Crea il reperto REP-BENCH-CALIPER (se assente) tramite test funzionale ciclo vita.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
BENCH_ID="${BENCH_REPERTO_ID:-REP-BENCH-CALIPER}"
CICLO_VITA="$ROOT_DIR/tests/ciclo_vita/test_ciclo_vita_reperto.sh"

cd "$ROOT_DIR"

if ! docker ps --format '{{.Names}}' | grep -qx 'peer0.pg.it'; then
  echo "Rete Fabric non attiva. Esegui prima:" >&2
  echo "  bash scripts/network/bootstrap_rete.sh" >&2
  echo "  bash scripts/network/deploy_lifecycle_reperto.sh" >&2
  exit 1
fi

echo "Reperto benchmark: $BENCH_ID"

if docker exec -e CORE_PEER_LOCALMSPID=PGMSP \
  -e CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/coc-pki/certificati_pki/peerOrganizations/pg.it/users/Admin@pg.it/msp \
  -e CORE_PEER_ADDRESS=peer0.pg.it:7051 peer0.pg.it \
  peer chaincode query -C canale-coc -n reperto \
  -c "{\"Args\":[\"RepertoExists\",\"$BENCH_ID\"]}" 2>/dev/null | grep -q 'true'; then
  echo "Reperto già presente, skip setup."
  exit 0
fi

echo "Creazione reperto con ciclo vita completo..."
ID_REPERTO="$BENCH_ID" bash "$CICLO_VITA"
echo "Setup completato."
