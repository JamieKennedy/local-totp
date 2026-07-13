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
		if requested == "." || requested == "" {
			requested = "index.html"
		}
		if _, err := fs.Stat(content, requested); err != nil {
			request.URL.Path = "/"
		} else {
			request.URL.Path = "/" + requested
		}
		files.ServeHTTP(response, request)
	})
}
