package webui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestSPAHandlerServesIndexWithoutRedirects(t *testing.T) {
	t.Parallel()
	handler := spaHandler(fstest.MapFS{
		"index.html":    {Data: []byte("workbench")},
		"assets/app.js": {Data: []byte("application")},
	})

	for _, target := range []string{"/", "/index.html", "/credentials/example"} {
		t.Run(target, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, target, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
			}
			if response.Body.String() != "workbench" {
				t.Fatalf("body = %q, want workbench", response.Body.String())
			}
		})
	}

	request := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	body, err := io.ReadAll(response.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "application" {
		t.Fatalf("asset body = %q, want application", body)
	}
}
