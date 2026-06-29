#!/usr/bin/env bash

align_csv() {
  local file="$1"
  [[ -s "$file" ]] || return 0
  local tmp="${file}.tmp.$$"
  awk -F',' '
    function isnum(s) { return s ~ /^-?[0-9]+(\.[0-9]+)?$/ }
    NR==FNR {
      if (NF > maxnf) maxnf = NF
      for (i=1; i<=NF; i++) {
        if (length($i) > w[i]) w[i] = length($i)
        if (FNR > 1 && $i != "" && !isnum($i)) nonnum[i] = 1
      }
      next
    }
    {
      line = ""
      for (i=1; i<=maxnf; i++) {
        val = (i <= NF) ? $i : ""
        if (FNR > 1 && !nonnum[i] && isnum(val)) {
          line = line sprintf("%*s", w[i], val)
        } else {
          line = line sprintf("%-*s", w[i], val)
        }
        if (i < maxnf) line = line ","
      }
      print line
    }
  ' "$file" "$file" > "$tmp" && mv "$tmp" "$file"
}
