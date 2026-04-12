package server

import "embed"

//go:embed webui-dist/**
var webuiDist embed.FS

//go:embed web/index.html
var legacyIndex []byte
