# NasPanel

**English** | [中文](#naspanel-1)

A **lightweight** **X11-less, DRM-direct Linux status display panel** for **home NAS / homelab**: it **renders the dashboard on a physical monitor** (KMS/DRM, no X11 / no full desktop) and includes a **browser-based layout editor** for widgets, scenes, and data sources.

---

## Features

- **Direct rendering**: [Raylib](https://www.raylib.com/) **DRM/KMS** backend (`-tags drm`) for bare-metal or embedded panels.
- **Data sources**: backends in `configs/nodes.json`; widgets bind chart IDs and dimensions (**Netdata API only for now**).
- **Widget types**: text, gauge, line chart, progress bar, histogram.
- **Scenes**: multiple screens with timed rotation and cross-fade.
- **Web UI** (embedded in the binary): visual editor, settings (canvas, rotation, intervals), data source probes, i18n (Chinese / English).
- **HTTP API**: Gin serves the SPA and REST endpoints for layout and nodes; optional to disable.
- **Ops-oriented extras**: optional supervisor hooks (TTY / VCS, `chvt`) for coexisting with a text console on appliances—see code and env vars if you need them.

---

## Quick start

### Requirements

- **Go** 1.22+ (see `go.mod`)
- **Node.js** + **npm** (only to build the Web UI)
- **Linux** with DRM stack and **CGO** + Raylib **DRM** build dependencies (see `build.sh` comments)
- A reachable **metrics** HTTP endpoint (**Netdata API only for now**; see `configs/nodes.json`)

### Build

```bash
./build.sh
```

This runs `npm install && npm run build` in `webui/`, then:

```bash
CGO_ENABLED=1 go build -tags drm -trimpath -o naspanel ./cmd/naspanel
```

To skip the frontend when `internal/server/webui-dist` is already present:

```bash
SKIP_WEBUI=1 ./build.sh
```

### Run

```bash
./naspanel
```

- Open **`http://<host>:8090/`** for the Web UI (default listen `:8090`).
- Layout file: **`configs/layout.json`** (override with `NASPANEL_LAYOUT_PATH`).
- Nodes file: **`configs/nodes.json`** (override with `NASPANEL_NODES_PATH`).

### DRM text / CJK

The default Raylib font is Latin-only. For **Chinese (or full Unicode) titles**, set a font path:

```bash
export NASPANEL_FONT=/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc
```

More runtime options: **`docs/naspanel-run.md`**.

---

## Repository layout (brief)

| Path | Role |
|------|------|
| `cmd/naspanel` | Main binary |
| `internal/raypanel` | DRM render loop, widgets, metrics polling |
| `internal/server` | Gin + embedded SPA |
| `webui/` | React (Vite) editor |
| `configs/` | Default `layout.json` / `nodes.json` location |

---

## License

**MIT License** — this project is released under the MIT license.

---

# NasPanel

[English](#naspanel) | **中文**

面向 **家庭 NAS / 实验室** 的轻量 **无 X11、DRM 直连** **Linux 状态显示面板**：在 **显示器** 上呈现状态界面，经 KMS/DRM 输出，无需完整图形桌面；内置 **浏览器编排界面**，用于布局、场景与数据源配置。

---

## 功能概览

- **直连显示**：基于 [Raylib](https://www.raylib.com/) 的 **DRM/KMS** 渲染（构建时使用 `-tags drm`）。
- **数据源**：在 `configs/nodes.json` 中配置多个后端；组件绑定 chart 与维度（**目前仅支持 Netdata API**，见 `configs/nodes.json`）。
- **组件类型**：文本、仪表盘、折线图、进度条、柱状分布。
- **多场景**：支持轮播间隔与渐变切换。
- **Web 界面**（嵌入二进制）：可视化编排、设置（画布/旋转/轮播）、数据源探测、中 / 英界面。
- **HTTP API**：Gin 提供 SPA 与布局、节点等 REST；可关闭 HTTP 仅保留 DRM。
- **运维向能力**：可选的 TTY / 虚拟控制台与 `chvt` 等逻辑，便于与设备控制台共存——详见源码与环境变量。

---

## 使用说明

### 依赖

- **Go** 1.22+（见 `go.mod`）
- **Node.js** + **npm**（仅构建 Web UI）
- 带 **DRM** 的 **Linux**，并安装 **CGO** 与 Raylib **DRM** 所需头文件/库（见 `build.sh` 注释）
- 至少一个可在本机访问的 **监控数据 HTTP 端点**（**目前仅支持 Netdata API**，见 `configs/nodes.json`）

### 构建

```bash
./build.sh
```

等价于先构建前端，再执行：

```bash
CGO_ENABLED=1 go build -tags drm -trimpath -o naspanel ./cmd/naspanel
```

若已有 `internal/server/webui-dist`，可跳过前端：

```bash
SKIP_WEBUI=1 ./build.sh
```

### 运行

```bash
./naspanel
```

- 浏览器打开 **`http://<主机>:8090/`**（默认监听 `:8090`）。
- 布局文件：**`configs/layout.json`**（可用 `NASPANEL_LAYOUT_PATH` 覆盖）。
- 节点文件：**`configs/nodes.json`**（可用 `NASPANEL_NODES_PATH` 覆盖）。

### 中文显示

默认字库对中文不友好，标题等会出现方块或问号。请指定含中文的字体，例如：

```bash
export NASPANEL_FONT=/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc
```

更多环境变量说明见 **`docs/naspanel-run.md`**。

---

## 目录结构（摘要）

| 路径 | 说明 |
|------|------|
| `cmd/naspanel` | 主程序入口 |
| `internal/raypanel` | DRM 主循环、组件绘制、数据轮询 |
| `internal/server` | Gin 与嵌入的前端资源 |
| `webui/` | React（Vite）编排器 |
| `configs/` | 默认 `layout.json`、`nodes.json` |

---

## 开源许可

本项目采用 **MIT许可** 。
