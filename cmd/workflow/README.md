# Workflow

Client Go per il ciclo di vita del reperto. Solo Fabric, niente IPFS.

## Modalità

| Mode | Chaincode | Org default |
|------|-----------|-------------|
| `leggi-reperto` | `ReadReperto` | pg |
| `storia-reperto` | `OttieniStoriaReperto` | pg |
| `richiedi-analisi` | `RichiediAnalisi` | pm |
| `avvia-trasporto` | `AvviaTrasporto` | pg |
| `ricevi-laboratorio` | `RiceviInLaboratorio` | lab |
| `completa-analisi` | `CompletaAnalisi` | lab |
| `prepara-riconsegna` | `PreparaRiconsegna` | lab |
| `deposita-sede` | `DepositaInSede` | pg |

Override org: `-org pg|pm|lab`. Output JSON con `transactionId` e tempi, oppure payload query.

## Build

```bash
cd cmd/workflow
go build -o ../../bin/workflow .
```

## Esempi

```bash
./bin/workflow -mode leggi-reperto -id-reperto REP-001

./bin/workflow -mode richiedi-analisi -id-reperto REP-001 \
  -id-lab LAB-1 -tipo-analisi IMPRONTA_DIGITALE \
  -cid-decreto bafybei... -chiave-decreto base64==

./bin/workflow -mode avvia-trasporto -id-reperto REP-001 -id-agente-pg AG-7
./bin/workflow -mode ricevi-laboratorio -id-reperto REP-001 -id-laboratorio LAB-1
./bin/workflow -mode completa-analisi -id-reperto REP-001 \
  -cid-relazione bafybei... -chiave-relazione base64==
./bin/workflow -mode deposita-sede -id-reperto REP-001 \
  -cid-verbale-riconsegna bafybei... -chiave-verbale-riconsegna base64==

./bin/workflow -mode storia-reperto -id-reperto REP-001
```

I CID e le chiavi dei documenti si ottengono da `cmd/upload` prima di `richiedi-analisi` / `completa-analisi` / `deposita-sede`.
