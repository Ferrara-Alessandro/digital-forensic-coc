# Funzioni condivise tra generatore testo e immagini.

from __future__ import annotations

import hashlib
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
MODULO = Path(__file__).resolve().parent


def sha256_file(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(65536), b""):
            h.update(chunk)
    return h.hexdigest()


def path_relativo(path: Path, base: Path = ROOT) -> str:
    try:
        return str(path.resolve().relative_to(base.resolve()))
    except ValueError:
        return str(path)


def risultato(path: Path, source: str, **extra: str | int) -> dict:
    """Risposta standard: percorso relativo alla root del repo, hash e metadati."""
    out: dict = {
        "file": path_relativo(path),
        "sha256": sha256_file(path),
        "bytes": path.stat().st_size,
        "source": source,
    }
    out.update(extra)
    return out
