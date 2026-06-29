# Chaincode `reperto`

Smart contract Fabric (Go, contract-api) per la custodia del reperto fisico e per la registrazione di documenti tipizzati ed evidenze collegate a un caso (contenuti su IPFS, puntatori e chiavi in PDC).

## Struttura

| File | Ruolo |
|------|--------|
| `main.go` | Entry point del chaincode |
| `internal/contract/contract.go` | Struct `SmartContract` |
| `reperto_contract.go` | `CreaReperto`, `ReadReperto`, `RepertoExists` |
| `reperto_helpers.go` | MSP, lettura/scrittura ledger e PDC |
| `reperto_workflow.go` | Macchina a stati (analisi, trasporto, laboratorio, riconsegna) |
| `documento.go` | Verbali, decreto, relazione tecnica |
| `evidenza.go` | Materiale digitale generico |
| `storage_common.go` | Nomi PDC, transient, validazione chiavi |
| `internal/model/` | Struct JSON sul ledger |

## Tre entità

1. **Reperto** — chiave world state `idReperto`; metadati investigativi in `collezione_PG_PM` (transient `reperto_privato` alla creazione).
2. **Documento** — chiave composite `DOC`; tipo atto fisso; PDC `collezione_PG_PM` o `collezione_PM_LAB`; transient `documento` con cid e `chiaveCifrata`.
3. **Evidenza** — chiave `EVI`; classe libera; PDC sempre `collezione_PG_PM`; transient `evidenza`.

`CreaReperto` non carica file: dopo la creazione registro allegati con `RegistraDocumentoConTransient` / `RegistraEvidenzaConTransient` (client in `cmd/upload`).

## Build

```bash
cd chaincode && go build -o /dev/null .
```

Al deploy lifecycle passo `infrastruttura_blockchain/collections_config.json` (nomi collezioni allineati a `storage_common.go`).
