#!/usr/bin/env bash
# Crea generazione_reperti/.venv e installa torch + diffusers.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VENV="$ROOT/.venv"

if ! python3 -c "import venv" 2>/dev/null; then
  echo "Serve python3-venv: sudo apt install python3-venv python3-full" >&2
  exit 1
fi

if [[ ! -d "$VENV" ]]; then
  python3 -m venv "$VENV"
fi

"$VENV/bin/pip" install --upgrade pip
"$VENV/bin/pip" install -r "$ROOT/requirements.txt"

echo "OK: $VENV/bin/python generazione_reperti/immagini/generatore.py --descrizione \"...\""
