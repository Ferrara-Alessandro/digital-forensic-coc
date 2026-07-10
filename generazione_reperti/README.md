# Generazione reperti

Script per file di evidenza sintetici nei test di upload/registrazione.

| Script | Output |
|--------|--------|
| `testi/generatore_testo.py` | `.txt` in `testi/` |
| `immagini/generatore.py` | `.png` in `immagini/` |
| `immagini/generatore_master.py` | `immagine_master.png` (copia o SD) |
| `immagini/modificatore_inquadratura.py` | `variante_*.png`, due `dettaglio_*.png` |

Anche via MCP: `genera_evidenza_testo`, `genera_evidenza_immagine` in `agente_coc/server.py`.

## Setup

Testi: Ollama su `127.0.0.1:11434`:

```bash
ollama pull llama3
```

Immagini: venv locale (non usare pip di sistema su Debian/WSL):

```bash
bash generazione_reperti/setup.sh
```

Primo avvio SD: download modello v1-5 (~4 GB). Senza GPU recente resta su CPU.

## Esempi

```bash
python3 generazione_reperti/testi/generatore_testo.py \
  --descrizione "Log accessi al NAS caso Rossi"

bash generazione_reperti/run_python.sh generazione_reperti/immagini/generatore.py \
  --descrizione "impronta digitale su schermo smartphone, foto macro"

python3 generazione_reperti/immagini/generatore_master.py --input /percorso/foto.png
python3 generazione_reperti/immagini/modificatore_inquadratura.py
```

Il campo `file` nell'output va passato a `registra_evidenza` o a `cmd/upload -mode evidenza`.
