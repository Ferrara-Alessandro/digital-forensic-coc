# Test e benchmark

Tre suite indipendenti: correttezza del workflow, prestazioni chaincode, prestazioni off-chain.

## Prerequisiti

```bash
bash scripts/network/bootstrap_rete.sh
bash scripts/network/deploy_lifecycle_reperto.sh
ipfs daemon   # terminale separato, porta 5001
```

I test controllano peer Docker attivi, allineamento blocco PG/PM/LAB e IPFS raggiungibile.

## Struttura

| Cartella | Scopo | Strumento |
|----------|-------|-----------|
| `ciclo_vita/` | Workflow reperto PG → PM → LAB | bash + `peer` in Docker |
| `fabric_caliper/` | Throughput/latenza chaincode | Hyperledger Caliper |
| `cifratura_ipfs/` | Cifratura, IPFS, round-trip | `bin/upload`, `bin/download` |

## Esecuzione

```bash
bash tests/run_all.sh
bash tests/ciclo_vita/run.sh
bash tests/fabric_caliper/run.sh read       # ~12 min
bash tests/fabric_caliper/run.sh query-mix  # ~12 min
bash tests/fabric_caliper/run.sh create     # ~14 min
bash tests/fabric_caliper/run.sh all        # ~40–50 min
bash tests/cifratura_ipfs/run.sh upload
bash tests/cifratura_ipfs/run.sh download
```

## Cosa copre cosa

| Funzionalità | Suite |
|--------------|-------|
| Workflow multi-step (`RichiediAnalisi`, `RiceviInLaboratorio`, …) | `ciclo_vita/` |
| `ReadReperto`, mix query, `CreaReperto` sotto carico | `fabric_caliper/` |
| Cifratura AES, upload/download IPFS | `cifratura_ipfs/` |
| `RegistraDocumentoConTransient` nel flusso reale | `ciclo_vita/` + `cifratura_ipfs/` |

Il workload Caliper `create` usa endorsement PG+PM (`targetPeers` nel connettore fabric-network 2.2). Non usare il bind `fabric-gateway` per invoke multi-peer.

## Output in `tests/results/`

| File | Contenuto |
|------|-----------|
| `ciclo_vita_workflow.log` | Log completo validazione funzionale |
| `caliper/caliper_summary.csv` | Tutte le run Caliper aggregate |
| `caliper/report-*.html` | Report HTML per campagna |
| `cifratura_upload_20260630-125941.csv` | Benchmark upload (10–1024 MB) |
| `cifratura_download_20260702-083516.csv` | Benchmark round-trip download |

Variabili utili: `BENCH_REPERTO_ID`, `BENCH_SIZES_MB`, `FABRIC_SYNC_TIMEOUT_SEC`, `TEST_RESULTS_DIR`.

**Dimensioni default:** `10 100 500 1024` MB (upload e download). Timeout IPFS: `~60 + 3×size_MB` s (max 2 h), vedi `tests/lib/common.sh`.

Esempio rapido: `BENCH_SIZES_MB="1 5 10" bash tests/cifratura_ipfs/run.sh upload`

Dettagli matrici Caliper: `fabric_caliper/README.md`.
