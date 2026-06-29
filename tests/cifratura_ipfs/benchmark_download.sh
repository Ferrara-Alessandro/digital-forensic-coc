#!/usr/bin/env bash
# Benchmark: round-trip upload (cifra + IPFS + Fabric) e download (Fabric + IPFS + decifra).
# Strumento: bin/upload + bin/download (Go).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../lib/common.sh
source "$SCRIPT_DIR/../lib/common.sh"

SIZES_MB="${BENCH_SIZES_MB:-1 5}"
CASO_ID="${BENCH_CASO_ID:-CASO-BENCH-DL}"
REPERTO_ID="${BENCH_REPERTO_ID:-REP-BENCH-CALIPER}"
WORKDIR="${BENCH_WORKDIR:-$TESTS_DIR/.work/cifratura_ipfs}"
OUT_DIR="$WORKDIR/out"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
CSV="$RESULTS_DIR/cifratura_download_${TIMESTAMP}.csv"

require_cmd jq
require_cmd dd
require_cmd curl
require_docker_peer
require_ipfs
ensure_client_bins
require_fabric_peers_sync
mkdir_results
mkdir -p "$WORKDIR" "$OUT_DIR"

section "Download round-trip (client Go: upload + download)"

bash "$TESTS_DIR/fabric_caliper/caliper/scripts/prepara_dati.sh"

echo "size_mb,evidenza_id,upload_encrypt_ms,upload_ipfs_ms,upload_fabric_ms,download_fabric_ms,download_ipfs_ms,download_decrypt_ms,download_total_ms" >"$CSV"

for size_mb in $SIZES_MB; do
  input_file="$WORKDIR/payload_${size_mb}mb.bin"
  if [[ ! -f "$input_file" ]]; then
    dd if=/dev/urandom of="$input_file" bs=1M count="$size_mb" status=none
  fi

  evi_id="EVI-DL-${size_mb}M-$(date +%s)"
  up_json="$(
    "$UPLOAD_BIN" \
      -mode evidenza \
      -file "$input_file" \
      -id-caso "$CASO_ID" \
      -id-evidenza "$evi_id" \
      -id-reperto-evidenza "$REPERTO_ID" \
      -descrizione-evidenza "benchmark download ${size_mb}MB" \
      -classe-evidenza "DIGITALE" \
      -ingest-org pg
  )"

  up_enc_ms="$(echo "$up_json" | jq -r '.tempoEncryptOnlyMs // 0')"
  up_ipfs_ms="$(echo "$up_json" | jq -r '.tempoIpfsMs // 0')"
  up_fab_ms="$(echo "$up_json" | jq -r '.tempoFabricMs // 0')"

  sleep 2

  dl_json="$(
    "$DOWNLOAD_BIN" \
      -mode evidenza \
      -org pg \
      -id-caso "$CASO_ID" \
      -id-evidenza "$evi_id" \
      -out-dir "$OUT_DIR" \
      -out-file "$OUT_DIR/${evi_id}.bin"
  )"

  dl_fab_ms="$(echo "$dl_json" | jq -r '.tempoFabricMs // 0')"
  dl_ipfs_ms="$(echo "$dl_json" | jq -r '.tempoIpfsMs // 0')"
  dl_dec_ms="$(echo "$dl_json" | jq -r '.tempoDecryptMs // 0')"
  dl_tot_ms="$(echo "$dl_json" | jq -r '.tempoTotaleMs // 0')"

  echo "${size_mb},${evi_id},${up_enc_ms},${up_ipfs_ms},${up_fab_ms},${dl_fab_ms},${dl_ipfs_ms},${dl_dec_ms},${dl_tot_ms}" >>"$CSV"
  printf '  %s MB: upload(ipfs=%sms fabric=%sms) download(ipfs=%sms decrypt=%sms total=%sms)\n' \
    "$size_mb" "$up_ipfs_ms" "$up_fab_ms" "$dl_ipfs_ms" "$dl_dec_ms" "$dl_tot_ms"
done

echo "Risultati: $CSV"
