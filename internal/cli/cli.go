// Package cli implements the read-only HTTP client adapter.
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/JamieKennedy/local-totp/internal/application"
)

type config struct {
	URL        string `json:"url"`
	APIKeyFile string `json:"apiKeyFile"`
}

// Run executes a CLI client command.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		_, _ = fmt.Fprint(stdout, helpText)
		return nil
	}
	if args[0] == "configure" {
		return configure(args[1:], stdout)
	}
	outputJSON := false
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--json" {
			outputJSON = true
		} else {
			filtered = append(filtered, arg)
		}
	}
	settings, err := loadConfig()
	if err != nil {
		return err
	}
	client, err := newClient(settings)
	if err != nil {
		return err
	}
	switch filtered[0] {
	case "status":
		var status map[string]any
		if err := client.get(ctx, "/api/v1/status", false, &status); err != nil {
			return err
		}
		return printValue(stdout, status, outputJSON)
	case "list":
		var response struct {
			Credentials []application.CredentialView `json:"credentials"`
		}
		if err := client.get(ctx, "/api/v1/credentials", true, &response); err != nil {
			return err
		}
		if outputJSON {
			return printValue(stdout, response.Credentials, true)
		}
		for _, item := range response.Credentials {
			_, _ = fmt.Fprintf(stdout, "%s\t%s\n", item.ID, displayName(item))
		}
		return nil
	case "codes":
		var response application.CodesResponse
		if err := client.get(ctx, "/api/v1/codes", true, &response); err != nil {
			return err
		}
		return printValue(stdout, response, outputJSON)
	case "code":
		if len(filtered) != 2 {
			return errors.New("usage: local-totp code <uuid|exact-issuer/account> [--json]")
		}
		id, err := client.resolve(ctx, filtered[1])
		if err != nil {
			return err
		}
		var code application.CurrentCode
		if err := client.get(ctx, "/api/v1/credentials/"+id+"/code", true, &code); err != nil {
			return err
		}
		if outputJSON {
			return printValue(stdout, code, true)
		}
		_, err = fmt.Fprintln(stdout, code.Code)
		return err
	default:
		_, _ = fmt.Fprint(stderr, helpText)
		return fmt.Errorf("unknown command %q", filtered[0])
	}
}

type apiClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func newClient(settings config) (*apiClient, error) {
	key := ""
	if settings.APIKeyFile != "" {
		value, err := os.ReadFile(settings.APIKeyFile)
		if err != nil {
			return nil, fmt.Errorf("read API key file: %w", err)
		}
		key = strings.TrimSpace(string(value))
	}
	return &apiClient{baseURL: strings.TrimRight(settings.URL, "/"), apiKey: key, http: &http.Client{Timeout: 10 * time.Second}}, nil
}

func (client *apiClient) get(ctx context.Context, path string, authenticated bool, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+path, nil)
	if err != nil {
		return err
	}
	if authenticated {
		if client.apiKey == "" {
			return errors.New("no API key configured")
		}
		request.Header.Set("Authorization", "Bearer "+client.apiKey)
	}
	response, err := client.http.Do(request)
	if err != nil {
		return fmt.Errorf("request Local TOTP: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		var failure struct {
			Error struct{ Code, Message string } `json:"error"`
		}
		_ = json.NewDecoder(io.LimitReader(response.Body, 64*1024)).Decode(&failure)
		if failure.Error.Message == "" {
			failure.Error.Message = response.Status
		}
		return fmt.Errorf("%s: %s", failure.Error.Code, failure.Error.Message)
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func (client *apiClient) resolve(ctx context.Context, selector string) (string, error) {
	if len(selector) == 36 && strings.Count(selector, "-") == 4 {
		return selector, nil
	}
	var response struct {
		Credentials []application.CredentialView `json:"credentials"`
	}
	if err := client.get(ctx, "/api/v1/credentials", true, &response); err != nil {
		return "", err
	}
	matches := []application.CredentialView{}
	for _, item := range response.Credentials {
		if strings.EqualFold(displayName(item), selector) {
			matches = append(matches, item)
		}
	}
	if len(matches) == 0 {
		return "", errors.New("credential name was not found")
	}
	if len(matches) > 1 {
		buffer := bytes.NewBufferString("credential name is ambiguous; use one of:")
		for _, match := range matches {
			_, _ = fmt.Fprintf(buffer, "\n  %s", match.ID)
		}
		return "", errors.New(buffer.String())
	}
	return matches[0].ID, nil
}

func configure(args []string, stdout io.Writer) error {
	set := flag.NewFlagSet("configure", flag.ContinueOnError)
	urlValue := set.String("url", "http://localhost:8080", "Local TOTP URL")
	keyFile := set.String("api-key-file", "", "file containing a named API key")
	if err := set.Parse(args); err != nil {
		return err
	}
	if *keyFile == "" {
		return errors.New("--api-key-file is required")
	}
	absolute, err := filepath.Abs(*keyFile)
	if err != nil {
		return err
	}
	settings := config{URL: strings.TrimRight(*urlValue, "/"), APIKeyFile: absolute}
	encoded, _ := json.MarshalIndent(settings, "", "  ")
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "Configuration written to %s\n", path)
	return err
}

func loadConfig() (config, error) {
	path, err := configPath()
	if err != nil {
		return config{}, err
	}
	settings := config{URL: "http://localhost:8080"}
	if value, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(value, &settings); err != nil {
			return config{}, fmt.Errorf("decode CLI configuration: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return config{}, err
	}
	if value := os.Getenv("LOCAL_TOTP_URL"); value != "" {
		settings.URL = value
	}
	if value := os.Getenv("LOCAL_TOTP_API_KEY_FILE"); value != "" {
		settings.APIKeyFile = value
	}
	return settings, nil
}

func configPath() (string, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, "local-totp", "config.json"), nil
}

func printValue(writer io.Writer, value any, asJSON bool) error {
	if asJSON {
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(writer, string(encoded))
	return err
}

func displayName(item application.CredentialView) string {
	if item.Issuer == "" {
		return item.Account
	}
	return item.Issuer + ":" + item.Account
}

const helpText = `Local TOTP developer workbench

Usage:
  local-totp serve
  local-totp configure --url http://localhost:8080 --api-key-file PATH
  local-totp status [--json]
  local-totp list [--json]
  local-totp codes [--json]
  local-totp code <uuid|exact-issuer/account> [--json]
  local-totp version
`
