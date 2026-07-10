# Upload

Cifratura AES-256-GCM in streaming, upload IPFS (Kubo), invoke chaincode `reperto` via Fabric Gateway.

## Modalità

| Mode | Chaincode | File |
|------|-----------|------|
| `reperto` | `CreaReperto` | no |
| `documento` | `RegistraDocumentoConTransient` | sì |
| `evidenza` | `RegistraEvidenzaConTransient` | sì |

Transient:
- `reperto` → `reperto_privato` (JSON metadati caso/agente/distretto)
- `documento` / `evidenza` → `{"cid":"...","chiaveCifrata":"..."}` (base64)

Alcune invoke richiedono endorsement su più peer (es. PG+PM per `collezione_PG_PM`). Org firmataria: `-ingest-org` o default per tipo documento.

## Build

```bash
cd cmd/upload
go build -o ../../bin/upload .
```

## Esempi

```bash
./bin/upload -mode reperto \
  -id-reperto REP-001 -id-caso CASO-1 -id-agente AG-1 -id-distretto DIST-1 \
  -descrizione-bene "Descrizione" \
  -pki ./infrastruttura_blockchain/certificati_pki

./bin/upload -mode documento \
  -file ./verbale.pdf -id-caso CASO-1 -id-documento DOC-SEQ-001 \
  -tipo-documento VERBALE_SEQUESTRO -id-reperto-documento REP-001 \
  -descrizione-documento "Verbale di sequestro" \
  -pki ./infrastruttura_blockchain/certificati_pki
```

`-skip-chaincode`: solo cifratura + IPFS, senza invoke (benchmark off-chain).
