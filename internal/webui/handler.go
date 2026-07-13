package webui

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

func spaHandler(content fs.FS) http.Handler {
	files := http.FileServer(http.FS(content))
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requested := strings.TrimPrefix(path.Clean(request.URL.Path), "/")
		if requested == "." || requested == "" || requested == "index.html" {
			serveIndex(response, content)
			return
		}
		if _, err := fs.Stat(content, requested); err != nil {
			serveIndex(response, content)
			return
		}
		request.URL.Path = "/" + requested
		files.ServeHTTP(response, request)
	})
}

func serveIndex(response http.ResponseWriter, content fs.FS) {
	index, err := fs.ReadFile(content, "index.html")
	if err != nil {
		http.Error(response, "web interface is unavailable", http.StatusInternalServerError)
		return
	}
	response.Header().Set("Cache-Control", "no-cache")
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = response.Write(index)
}
