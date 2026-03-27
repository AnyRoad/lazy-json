#!/bin/sh
set -eu

profile=${1:-.coverage/coverage.out}
output=${2:-docs/badges/coverage.svg}
repo_root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)

profile_path=$profile
case "$profile_path" in
  /*) ;;
  *) profile_path="$repo_root/$profile_path" ;;
esac

output_path=$output
case "$output_path" in
  /*) ;;
  *) output_path="$repo_root/$output_path" ;;
esac

mkdir -p "$(dirname "$output_path")"

percent=$(GOCACHE=${GOCACHE:-$repo_root/.gocache} go tool cover -func="$profile_path" | awk '/^total:/{print $3}')
value_width=46
total_width=114

percent_value=${percent%%%}
if awk "BEGIN { exit !($percent_value >= 80) }"; then
  color="#97CA00"
elif awk "BEGIN { exit !($percent_value >= 60) }"; then
  color="#dfb317"
else
  color="#e05d44"
fi

cat >"$output_path" <<EOF
<svg xmlns="http://www.w3.org/2000/svg" width="$total_width" height="20" role="img" aria-label="coverage: $percent">
  <title>coverage: $percent</title>
  <linearGradient id="b" x2="0" y2="100%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <mask id="a">
    <rect width="$total_width" height="20" rx="3" fill="#fff"/>
  </mask>
  <g mask="url(#a)">
    <rect width="68" height="20" fill="#555"/>
    <rect x="68" width="$value_width" height="20" fill="$color"/>
    <rect width="$total_width" height="20" fill="url(#b)"/>
  </g>
  <g fill="#fff" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" font-size="11" text-anchor="middle">
    <text x="34" y="15" fill="#010101" fill-opacity=".3">coverage</text>
    <text x="34" y="14">coverage</text>
    <text x="91" y="15" fill="#010101" fill-opacity=".3">$percent</text>
    <text x="91" y="14">$percent</text>
  </g>
</svg>
EOF

printf '%s\n' "$percent"
