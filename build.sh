#!/bin/sh
#
# =============================================================================
# NasPanel Pro — 构建与启动（本脚本即「官方」构建方式）
# =============================================================================
#
# 依赖：
#   - Go 1.22+（toolchain 见 go.mod）
#   - Node.js + npm（用于 WebUI；没有 npm 时跳过前端构建，依赖已存在的
#     internal/server/webui-dist）
#   - 构建 DRM 二进制：必须 CGO_ENABLED=1，且本机具备 Linux DRM 相关头文件/库
#     （TrueNAS/Debian 等需已安装 raylib DRM 构建依赖；缺 Wayland 头文件时请用
#     带 drm 标签的构建环境，勿在无头文件机器上强编 GLFW 路径）
#
# 步骤（本脚本自动执行）：
#   1. （可选）结束已运行的 naspanel 进程，避免占用显示或端口
#   2. cd 到仓库根目录
#   3. WebUI：cd webui && npm install && npm run build
#      → Vite 产出写入 internal/server/webui-dist（由 vite.config 的 outDir 指定）
#      → Go 通过 internal/server/embed.go 嵌入该目录为单文件二进制中的 SPA
#   4. Go：CGO_ENABLED=1 go build -tags drm -trimpath -o naspanel ./cmd/naspanel
#      → 必须 -tags drm：Linux 纯 DRM；否则 raylib 走 GLFW，需 X11/Wayland 开发包
#   5. exec ./naspanel "$@" — 启动进程（可跟命令行参数）
#
# 环境变量（节选，与二进制运行时一致）：
#   NASPANEL_LAYOUT_PATH   layout.json 路径，默认 configs/layout.json
#   NASPANEL_NODES_PATH    节点配置，默认 configs/nodes.json
#   NASPANEL_HTTP_ADDR     HTTP 监听，默认 :8090；设为 - 关闭 Web
#   Netdata 地址见 configs/nodes.json；未指定 node_id 的组件使用默认（首条）节点
#
# 仅构建、不启动：
#   注释掉本文件最后一行 exec，或手动执行：
#   (cd webui && npm install && npm run build)
#   CGO_ENABLED=1 go build -tags drm -trimpath -o naspanel ./cmd/naspanel
#
# 跳过前端（已有 webui-dist 或离线环境）：
#   SKIP_WEBUI=1 ./build.sh
#
# =============================================================================

# 结束旧进程（无匹配时不调用 kill，避免 set -e 失败）
_old_pids=$(ps -ef | grep '[n]aspanel' | awk '{print $2}')
if [ -n "${_old_pids:-}" ]; then
  echo "$_old_pids" | xargs kill -9 2>/dev/null || true
fi
unset _old_pids
set -eu
cd "$(dirname "$0")"

if [ "${SKIP_WEBUI:-0}" != "1" ] && command -v npm >/dev/null 2>&1; then
  echo "==> WebUI: npm install -v && npm run build -v -> internal/server/webui-dist"
  (cd webui && npm install && npm run build)
elif [ "${SKIP_WEBUI:-0}" = "1" ]; then
  echo "==> 跳过 WebUI（SKIP_WEBUI=1）" >&2
else
  echo "warn: 未找到 npm，跳过 WebUI 构建（使用已有 internal/server/webui-dist）" >&2
fi

echo "==> Go: CGO_ENABLED=1 go build -tags drm -o naspanel ./cmd/naspanel"
CGO_ENABLED=1 go build -tags drm -trimpath -o naspanel ./cmd/naspanel

exec ./naspanel "$@"
