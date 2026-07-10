#!/usr/bin/env python3
# Varianti di inquadratura e zoom da immagine_master.png.

from __future__ import annotations

import argparse
import sys
from datetime import datetime
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))

from generazione_reperti.common import MODULO, path_relativo, risultato

OUTPUT_DIR = MODULO / "immagini"
MASTER_DEFAULT = OUTPUT_DIR / "immagine_master.png"

# Variante con inquadratura obliqua rispetto alla master.
INQUADRATURA_ROTATION = -9.0
INQUADRATURA_ZOOM = 1.18
INQUADRATURA_CROP_RATIO = 0.96
INQUADRATURA_OFFSET_X = 0.50
INQUADRATURA_OFFSET_Y = 0.45

# Primo dettaglio: crop centrale-basso.
DETTAGLIO_CROP_RATIO = 0.38
DETTAGLIO_OFFSET_X = 0.42
DETTAGLIO_OFFSET_Y = 0.58

# Secondo dettaglio: crop sulla parte destra del reperto.
DETTAGLIO_2_CROP_RATIO = 0.30
DETTAGLIO_2_OFFSET_X = 0.82
DETTAGLIO_2_OFFSET_Y = 0.52


def _ritaglio_centrale(img, width: int, height: int, *, zoom: float = 1.0):
    from PIL import Image

    zoom = max(1.0, zoom)
    w, h = img.size
    crop_w = max(1, int(width / zoom))
    crop_h = max(1, int(height / zoom))
    crop_w = min(crop_w, w)
    crop_h = min(crop_h, h)
    left = max(0, (w - crop_w) // 2)
    top = max(0, (h - crop_h) // 2)
    cropped = img.crop((left, top, left + crop_w, top + crop_h))
    return cropped.resize((width, height), Image.Resampling.LANCZOS)


def _inquadratura_obliqua(
    img,
    *,
    width: int,
    height: int,
    rotation: float,
    zoom: float,
    crop_ratio: float,
    offset_x: float,
    offset_y: float,
):
    from PIL import Image

    rotated = img.rotate(
        rotation,
        resample=Image.Resampling.BICUBIC,
        expand=True,
        fillcolor=img.getpixel((width // 2, height // 2)),
    )
    rotated = _ritaglio_centrale(rotated, width, height, zoom=zoom)
    return _trasforma(
        rotated,
        width=width,
        height=height,
        crop_ratio=crop_ratio,
        offset_x=offset_x,
        offset_y=offset_y,
    )


def _trasforma(
    img,
    *,
    width: int,
    height: int,
    crop_ratio: float,
    offset_x: float,
    offset_y: float,
):
    from PIL import Image

    crop_ratio = max(0.2, min(crop_ratio, 0.98))
    offset_x = max(0.0, min(offset_x, 1.0))
    offset_y = max(0.0, min(offset_y, 1.0))

    crop_w = max(1, int(width * crop_ratio))
    crop_h = max(1, int(height * crop_ratio))
    margin_x = width - crop_w
    margin_y = height - crop_h

    left = int(margin_x * offset_x)
    top = int(margin_y * offset_y)
    box = (left, top, left + crop_w, top + crop_h)
    return img.crop(box).resize((width, height), Image.Resampling.LANCZOS)


def _salva_variante(img, prefix: str, nome_file: str = "") -> Path:
    stamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    if nome_file.strip():
        name = nome_file.strip()
        if not name.lower().endswith(".png"):
            name += ".png"
    else:
        name = f"{prefix}_{stamp}.png"

    out_path = OUTPUT_DIR / name
    out_path.parent.mkdir(parents=True, exist_ok=True)
    img.save(out_path, format="PNG")
    return out_path


def crea_variante_inquadratura(
    master_path: Path | None = None,
    *,
    rotation: float = INQUADRATURA_ROTATION,
    zoom: float = INQUADRATURA_ZOOM,
    crop_ratio: float = INQUADRATURA_CROP_RATIO,
    offset_x: float = INQUADRATURA_OFFSET_X,
    offset_y: float = INQUADRATURA_OFFSET_Y,
    nome_file: str = "",
) -> dict:
    try:
        from PIL import Image
    except ImportError as exc:
        raise RuntimeError("Manca Pillow. Installa con: pip install pillow") from exc

    src = master_path or MASTER_DEFAULT
    if not src.is_file():
        raise FileNotFoundError(f"Immagine master non trovata: {src}")

    with Image.open(src) as img:
        img = img.convert("RGB")
        width, height = img.size
        variant = _inquadratura_obliqua(
            img,
            width=width,
            height=height,
            rotation=rotation,
            zoom=zoom,
            crop_ratio=crop_ratio,
            offset_x=offset_x,
            offset_y=offset_y,
        )
        out_path = _salva_variante(variant, "variante", nome_file)

    return risultato(
        out_path,
        "augmentation:inquadratura",
        master=path_relativo(src),
        tipo="inquadratura",
        rotation=rotation,
        zoom=zoom,
        crop_ratio=crop_ratio,
        offset_x=offset_x,
        offset_y=offset_y,
        width=width,
        height=height,
    )


def crea_variante_dettaglio(
    master_path: Path | None = None,
    *,
    crop_ratio: float = DETTAGLIO_CROP_RATIO,
    offset_x: float = DETTAGLIO_OFFSET_X,
    offset_y: float = DETTAGLIO_OFFSET_Y,
    nome_file: str = "",
) -> dict:
    try:
        from PIL import Image
    except ImportError as exc:
        raise RuntimeError("Manca Pillow. Installa con: pip install pillow") from exc

    src = master_path or MASTER_DEFAULT
    if not src.is_file():
        raise FileNotFoundError(f"Immagine master non trovata: {src}")

    with Image.open(src) as img:
        img = img.convert("RGB")
        width, height = img.size
        variant = _trasforma(
            img,
            width=width,
            height=height,
            crop_ratio=crop_ratio,
            offset_x=offset_x,
            offset_y=offset_y,
        )
        out_path = _salva_variante(variant, "dettaglio", nome_file)

    return risultato(
        out_path,
        "augmentation:dettaglio",
        master=path_relativo(src),
        tipo="dettaglio",
        crop_ratio=crop_ratio,
        offset_x=offset_x,
        offset_y=offset_y,
        width=width,
        height=height,
    )


def crea_variante_dettaglio_2(
    master_path: Path | None = None,
    *,
    crop_ratio: float = DETTAGLIO_2_CROP_RATIO,
    offset_x: float = DETTAGLIO_2_OFFSET_X,
    offset_y: float = DETTAGLIO_2_OFFSET_Y,
    nome_file: str = "",
) -> dict:
    return crea_variante_dettaglio(
        master_path,
        crop_ratio=crop_ratio,
        offset_x=offset_x,
        offset_y=offset_y,
        nome_file=nome_file,
    )


def crea_varianti(master_path: Path | None = None) -> list[dict]:
    """Dalla master produce inquadratura spostata e due zoom su dettaglio."""
    master = master_path or MASTER_DEFAULT
    return [
        crea_variante_inquadratura(master),
        crea_variante_dettaglio(master),
        crea_variante_dettaglio_2(master),
    ]


def main() -> None:
    parser = argparse.ArgumentParser(
        description=(
            "Crea varianti da immagine_master.png: inquadratura spostata "
            "e due zoom su dettaglio."
        )
    )
    parser.add_argument(
        "--master",
        default="",
        help="Percorso immagine sorgente (default: immagini/immagine_master.png)",
    )
    parser.add_argument(
        "--solo",
        choices=("tutte", "inquadratura", "dettaglio", "dettaglio2"),
        default="tutte",
        help="Quale variante generare (default: tutte)",
    )
    args = parser.parse_args()

    master = Path(args.master) if args.master.strip() else None
    if args.solo == "tutte":
        results = crea_varianti(master)
    elif args.solo == "inquadratura":
        results = [crea_variante_inquadratura(master)]
    elif args.solo == "dettaglio":
        results = [crea_variante_dettaglio(master)]
    else:
        results = [crea_variante_dettaglio_2(master)]

    for result in results:
        print("---")
        for key, val in result.items():
            print(f"{key.upper()}={val}")


if __name__ == "__main__":
    main()
