#!/bin/sh
# Debian/Ubuntu：安装 naspanel（Raylib DRM）编译运行依赖（需 root）。
set -eu
apt-get update
apt-get install -y --no-install-recommends \
  build-essential pkg-config \
  libgl1-mesa-dri \
  libgbm-dev libdrm-dev \
  libgles2-mesa-dev libegl1-mesa-dev
