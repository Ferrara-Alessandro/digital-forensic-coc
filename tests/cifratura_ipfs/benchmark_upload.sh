#!/usr/bin/env bash
# Benchmark: cifratura AES + upload IPFS (senza scrittura su Fabric).
# Strumento: bin/upload (Go), mode evidenza, -skip-chaincode.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../lib/common.sh
source "$SCRIPT_DIR/../lib/common.sh"

REPEATS="${BENCH_REPEATS:-3}"
SIZES_MB="${BENCH_SIZES_MB:-1 5 10}"
CASO_ID="${BENCH_CASO_ID:-CASO-BENCH-IPFS}"
REPERTO_ID="${BENCH_REPERTO_ID:-REP-BENCH-CALIPER}"
WORKDIR="${BENCH_WORKDIR:-$TESTS_DIR/.work/cifratura_ipfs}"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
CSV="$RESULTS_DIR/cifratura_upload_${TIMESTAMP}.csv"

require_cmd jq
require_cmd dd
require_cmd curl
require_ipfs
ensure_upload_bin
mkdir_results
mkdir -p "$WORKDIR"

section "Cifratura + upload IPFS (client Go, no chaincode)"

echo "size_mb,repeat,input_bytes,encrypted_bytes,encrypt_ms,ipfs_ms" >"$CSV"

for size_mb in $SIZES_MB; do
  input_file="$WORKDIR/payload_${size_mb}mb.bin"
  if [[ ! -f "$input_file" ]]; then
    dd if=/dev/urandom of="$input_file" bs=1M count="$size_mb" status=none
  fi

  for ((r = 1; r <= REPEATS; r++)); do
    evi_id="EVI-UP-${size_mb}M-r${r}-$(date +%s)"
    json_out="$(
      "$UPLOAD_BIN" \
        -mode evidenza \
        -skip-chaincode \
        -file "$input_file" \
        -id-caso "$CASO_ID" \
        -id-evidenza "$evi_id" \
        -id-reperto-evidenza "$REPERTO_ID" \
        -descrizione-evidenza "benchmark upload ${size_mb}MB run $r" \
        -classe-evidenza "DIGITALE" \
        -ingest-org pg 2>/dev/null
    )"

    input_bytes="$(echo "$json_out" | jq -r '.inputSizeBytes')"
    enc_bytes="$(echo "$json_out" | jq -r '.encryptedSizeBytes')"
    enc_ms="$(echo "$json_out" | jq -r '.tempoEncryptOnlyMs // 0')"
    ipfs_ms="$(echo "$json_out" | jq -r '.tempoIpfsMs // 0')"

    echo "${size_mb},${r},${input_bytes},${enc_bytes},${enc_ms},${ipfs_ms}" >>"$CSV"
    printf '  %s MB run %d: encrypt=%sms ipfs=%sms\n' "$size_mb" "$r" "$enc_ms" "$ipfs_ms"
  done
done

echo "Risultati: $CSV"
