// Package vault owns encrypted persistence and vault-key lifecycle.
package vault

import (
	"errors"
	"time"

	"github.com/JamieKennedy/local-totp/internal/totp"
)

var (
	// ErrNotSetup indicates that the vault has not been initialized.
	ErrNotSetup = errors.New("vault is not set up")
	// ErrAlreadySetup indicates that setup has already completed.
	ErrAlreadySetup = errors.New("vault is already set up")
	// ErrLocked indicates that an operation requires an unlocked vault.
	ErrLocked = errors.New("vault is locked")
	// ErrUnauthorized indicates that supplied credentials could not unwrap the vault key.
	ErrUnauthorized = errors.New("credentials are invalid")
	// ErrNotFound indicates that the requested record does not exist.
	ErrNotFound = errors.New("record not found")
	// ErrConflict indicates a unique record conflict.
	ErrConflict = errors.New("record conflicts with existing data")
)

// Credential is an encrypted TOTP record. Secret is never serialized by normal HTTP responses.
type Credential struct {
	ID        string         `json:"id"`
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
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

// TOTPConfig returns the generation configuration held by a credential.
func (credential Credential) TOTPConfig() totp.Config {
	return totp.Config{
		Secret: credential.Secret, Issuer: credential.Issuer, Account: credential.Account,
		Algorithm: credential.Algorithm, Digits: credential.Digits, Period: credential.Period,
	}
}

// Group is a single folder-like credential category.
type Group struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// APIKey describes a named read-only automation credential.
type APIKey struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"createdAt"`
	LastUsed  *time.Time `json:"lastUsed,omitempty"`
}

// CreatedAPIKey includes plaintext key material returned once at creation.
type CreatedAPIKey struct {
	APIKey
	Key string `json:"key"`
}

// BackupPreview describes the effect of applying an encrypted backup.
type BackupPreview struct {
	ID            string `json:"id"`
	Credentials   int    `json:"credentials"`
	Groups        int    `json:"groups"`
	Duplicates    int    `json:"duplicates"`
	NameConflicts int    `json:"nameConflicts"`
	CreatedAt     string `json:"createdAt"`
	plan          backupPayload
}
