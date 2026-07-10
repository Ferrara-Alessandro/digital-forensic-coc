#!/usr/bin/env bash
# Benchmark: cifratura AES + upload IPFS (senza scrittura su Fabric).
# Strumento: bin/upload (Go), mode evidenza, -skip-chaincode.
# Dimensioni predefinite orientate a evidenze forensi reali (backup parziali, export app, …).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../lib/common.sh
source "$SCRIPT_DIR/../lib/common.sh"

SIZES_MB="${BENCH_SIZES_MB:-10 100 500 1024}"
CASO_ID="${BENCH_CASO_ID:-CASO-BENCH-IPFS}"
REPERTO_ID="${BENCH_REPERTO_ID:-REP-BENCH-CALIPER}"
WORKDIR="${BENCH_WORKDIR:-$TESTS_DIR/.work/cifratura_ipfs}"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
CSV="$RESULTS_DIR/cifratura_upload_${TIMESTAMP}.csv"

require_cmd jq
require_cmd dd
require_ipfs
ensure_upload_bin
mkdir_results
mkdir -p "$WORKDIR"

section "Cifratura + upload IPFS (client Go, no chaincode)"
echo "Dimensioni (MB): $SIZES_MB"
echo "Output: $CSV"
echo ""

echo "size_mb,input_bytes,encrypted_bytes,encrypt_ms,ipfs_ms,ipfs_timeout_sec" >"$CSV"

for size_mb in $SIZES_MB; do
  ipfs_to="$(bench_ipfs_timeout_for_mb "$size_mb")"
  input_file="$WORKDIR/payload_${size_mb}mb.bin"
  if [[ ! -f "$input_file" ]]; then
    echo "  Generazione payload ${size_mb} MB..."
    dd if=/dev/urandom of="$input_file" bs=1M count="$size_mb" status=none 2>/dev/null \
      || dd if=/dev/urandom of="$input_file" bs=1048576 count="$size_mb" status=none
  fi

  echo "  → ${size_mb} MB (ipfs-timeout=${ipfs_to}s)"

  evi_id="EVI-UP-${size_mb}M-$(date +%s)"
  json_out="$(
    "$UPLOAD_BIN" \
      -mode evidenza \
      -skip-chaincode \
      -file "$input_file" \
      -id-caso "$CASO_ID" \
      -id-evidenza "$evi_id" \
      -id-reperto-evidenza "$REPERTO_ID" \
      -descrizione-evidenza "benchmark upload ${size_mb}MB" \
      -classe-evidenza "DIGITALE" \
      -ingest-org pg \
      -ipfs-timeout "${ipfs_to}s"
  )"

  input_bytes="$(echo "$json_out" | jq -r '.inputSizeBytes')"
  enc_bytes="$(echo "$json_out" | jq -r '.encryptedSizeBytes')"
  enc_ms="$(echo "$json_out" | jq -r '.tempoEncryptOnlyMs // 0')"
  ipfs_ms="$(echo "$json_out" | jq -r '.tempoIpfsMs // 0')"

  echo "${size_mb},${input_bytes},${enc_bytes},${enc_ms},${ipfs_ms},${ipfs_to}" >>"$CSV"
  printf '    encrypt=%sms ipfs=%sms\n' "$enc_ms" "$ipfs_ms"
done

echo ""
echo "Risultati: $CSV"
