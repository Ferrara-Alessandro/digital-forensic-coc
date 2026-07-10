#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/../lib/bash_ui.sh"

TIPO=""
DESCRIZIONE=""
MODEL="llama3"
NOME_FILE=""


read_nonempty TIPO "Tipo di reperto (es. Email, Chat, Log): "
read_nonempty DESCRIZIONE "Descrizione tecnica: "

read -r -p "Modello LLM [llama3]: " MODEL
MODEL="${MODEL:-llama3}"
[[ -n "${MODEL// }" ]] || {
  echo "Errore: modello non valido." >&2
  exit 1
}

read -r -p "Nome file output in generazione_reperti/testi/ [lascia vuoto = automatico]: " NOME_FILE

cd "$ROOT_DIR" || {
  echo "Errore: cartella progetto non accessibile: $ROOT_DIR" >&2
  exit 1
}

echo "Generazione avviata (tipo=$TIPO, model=$MODEL)..."

ARGS=(
  "--tipo" "$TIPO"
  "--descrizione-tecnica" "$DESCRIZIONE"
  "--model" "$MODEL"
)

if [[ -n "${NOME_FILE// }" ]]; then
  ARGS+=("--nome-file" "$NOME_FILE")
fi

python3 generazione_reperti/testi/generatore_testo.py "${ARGS[@]}"

if [[ -n "${NOME_FILE// }" ]]; then
  OUTPUT_FILE="generazione_reperti/testi/$NOME_FILE"
else
  OUTPUT_FILE="$(ls -1t generazione_reperti/testi/reperto_*.txt | head -n 1)"
fi

if [[ ! -f "$OUTPUT_FILE" ]]; then
  echo "Errore: file output non trovato: $OUTPUT_FILE" >&2
  exit 1
fi

SHA256="$(sha256sum "$OUTPUT_FILE" | awk '{print $1}')"

printf '\n[OK] Reperto testuale generato\n'
printf 'FILE=%s\n' "$OUTPUT_FILE"
printf 'SHA256=%s\n' "$SHA256"
