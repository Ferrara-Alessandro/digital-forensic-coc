# Chain of custody (CoC) su Hyperledger Fabric

Sistema di chain of custody digitale per reperti fisici in ambito giudiziario. Ogni passaggio di custodia, dal sequestro PG all'analisi in laboratorio e alla riconsegna, è tracciato su un ledger Hyperledger Fabric condiviso tra tre organizzazioni (PG, PM, LAB).

I file (verbali, relazioni, evidenze digitali) vengono cifrati e caricati su IPFS locale; sul ledger restano CID e metadati. I dati sensibili e le chiavi AES stanno nelle collezioni dati privati (PDC) di Fabric, accessibili solo alle org autorizzate.

Il server MCP in `agente_coc/` espone le stesse operazioni dei client Go come tool invocabili da un agente conversazionale. Il modulo `generazione_reperti/` produce file di prova (testo via Ollama, immagini via Stable Diffusion) per testare upload e registrazione senza dati reali.

---

## Struttura del repository

| Cartella | Contenuto |
|----------|-----------|
| `chaincode/` | Smart contract Go: stati reperto, documenti tipizzati, evidenze, PDC |
| `cmd/upload/` | Cifra, carica su IPFS, registra sul ledger |
| `cmd/download/` | Legge dal ledger, scarica da IPFS, decifra |
| `cmd/workflow/` | Ciclo di vita reperto e query (senza IPFS) |
| `agente_coc/` | Server MCP sopra i binari Go |
| `generazione_reperti/` | Generatori sintetici testo/immagini |
| `infrastruttura_blockchain/` | `configtx.yaml`, Docker Compose, `collections_config.json` |
| `scripts/` | Helper operativi PG/PM/LAB e deploy rete |
| `tests/` | Validazione funzionale (`ciclo_vita/`), benchmark Caliper e cifratura/IPFS |

Dettagli per modulo: README nelle rispettive cartelle.

---

## Avvio rapido

Prerequisiti: Docker, Go 1.23+, IPFS Kubo (`ipfs daemon` su porta 5001).

```bash
# 1. Rete Fabric + chaincode
bash scripts/network/bootstrap_rete.sh
bash scripts/network/deploy_lifecycle_reperto.sh

# 2. Binari client
cd cmd/upload   && go build -o ../../bin/upload .
cd ../download  && go build -o ../../bin/download .
cd ../workflow  && go build -o ../../bin/workflow .

# 3. Smoke test workflow PG → PM → LAB
bash tests/ciclo_vita/run.sh
```

Per il server MCP: `pip install fastmcp`, poi `python3 agente_coc/server.py` (vedi `agente_coc/README.md`).

---

## Le tre entità sul ledger

### Reperto

Oggetto centrale. `CreaReperto` scrive sul world state la parte pubblica (`RepertoPublic`: stato, custode, riferimenti atti) e in `collezione_PG_PM` la parte riservata (`RepertoPrivate`: caso, agente, distretto, descrizione) via transient `reperto_privato`.

Ciclo di vita:

```
SEQUESTRATO → ATTESA_TRASPORTO → IN_TRANSITO → IN_ANALISI → ATTESA_RITIRO → IN_TRANSITO → SEQUESTRATO
```

Ogni transizione è vincolata per org: solo il PM chiama `RichiediAnalisi`, solo la PG `AvviaTrasporto`, solo il LAB `RiceviInLaboratorio` / `CompletaAnalisi`, ecc.

### Documento

Atto giudiziario tipizzato (verbale sopralluogo/sequestro/riconsegna, decreto, relazione). Il tipo è una costante; stringhe arbitrarie non sono accettate.

| Tipo | Collezione PDC |
|------|----------------|
| `VERBALE_SOPRALLUOGO`, `VERBALE_SEQUESTRO`, `VERBALE_RICONSEGNA` | `collezione_PG_PM` |
| `DECRETO_ACCERTAMENTO`, `RELAZIONE_TECNICA` | `collezione_PM_LAB` |

Alcuni documenti nascono dal workflow (`RichiediAnalisi` registra il decreto); altri si caricano con `cmd/upload`.

### Evidenza

Materiale digitale generico (immagini, dump, log). Classe libera, sempre in `collezione_PG_PM`.

---

## PDC e transient

| Collezione | Org | Contenuto |
|------------|-----|-----------|
| `collezione_PG_PM` | PG, PM | Dati reperto, verbali PG, evidenze, verbale riconsegna |
| `collezione_PM_LAB` | PM, LAB | Decreto, relazione, copia dati reperto per il lab |

CID e chiave AES viaggiano nel campo **transient** della transazione (non finiscono sul ledger pubblico). Configurazione: `infrastruttura_blockchain/collections_config.json`.

---

## IPFS e cifratura

1. `cmd/upload` cifra in streaming AES-256-GCM (formato `EV2`, chunk 4 MB).
2. Il blob va su IPFS locale; si ottiene il CID.
3. Invoke chaincode con CID + chiave in transient; la chiave resta in PDC.
4. `cmd/download` legge PDC, scarica da IPFS, decifra.

---

## Infrastruttura rete

| File | Ruolo |
|------|--------|
| `definizione_organizzazioni.yaml` | Input cryptogen (peer + Admin per org) |
| `configtx.yaml` | Genesis e canale `canale-coc` |
| `collections_config.json` | Definizione PDC per il lifecycle |
| `avvio_nodi.yaml` / `peer_nodi.yaml` | Docker Compose orderer e peer |

Generati in locale (non in Git): `certificati_pki/`, `*.block`, `reperto_*.tar.gz`. Si rigenerano dagli yaml con `scripts/network/bootstrap_rete.sh`.

---

## Esempio d'uso

Scheda reperto (senza file):

```bash
./bin/upload -mode reperto \
  -id-reperto REP-001 -id-caso CASO-1 -id-agente AG-1 -id-distretto DIST-1 \
  -descrizione-bene "Smartphone sequestrato" \
  -pki ./infrastruttura_blockchain/certificati_pki
```

Verbale cifrato su IPFS + ledger:

```bash
./bin/upload -mode documento \
  -file ./verbale_sequestro.pdf -id-caso CASO-1 -id-documento DOC-SEQ-001 \
  -tipo-documento VERBALE_SEQUESTRO -id-reperto-documento REP-001 \
  -descrizione-documento "Verbale di sequestro" \
  -pki ./infrastruttura_blockchain/certificati_pki
```

Recupero:

```bash
./bin/download -mode documento \
  -id-caso CASO-1 -id-documento DOC-SEQ-001 \
  -ipfs-api http://127.0.0.1:5001 \
  -pki ./infrastruttura_blockchain/certificati_pki \
  -out-dir ./downloads
```

---

## Test e risultati

```bash
bash tests/ciclo_vita/run.sh              # validazione funzionale end-to-end
bash tests/fabric_caliper/run.sh read     # benchmark query (Caliper)
bash tests/cifratura_ipfs/run.sh all      # benchmark cifratura + IPFS
```

Output di riferimento in `tests/results/` (log workflow, CSV Caliper/cifratura, report HTML). Vedi `tests/README.md`.
