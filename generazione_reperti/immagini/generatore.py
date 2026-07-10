#!/usr/bin/env python3
# Genera PNG di evidenza con Stable Diffusion. Output in generazione_reperti/immagini/.

from __future__ import annotations

import argparse
import sys
from datetime import datetime
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))

from generazione_reperti.common import MODULO, risultato

OUTPUT_DIR = MODULO / "immagini"
MODEL_ID = "runwayml/stable-diffusion-v1-5"

_pipeline = None
_device: str | None = None


def _scegli_device() -> str:
    """CUDA se disponibile, altrimenti CPU."""
    import torch

    if not torch.cuda.is_available():
        return "cpu"

    # PyTorch recente non supporta GPU con compute capability < 7.5 (es. MX250).
    major, minor = torch.cuda.get_device_capability(0)
    if (major, minor) < (7, 5):
        print(
            f"GPU non supportata (CC {major}.{minor}); uso CPU.",
            file=sys.stderr,
        )
        return "cpu"

    try:
        torch.zeros(1, device="cuda")
        return "cuda"
    except Exception:
        print("CUDA non utilizzabile; uso CPU.", file=sys.stderr)
        return "cpu"


def _carica_pipeline():
    """Singleton pipeline SD (download ~4 GB al primo avvio)."""
    global _pipeline, _device
    if _pipeline is not None:
        return _pipeline, _device

    try:
        import torch
        from diffusers import StableDiffusionPipeline
    except ImportError as exc:
        raise RuntimeError(
            "Mancano torch/diffusers. Esegui: bash generazione_reperti/setup.sh"
        ) from exc

    _device = _scegli_device()
    dtype = torch.float16 if _device == "cuda" else torch.float32

    pipe = StableDiffusionPipeline.from_pretrained(MODEL_ID, torch_dtype=dtype)
    pipe = pipe.to(_device)
    if _device == "cpu":
        pipe.enable_attention_slicing()

    _pipeline = pipe
    return _pipeline, _device


def genera_immagine(
    descrizione: str,
    *,
    seed: int = 12345,
    num_inference_steps: int = 20,
    nome_file: str = "",
) -> dict:
    prompt = descrizione.strip()
    if not prompt:
        raise ValueError("descrizione vuota")

    pipe, device = _carica_pipeline()

    import torch

    generator = torch.Generator(device).manual_seed(seed)
    image = pipe(
        prompt,
        generator=generator,
        num_inference_steps=num_inference_steps,
    ).images[0]

    stamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    if nome_file.strip():
        name = nome_file.strip()
        if not name.lower().endswith(".png"):
            name += ".png"
    else:
        name = f"evidenza_{stamp}.png"

    out_path = OUTPUT_DIR / name
    out_path.parent.mkdir(parents=True, exist_ok=True)
    image.save(out_path)

    return risultato(
        out_path,
        f"stable-diffusion:{MODEL_ID}",
        prompt=prompt,
        seed=seed,
        device=device,
        steps=num_inference_steps,
    )


def main() -> None:
    parser = argparse.ArgumentParser(description="Genera immagine evidenza (Stable Diffusion).")
    parser.add_argument("--descrizione", required=True, help="Prompt")
    parser.add_argument("--seed", type=int, default=12345)
    parser.add_argument("--steps", type=int, default=20)
    parser.add_argument("--nome-file", default="")
    args = parser.parse_args()

    result = genera_immagine(
        args.descrizione,
        seed=args.seed,
        num_inference_steps=args.steps,
        nome_file=args.nome_file,
    )
    for key, val in result.items():
        print(f"{key.upper()}={val}")


if __name__ == "__main__":
    main()
