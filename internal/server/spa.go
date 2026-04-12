package server

import (
	"io/fs"
)

func loadWebUI() (index []byte, root fs.FS) {
	b, err := webuiDist.ReadFile("webui-dist/index.html")
	if err != nil {
		return legacyIndex, nil
	}
	sub, err := fs.Sub(webuiDist, "webui-dist")
	if err != nil {
		return b, nil
	}
	return b, sub
}
