#!/usr/bin/env bash
# Lancia python dal venv del modulo, se esiste.
set -euo pipefail
MODULO="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -x "$MODULO/.venv/bin/python" ]]; then
  exec "$MODULO/.venv/bin/python" "$@"
fi
exec python3 "$@"
