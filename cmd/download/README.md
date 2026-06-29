# Download

Client Go: query su Fabric (`LeggiDocumento` / `LeggiEvidenza`), download del blob da IPFS, decifratura AES-256-GCM (formato `EV2` allineato a `cmd/upload`).

## Modalità

| Mode | Query | Argomenti |
|------|-------|-----------|
| `documento` | `LeggiDocumento` | `-id-caso`, `-id-documento` |
| `evidenza` | `LeggiEvidenza` | `-id-caso`, `-id-evidenza` |

Uso l’Admin PG su `peer0.pg.it` per le evaluate (lettura metadati in PDC `collezione_PG_PM`).

## Build

```bash
cd cmd/download
go build -o ../../bin/download .
```

## Esempio

```bash
./bin/download -mode documento \
  -id-caso CASO-1 -id-documento DOC-SEQ-001 \
  -ipfs-api http://127.0.0.1:5001 \
  -pki ./infrastruttura_blockchain/certificati_pki \
  -out-dir ./downloads
```

`-no-write` evita la scrittura su disco e stampa solo i tempi nel JSON di uscita.
