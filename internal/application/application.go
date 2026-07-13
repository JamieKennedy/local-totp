// Package application owns Local TOTP user workflows.
package application

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/JamieKennedy/local-totp/internal/totp"
	"github.com/JamieKennedy/local-totp/internal/vault"
)

// App coordinates vault persistence with pure TOTP generation.
type App struct {
	vault *vault.Vault
}

// Status reports setup and lock state.
type Status struct {
	Setup  bool `json:"setup"`
	Locked bool `json:"locked"`
}

// CredentialInput accepts manual, URI, or generated credential creation.
type CredentialInput struct {
	Source    string         `json:"source"`
	URI       string         `json:"uri,omitempty"`
	Issuer    string         `json:"issuer"`
	Account   string         `json:"account"`
	Secret    string         `json:"secret,omitempty"`
	Algorithm totp.Algorithm `json:"algorithm"`
	Digits    int            `json:"digits"`
	Period    int            `json:"period"`
	Favorite  bool           `json:"favorite"`
	GroupID   string         `json:"groupId,omitempty"`
	Tags      []string       `json:"tags"`
	Notes     string         `json:"notes"`
}

// CredentialView is metadata safe for normal dashboard and API-key responses.
type CredentialView struct {
	ID        string         `json:"id"`
	Issuer    string         `json:"issuer"`
	Account   string         `json:"account"`
	Algorithm totp.Algorithm `json:"algorithm"`
	Digits    int            `json:"digits"`
	Period    int            `json:"period"`
	Favorite  bool           `json:"favorite"`
	GroupID   string         `json:"groupId,omitempty"`
	Tags      []string       `json:"tags"`
	Notes     string         `json:"notes"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

// SecretView is returned only after an explicit authenticated reveal.
type SecretView struct {
	Secret string `json:"secret"`
	URI    string `json:"uri"`
}

// CurrentCode identifies a code and its server-derived validity interval.
type CurrentCode struct {
	CredentialID string    `json:"credentialId"`
	Code         string    `json:"code"`
	ValidFrom    time.Time `json:"validFrom"`
	ValidUntil   time.Time `json:"validUntil"`
}

// CodesResponse includes server time for client drift detection.
type CodesResponse struct {
	ServerTime time.Time     `json:"serverTime"`
	Codes      []CurrentCode `json:"codes"`
}

// New constructs the application module around an open vault.
func New(vaultStore *vault.Vault) *App { return &App{vault: vaultStore} }

// Status returns current lifecycle state.
func (app *App) Status(ctx context.Context) (Status, error) {
	setup, err := app.vault.IsSetup(ctx)
	return Status{Setup: setup, Locked: app.vault.IsLocked()}, err
}

// Setup initializes the vault.
func (app *App) Setup(ctx context.Context, password string) (string, error) {
	return app.vault.Setup(ctx, password)
}

// Unlock unlocks the vault.
func (app *App) Unlock(ctx context.Context, password string) error {
	return app.vault.Unlock(ctx, password)
}

// Lock locks the vault.
func (app *App) Lock() { app.vault.Lock() }

// Recover replaces the password and recovery key.
func (app *App) Recover(ctx context.Context, recovery, password string) (string, error) {
	return app.vault.Recover(ctx, recovery, password)
}

// ChangePassword rewraps the vault key.
func (app *App) ChangePassword(ctx context.Context, current, replacement string) error {
	return app.vault.ChangePassword(ctx, current, replacement)
}

// RotateRecovery rotates the recovery key.
func (app *App) RotateRecovery(ctx context.Context) (string, error) {
	return app.vault.RotateRecovery(ctx)
}

// CreateCredential creates a credential from the selected source.
func (app *App) CreateCredential(ctx context.Context, input CredentialInput) (CredentialView, error) {
	credential, err := app.credentialFromInput(input, false)
	if err != nil {
		return CredentialView{}, err
	}
	stored, err := app.vault.CreateCredential(ctx, credential)
	return view(stored), err
}

// UpdateCredential replaces a credential, retaining its seed when omitted.
func (app *App) UpdateCredential(ctx context.Context, id string, input CredentialInput) (CredentialView, error) {
	credential, err := app.credentialFromInput(input, true)
	if err != nil {
		return CredentialView{}, err
	}
	stored, err := app.vault.UpdateCredential(ctx, id, credential)
	return view(stored), err
}

// DeleteCredential deletes a credential permanently.
func (app *App) DeleteCredential(ctx context.Context, id string) error {
	return app.vault.DeleteCredential(ctx, id)
}

// Credentials returns seed-free metadata.
func (app *App) Credentials(ctx context.Context) ([]CredentialView, error) {
	credentials, err := app.vault.ListCredentials(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]CredentialView, len(credentials))
	for index, credential := range credentials {
		views[index] = view(credential)
	}
	return views, nil
}

// RevealSecret returns seed material for one explicitly requested credential.
func (app *App) RevealSecret(ctx context.Context, id string) (SecretView, error) {
	credential, err := app.vault.GetCredential(ctx, id)
	if err != nil {
		return SecretView{}, err
	}
	uri, err := totp.BuildURI(credential.TOTPConfig())
	if err != nil {
		return SecretView{}, err
	}
	return SecretView{Secret: credential.Secret, URI: uri}, nil
}

// Codes generates every current code at one shared instant.
func (app *App) Codes(ctx context.Context, now time.Time) (CodesResponse, error) {
	credentials, err := app.vault.ListCredentials(ctx)
	if err != nil {
		return CodesResponse{}, err
	}
	response := CodesResponse{ServerTime: now.UTC(), Codes: make([]CurrentCode, 0, len(credentials))}
	for _, credential := range credentials {
		code, err := totp.Generate(credential.TOTPConfig(), now)
		if err != nil {
			return CodesResponse{}, fmt.Errorf("generate code for %s: %w", credential.ID, err)
		}
		response.Codes = append(response.Codes, CurrentCode{CredentialID: credential.ID, Code: code.Value, ValidFrom: code.ValidFrom, ValidUntil: code.ValidUntil})
	}
	return response, nil
}

// Code returns one current code.
func (app *App) Code(ctx context.Context, id string, now time.Time) (CurrentCode, error) {
	credential, err := app.vault.GetCredential(ctx, id)
	if err != nil {
		return CurrentCode{}, err
	}
	code, err := totp.Generate(credential.TOTPConfig(), now)
	if err != nil {
		return CurrentCode{}, err
	}
	return CurrentCode{CredentialID: id, Code: code.Value, ValidFrom: code.ValidFrom, ValidUntil: code.ValidUntil}, nil
}

// Groups returns all groups.
func (app *App) Groups(ctx context.Context) ([]vault.Group, error) { return app.vault.ListGroups(ctx) }

// CreateGroup creates a group.
func (app *App) CreateGroup(ctx context.Context, group vault.Group) (vault.Group, error) {
	return app.vault.CreateGroup(ctx, group)
}

// UpdateGroup updates a group.
func (app *App) UpdateGroup(ctx context.Context, id string, group vault.Group) (vault.Group, error) {
	return app.vault.UpdateGroup(ctx, id, group)
}

// DeleteGroup deletes a group.
func (app *App) DeleteGroup(ctx context.Context, id string) error {
	return app.vault.DeleteGroup(ctx, id)
}

// APIKeys lists automation keys.
func (app *App) APIKeys(ctx context.Context) ([]vault.APIKey, error) {
	return app.vault.ListAPIKeys(ctx)
}

// CreateAPIKey creates an automation key.
func (app *App) CreateAPIKey(ctx context.Context, name string) (vault.CreatedAPIKey, error) {
	return app.vault.CreateAPIKey(ctx, name)
}

// DeleteAPIKey revokes an automation key.
func (app *App) DeleteAPIKey(ctx context.Context, id string) error {
	return app.vault.DeleteAPIKey(ctx, id)
}

// AuthenticateAPIKey validates a read-only key.
func (app *App) AuthenticateAPIKey(ctx context.Context, key string) (bool, error) {
	return app.vault.AuthenticateAPIKey(ctx, key)
}

// ExportBackup creates an encrypted backup.
func (app *App) ExportBackup(ctx context.Context, password string) ([]byte, error) {
	return app.vault.ExportBackup(ctx, password)
}

// PreviewBackup decrypts and validates a backup.
func (app *App) PreviewBackup(ctx context.Context, value []byte, password string) (vault.BackupPreview, error) {
	return app.vault.PreviewBackup(ctx, value, password)
}

// ApplyBackup transactionally applies a backup preview.
func (app *App) ApplyBackup(ctx context.Context, preview vault.BackupPreview, mode string) error {
	return app.vault.ApplyBackup(ctx, preview, mode)
}

func (app *App) credentialFromInput(input CredentialInput, update bool) (vault.Credential, error) {
	var config totp.Config
	var err error
	switch input.Source {
	case "uri":
		config, err = totp.ParseURI(input.URI)
	case "generate":
		secret := make([]byte, 20)
		if _, err = rand.Read(secret); err == nil {
			config = totp.Config{Secret: base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), Issuer: input.Issuer, Account: input.Account, Algorithm: input.Algorithm, Digits: input.Digits, Period: input.Period}
		}
	case "", "manual":
		config = totp.Config{Secret: input.Secret, Issuer: input.Issuer, Account: input.Account, Algorithm: input.Algorithm, Digits: input.Digits, Period: input.Period}
		if update && input.Secret == "" {
			if strings.TrimSpace(input.Account) == "" {
				return vault.Credential{}, errors.New("account is required")
			}
			return vault.Credential{Issuer: strings.TrimSpace(input.Issuer), Account: strings.TrimSpace(input.Account), Algorithm: input.Algorithm, Digits: input.Digits, Period: input.Period, Favorite: input.Favorite, GroupID: input.GroupID, Tags: input.Tags, Notes: input.Notes}, nil
		}
	default:
		return vault.Credential{}, errors.New("source must be manual, uri, or generate")
	}
	if err != nil {
		return vault.Credential{}, err
	}
	config, err = totp.Validate(config)
	if err != nil {
		return vault.Credential{}, err
	}
	return vault.Credential{Issuer: config.Issuer, Account: config.Account, Secret: config.Secret, Algorithm: config.Algorithm, Digits: config.Digits, Period: config.Period, Favorite: input.Favorite, GroupID: input.GroupID, Tags: input.Tags, Notes: input.Notes}, nil
}

func view(credential vault.Credential) CredentialView {
	return CredentialView{ID: credential.ID, Issuer: credential.Issuer, Account: credential.Account, Algorithm: credential.Algorithm, Digits: credential.Digits, Period: credential.Period, Favorite: credential.Favorite, GroupID: credential.GroupID, Tags: credential.Tags, Notes: credential.Notes, CreatedAt: credential.CreatedAt, UpdatedAt: credential.UpdatedAt}
}
