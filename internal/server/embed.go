package server

import "embed"

//go:embed webui-dist/**
var webuiDist embed.FS
