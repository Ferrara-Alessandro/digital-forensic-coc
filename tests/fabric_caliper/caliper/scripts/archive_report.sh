#!/usr/bin/env bash
# Copia report.html e appende righe summary a CSV (per tesi).
set -euo pipefail

CALIPER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TESTS_DIR="$(cd "$CALIPER_DIR/../.." && pwd)"
RESULTS="${TEST_RESULTS_DIR:-$TESTS_DIR/results}/caliper"
BENCH_TAG="${1:?usage: archive_report.sh <tag es. read>}"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
REPORT_SRC="$CALIPER_DIR/report.html"

mkdir -p "$RESULTS"
REPORT_DST="$RESULTS/report-${BENCH_TAG}-${TIMESTAMP}.html"
CSV="$RESULTS/caliper_summary.csv"

cp "$REPORT_SRC" "$REPORT_DST"

if [[ ! -f "$CSV" ]]; then
  echo "timestamp,benchmark,round,succ,fail,send_tps,max_lat_s,min_lat_s,avg_lat_s,throughput_tps" >"$CSV"
fi

python3 << PY
import re
from pathlib import Path

html = Path("$REPORT_DST").read_text(encoding="utf-8")
# Tabella summary: righe <td>name</td> <td>304</td> ...
row_re = re.compile(
    r"<td>([^<]+)</td>\s*<td>(\d+)</td>\s*<td>(\d+)</td>\s*"
    r"<td>([\d.]+)</td>\s*<td>([\d.]+|-)</td>\s*<td>([\d.]+|-)</td>\s*"
    r"<td>([\d.]+|-)</td>\s*<td>([\d.]+)</td>"
)
ts = "$TIMESTAMP"
tag = "$BENCH_TAG"
out = Path("$CSV")
lines = []
seen = set()
for m in row_re.finditer(html):
    name, succ, fail, send, mx, mn, avg, thr = m.groups()
    if not name.startswith(("read-", "querymix-", "create-")):
        continue
    if name in seen:
        continue
    seen.add(name)
    lines.append(
        f'{ts},{tag},{name},{succ},{fail},{send},{mx},{mn},{avg},{thr}\n'
    )
if lines:
    with out.open("a", encoding="utf-8") as f:
        f.writelines(lines)
    print(f"CSV: {len(lines)} righe append a {out}")
else:
    print("CSV: nessuna riga estratta (report ok?)", file=__import__("sys").stderr)
PY

echo "Report HTML: $REPORT_DST"
echo "Report latest: $CALIPER_DIR/report.html"
