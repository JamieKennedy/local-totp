package vault

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/JamieKennedy/local-totp/internal/totp"
)

const (
	backupMagic   = "LOCAL-TOTP-BACKUP"
	backupVersion = 1
)

type backupEnvelope struct {
	Magic      string    `json:"magic"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"createdAt"`
	KDF        kdfParams `json:"kdf"`
	Nonce      []byte    `json:"nonce"`
	Ciphertext []byte    `json:"ciphertext"`
}

type backupPayload struct {
	Credentials []Credential `json:"credentials"`
	Groups      []Group      `json:"groups"`
}

// ExportBackup returns a versioned backup encrypted independently with the supplied master password.
func (vault *Vault) ExportBackup(ctx context.Context, password string) ([]byte, error) {
	if err := vault.verifyPassword(ctx, password); err != nil {
		return nil, err
	}
	credentials, err := vault.ListCredentials(ctx)
	if err != nil {
		return nil, err
	}
	groups, err := vault.ListGroups(ctx)
	if err != nil {
		return nil, err
	}
	plaintext, err := json.Marshal(backupPayload{Credentials: credentials, Groups: groups})
	if err != nil {
		return nil, fmt.Errorf("encode backup: %w", err)
	}
	params, err := defaultKDF()
	if err != nil {
		return nil, err
	}
	key := passwordKey(password, params)
	nonce, ciphertext, err := seal(key, plaintext, []byte("local-totp-backup:v1"))
	wipe(key)
	if err != nil {
		return nil, err
	}
	envelope := backupEnvelope{Magic: backupMagic, Version: backupVersion, CreatedAt: time.Now().UTC(), KDF: params, Nonce: nonce, Ciphertext: ciphertext}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("encode backup envelope: %w", err)
	}
	return encoded, nil
}

// PreviewBackup validates and decrypts an encrypted backup without changing stored data.
func (vault *Vault) PreviewBackup(ctx context.Context, encoded []byte, password string) (BackupPreview, error) {
	if len(encoded) > 10*1024*1024 {
		return BackupPreview{}, errors.New("backup exceeds 10 MiB limit")
	}
	var envelope backupEnvelope
	if err := json.Unmarshal(encoded, &envelope); err != nil {
		return BackupPreview{}, errors.New("backup envelope is malformed")
	}
	if envelope.Magic != backupMagic || envelope.Version != backupVersion {
		return BackupPreview{}, errors.New("backup format or version is unsupported")
	}
	if envelope.KDF.Memory < 8*1024 || envelope.KDF.Memory > 256*1024 || envelope.KDF.Time < 1 || envelope.KDF.Time > 10 || envelope.KDF.Threads < 1 || envelope.KDF.Threads > 16 || len(envelope.KDF.Salt) != 16 {
		return BackupPreview{}, errors.New("backup KDF parameters are outside safe limits")
	}
	key := passwordKey(password, envelope.KDF)
	plaintext, err := openSealed(key, envelope.Nonce, envelope.Ciphertext, []byte("local-totp-backup:v1"))
	wipe(key)
	if err != nil {
		return BackupPreview{}, ErrUnauthorized
	}
	var payload backupPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return BackupPreview{}, errors.New("backup payload is malformed")
	}
	if len(payload.Credentials) > 5000 || len(payload.Groups) > 500 {
		return BackupPreview{}, errors.New("backup contains too many records")
	}
	for index := range payload.Credentials {
		validated, err := validateBackupCredential(payload.Credentials[index])
		if err != nil {
			return BackupPreview{}, fmt.Errorf("credential %d: %w", index+1, err)
		}
		payload.Credentials[index] = validated
	}
	for index := range payload.Groups {
		payload.Groups[index].Name = strings.TrimSpace(payload.Groups[index].Name)
		if payload.Groups[index].Name == "" {
			return BackupPreview{}, fmt.Errorf("group %d has no name", index+1)
		}
	}
	existing, err := vault.ListCredentials(ctx)
	if err != nil {
		return BackupPreview{}, err
	}
	duplicates, conflicts := 0, 0
	for _, imported := range payload.Credentials {
		for _, current := range existing {
			if sameTOTP(imported, current) {
				duplicates++
				break
			}
			if strings.EqualFold(displayName(imported), displayName(current)) {
				conflicts++
				break
			}
		}
	}
	id, err := newID()
	if err != nil {
		return BackupPreview{}, err
	}
	return BackupPreview{ID: id, Credentials: len(payload.Credentials), Groups: len(payload.Groups), Duplicates: duplicates, NameConflicts: conflicts, CreatedAt: envelope.CreatedAt.Format(time.RFC3339), plan: payload}, nil
}

// ApplyBackup applies a previously previewed backup in merge or replace mode.
func (vault *Vault) ApplyBackup(ctx context.Context, preview BackupPreview, mode string) error {
	if mode != "merge" && mode != "replace" {
		return errors.New("backup mode must be merge or replace")
	}
	key, err := vault.copyKey()
	if err != nil {
		return err
	}
	defer wipe(key)
	existingCredentials, err := vault.ListCredentials(ctx)
	if err != nil {
		return err
	}
	existingGroups, err := vault.ListGroups(ctx)
	if err != nil {
		return err
	}
	tx, err := vault.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin backup import: %w", err)
	}
	defer tx.Rollback()
	if mode == "replace" {
		if _, err := tx.ExecContext(ctx, `DELETE FROM credentials`); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM groups_data`); err != nil {
			return err
		}
		existingCredentials, existingGroups = nil, nil
	}
	groupMap := make(map[string]string)
	usedGroupIDs := make(map[string]bool)
	for _, group := range existingGroups {
		usedGroupIDs[group.ID] = true
	}
	for _, imported := range preview.plan.Groups {
		matched := ""
		for _, current := range existingGroups {
			if strings.EqualFold(current.Name, imported.Name) {
				matched = current.ID
				break
			}
		}
		if matched != "" {
			groupMap[imported.ID] = matched
			continue
		}
		originalID := imported.ID
		if imported.ID == "" || usedGroupIDs[imported.ID] {
			imported.ID, err = newID()
			if err != nil {
				return err
			}
		}
		if imported.CreatedAt.IsZero() {
			imported.CreatedAt = time.Now().UTC()
		}
		imported.UpdatedAt = time.Now().UTC()
		if err := putEncryptedTx(ctx, tx, key, "groups_data", imported.ID, imported, imported.CreatedAt, imported.UpdatedAt); err != nil {
			return err
		}
		groupMap[originalID] = imported.ID
		usedGroupIDs[imported.ID] = true
		existingGroups = append(existingGroups, imported)
	}
	usedCredentialIDs := make(map[string]bool)
	for _, credential := range existingCredentials {
		usedCredentialIDs[credential.ID] = true
	}
	for _, imported := range preview.plan.Credentials {
		duplicate := false
		for _, current := range existingCredentials {
			if sameTOTP(imported, current) {
				duplicate = true
				break
			}
		}
		if duplicate && mode == "merge" {
			continue
		}
		if mapped, ok := groupMap[imported.GroupID]; ok {
			imported.GroupID = mapped
		} else {
			imported.GroupID = ""
		}
		if imported.ID == "" || usedCredentialIDs[imported.ID] {
			imported.ID, err = newID()
			if err != nil {
				return err
			}
		}
		if imported.CreatedAt.IsZero() {
			imported.CreatedAt = time.Now().UTC()
		}
		imported.UpdatedAt = time.Now().UTC()
		if err := putEncryptedTx(ctx, tx, key, "credentials", imported.ID, imported, imported.CreatedAt, imported.UpdatedAt); err != nil {
			return err
		}
		usedCredentialIDs[imported.ID] = true
		existingCredentials = append(existingCredentials, imported)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit backup import: %w", err)
	}
	return nil
}

func (vault *Vault) verifyPassword(ctx context.Context, password string) error {
	metadata, err := vault.loadMetadata(ctx)
	if err != nil {
		return err
	}
	key := passwordKey(password, metadata.KDF)
	defer wipe(key)
	plaintext, err := openSealed(key, metadata.MasterNonce, metadata.MasterWrapped, []byte("vault-key:master:v1"))
	if err != nil {
		return ErrUnauthorized
	}
	wipe(plaintext)
	return nil
}

func putEncryptedTx(ctx context.Context, tx *sql.Tx, key []byte, table, id string, value any, createdAt, updatedAt time.Time) error {
	plaintext, err := json.Marshal(value)
	if err != nil {
		return err
	}
	nonce, payload, err := seal(key, plaintext, []byte(table+":"+id+":"+schemaVersion))
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO `+table+`(id,nonce,payload,created_at,updated_at) VALUES(?,?,?,?,?)`, id, nonce, payload, createdAt.UnixNano(), updatedAt.UnixNano())
	return err
}

func validateBackupCredential(credential Credential) (Credential, error) {
	config, err := totp.Validate(credential.TOTPConfig())
	if err != nil {
		return Credential{}, err
	}
	credential.Issuer, credential.Account, credential.Secret = config.Issuer, config.Account, config.Secret
	credential.Algorithm, credential.Digits, credential.Period = config.Algorithm, config.Digits, config.Period
	credential.Tags = normalizeTags(credential.Tags)
	return credential, validateCredentialMetadata(credential)
}

func sameTOTP(left, right Credential) bool {
	return left.Secret == right.Secret && left.Algorithm == right.Algorithm && left.Digits == right.Digits && left.Period == right.Period
}

func displayName(credential Credential) string {
	if credential.Issuer == "" {
		return credential.Account
	}
	return credential.Issuer + ":" + credential.Account
}
