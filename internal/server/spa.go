package server

import (
	"io/fs"
)

// fallbackHTML 当 webui-dist 未构建时返回的占位页面。
var fallbackHTML = []byte(`<!doctype html><html><head><meta charset="utf-8"><title>NasPanel</title></head><body><p>WebUI not built. Run <code>./build.sh</code> first.</p></body></html>`)

func loadWebUI() (index []byte, root fs.FS) {
	b, err := webuiDist.ReadFile("webui-dist/index.html")
	if err != nil {
		return fallbackHTML, nil
	}
	sub, err := fs.Sub(webuiDist, "webui-dist")
	if err != nil {
		return b, nil
	}
	return b, sub
}
