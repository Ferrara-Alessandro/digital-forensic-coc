# Server MCP per la catena di custodia

Server MCP in Python che espone le operazioni del chaincode `reperto` come tool utilizzabili da un agente AI. Qualsiasi client compatibile con il protocollo MCP (Cursor, Claude Desktop, ecc.) può invocare le funzioni Fabric in linguaggio naturale senza conoscere la CLI o la struttura del ledger.

## Come funziona

### Architettura a strati

```
agente AI (Cursor)
    ↓  MCP tool call  (JSON-RPC su stdio, protocollo MCP)
server.py  (FastMCP)
    ↓  subprocess
cmd/upload    →  cifra + IPFS + Fabric   (reperto, documenti, evidenze)
cmd/download  →  Fabric + IPFS + decifra
cmd/workflow  →  solo Fabric             (ciclo di vita reperto)
    ↓
Fabric Gateway  →  chaincode reperto  (gRPC + TLS)
```

L'agente non conosce certificati, porte o flag: scrive in linguaggio naturale e il server traduce in invocazioni concrete. L'autorizzazione rimane interamente a carico di Fabric — ogni operazione è firmata con il certificato Admin dell'organizzazione corretta.

### Come l'agente si approccia al chaincode

L'agente AI non può invocare Hyperledger Fabric direttamente: Fabric espone un'interfaccia gRPC con autenticazione TLS mutual e certificati X.509, incompatibile con le capacità native di un LLM. Il server MCP risolve questo disaccoppiamento esponendo ogni operazione del chaincode come un **tool** con nome, descrizione e schema dei parametri leggibili dall'agente.

Quando l'agente decide di eseguire un'operazione (es. registrare un documento), il client MCP nel suo runtime:
1. Invia una chiamata `tools/call` al server con nome del tool e argomenti in JSON
2. `server.py` traduce la chiamata nei flag del binario Go corrispondente e lancia il subprocess
3. Il binario apre una connessione Fabric Gateway verso `peer0.pg.it`, firma la transazione con il certificato Admin dell'org appropriata e attende il commit
4. L'output JSON del binario viene restituito all'agente come risultato del tool

L'agente riceve quindi una risposta strutturata (transaction ID, CID IPFS, tempi) e può usarla per i passi successivi del flusso.

### Routing delle org e accesso ai dati privati

Le operazioni sono firmate con l'identità dell'org che ha autorità su quella transizione di stato. Tutte le connessioni fisiche transitano per `peer0.pg.it`, che è il peer con i dati PDC sempre aggiornati:

| Org firmataria | Peer fisico usato | Motivo |
|---|---|---|
| PG | peer0.pg.it | peer di appartenenza |
| PM | peer0.pg.it | PM non riceve i dati PDC via gossip in modo sincrono; PG ha sempre `collezione_PG_PM` |
| LAB | peer0.lab.it | LAB scrive direttamente in `collezione_PM_LAB`, i dati sono immediatamente disponibili sul suo peer |

Fabric valuta la collection policy basandosi sul MSP del firmatario, non sul peer fisico che riceve la richiesta: PM ottiene solo i dati di `collezione_PG_PM`, LAB ottiene solo i dati di `collezione_PM_LAB`.

## Tool disponibili

### Reperto

| Tool | Org | Funzione chaincode | Descrizione |
|------|-----|--------------------|-------------|
| `crea_reperto` | PG | `CreaReperto` | Registra un nuovo reperto sequestrato sul ledger |
| `leggi_reperto` | qualsiasi | `ReadReperto` | Legge stato, custode e riferimenti documenti |
| `storia_reperto` | qualsiasi | `OttieniStoriaReperto` | Cronologia completa delle transazioni sul reperto |

### Ciclo di vita

| Tool | Org | Funzione chaincode | Stato richiesto → risultante |
|------|-----|--------------------|------------------------------|
| `richiedi_analisi` | PM | `RichiediAnalisi` | SEQUESTRATO → ATTESA_TRASPORTO |
| `avvia_trasporto` | PG | `AvviaTrasporto` | ATTESA_TRASPORTO / ATTESA_RITIRO → IN_TRANSITO |
| `ricevi_in_laboratorio` | LAB | `RiceviInLaboratorio` | IN_TRANSITO → IN_ANALISI |
| `completa_analisi` | LAB | `CompletaAnalisi` | IN_ANALISI → ATTESA_RITIRO |
| `prepara_riconsegna` | LAB | `PreparaRiconsegna` | ATTESA_RITIRO → ATTESA_RITIRO (prepara) |
| `deposita_in_sede` | PG | `DepositaInSede` | IN_TRANSITO → SEQUESTRATO |

### Documenti ed evidenze

| Tool | Org | Binario | Descrizione |
|------|-----|---------|-------------|
| `registra_documento` | varia | upload | Cifra file, carica su IPFS, registra atto tipizzato; restituisce CID e `chiaveB64` |
| `registra_evidenza` | PG | upload | Cifra file, carica su IPFS, registra materiale generico; restituisce CID e `chiaveB64` |
| `leggi_documento` | varia | workflow | Legge metadati e chiave cifrata di un documento dalla PDC |
| `leggi_evidenza` | varia | workflow | Legge metadati e chiave cifrata di un'evidenza dalla PDC |
| `scarica_documento` | varia | download | Recupera dal ledger, scarica da IPFS e decifra un documento |
| `scarica_evidenza` | varia | download | Recupera dal ledger, scarica da IPFS e decifra un'evidenza |

### Tipi di documento supportati

| Tipo | Org firmataria | Collezione PDC |
|------|----------------|----------------|
| `VERBALE_SOPRALLUOGO` | PG | `collezione_PG_PM` |
| `VERBALE_SEQUESTRO` | PG | `collezione_PG_PM` |
| `DECRETO_ACCERTAMENTO` | PM | `collezione_PM_LAB` |
| `RELAZIONE_TECNICA` | LAB | `collezione_PM_LAB` |
| `VERBALE_RICONSEGNA` | PG | `collezione_PG_PM` |

## Prerequisiti

- Rete Fabric attiva (`scripts/network/bootstrap_rete.sh` + `deploy_lifecycle_reperto.sh`)
- IPFS in esecuzione locale (`ipfs daemon`) per le operazioni su documenti ed evidenze
- Binari Go compilati in `bin/` (`go build` in `cmd/upload`, `cmd/download`, `cmd/workflow`)
- Python 3 con `fastmcp` installato (`pip install fastmcp`)