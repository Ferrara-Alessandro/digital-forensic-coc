# Workflow

Client Go per le operazioni di ciclo di vita del reperto su Hyperledger Fabric. A differenza di `cmd/upload` e `cmd/download`, non interagisce con IPFS: invoca solo funzioni del chaincode che cambiano lo stato del reperto o lo leggono.

## Modalità

| Mode | Funzione chaincode | Org | Argomenti principali |
|------|--------------------|-----|----------------------|
| `leggi-reperto` | `ReadReperto` | qualsiasi | `-id-reperto` |
| `storia-reperto` | `OttieniStoriaReperto` | qualsiasi | `-id-reperto` |
| `richiedi-analisi` | `RichiediAnalisi` | PM | `-id-reperto`, `-id-lab`, `-tipo-analisi`, `-cid-decreto`, `-chiave-decreto` |
| `avvia-trasporto` | `AvviaTrasporto` | PG | `-id-reperto`, `-id-agente-pg` |
| `ricevi-laboratorio` | `RiceviInLaboratorio` | LAB | `-id-reperto`, `-id-laboratorio` |
| `completa-analisi` | `CompletaAnalisi` | LAB | `-id-reperto`, `-cid-relazione`, `-chiave-relazione` |
| `prepara-riconsegna` | `PreparaRiconsegna` | LAB | `-id-reperto` |
| `deposita-sede` | `DepositaInSede` | PG | `-id-reperto`, `-cid-verbale-riconsegna`, `-chiave-verbale-riconsegna` |

L'org predefinita è già quella corretta per ogni mode (pm per `richiedi-analisi`, lab per i tre di laboratorio, pg per il resto). Si può forzare con `-org pg|pm|lab`.

Ogni comando stampa un JSON con `transactionId` e tempi, oppure il payload restituito dal chaincode per le query.

## Build

```bash
cd cmd/workflow
go build -o ../../bin/workflow .
```

## Esempi

```bash
# Leggi lo stato attuale
./bin/workflow -mode leggi-reperto -id-reperto REP-001

# Il PM autorizza l'analisi (il decreto è già stato caricato su IPFS con cmd/upload)
./bin/workflow -mode richiedi-analisi -id-reperto REP-001 \
  -id-lab LAB-1 -tipo-analisi IMPRONTA_DIGITALE \
  -cid-decreto bafybeiabc123... -chiave-decreto base64==

# La PG avvia il trasporto
./bin/workflow -mode avvia-trasporto -id-reperto REP-001 -id-agente-pg AG-7

# Il LAB riceve, analizza, completa
./bin/workflow -mode ricevi-laboratorio -id-reperto REP-001 -id-laboratorio LAB-1
./bin/workflow -mode completa-analisi   -id-reperto REP-001 \
  -cid-relazione bafybeiabc456... -chiave-relazione base64==

# La PG chiude il ciclo
./bin/workflow -mode deposita-sede -id-reperto REP-001 \
  -cid-verbale-riconsegna bafybeiabc789... -chiave-verbale-riconsegna base64==

# Audit completo
./bin/workflow -mode storia-reperto -id-reperto REP-001
```
