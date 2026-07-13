// Package httpapi adapts HTTP requests to application workflows.
package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/JamieKennedy/local-totp/internal/application"
	"github.com/JamieKennedy/local-totp/internal/vault"
	"github.com/JamieKennedy/local-totp/internal/webui"
)

const sessionCookie = "local_totp_session"

// Server is the HTTP adapter and in-memory session owner.
type Server struct {
	app       *application.App
	version   string
	logger    *slog.Logger
	sessions  map[string]string
	previews  map[string]storedPreview
	mu        sync.Mutex
	static    http.Handler
	openAPI   []byte
	loginMu   sync.Mutex
	nextLogin time.Time
}

type storedPreview struct {
	value   vault.BackupPreview
	expires time.Time
}

// New constructs the HTTP adapter.
func New(app *application.App, version string, logger *slog.Logger, openAPI []byte) *Server {
	return &Server{app: app, version: version, logger: logger, sessions: map[string]string{}, previews: map[string]storedPreview{}, static: webui.Handler(), openAPI: openAPI}
}

// Handler returns the complete HTTP handler.
func (server *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.health)
	mux.HandleFunc("GET /api/v1/openapi.json", server.openAPISpec)
	mux.HandleFunc("GET /api/v1/status", server.status)
	mux.HandleFunc("POST /api/v1/setup", server.setup)
	mux.HandleFunc("POST /api/v1/session/unlock", server.unlock)
	mux.HandleFunc("POST /api/v1/session/lock", server.lock)
	mux.HandleFunc("POST /api/v1/session/recover", server.recover)
	mux.HandleFunc("GET /api/v1/credentials", server.listCredentials)
	mux.HandleFunc("POST /api/v1/credentials", server.createCredential)
	mux.HandleFunc("GET /api/v1/credentials/{id}/secret", server.revealSecret)
	mux.HandleFunc("GET /api/v1/credentials/{id}/code", server.oneCode)
	mux.HandleFunc("PATCH /api/v1/credentials/{id}", server.updateCredential)
	mux.HandleFunc("DELETE /api/v1/credentials/{id}", server.deleteCredential)
	mux.HandleFunc("GET /api/v1/codes", server.codes)
	mux.HandleFunc("GET /api/v1/groups", server.listGroups)
	mux.HandleFunc("POST /api/v1/groups", server.createGroup)
	mux.HandleFunc("PATCH /api/v1/groups/{id}", server.updateGroup)
	mux.HandleFunc("DELETE /api/v1/groups/{id}", server.deleteGroup)
	mux.HandleFunc("GET /api/v1/settings/api-keys", server.listAPIKeys)
	mux.HandleFunc("POST /api/v1/settings/api-keys", server.createAPIKey)
	mux.HandleFunc("DELETE /api/v1/settings/api-keys/{id}", server.deleteAPIKey)
	mux.HandleFunc("POST /api/v1/settings/password", server.changePassword)
	mux.HandleFunc("POST /api/v1/settings/recovery/rotate", server.rotateRecovery)
	mux.HandleFunc("POST /api/v1/backups/export", server.exportBackup)
	mux.HandleFunc("POST /api/v1/backups/preview", server.previewBackup)
	mux.HandleFunc("POST /api/v1/backups/{id}/apply", server.applyBackup)
	mux.Handle("/", server.static)
	return server.securityHeaders(server.validateHost(mux))
}

func (server *Server) health(response http.ResponseWriter, _ *http.Request) {
	writeJSON(response, http.StatusOK, map[string]any{"status": "ok", "version": server.version})
}

func (server *Server) openAPISpec(response http.ResponseWriter, _ *http.Request) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusOK)
	_, _ = response.Write(server.openAPI)
}

func (server *Server) status(response http.ResponseWriter, request *http.Request) {
	status, err := server.app.Status(request.Context())
	if err != nil {
		server.writeError(response, err)
		return
	}
	authenticated, csrf := server.session(request)
	writeJSON(response, http.StatusOK, map[string]any{"setup": status.Setup, "locked": status.Locked, "authenticated": authenticated, "csrfToken": csrf, "version": server.version, "testOnly": true})
}

func (server *Server) setup(response http.ResponseWriter, request *http.Request) {
	var input struct {
		Password string `json:"password"`
	}
	if !decodeJSON(response, request, &input) {
		return
	}
	recovery, err := server.app.Setup(request.Context(), input.Password)
	if err != nil {
		server.writeError(response, err)
		return
	}
	csrf, err := server.createSession(response)
	if err != nil {
		server.writeError(response, err)
		return
	}
	writeJSON(response, http.StatusCreated, map[string]string{"recoveryKey": recovery, "csrfToken": csrf})
}

func (server *Server) unlock(response http.ResponseWriter, request *http.Request) {
	if !server.loginAllowed(response) {
		return
	}
	var input struct {
		Password string `json:"password"`
	}
	if !decodeJSON(response, request, &input) {
		return
	}
	if err := server.app.Unlock(request.Context(), input.Password); err != nil {
		server.recordLoginFailure()
		server.writeError(response, err)
		return
	}
	server.clearLoginFailures()
	csrf, err := server.createSession(response)
	if err != nil {
		server.writeError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]string{"csrfToken": csrf})
}

func (server *Server) lock(response http.ResponseWriter, request *http.Request) {
	if !server.requireSession(response, request, true) {
		return
	}
	server.app.Lock()
	server.revokeSessions()
	http.SetCookie(response, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	response.WriteHeader(http.StatusNoContent)
}

func (server *Server) recover(response http.ResponseWriter, request *http.Request) {
	if !server.loginAllowed(response) {
		return
	}
	var input struct {
		RecoveryKey string `json:"recoveryKey"`
		Password    string `json:"password"`
	}
	if !decodeJSON(response, request, &input) {
		return
	}
	recovery, err := server.app.Recover(request.Context(), input.RecoveryKey, input.Password)
	if err != nil {
		server.recordLoginFailure()
		server.writeError(response, err)
		return
	}
	server.revokeSessions()
	csrf, err := server.createSession(response)
	if err != nil {
		server.writeError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]string{"recoveryKey": recovery, "csrfToken": csrf})
}

func (server *Server) listCredentials(response http.ResponseWriter, request *http.Request) {
	if !server.requireRead(response, request) {
		return
	}
	items, err := server.app.Credentials(request.Context())
	if err != nil {
		server.writeError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"credentials": items})
}

func (server *Server) createCredential(response http.ResponseWriter, request *http.Request) {
	if !server.requireSession(response, request, true) {
		return
	}
	var input application.CredentialInput
	if !decodeJSON(response, request, &input) {
		return
	}
	item, err := server.app.CreateCredential(request.Context(), input)
	if err != nil {
		server.writeError(response, err)
		return
	}
	writeJSON(response, http.StatusCreated, item)
}

func (server *Server) updateCredential(response http.ResponseWriter, request *http.Request) {
	if !server.requireSession(response, request, true) {
		return
	}
	var input application.CredentialInput
	if !decodeJSON(response, request, &input) {
		return
	}
	item, err := server.app.UpdateCredential(request.Context(), request.PathValue("id"), input)
	if err != nil {
		server.writeError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, item)
}

func (server *Server) deleteCredential(response http.ResponseWriter, request *http.Request) {
	if !server.requireSession(response, request, true) {
		return
	}
	if err := server.app.DeleteCredential(request.Context(), request.PathValue("id")); err != nil {
		server.writeError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func (server *Server) revealSecret(response http.ResponseWriter, request *http.Request) {
	if !server.requireSession(response, request, false) {
		return
	}
	secret, err := server.app.RevealSecret(request.Context(), request.PathValue("id"))
	if err != nil {
		server.writeError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, secret)
}

func (server *Server) codes(response http.ResponseWriter, request *http.Request) {
	if !server.requireRead(response, request) {
		return
	}
	codes, err := server.app.Codes(request.Context(), time.Now().UTC())
	if err != nil {
		server.writeError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, codes)
}

func (server *Server) oneCode(response http.ResponseWriter, request *http.Request) {
	if !server.requireRead(response, request) {
		return
	}
	code, err := server.app.Code(request.Context(), request.PathValue("id"), time.Now().UTC())
	if err != nil {
		server.writeError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, code)
}

func (server *Server) listGroups(response http.ResponseWriter, request *http.Request) {
	if !server.requireRead(response, request) {
		return
	}
	groups, err := server.app.Groups(request.Context())
	if err != nil {
		server.writeError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"groups": groups})
}

func (server *Server) createGroup(response http.ResponseWriter, request *http.Request) {
	if !server.requireSession(response, request, true) {
		return
	}
	var input vault.Group
	if !decodeJSON(response, request, &input) {
		return
	}
	group, err := server.app.CreateGroup(request.Context(), input)
	if err != nil {
		server.writeError(response, err)
		return
	}
	writeJSON(response, http.StatusCreated, group)
}

func (server *Server) updateGroup(response http.ResponseWriter, request *http.Request) {
	if !server.requireSession(response, request, true) {
		return
	}
	var input vault.Group
	if !decodeJSON(response, request, &input) {
		return
	}
	group, err := server.app.UpdateGroup(request.Context(), request.PathValue("id"), input)
	if err != nil {
		server.writeError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, group)
}

func (server *Server) deleteGroup(response http.ResponseWriter, request *http.Request) {
	if !server.requireSession(response, request, true) {
		return
	}
	if err := server.app.DeleteGroup(request.Context(), request.PathValue("id")); err != nil {
		server.writeError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func (server *Server) listAPIKeys(response http.ResponseWriter, request *http.Request) {
	if !server.requireSession(response, request, false) {
		return
	}
	keys, err := server.app.APIKeys(request.Context())
	if err != nil {
		server.writeError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"apiKeys": keys})
}

func (server *Server) createAPIKey(response http.ResponseWriter, request *http.Request) {
	if !server.requireSession(response, request, true) {
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if !decodeJSON(response, request, &input) {
		return
	}
	key, err := server.app.CreateAPIKey(request.Context(), input.Name)
	if err != nil {
		server.writeError(response, err)
		return
	}
	writeJSON(response, http.StatusCreated, key)
}

func (server *Server) deleteAPIKey(response http.ResponseWriter, request *http.Request) {
	if !server.requireSession(response, request, true) {
		return
	}
	if err := server.app.DeleteAPIKey(request.Context(), request.PathValue("id")); err != nil {
		server.writeError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func (server *Server) changePassword(response http.ResponseWriter, request *http.Request) {
	if !server.requireSession(response, request, true) {
		return
	}
	var input struct {
		Current     string `json:"current"`
		Replacement string `json:"replacement"`
	}
	if !decodeJSON(response, request, &input) {
		return
	}
	if err := server.app.ChangePassword(request.Context(), input.Current, input.Replacement); err != nil {
		server.writeError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func (server *Server) rotateRecovery(response http.ResponseWriter, request *http.Request) {
	if !server.requireSession(response, request, true) {
		return
	}
	recovery, err := server.app.RotateRecovery(request.Context())
	if err != nil {
		server.writeError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]string{"recoveryKey": recovery})
}

func (server *Server) exportBackup(response http.ResponseWriter, request *http.Request) {
	if !server.requireSession(response, request, true) {
		return
	}
	var input struct {
		Password string `json:"password"`
	}
	if !decodeJSON(response, request, &input) {
		return
	}
	value, err := server.app.ExportBackup(request.Context(), input.Password)
	if err != nil {
		server.writeError(response, err)
		return
	}
	response.Header().Set("Content-Type", "application/vnd.local-totp.backup+json")
	response.Header().Set("Content-Disposition", `attachment; filename="local-totp-backup.ltotp"`)
	response.WriteHeader(http.StatusOK)
	_, _ = response.Write(value)
}

func (server *Server) previewBackup(response http.ResponseWriter, request *http.Request) {
	if !server.requireSession(response, request, true) {
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, 11*1024*1024)
	if err := request.ParseMultipartForm(10 * 1024 * 1024); err != nil {
		server.writeError(response, errors.New("invalid backup upload"))
		return
	}
	file, _, err := request.FormFile("file")
	if err != nil {
		server.writeError(response, errors.New("backup file is required"))
		return
	}
	defer file.Close()
	value, err := io.ReadAll(io.LimitReader(file, 10*1024*1024+1))
	if err != nil {
		server.writeError(response, err)
		return
	}
	preview, err := server.app.PreviewBackup(request.Context(), value, request.FormValue("password"))
	if err != nil {
		server.writeError(response, err)
		return
	}
	server.mu.Lock()
	server.previews[preview.ID] = storedPreview{value: preview, expires: time.Now().Add(5 * time.Minute)}
	server.mu.Unlock()
	writeJSON(response, http.StatusOK, preview)
}

func (server *Server) applyBackup(response http.ResponseWriter, request *http.Request) {
	if !server.requireSession(response, request, true) {
		return
	}
	var input struct {
		Mode string `json:"mode"`
	}
	if !decodeJSON(response, request, &input) {
		return
	}
	server.mu.Lock()
	stored, ok := server.previews[request.PathValue("id")]
	delete(server.previews, request.PathValue("id"))
	server.mu.Unlock()
	if !ok || time.Now().After(stored.expires) {
		server.writeError(response, vault.ErrNotFound)
		return
	}
	if err := server.app.ApplyBackup(request.Context(), stored.value, input.Mode); err != nil {
		server.writeError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func (server *Server) requireRead(response http.ResponseWriter, request *http.Request) bool {
	if authenticated, _ := server.session(request); authenticated {
		return true
	}
	value := strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer ")
	valid, err := server.app.AuthenticateAPIKey(request.Context(), value)
	if err != nil {
		server.writeError(response, err)
		return false
	}
	if !valid {
		server.writeError(response, vault.ErrUnauthorized)
		return false
	}
	return true
}

func (server *Server) requireSession(response http.ResponseWriter, request *http.Request, csrfRequired bool) bool {
	authenticated, csrf := server.session(request)
	if !authenticated {
		server.writeError(response, vault.ErrUnauthorized)
		return false
	}
	if csrfRequired && request.Header.Get("X-CSRF-Token") != csrf {
		writeAPIError(response, http.StatusForbidden, "csrf_failed", "CSRF token is missing or invalid")
		return false
	}
	return true
}

func (server *Server) createSession(response http.ResponseWriter) (string, error) {
	sessionID, err := secureToken()
	if err != nil {
		return "", err
	}
	csrf, err := secureToken()
	if err != nil {
		return "", err
	}
	server.mu.Lock()
	server.sessions[sessionID] = csrf
	server.mu.Unlock()
	http.SetCookie(response, &http.Cookie{Name: sessionCookie, Value: sessionID, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
	return csrf, nil
}

func (server *Server) session(request *http.Request) (bool, string) {
	cookie, err := request.Cookie(sessionCookie)
	if err != nil {
		return false, ""
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	csrf, ok := server.sessions[cookie.Value]
	return ok, csrf
}

func (server *Server) revokeSessions() {
	server.mu.Lock()
	server.sessions = map[string]string{}
	server.previews = map[string]storedPreview{}
	server.mu.Unlock()
}

func secureToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func (server *Server) validateHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		host := request.Host
		if parsed, _, err := net.SplitHostPort(request.Host); err == nil {
			host = parsed
		}
		host = strings.Trim(host, "[]")
		if host != "localhost" && host != "127.0.0.1" && host != "::1" {
			writeAPIError(response, http.StatusForbidden, "invalid_host", "Host must be loopback")
			return
		}
		if origin := request.Header.Get("Origin"); origin != "" {
			parsed, err := url.Parse(origin)
			originHost := ""
			if err == nil {
				originHost = parsed.Hostname()
			}
			if originHost != "localhost" && originHost != "127.0.0.1" && originHost != "::1" {
				writeAPIError(response, http.StatusForbidden, "invalid_origin", "Origin must be loopback")
				return
			}
		}
		next.ServeHTTP(response, request)
	})
}

func (server *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: blob:; style-src 'self'; script-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.Header().Set("Referrer-Policy", "no-referrer")
		response.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(response, request)
	})
}

func (server *Server) loginAllowed(response http.ResponseWriter) bool {
	server.loginMu.Lock()
	defer server.loginMu.Unlock()
	if time.Now().Before(server.nextLogin) {
		response.Header().Set("Retry-After", "1")
		writeAPIError(response, http.StatusTooManyRequests, "rate_limited", "Try again shortly")
		return false
	}
	return true
}

func (server *Server) recordLoginFailure() {
	server.loginMu.Lock()
	server.nextLogin = time.Now().Add(time.Second)
	server.loginMu.Unlock()
}
func (server *Server) clearLoginFailures() {
	server.loginMu.Lock()
	server.nextLogin = time.Time{}
	server.loginMu.Unlock()
}

func (server *Server) writeError(response http.ResponseWriter, err error) {
	status, code, message := http.StatusUnprocessableEntity, "validation_failed", err.Error()
	switch {
	case errors.Is(err, vault.ErrLocked):
		status, code, message = http.StatusLocked, "vault_locked", "The vault is locked"
	case errors.Is(err, vault.ErrUnauthorized):
		status, code, message = http.StatusUnauthorized, "unauthorized", "Authentication failed"
	case errors.Is(err, vault.ErrNotFound):
		status, code, message = http.StatusNotFound, "not_found", "The requested record was not found"
	case errors.Is(err, vault.ErrConflict), errors.Is(err, vault.ErrAlreadySetup):
		status, code = http.StatusConflict, "conflict"
	case errors.Is(err, vault.ErrNotSetup):
		status, code = http.StatusPreconditionFailed, "not_setup"
	}
	if status >= 500 {
		server.logger.Error("request failed", "error", err)
	}
	writeAPIError(response, status, code, message)
}

func decodeJSON(response http.ResponseWriter, request *http.Request, target any) bool {
	if contentType := request.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		writeAPIError(response, http.StatusUnsupportedMediaType, "invalid_content_type", "Content-Type must be application/json")
		return false
	}
	request.Body = http.MaxBytesReader(response, request.Body, 1024*1024)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeAPIError(response, http.StatusBadRequest, "invalid_json", "Request body is invalid")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeAPIError(response, http.StatusBadRequest, "invalid_json", "Request body must contain one JSON value")
		return false
	}
	return true
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

func writeAPIError(response http.ResponseWriter, status int, code, message string) {
	writeJSON(response, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
