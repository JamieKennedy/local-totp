package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	apispec "github.com/JamieKennedy/local-totp/api"
	"github.com/JamieKennedy/local-totp/internal/application"
	"github.com/JamieKennedy/local-totp/internal/vault"
)

const (
	syntheticPassword = "synthetic-test-password"
	rfc6238SHA1Secret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
)

type testAPI struct {
	handler http.Handler
	server  *httptest.Server
	client  *http.Client
	logs    *bytes.Buffer
}

func newTestAPI(t *testing.T) *testAPI {
	t.Helper()
	store, err := vault.Open(context.Background(), filepath.Join(t.TempDir(), "vault.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logs, nil))
	server := New(application.New(store), "1.0.0-test", logger, apispec.Document())
	handler := server.Handler()
	testServer := httptest.NewServer(handler)
	t.Cleanup(testServer.Close)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := testServer.Client()
	client.Jar = jar
	return &testAPI{handler: handler, server: testServer, client: client, logs: logs}
}

func (api *testAPI) setup(t *testing.T) string {
	t.Helper()
	response := api.json(t, api.client, http.MethodPost, "/api/v1/setup", map[string]string{"password": syntheticPassword}, "", "")
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("setup status = %d, want %d", response.StatusCode, http.StatusCreated)
	}
	var body struct {
		CSRFToken string `json:"csrfToken"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.CSRFToken == "" {
		t.Fatal("setup did not return a CSRF token")
	}
	return body.CSRFToken
}

func (api *testAPI) json(t *testing.T, client *http.Client, method, path string, body any, csrf, bearer string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(context.Background(), method, api.server.URL+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if csrf != "" {
		request.Header.Set("X-CSRF-Token", csrf)
	}
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func TestLifecycleCookiesCSRFAndHeaders(t *testing.T) {
	api := newTestAPI(t)
	response := api.json(t, api.client, http.MethodPost, "/api/v1/setup", map[string]string{"password": syntheticPassword}, "", "")
	if response.StatusCode != http.StatusCreated {
		response.Body.Close()
		t.Fatalf("setup status = %d, want %d", response.StatusCode, http.StatusCreated)
	}
	var setup struct {
		CSRFToken string `json:"csrfToken"`
	}
	if err := json.NewDecoder(response.Body).Decode(&setup); err != nil {
		response.Body.Close()
		t.Fatal(err)
	}
	response.Body.Close()

	cookies := response.Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookie || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode || cookies[0].Path != "/" {
		t.Fatal("session cookie is missing HttpOnly, SameSite=Strict, or root-path protection")
	}
	for _, header := range []string{"Content-Security-Policy", "X-Content-Type-Options", "Referrer-Policy", "Permissions-Policy"} {
		if response.Header.Get(header) == "" {
			t.Fatalf("security header %s is missing", header)
		}
	}

	withoutCSRF := api.json(t, api.client, http.MethodPost, "/api/v1/groups", map[string]string{"name": "Synthetic", "color": "teal"}, "", "")
	withoutCSRF.Body.Close()
	if withoutCSRF.StatusCode != http.StatusForbidden {
		t.Fatalf("missing-CSRF status = %d, want %d", withoutCSRF.StatusCode, http.StatusForbidden)
	}

	withCSRF := api.json(t, api.client, http.MethodPost, "/api/v1/groups", map[string]string{"name": "Synthetic", "color": "teal"}, setup.CSRFToken, "")
	withCSRF.Body.Close()
	if withCSRF.StatusCode != http.StatusCreated {
		t.Fatalf("valid-CSRF status = %d, want %d", withCSRF.StatusCode, http.StatusCreated)
	}

	locked := api.json(t, api.client, http.MethodPost, "/api/v1/session/lock", nil, setup.CSRFToken, "")
	locked.Body.Close()
	if locked.StatusCode != http.StatusNoContent {
		t.Fatalf("lock status = %d, want %d", locked.StatusCode, http.StatusNoContent)
	}
	status := api.json(t, api.client, http.MethodGet, "/api/v1/status", nil, "", "")
	defer status.Body.Close()
	var lifecycle struct {
		Locked        bool `json:"locked"`
		Authenticated bool `json:"authenticated"`
	}
	if err := json.NewDecoder(status.Body).Decode(&lifecycle); err != nil {
		t.Fatal(err)
	}
	if !lifecycle.Locked || lifecycle.Authenticated {
		t.Fatal("locking the vault did not revoke the browser session")
	}
}

func TestHostOriginAndOpenAPI(t *testing.T) {
	api := newTestAPI(t)

	invalidHost := httptest.NewRequest(http.MethodGet, "http://example.test/api/v1/status", nil)
	invalidHost.Host = "example.test"
	hostResponse := httptest.NewRecorder()
	api.handler.ServeHTTP(hostResponse, invalidHost)
	if hostResponse.Code != http.StatusForbidden {
		t.Fatalf("invalid Host status = %d, want %d", hostResponse.Code, http.StatusForbidden)
	}

	invalidOrigin := httptest.NewRequest(http.MethodGet, "http://localhost/api/v1/status", nil)
	invalidOrigin.Header.Set("Origin", "https://example.test")
	originResponse := httptest.NewRecorder()
	api.handler.ServeHTTP(originResponse, invalidOrigin)
	if originResponse.Code != http.StatusForbidden {
		t.Fatalf("invalid Origin status = %d, want %d", originResponse.Code, http.StatusForbidden)
	}

	response := api.json(t, api.client, http.MethodGet, "/api/v1/openapi.json", nil, "", "")
	defer response.Body.Close()
	served, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !bytes.Equal(served, apispec.Document()) {
		t.Fatal("served OpenAPI document differs from the canonical embedded document")
	}
}

func TestBearerIsReadOnlyAndCannotRevealSecrets(t *testing.T) {
	api := newTestAPI(t)
	csrf := api.setup(t)
	credential := api.json(t, api.client, http.MethodPost, "/api/v1/credentials", map[string]any{
		"source": "manual", "issuer": "RFC", "account": "6238@example.test", "secret": rfc6238SHA1Secret,
		"algorithm": "SHA1", "digits": 8, "period": 30, "favorite": false, "tags": []string{"synthetic"}, "notes": "RFC 6238 test vector",
	}, csrf, "")
	if credential.StatusCode != http.StatusCreated {
		credential.Body.Close()
		t.Fatalf("credential creation status = %d, want %d", credential.StatusCode, http.StatusCreated)
	}
	var createdCredential struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(credential.Body).Decode(&createdCredential); err != nil {
		credential.Body.Close()
		t.Fatal(err)
	}
	credential.Body.Close()

	keyResponse := api.json(t, api.client, http.MethodPost, "/api/v1/settings/api-keys", map[string]string{"name": "synthetic-test-client"}, csrf, "")
	if keyResponse.StatusCode != http.StatusCreated {
		keyResponse.Body.Close()
		t.Fatalf("API key creation status = %d, want %d", keyResponse.StatusCode, http.StatusCreated)
	}
	var createdKey struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(keyResponse.Body).Decode(&createdKey); err != nil {
		keyResponse.Body.Close()
		t.Fatal(err)
	}
	keyResponse.Body.Close()
	if createdKey.Key == "" {
		t.Fatal("API key creation did not return key material")
	}

	bearerClient := &http.Client{Transport: api.server.Client().Transport}
	list := api.json(t, bearerClient, http.MethodGet, "/api/v1/credentials", nil, "", createdKey.Key)
	listBody, err := io.ReadAll(list.Body)
	list.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if list.StatusCode != http.StatusOK || bytes.Contains(listBody, []byte(rfc6238SHA1Secret)) || bytes.Contains(listBody, []byte(`"secret"`)) {
		t.Fatal("bearer credential listing failed or disclosed seed material")
	}

	code := api.json(t, bearerClient, http.MethodGet, "/api/v1/credentials/"+createdCredential.ID+"/code", nil, "", createdKey.Key)
	code.Body.Close()
	if code.StatusCode != http.StatusOK {
		t.Fatalf("bearer code-read status = %d, want %d", code.StatusCode, http.StatusOK)
	}

	reveal := api.json(t, bearerClient, http.MethodGet, "/api/v1/credentials/"+createdCredential.ID+"/secret", nil, "", createdKey.Key)
	revealBody, err := io.ReadAll(reveal.Body)
	reveal.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if reveal.StatusCode != http.StatusUnauthorized || bytes.Contains(revealBody, []byte(rfc6238SHA1Secret)) {
		t.Fatal("bearer key was not denied seed reveal")
	}

	write := api.json(t, bearerClient, http.MethodPost, "/api/v1/groups", map[string]string{"name": "Denied", "color": "red"}, "", createdKey.Key)
	write.Body.Close()
	if write.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bearer management status = %d, want %d", write.StatusCode, http.StatusUnauthorized)
	}
}

func TestUnlockRateLimitAndLoggingRedaction(t *testing.T) {
	api := newTestAPI(t)
	csrf := api.setup(t)
	lock := api.json(t, api.client, http.MethodPost, "/api/v1/session/lock", nil, csrf, "")
	lock.Body.Close()
	if lock.StatusCode != http.StatusNoContent {
		t.Fatalf("lock status = %d, want %d", lock.StatusCode, http.StatusNoContent)
	}

	failed := api.json(t, api.client, http.MethodPost, "/api/v1/session/unlock", map[string]string{"password": "synthetic-incorrect-password"}, "", "")
	failed.Body.Close()
	if failed.StatusCode != http.StatusUnauthorized {
		t.Fatalf("failed-unlock status = %d, want %d", failed.StatusCode, http.StatusUnauthorized)
	}
	rateLimited := api.json(t, api.client, http.MethodPost, "/api/v1/session/unlock", map[string]string{"password": "synthetic-incorrect-password"}, "", "")
	rateLimited.Body.Close()
	if rateLimited.StatusCode != http.StatusTooManyRequests || rateLimited.Header.Get("Retry-After") == "" {
		t.Fatalf("rate-limit status = %d with Retry-After %q", rateLimited.StatusCode, rateLimited.Header.Get("Retry-After"))
	}

	marker := "synthetic-redaction-marker"
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, api.server.URL+"/api/v1/setup", strings.NewReader(`{"password":"synthetic-test-password","unknown":"`+marker+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := api.client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusBadRequest || bytes.Contains(body, []byte(marker)) || strings.Contains(api.logs.String(), marker) {
		t.Fatal("invalid input was not rejected with redacted output and logs")
	}
}
