# NasPanel Pro 构建与运行

## 编译

推荐使用根目录 `build.sh`：会先执行 `webui` 的 `npm install && npm run build`（输出到 `internal/server/webui-dist`），再编译带嵌入 SPA 的二进制。

```bash
./build.sh
```

仅手动编译 Go（不更新前端）时：

```bash
CGO_ENABLED=1 go build -tags drm -trimpath -o naspanel ./cmd/naspanel
```

## 环境变量（节选）

| 变量 | 说明 |
|------|------|
| `NASPANEL_LAYOUT_PATH` | `layout.json` 路径，默认 `configs/layout.json` |
| `NASPANEL_NODES_PATH` | Netdata 节点表路径，默认 `configs/nodes.json` |
| `NASPANEL_HTTP_ADDR` | Gin 监听地址，默认 `:8090`；设为 `-` 关闭 WebUI |
| `NASPANEL_MAXTPS` | 帧率上限，`0` 为不限 |
| `NASPANEL_DRM_KEEP_STDIN` | 设为 `1` 时不在启动时把 fd0 接到 `/dev/null`（默认会分离 fd0，但会 **dup 保留** 原 PTY 引用，避免 SSH 下 Ctrl+C 的 SIGINT 递不到进程） |
| `NASPANEL_BLOCKED_EXIT_MS` | 收到 SIGINT/SIGTERM 后若仍卡在 DRM Present，经过多少毫秒后 `os.Exit(0)`；默认 `600`，`0` 表示关闭（调试用） |
| `NASPANEL_ROTATE` / `NASPANEL_DRM_*` | 见 raypanel 注释 |
