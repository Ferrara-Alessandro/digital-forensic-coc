# Benchmark Fabric (Caliper)

## Matrice di carico (config attuale)

| Script | Round | Durata | TPS target | Worker |
|--------|-------|--------|------------|--------|
| **read** | 5 | 60 s | 5, 10, 25, 50, 75 | 4 |
| **query-mix** | 3 | 60 s | 10, 25, 50 | 4 |
| **create** | 3 | 60 s | 0,5, 1, 2 | 1 |

Modifica i valori in `caliper/benchmarks/*.yaml`.

## Esecuzione

```bash
bash tests/fabric_caliper/run.sh read
bash tests/fabric_caliper/run.sh query-mix
bash tests/fabric_caliper/run.sh create
bash tests/fabric_caliper/run.sh all    # ~25–40 min
```

## Output

Dopo ogni run:

| File | Contenuto |
|------|-----------|
| `tests/results/caliper/report-<tipo>-<timestamp>.html` | Report Caliper (apri nel browser) |
| `tests/results/caliper/caliper_summary.csv` | Tutte le run accumulate — per tabelle LaTeX/Excel |
| `caliper/report.html` | Ultimo run (sovrascritto) |

Aprire HTML su WSL:
```bash
explorer.exe tests/results/caliper/report-read-*.html
```

## Colonne CSV

`timestamp`, `benchmark`, `round`, `succ`, `fail`, `send_tps`, `max_lat_s`, `min_lat_s`, `avg_lat_s`, `throughput_tps`
