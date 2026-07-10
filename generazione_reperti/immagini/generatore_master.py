#!/usr/bin/env python3
# Prepara immagine_master.png: copia un file fornito o genera con Stable Diffusion.

from __future__ import annotations

import argparse
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))

from generazione_reperti.common import MODULO, path_relativo, risultato
from generazione_reperti.immagini.generatore import genera_immagine

OUTPUT_DIR = MODULO / "immagini"
MASTER_PATH = OUTPUT_DIR / "immagine_master.png"

PROMPT_DEFAULT = (
    "smartphone Android su tavolo da perquisizione, foto forense, luce neutra"
)


def _copia_come_master(src: Path) -> dict:
    try:
        from PIL import Image
    except ImportError as exc:
        raise RuntimeError("Manca Pillow. Installa con: pip install pillow") from exc

    if not src.is_file():
        raise FileNotFoundError(f"File sorgente non trovato: {src}")

    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    with Image.open(src) as img:
        img.convert("RGB").save(MASTER_PATH, format="PNG")

    return risultato(MASTER_PATH, f"copy:{path_relativo(src)}")


def prepara_master(
    *,
    input_path: str = "",
    descrizione: str = PROMPT_DEFAULT,
    seed: int = 12345,
    steps: int = 20,
) -> dict:
    if input_path.strip():
        return _copia_come_master(Path(input_path.strip()))

    if MASTER_PATH.is_file():
        return risultato(MASTER_PATH, "existing:immagine_master.png")

    return genera_immagine(
        descrizione,
        seed=seed,
        num_inference_steps=steps,
        nome_file="immagine_master.png",
    )


def main() -> None:
    parser = argparse.ArgumentParser(description="Prepara generazione_reperti/immagini/immagine_master.png")
    parser.add_argument(
        "--input",
        default="",
        help="Copia un'immagine esistente come master (salta Stable Diffusion)",
    )
    parser.add_argument("--descrizione", default=PROMPT_DEFAULT)
    parser.add_argument("--seed", type=int, default=12345)
    parser.add_argument("--steps", type=int, default=20)
    parser.add_argument(
        "--forza",
        action="store_true",
        help="Rigenera il master anche se immagine_master.png esiste già",
    )
    args = parser.parse_args()

    if args.forza and MASTER_PATH.is_file() and not args.input.strip():
        MASTER_PATH.unlink()

    result = prepara_master(
        input_path=args.input,
        descrizione=args.descrizione,
        seed=args.seed,
        steps=args.steps,
    )
    for key, val in result.items():
        print(f"{key.upper()}={val}")


if __name__ == "__main__":
    main()
