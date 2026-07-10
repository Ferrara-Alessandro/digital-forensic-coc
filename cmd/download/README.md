# Download

Query su Fabric (`LeggiDocumento` / `LeggiEvidenza`), prelievo da IPFS in streaming, decifratura AES-256-GCM (`EV2`, allineato a `cmd/upload`).

## Modalità

| Mode | Query | Argomenti |
|------|-------|-----------|
| `documento` | `LeggiDocumento` | `-id-caso`, `-id-documento` |
| `evidenza` | `LeggiEvidenza` | `-id-caso`, `-id-evidenza` |

Le evaluate passano da `peer0.pg.it` con identità Admin dell'org indicata (`-org`, default `pg`). Fabric valuta l'accesso PDC sull'MSP del firmatario.

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

`-no-write` salta la scrittura su disco. `-skip-chaincode` con `-cid` e `-key-b64` salta Fabric (solo IPFS + decifra, per benchmark).
