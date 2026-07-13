//go:build !webui_dist

package webui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed fallback/*
var embedded embed.FS

// Handler returns the development fallback interface.
func Handler() http.Handler {
	sub, _ := fs.Sub(embedded, "fallback")
	return spaHandler(sub)
}
