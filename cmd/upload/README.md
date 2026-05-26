# Upload

Client Go: cifratura AES-256-GCM, caricamento su IPFS (Kubo), invoke sul chaincode `reperto` via Fabric Gateway.

## Modalità

| Mode | Chaincode | File |
|------|-----------|------|
| `reperto` | `CreaReperto` | no |
| `documento` | `RegistraDocumentoConTransient` | sì (cifrato) |
| `evidenza` | `RegistraEvidenzaConTransient` | sì (cifrato) |

Transient:

- `reperto`: `reperto_privato` (JSON metadati caso / agente / distretto)
- `documento` / `evidenza`: `{"cid":"...","chiaveCifrata":"..."}` (base64, obbligatoria)

Per alcuni atti nella vita reale servono due peer nella stessa chiamata Fabric; il programma di upload firma come Admin dell’organizzazione scelta (`-ingest-org` o default in base al tipo documento).

## Build

```bash
cd cmd/upload
go build -o ../../bin/upload .
```

## Esempi

Scheda reperto (solo metadati privati in transient):

```bash
./bin/upload -mode reperto \
  -id-reperto REP-001 -id-caso CASO-1 -id-agente AG-1 -id-distretto DIST-1 \
  -descrizione-bene "Descrizione" \
  -pki ./infrastruttura_blockchain/certificati_pki
```

Verbale di sequestro:

```bash
./bin/upload -mode documento \
  -file ./verbale.pdf -id-caso CASO-1 -id-documento DOC-SEQ-001 \
  -tipo-documento VERBALE_SEQUESTRO -id-reperto-documento REP-001 \
  -descrizione-documento "Verbale di sequestro" \
  -pki ./infrastruttura_blockchain/certificati_pki
```

`-skip-chaincode` lascia solo cifratura e IPFS (prove locali senza Fabric).
