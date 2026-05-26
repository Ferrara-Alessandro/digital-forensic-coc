# Server MCP per la catena di custodia

Server MCP in Python che espone le operazioni del chaincode `reperto` come tool utilizzabili da un agente AI. Qualsiasi client compatibile con il protocollo MCP (Cursor, Claude Desktop, ecc.) può invocare le funzioni Fabric in linguaggio naturale senza conoscere la CLI o la struttura del ledger.

## Come funziona

Il server fa da ponte tra l'agente e i client Go già esistenti:

```
agente AI (Cursor)
    ↓  MCP tool call
server.py
    ↓  subprocess
cmd/upload    →  cifra + IPFS + Fabric   (reperto, documenti, evidenze)
cmd/download  →  Fabric + IPFS + decifra
cmd/workflow  →  solo Fabric             (ciclo di vita reperto)
```

L'agente non conosce certificati, porte o flag: scrive in italiano e il server traduce in invocazioni concrete. L'autorizzazione rimane interamente a carico di Fabric — ogni operazione è firmata con il certificato Admin dell'organizzazione corretta.

## Tool disponibili

| Tool | Org | Descrizione |
|------|-----|-------------|
| `crea_reperto` | PG | Registra un nuovo reperto sequestrato sul ledger |
| `leggi_reperto` | qualsiasi | Legge stato, custode e riferimenti documenti |
| `storia_reperto` | qualsiasi | Cronologia completa delle transazioni sul reperto |
| `richiedi_analisi` | PM | Collega decreto e laboratorio, porta a ATTESA_TRASPORTO |
| `avvia_trasporto` | PG | Segna il reperto in viaggio, porta a IN_TRANSITO |
| `ricevi_in_laboratorio` | LAB | Conferma ricezione, porta a IN_ANALISI |
| `completa_analisi` | LAB | Carica relazione tecnica, porta a ATTESA_RITIRO |
| `prepara_riconsegna` | LAB | Verifica che il reperto sia pronto per il ritiro |
| `deposita_in_sede` | PG | Registra verbale di riconsegna e chiude il ciclo |
| `registra_documento` | varia | Cifra file, carica su IPFS, registra atto tipizzato |
| `registra_evidenza` | PG | Cifra file, carica su IPFS, registra materiale generico |
| `scarica_documento` | PG/PM | Recupera e decifra un documento dal ledger e IPFS |
| `scarica_evidenza` | PG/PM | Recupera e decifra un'evidenza dal ledger e IPFS |

## Prerequisiti

- Rete Fabric attiva (`scripts/network/bootstrap_rete.sh` + `deploy_lifecycle_reperto.sh`)
- IPFS in esecuzione locale (`ipfs daemon`) per le operazioni su documenti ed evidenze
- Binari Go compilati in `bin/` (`go build` in `cmd/upload`, `cmd/download`, `cmd/workflow`)
- Python 3 con `fastmcp` installato (`pip install fastmcp`)

## Configurazione in Cursor

Aggiungi a `~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "coc-fabric": {
      "command": "python3",
      "args": ["/percorso/assoluto/agente_coc/server.py"]
    }
  }
}
```

Riavvia Cursor. I tool compaiono in `Settings → MCP → coc-fabric`.

## Avvio manuale (debug)

```bash
python3 agente_coc/server.py
```

## Note

I client Go leggono i certificati da `infrastruttura_blockchain/certificati_pki/`. Le chiavi private generate da Docker appartengono a `root`: se il server restituisce *permission denied*, esegui:

```bash
sudo find infrastruttura_blockchain/certificati_pki -name "priv_sk" \
  -exec chown $USER {} \; -exec chmod 600 {} \;
```
