#!/usr/bin/env python3
# Genera file testuali di evidenza per test e demo. Output in generazione_reperti/testi/.

from __future__ import annotations

import argparse
import json
import sys
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))

from generazione_reperti.common import risultato

OUTPUT_DIR = Path(__file__).resolve().parent
OLLAMA_HOST_DEFAULT = "http://127.0.0.1:11434"


def _infer_tipo(descrizione: str) -> str:
    d = descrizione.lower()
    if any(w in d for w in ("email", "mail", "posta")):
        return "Email"
    if any(w in d for w in ("chat", "whatsapp", "telegram", "messagg")):
        return "Chat"
    if any(w in d for w in ("log", "access", "syslog")):
        return "Log"
    return "Documento"


def _template(tipo: str, descrizione: str) -> str:
    """Fallback se Ollama non risponde: contenuto minimo ma coerente col tipo."""
    ts = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    if tipo == "Email":
        return (
            f"From: mittente@example.it\n"
            f"To: destinatario@example.it\n"
            f"Date: {ts}\n"
            f"Subject: {descrizione}\n\n"
            "Messaggio di prova per test catena di custodia.\n"
        )
    if tipo == "Chat":
        return (
            f"[{ts}] utente_A: messaggio di prova\n"
            f"[{ts}] utente_B: ok\n"
            f"Contesto: {descrizione}\n"
        )
    if tipo == "Log":
        return (
            f"{ts} INFO  avvio sessione\n"
            f"{ts} WARN  tentativo accesso non autorizzato\n"
            f"{ts} NOTE  contesto={descrizione}\n"
        )
    return f"Tipo: {tipo}\nContesto: {descrizione}\nGenerato: {ts}\n"


def _try_ollama(tipo: str, descrizione: str, model: str, host: str) -> str | None:
    prompt = (
        f"Scrivi un {tipo} fittizio in italiano, adatto a un test forense. "
        f"Contesto: {descrizione}. "
        "Restituisci solo il contenuto del file, senza spiegazioni."
    )
    payload = json.dumps(
        {
            "model": model,
            "prompt": prompt,
            "stream": False,
            "options": {"temperature": 0.4, "num_predict": 600},
        }
    ).encode()
    req = urllib.request.Request(
        f"{host.rstrip('/')}/api/generate",
        data=payload,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            data = json.loads(resp.read().decode())
        text = (data.get("response") or "").strip()
        return text or None
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError, KeyError):
        return None


def genera_testo(
    descrizione: str,
    *,
    tipo: str = "",
    model: str = "llama3",
    nome_file: str = "",
    ollama_host: str = OLLAMA_HOST_DEFAULT,
    no_llm: bool = False,
) -> dict:
    tipo_eff = (tipo or _infer_tipo(descrizione)).strip() or "Documento"
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

    body: str | None = None
    source = "template"
    if not no_llm:
        body = _try_ollama(tipo_eff, descrizione, model, ollama_host)
        if body:
            source = f"ollama:{model}"

    if body is None:
        body = _template(tipo_eff, descrizione)

    stamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    if nome_file.strip():
        out_name = nome_file.strip()
        if not out_name.endswith(".txt"):
            out_name += ".txt"
    else:
        safe_tipo = "".join(c if c.isalnum() else "_" for c in tipo_eff)[:24]
        out_name = f"reperto_{safe_tipo}_{stamp}.txt"

    out_path = OUTPUT_DIR / out_name
    out_path.write_text(body, encoding="utf-8")
    return risultato(out_path, source, tipo=tipo_eff)


def main() -> None:
    parser = argparse.ArgumentParser(description="Genera reperto testuale simulato.")
    parser.add_argument("--tipo", default="", help="Email, Chat, Log (opzionale)")
    parser.add_argument(
        "--descrizione-tecnica",
        "--descrizione",
        dest="descrizione_tecnica",
        required=True,
        help="Contesto o richiesta operatore",
    )
    parser.add_argument("--model", default="llama3", help="Modello Ollama")
    parser.add_argument("--nome-file", default="", help="Nome file in testi/")
    parser.add_argument("--ollama-host", default=OLLAMA_HOST_DEFAULT)
    parser.add_argument(
        "--no-llm",
        action="store_true",
        help="Salta Ollama e usa solo il template",
    )
    args = parser.parse_args()

    result = genera_testo(
        args.descrizione_tecnica,
        tipo=args.tipo,
        model=args.model,
        nome_file=args.nome_file,
        ollama_host=args.ollama_host,
        no_llm=args.no_llm,
    )
    for key, val in result.items():
        print(f"{key.upper()}={val}")


if __name__ == "__main__":
    main()
