# Generazione reperti

Script Python per produrre file di evidenza sintetici (testo e immagini) da usare nei test del flusso upload/registrazione. 

## Componenti

| Script | Output | Dipendenze |
|--------|--------|------------|
| `testi/generatore_testo.py` | `.txt` in `testi/` | Ollama opzionale; senza Ollama usa template |
| `immagini/generatore.py` | `.png` in `immagini/` | torch, diffusers (venv locale) |

Il server MCP (`agente_coc/server.py`) richiama le stesse funzioni tramite i tool `genera_evidenza_testo` e `genera_evidenza_immagine`.

## Setup

Testi — [Ollama](https://ollama.com/) in ascolto su `127.0.0.1:11434`:

```bash
ollama pull llama3
```

Immagini — venv in `generazione_reperti/.venv` (non usare `pip` di sistema su Debian/WSL):

```bash
bash generazione_reperti/setup.sh
```

Al primo avvio il generatore immagini scarica il modello SD v1-5 (~4 GB). Senza GPU NVIDIA recente la generazione resta su CPU.

## Esempi

```bash
python3 generazione_reperti/testi/generatore_testo.py \
  --descrizione "Log accessi al NAS caso Rossi"

bash generazione_reperti/run_python.sh generazione_reperti/immagini/generatore.py \
  --descrizione "impronta digitale su schermo smartphone, foto macro"
```

Il campo `file` restituito va passato a `registra_evidenza` o al client upload.
