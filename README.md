# Chain of custody (CoC) su Hyperledger Fabric

Questo progetto realizza un sistema di chain of custody digitale per reperti fisici in ambito giudiziario, basato su Hyperledger Fabric. L'idea è tracciare in modo immutabile ogni passaggio di custodia — dal sequestro iniziale da parte della PG fino all'analisi in laboratorio e alla riconsegna — senza affidarsi a un'unica autorità centrale. I documenti e le evidenze digitali collegati al reperto vengono cifrati e archiviati su IPFS; sul ledger rimangono solo i riferimenti e i metadati, con i dati sensibili accessibili esclusivamente alle organizzazioni autorizzate tramite le collezioni dati privati di Fabric.

L'automazione del processo è già integrata: il server MCP in `agente_coc/` espone tutte le operazioni Fabric come tool per agenti AI. Un agente può creare un reperto, avanzarne il ciclo di vita e interrogare il ledger in linguaggio naturale, senza conoscere la CLI o la struttura dei certificati. Il modulo di generazione automatica dei contenuti verrà aggiunto in una fase successiva.

---

## Contenuto del repository

| Cartella | Contenuto |
|----------|-----------|
| `chaincode/` | Smart contract Go: stati del reperto, documenti tipizzati, evidenze, collezioni PDC |
| `cmd/upload/` | Client caricamento: cifra il file, lo carica su IPFS, registra sul ledger |
| `cmd/download/` | Client recupero: legge i metadati dal ledger, scarica da IPFS, decifra |
| `cmd/workflow/` | Client operazioni di ciclo di vita: avanzamento stati, query reperto, storia transazioni |
| `agente_coc/` | Server MCP: espone le operazioni Fabric come tool per agenti AI (Cursor, Claude Desktop, ecc.) |
| `infrastruttura_blockchain/` | `configtx.yaml`, Docker Compose, cryptogen, definizione collezioni PDC |

Il modulo di generazione dei contenuti verrà aggiunto al repository in una fase successiva.

---

## Le tre entità sul ledger

Il chaincode distingue tre strutture separate, ognuna con la propria chiave e le proprie regole di accesso.

### Reperto

È l'oggetto centrale. Quando la PG sequestra un reperto fisico chiama `CreaReperto`: sul world state viene scritta la parte pubblica (`RepertoPublic`) con lo stato corrente e il custode; nella collezione privata `collezione_PG_PM` viene salvata la parte riservata (`RepertoPrivate`) con i dati del caso, l'agente, il distretto e la descrizione del bene.

La parte pubblica è visibile a tutti i peer della rete. La parte privata la leggono solo PG e PM.

Il reperto ha un ciclo di vita a stati fissi:

```
SEQUESTRATO → ATTESA_TRASPORTO → IN_TRANSITO → IN_ANALISI → ATTESA_RITIRO → IN_TRANSITO → SEQUESTRATO
```

Ogni passaggio di stato richiede che sia la giusta organizzazione a invocare la funzione: ad esempio solo il PM può chiamare `RichiediAnalisi`, solo la PG può chiamare `AvviaTrasporto`, solo il LAB può chiamare `RiceviInLaboratorio` e `CompletaAnalisi`.

### Documento

Rappresenta un atto giudiziario specifico collegato al caso: verbale di sopralluogo, verbale di sequestro, decreto di accertamento, relazione tecnica, verbale di riconsegna. Il tipo è una costante fissa; non accetto stringhe arbitrarie.

Sul world state va la parte pubblica (`DocumentoPublic`) con l'id documento e il CID IPFS. Nella collezione privata (`collezione_PG_PM` o `collezione_PM_LAB` a seconda del tipo) finiscono il tipo atto, la descrizione, l'autore, il timestamp e la chiave AES usata per cifrare il file.

La collezione dipende dal tipo documento:
- `VERBALE_SOPRALLUOGO`, `VERBALE_SEQUESTRO`, `VERBALE_RICONSEGNA` → `collezione_PG_PM`
- `DECRETO_ACCERTAMENTO`, `RELAZIONE_TECNICA` → `collezione_PM_LAB`

Alcuni documenti vengono registrati automaticamente dalle funzioni di workflow (ad esempio `RichiediAnalisi` registra il decreto); altri vengono caricati esplicitamente tramite `cmd/upload`.

### Evidenza

È materiale digitale generico legato al caso: immagini, dump, log, qualunque file che non rientra negli atti tipizzati. A differenza del documento non ha un tipo fisso ma una classe libera. Va sempre in `collezione_PG_PM`.

---

## Dati privati (PDC)

Le collezioni private di Fabric permettono che certi dati non transitino sulla rete pubblica tra i peer ma vengano distribuiti solo alle organizzazioni che ne hanno diritto.

| Collezione | Organizzazioni | Cosa contiene |
|------------|---------------|---------------|
| `collezione_PG_PM` | PG, PM | Dati investigativi del reperto, verbali PG, evidenze, verbale di riconsegna |
| `collezione_PM_LAB` | PM, LAB | Decreto di accertamento, relazione tecnica, copia dati reperto per il laboratorio |

Quando chiamo una funzione che scrive dati privati passo i dati in `transient` (non negli argomenti della transazione): in questo modo non compaiono mai nella proposta di transazione che gira tra i peer.

La configurazione delle collezioni è in `infrastruttura_blockchain/collections_config.json` e va passata al lifecycle durante il deploy del chaincode.

---

## IPFS e cifratura

I file non vanno mai sul ledger. Il flusso è:

1. `cmd/upload` cifra il file con AES-256-GCM a chunk (magic `EV2`, chunk di default 4 MB).
2. Il blob cifrato viene caricato su un nodo IPFS locale (Kubo API su `127.0.0.1:5001`); Kubo restituisce il CID.
3. `cmd/upload` invoca il chaincode via Fabric Gateway, passando CID e chiave AES in transient. La chiave viene salvata nella PDC; sul world state resta solo il CID.
4. Per recuperare: `cmd/download` legge i metadati dalla PDC (CID + chiave), scarica il blob da IPFS, decifra e scrive il file originale su disco.

La chiave AES non esce mai in chiaro dalla transazione: viaggia solo nel campo transient, che Fabric non scrive sul ledger pubblico.

---

## Infrastruttura rete

| File | Ruolo |
|------|-------|
| `definizione_organizzazioni.yaml` | Input `cryptogen`: definisce un peer per org e le identità Admin |
| `configtx.yaml` | Profili genesis e canale (`CocGenesis`, `CocChannel`), policy MSP, capabilities |
| `collections_config.json` | Definizione delle collezioni PDC da passare al lifecycle |
| `avvio_nodi.yaml` | Docker Compose: solo l'orderer (usa `genesis.block`) |
| `peer_nodi.yaml` | Docker Compose: peer0 di PG, PM e LAB |

Generati in locale (non in Git): `certificati_pki/`, `genesis.block`, `canale-coc.block`, `atti_canale/`, pacchetti chaincode `reperto_*.tar.gz`.

Per ricostruire la rete su un altro PC bastano i file di configurazione:

```
definizione_organizzazioni.yaml  →  cryptogen      →  certificati_pki/
configtx.yaml                    →  configtxgen    →  genesis.block, canale-coc.block
collections_config.json          →  lifecycle peer →  chaincode installato e approvato
```

---

## Build

```bash
cd chaincode && go build -o /dev/null .
cd cmd/upload && go build -o /dev/null .
cd cmd/download && go build -o /dev/null .
cd cmd/workflow && go build -o /dev/null .
```

---

## Esempio d'uso

Creo la scheda reperto (metadati riservati in transient, nessun file):

```bash
./bin/upload -mode reperto \
  -id-reperto REP-001 -id-caso CASO-1 -id-agente AG-1 -id-distretto DIST-1 \
  -descrizione-bene "Smartphone sequestrato" \
  -pki ./infrastruttura_blockchain/certificati_pki
```

Carico il verbale di sequestro cifrato su IPFS e lo registro sul ledger:

```bash
./bin/upload -mode documento \
  -file ./verbale_sequestro.pdf -id-caso CASO-1 -id-documento DOC-SEQ-001 \
  -tipo-documento VERBALE_SEQUESTRO -id-reperto-documento REP-001 \
  -descrizione-documento "Verbale di sequestro" \
  -pki ./infrastruttura_blockchain/certificati_pki
```

Recupero il documento:

```bash
./bin/download -mode documento \
  -id-caso CASO-1 -id-documento DOC-SEQ-001 \
  -ipfs-api http://127.0.0.1:5001 \
  -pki ./infrastruttura_blockchain/certificati_pki \
  -out-dir ./downloads
```
