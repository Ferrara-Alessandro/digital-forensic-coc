#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"


ask_yes_no_default_no() {
  local prompt="$1"
  local answer
  read -r -p "$prompt [y/N]: " answer
  case "${answer,,}" in
    y|yes) echo "true" ;;
    *) echo "false" ;;
  esac
}

ask_hash_target_default_variante() {
  local answer
  while true; do
    read -r -p "Hash target (master/variante/dettaglio) [variante]: " answer
    answer="${answer,,}"
    answer="${answer:-variante}"
    if [[ "$answer" == "master" || "$answer" == "variante" || "$answer" == "dettaglio" ]]; then
      echo "$answer"
      return 0
    fi
    echo "Errore: inserisci master, variante o dettaglio." >&2
  done
}

SOLO_VARIANTE="$(ask_yes_no_default_no "Generare solo la variante (senza rigenerare il master)?")"
HASH_TARGET="$(ask_hash_target_default_variante)"

cd "$ROOT_DIR" || {
  echo "Errore: cartella progetto non accessibile: $ROOT_DIR" >&2
  exit 1
}

echo "Generazione immagini avviata (solo_variante=$SOLO_VARIANTE, hash_target=$HASH_TARGET)..."

PYTHON_BIN="python3"
if [[ -x "generazione_reperti/supporto_ia/bin/python3" ]]; then
  PYTHON_BIN="generazione_reperti/supporto_ia/bin/python3"
fi

if [[ "$SOLO_VARIANTE" == "false" ]]; then
  "$PYTHON_BIN" generazione_reperti/immagini/generatore_master.py
fi

"$PYTHON_BIN" generazione_reperti/immagini/modificatore_inquadratura.py

MASTER_FILE="generazione_reperti/immagini/immagine_master.png"
VARIANTE_FILE="$(ls -1t generazione_reperti/immagini/variante_*.png 2>/dev/null | head -n 1)"
DETTAGLIO_FILE="$(ls -1t generazione_reperti/immagini/dettaglio_*.png 2>/dev/null | head -n 1)"

if [[ "$HASH_TARGET" == "master" ]]; then
  TARGET_FILE="$MASTER_FILE"
elif [[ "$HASH_TARGET" == "dettaglio" ]]; then
  TARGET_FILE="$DETTAGLIO_FILE"
else
  TARGET_FILE="$VARIANTE_FILE"
fi

if [[ -z "${TARGET_FILE:-}" || ! -f "$TARGET_FILE" ]]; then
  echo "Errore: file target non trovato per hash." >&2
  exit 1
fi

SHA256="$(sha256sum "$TARGET_FILE" | awk '{print $1}')"

printf '\n[OK] Reperto immagine generato\n'
printf 'MASTER_FILE=%s\n' "$MASTER_FILE"
printf 'VARIANTE_FILE=%s\n' "${VARIANTE_FILE:-}"
printf 'DETTAGLIO_FILE=%s\n' "${DETTAGLIO_FILE:-}"
printf 'HASH_TARGET=%s\n' "$HASH_TARGET"
printf 'SHA256=%s\n' "$SHA256"
