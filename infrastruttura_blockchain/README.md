# Infrastruttura Hyperledger Fabric

Definizione della rete locale CoC (PG, PM, LAB) e delle collezioni dati privati usate dal chaincode `reperto`.

## File principali

| File | Ruolo |
|------|--------|
| `definizione_organizzazioni.yaml` | Input **cryptogen**: un peer per org, solo identità Admin (`Users: Count: 0`). |
| `configtx.yaml` | Profili genesis e canale applicativo, policy MSP, capabilities. |
| `collections_config.json` | Collezioni `collezione_PG_PM` e `collezione_PM_LAB` (deploy lifecycle). |
| `avvio_nodi.yaml` | Docker Compose: orderer (genesis `genesis.block`). |
| `peer_nodi.yaml` | Docker Compose: peer0 PG, PM, LAB. |

**Generati in locale (non in Git):** `certificati_pki/`, `genesis.block`, `canale-coc.block`, `atti_canale/`, `reperto_*.tar.gz`.

## Collezioni dati privati

- **collezione_PG_PM** — la vedono PG e PM: metadati del reperto, verbali PG, evidenze, verbale di riconsegna.
- **collezione_PM_LAB** — la vedono PM e LAB: decreto, relazione, copia dati reperto per il laboratorio.

Nel chaincode scelgo la collezione in base al tipo di documento. I nomi devono coincidere con `collections_config.json` e con le costanti in `storage_common.go`.

## Endorsement (sintesi)

In deploy ho messo una policy che accetta la firma di un solo peer tra PG, PM e LAB. Nel codice controllo comunque quale organizzazione invia la transazione.

Quando scrivo dati privati che devono finire subito su più nodi, nella pratica chiamo più peer nella stessa invoke (ad esempio PG e PM per la collezione PG-PM).

Per upload e download uso il certificato Admin di ciascuna organizzazione (`cmd/upload`, `cmd/download`).

## Ricostruzione su un altro PC

```text
definizione_organizzazioni.yaml  →  cryptogen  →  certificati_pki/
configtx.yaml                    →  configtxgen  →  blocchi canale
collections_config.json        →  lifecycle chaincode reperto
```

Non serve copiare certificati o `.block` da un repository: li rigenero con gli stessi file di configurazione.
