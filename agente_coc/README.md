# Server MCP

Espone le operazioni del chaincode `reperto` e i generatori in `generazione_reperti/` come tool MCP. Un client MCP (IDE, desktop agent, ecc.) invoca i tool; `server.py` traduce in subprocess verso `bin/upload`, `bin/download`, `bin/workflow` o verso i moduli Python.

L'autorizzazione resta su Fabric: ogni transazione è firmata con il certificato Admin dell'org competente, come da CLI.

## Architettura

```
client MCP
    ↓  tool call (stdio)
server.py
    ├→ bin/upload, bin/download, bin/workflow  →  Fabric Gateway
    └→ generazione_reperti/                    →  Ollama / Stable Diffusion
```

### Routing peer

Le connessioni gRPC passano da `peer0.pg.it` per PG e PM (il peer PM non ha sempre la PDC PG-PM aggiornata via gossip). Per il LAB si usa `peer0.lab.it` quando scrive in `collezione_PM_LAB`. La collection policy valuta l'MSP del firmatario, non il peer fisico.

## Avvio

Prerequisiti: rete Fabric attiva, IPFS per documenti/evidenze, binari in `bin/`.

```bash
pip install fastmcp
python3 agente_coc/server.py
```

Configurare il client MCP con comando `python3` e argomento il path assoluto di `server.py` (transport stdio).

## Tool

### Reperto e ciclo di vita

| Tool | Org | Chaincode |
|------|-----|-----------|
| `crea_reperto` | PG | `CreaReperto` |
| `leggi_reperto` | * | `ReadReperto` |
| `storia_reperto` | * | `OttieniStoriaReperto` |
| `richiedi_analisi` | PM | `RichiediAnalisi` |
| `avvia_trasporto` | PG | `AvviaTrasporto` |
| `ricevi_in_laboratorio` | LAB | `RiceviInLaboratorio` |
| `completa_analisi` | LAB | `CompletaAnalisi` |
| `prepara_riconsegna` | LAB | `PreparaRiconsegna` |
| `deposita_in_sede` | PG | `DepositaInSede` |

### Documenti, evidenze, generazione

| Tool | Binario / modulo |
|------|------------------|
| `registra_documento` | upload |
| `registra_evidenza` | upload |
| `leggi_documento`, `leggi_evidenza` | workflow |
| `scarica_documento`, `scarica_evidenza` | download |
| `genera_evidenza_testo`, `genera_evidenza_immagine` | generazione_reperti |

Tipi documento: `VERBALE_SOPRALLUOGO`, `VERBALE_SEQUESTRO`, `DECRETO_ACCERTAMENTO`, `RELAZIONE_TECNICA`, `VERBALE_RICONSEGNA`.

Le descrizioni complete dei parametri sono nei docstring di `server.py` (usati dal protocollo MCP).
