#!/usr/bin/env bash
# Benchmark: prelievo IPFS + decifratura (senza lettura dal chaincode).
# Richiede blob gia' su IPFS: per ogni dimensione esegue un upload di preparazione
# (-skip-chaincode) se manca il manifest, poi misura il download isolato.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../lib/common.sh
source "$SCRIPT_DIR/../lib/common.sh"

SIZES_MB="${BENCH_SIZES_MB:-10 100 500 1024}"
CASO_ID="${BENCH_CASO_ID:-CASO-BENCH-IPFS}"
REPERTO_ID="${BENCH_REPERTO_ID:-REP-BENCH-CALIPER}"
WORKDIR="${BENCH_WORKDIR:-$TESTS_DIR/.work/cifratura_ipfs}"
OUT_DIR="$WORKDIR/out"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
CSV="$RESULTS_DIR/cifratura_download_${TIMESTAMP}.csv"

require_cmd jq
require_cmd dd
require_cmd cmp
require_ipfs
ensure_upload_bin
ensure_download_bin
mkdir_results
mkdir -p "$WORKDIR" "$OUT_DIR"

section "Prelievo IPFS + decifratura (client Go, no chaincode)"
echo "Dimensioni (MB): $SIZES_MB"
echo "Output: $CSV"
echo ""

echo "size_mb,input_bytes,encrypted_bytes,ipfs_ms,decrypt_ms,total_ms,ipfs_timeout_sec" >"$CSV"

prepare_blob_on_ipfs() {
  local size_mb="$1"
  local manifest="$WORKDIR/manifest_${size_mb}mb.json"
  local input_file="$WORKDIR/payload_${size_mb}mb.bin"

  if [[ ! -f "$input_file" ]]; then
    echo "  Generazione payload ${size_mb} MB..."
    dd if=/dev/urandom of="$input_file" bs=1M count="$size_mb" status=none 2>/dev/null \
      || dd if=/dev/urandom of="$input_file" bs=1048576 count="$size_mb" status=none
  fi

  if [[ -f "$manifest" ]]; then
    echo "  Manifest ${size_mb} MB gia' presente, skip preparazione upload"
    return 0
  fi

  local ipfs_to
  ipfs_to="$(bench_ipfs_timeout_for_mb "$size_mb")"
  local evi_id="EVI-PREP-${size_mb}M-$(date +%s)"
  echo "  Preparazione blob ${size_mb} MB su IPFS (upload -skip-chaincode)..."

  local json_out
  json_out="$(
    "$UPLOAD_BIN" \
      -mode evidenza \
      -skip-chaincode \
      -file "$input_file" \
      -id-caso "$CASO_ID" \
      -id-evidenza "$evi_id" \
      -id-reperto-evidenza "$REPERTO_ID" \
      -descrizione-evidenza "benchmark prep ${size_mb}MB" \
      -classe-evidenza "DIGITALE" \
      -ingest-org pg \
      -ipfs-timeout "${ipfs_to}s"
  )"

  echo "$json_out" | jq -c \
    --arg input "$input_file" \
    --arg ipfs_to "$ipfs_to" \
    '{input_file:$input,cid:.cid,key_b64:.chiaveB64,input_bytes:.inputSizeBytes,encrypted_bytes:.encryptedSizeBytes,ipfs_timeout_sec:($ipfs_to|tonumber)}' \
    >"$manifest"
}

for size_mb in $SIZES_MB; do
  prepare_blob_on_ipfs "$size_mb"

  manifest="$WORKDIR/manifest_${size_mb}mb.json"
  input_file="$(jq -r '.input_file' "$manifest")"
  cid="$(jq -r '.cid' "$manifest")"
  key_b64="$(jq -r '.key_b64' "$manifest")"
  ipfs_to="$(jq -r '.ipfs_timeout_sec' "$manifest")"
  read_to="$((ipfs_to + 120))"

  echo "  → ${size_mb} MB (ipfs-timeout=${ipfs_to}s)"

  out_file="$OUT_DIR/download_${size_mb}mb.bin"
  json_out="$(
    "$DOWNLOAD_BIN" \
      -mode evidenza \
      -skip-chaincode \
      -cid "$cid" \
      -key-b64 "$key_b64" \
      -id-evidenza "EVI-DL-${size_mb}M-$(date +%s)" \
      -out-file "$out_file" \
      -ipfs-timeout "${ipfs_to}s" \
      -read-timeout "${read_to}s"
  )"

  cmp -s "$input_file" "$out_file"

  input_bytes="$(echo "$json_out" | jq -r '.decryptedSizeBytes')"
  enc_bytes="$(echo "$json_out" | jq -r '.encryptedSizeBytes')"
  ipfs_ms="$(echo "$json_out" | jq -r '.tempoIpfsMs // 0')"
  dec_ms="$(echo "$json_out" | jq -r '.tempoDecryptMs // 0')"
  tot_ms="$(echo "$json_out" | jq -r '.tempoTotaleMs // 0')"

  echo "${size_mb},${input_bytes},${enc_bytes},${ipfs_ms},${dec_ms},${tot_ms},${ipfs_to}" >>"$CSV"
  printf '    ipfs=%sms decrypt=%sms total=%sms\n' "$ipfs_ms" "$dec_ms" "$tot_ms"
done

echo ""
echo "Risultati: $CSV"
