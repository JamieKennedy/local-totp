//go:build webui_dist

package webui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist/*
var embedded embed.FS

// Handler returns the embedded production React interface.
func Handler() http.Handler {
	sub, _ := fs.Sub(embedded, "dist")
	return spaHandler(sub)
}
