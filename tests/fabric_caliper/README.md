# Benchmark Fabric (Caliper)

Prima installazione: `cd caliper && npm install` (genera `node_modules/`, gitignored).

## Matrice di carico

| Script | Round | Durata | TPS | Worker |
|--------|-------|--------|-----|--------|
| read | 10 | 60 s | 5–400 | 4 |
| query-mix | 10 | 60 s | 5–400 | 4 |
| create | 11 | 60 s | 0,5–400 | 4 |

Parametri in `caliper/benchmarks/*.yaml`.

## Esecuzione

```bash
bash tests/fabric_caliper/run.sh read
bash tests/fabric_caliper/run.sh query-mix
bash tests/fabric_caliper/run.sh create
bash tests/fabric_caliper/run.sh all
```

## Output

| File | Contenuto |
|------|-----------|
| `tests/results/caliper/report-<tipo>-<timestamp>.html` | Report Caliper |
| `tests/results/caliper/caliper_summary.csv` | CSV cumulativo |
| `caliper/report.html` | Ultimo run (sovrascritto) |

CSV: `timestamp`, `benchmark`, `round`, `succ`, `fail`, `send_tps`, `max_lat_s`, `min_lat_s`, `avg_lat_s`, `throughput_tps`.
