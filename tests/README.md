# Suite di test e benchmark

Tre cartelle = tre strumenti.

## Prerequisiti

```bash
bash scripts/network/bootstrap_rete.sh
bash scripts/network/deploy_lifecycle_reperto.sh
ipfs daemon   # in un terminale separato (porta 5001)
```

I test verificano automaticamente:
- rete Fabric attiva e **peer allineati** (stessa altezza blocco su PG/PM/LAB);
- IPFS raggiungibile (POST su `/api/v0/id`).

Se `ciclo_vita` fallisce per peer disallineati, riavvia la rete (comandi sopra).

## Struttura

```
tests/
├── ciclo_vita/           workflow completo reperto (PG→PM→LAB)
├── fabric_caliper/       throughput/latenza chaincode
│   └── caliper/          workspace Hyperledger Caliper
├── cifratura_ipfs/       cifratura, upload IPFS, download, decifra
├── lib/                  funzioni condivise
└── results/              CSV generati al run
```

| Cartella | Cosa fa | Strumento |
|----------|---------|-----------|
| **`ciclo_vita/`** | Workflow end-to-end del reperto | Bash + `peer` Docker |
| **`fabric_caliper/`** | Benchmark query/invoke chaincode | **Hyperledger Caliper** |
| **`cifratura_ipfs/`** | Benchmark cifratura, IPFS, download | **`bin/upload`** + **`bin/download`** |

## Esecuzione

```bash
bash tests/run_all.sh
bash tests/ciclo_vita/run.sh
bash tests/fabric_caliper/run.sh read          # 5 livelli TPS, 60s ciascuno (~6 min)
bash tests/fabric_caliper/run.sh query-mix     # 3 livelli, 60s (~4 min)
bash tests/fabric_caliper/run.sh create        # 3 livelli invoke (~4 min)
bash tests/fabric_caliper/run.sh all           # tutto (~25–40 min)
bash tests/cifratura_ipfs/run.sh upload
bash tests/cifratura_ipfs/run.sh download
```

## Caliper — copertura e limiti

**Funzionante e previsto per la tesi (prestazioni chaincode):**

| Workload | Funzioni chaincode | Stato |
|----------|-------------------|--------|
| `read` | `ReadReperto` @ 5/10/25/50/75 TPS × 60s | OK |
| `query-mix` | mix query @ 10/25/50 TPS × 60s | OK |
| `create` | `CreaReperto` @ 0,5/1/2 TPS × 60s (PG+PM) | OK |

**Non in Caliper (by design — altro strumento):**

| Cosa | Dove |
|------|------|
| Workflow multi-step (`RichiediAnalisi`, `RiceviInLaboratorio`, …) | `ciclo_vita/` |
| Cifratura AES, IPFS, decifra | `cifratura_ipfs/` |
| `RegistraDocumentoConTransient` / `RegistraEvidenzaConTransient` | `cifratura_ipfs/` + `ciclo_vita/` |
| Query su documenti/evidenze (`LeggiDocumento`, …) | Coperte funzionalmente in `ciclo_vita/`, non benchmarkate con Caliper |

Il workload **`create`** invoca `CreaReperto` con endorsement **PG+PM** tramite `targetPeers` (connettore `fabric-network` 2.2). Non usare il bind `fabric-gateway`: quel connettore non supporta invoke multi-peer.

Report HTML: `fabric_caliper/caliper/report.html`

## Perché Caliper (e non solo script bash)

Caliper è lo strumento standard per **benchmark ripetibili** su Fabric: controllo del rate (TPS), worker paralleli, report HTML con latenza/throughput, mix di query sotto carico. Bash + `peer invoke` serve al **workflow corretto** (`ciclo_vita/`), non a misurare *N transazioni al secondo per 30 secondi* con statistiche aggregate.

## Output Caliper

- HTML archiviato: `tests/results/caliper/report-<tipo>-<timestamp>.html`
- CSV cumulativo: `tests/results/caliper/caliper_summary.csv` (per tabella in tesi)
- Dettagli matrice di carico: `tests/fabric_caliper/README.md`

## Output CSV cifratura

- `results/cifratura_upload_*.csv` — tempi encrypt/IPFS per dimensione file
- `results/cifratura_download_*.csv` — round-trip upload+download

Variabili: `BENCH_REPERTO_ID`, `BENCH_SIZES_MB`, `BENCH_REPEATS`, `FABRIC_SYNC_TIMEOUT_SEC`, `TEST_RESULTS_DIR`.
