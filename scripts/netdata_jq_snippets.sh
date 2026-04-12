#!/bin/sh
# 配合 netdata_probe_charts.py --json-out FILE 使用（需本机已安装 jq）。
# 用法: NETDATA_JSON=/tmp/netdata_charts.json ./scripts/netdata_jq_snippets.sh
set -eu
F="${NETDATA_JSON:-/tmp/netdata_charts.json}"
test -f "$F" || { echo "请先: python3 scripts/netdata_probe_charts.py --json-out $F" >&2; exit 1; }

echo "== chart 总数（顶层含 .charts 时）"
jq 'if .charts then (.charts | keys | length) else (keys | length) end' "$F"

echo "== 前 15 个 chart id"
jq -r 'if .charts then (.charts | keys[] | .) else (keys[] | .) end' "$F" | head -15

echo "== system.cpu 的 dimensions 键"
jq '.charts["system.cpu"].dimensions // .["system.cpu"].dimensions | keys' "$F"

echo "== 单个 dimension 条目原始结构（system.cpu 第一个键）"
jq '.charts["system.cpu"].dimensions // .["system.cpu"].dimensions | to_entries[0]' "$F"
