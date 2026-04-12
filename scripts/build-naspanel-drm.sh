#!/bin/sh
# 兼容旧脚本名：与仓库根目录 build.sh 相同，生成 ./naspanel。
set -eu
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
CGO_ENABLED=1 go build -tags drm -trimpath -o naspanel ./cmd/naspanel
echo "输出: $ROOT/naspanel"
