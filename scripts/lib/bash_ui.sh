# shellcheck shell=bash
# Funzioni riutilizzabili per input interattivo (da includere con: source "$SCRIPT_DIR/../lib/bash_ui.sh").

# Legge finche' la riga non e' vuota (solo spazi = vuoto).
read_nonempty() {
  local var_name="$1"
  local prompt="$2"
  local line
  while true; do
    read -r -p "$prompt" line
    line="${line#"${line%%[![:space:]]*}"}"
    line="${line%"${line##*[![:space:]]}"}"
    if [[ -n "$line" ]]; then
      printf -v "$var_name" '%s' "$line"
      return 0
    fi
    echo "Errore: campo obbligatorio vuoto." >&2
  done
}
