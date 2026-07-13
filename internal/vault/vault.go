package vault

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/JamieKennedy/local-totp/internal/totp"
	_ "modernc.org/sqlite"
)

const schemaVersion = "1"

// Vault provides encrypted local persistence and key lifecycle operations.
type Vault struct {
	db  *sql.DB
	mu  sync.RWMutex
	key []byte
}

type metadata struct {
	KDF             kdfParams
	MasterNonce     []byte
	MasterWrapped   []byte
	RecoverySalt    []byte
	RecoveryNonce   []byte
	RecoveryWrapped []byte
}

// Open opens or creates the SQLite file and applies forward-only migrations.
func Open(ctx context.Context, path string) (*Vault, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open SQLite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	for _, pragma := range []string{"PRAGMA foreign_keys = ON", "PRAGMA busy_timeout = 5000", "PRAGMA journal_mode = DELETE", "PRAGMA synchronous = FULL"} {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("configure SQLite: %w", err)
		}
	}
	vault := &Vault{db: db}
	if err := vault.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return vault, nil
}

// Close locks the vault and closes SQLite.
func (vault *Vault) Close() error {
	vault.Lock()
	return vault.db.Close()
}

func (vault *Vault) migrate(ctx context.Context) error {
	const migration = `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS vault_metadata (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  kdf_memory INTEGER NOT NULL,
  kdf_time INTEGER NOT NULL,
  kdf_threads INTEGER NOT NULL,
  kdf_salt BLOB NOT NULL,
  master_nonce BLOB NOT NULL,
  master_wrapped BLOB NOT NULL,
  recovery_salt BLOB NOT NULL,
  recovery_nonce BLOB NOT NULL,
  recovery_wrapped BLOB NOT NULL
);
CREATE TABLE IF NOT EXISTS credentials (
  id TEXT PRIMARY KEY,
  nonce BLOB NOT NULL,
  payload BLOB NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS groups_data (
  id TEXT PRIMARY KEY,
  nonce BLOB NOT NULL,
  payload BLOB NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS api_keys (
  id TEXT PRIMARY KEY,
  key_hash BLOB NOT NULL UNIQUE,
  name_nonce BLOB NOT NULL,
  name_payload BLOB NOT NULL,
  created_at INTEGER NOT NULL,
  last_used INTEGER
);
INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES (1, unixepoch());`
	if _, err := vault.db.ExecContext(ctx, migration); err != nil {
		return fmt.Errorf("apply migration 1: %w", err)
	}
	return nil
}

// IsSetup reports whether vault setup has completed.
func (vault *Vault) IsSetup(ctx context.Context) (bool, error) {
	var count int
	if err := vault.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM vault_metadata WHERE id = 1").Scan(&count); err != nil {
		return false, fmt.Errorf("check setup: %w", err)
	}
	return count == 1, nil
}

// IsLocked reports whether the data-encryption key is absent from memory.
func (vault *Vault) IsLocked() bool {
	vault.mu.RLock()
	defer vault.mu.RUnlock()
	return len(vault.key) == 0
}

// Setup creates the encrypted vault and returns a one-time recovery key.
func (vault *Vault) Setup(ctx context.Context, password string) (string, error) {
	if len(password) < 12 {
		return "", errors.New("master password must contain at least 12 characters")
	}
	setup, err := vault.IsSetup(ctx)
	if err != nil {
		return "", err
	}
	if setup {
		return "", ErrAlreadySetup
	}
	dataKey, err := randomBytes(keySize)
	if err != nil {
		return "", err
	}
	defer wipe(dataKey)
	params, err := defaultKDF()
	if err != nil {
		return "", err
	}
	masterKey := passwordKey(password, params)
	masterNonce, masterWrapped, err := seal(masterKey, dataKey, []byte("vault-key:master:v1"))
	wipe(masterKey)
	if err != nil {
		return "", err
	}
	recoveryDisplay, recoveryRaw, err := generateRecovery()
	if err != nil {
		return "", err
	}
	defer wipe(recoveryRaw)
	recoverySalt, err := randomBytes(16)
	if err != nil {
		return "", err
	}
	recoveryDerived := recoveryKey(recoveryRaw, recoverySalt)
	recoveryNonce, recoveryWrapped, err := seal(recoveryDerived, dataKey, []byte("vault-key:recovery:v1"))
	wipe(recoveryDerived)
	if err != nil {
		return "", err
	}
	_, err = vault.db.ExecContext(ctx, `INSERT INTO vault_metadata(
id,kdf_memory,kdf_time,kdf_threads,kdf_salt,master_nonce,master_wrapped,recovery_salt,recovery_nonce,recovery_wrapped)
VALUES(1,?,?,?,?,?,?,?,?,?)`, params.Memory, params.Time, params.Threads, params.Salt, masterNonce, masterWrapped, recoverySalt, recoveryNonce, recoveryWrapped)
	if err != nil {
		return "", fmt.Errorf("store vault metadata: %w", err)
	}
	vault.setKey(dataKey)
	return recoveryDisplay, nil
}

// Unlock unwraps and retains the data key using the master password.
func (vault *Vault) Unlock(ctx context.Context, password string) error {
	metadata, err := vault.loadMetadata(ctx)
	if err != nil {
		return err
	}
	key := passwordKey(password, metadata.KDF)
	defer wipe(key)
	dataKey, err := openSealed(key, metadata.MasterNonce, metadata.MasterWrapped, []byte("vault-key:master:v1"))
	if err != nil {
		return ErrUnauthorized
	}
	defer wipe(dataKey)
	vault.setKey(dataKey)
	return nil
}

// Lock clears the in-memory data key on a best-effort basis.
func (vault *Vault) Lock() {
	vault.mu.Lock()
	defer vault.mu.Unlock()
	wipe(vault.key)
	vault.key = nil
}

// ChangePassword validates the current password and rewraps the data key.
func (vault *Vault) ChangePassword(ctx context.Context, current, replacement string) error {
	if len(replacement) < 12 {
		return errors.New("new master password must contain at least 12 characters")
	}
	metadata, err := vault.loadMetadata(ctx)
	if err != nil {
		return err
	}
	currentKey := passwordKey(current, metadata.KDF)
	dataKey, err := openSealed(currentKey, metadata.MasterNonce, metadata.MasterWrapped, []byte("vault-key:master:v1"))
	wipe(currentKey)
	if err != nil {
		return ErrUnauthorized
	}
	defer wipe(dataKey)
	params, err := defaultKDF()
	if err != nil {
		return err
	}
	newKey := passwordKey(replacement, params)
	nonce, wrapped, err := seal(newKey, dataKey, []byte("vault-key:master:v1"))
	wipe(newKey)
	if err != nil {
		return err
	}
	_, err = vault.db.ExecContext(ctx, `UPDATE vault_metadata SET kdf_memory=?,kdf_time=?,kdf_threads=?,kdf_salt=?,master_nonce=?,master_wrapped=? WHERE id=1`, params.Memory, params.Time, params.Threads, params.Salt, nonce, wrapped)
	if err != nil {
		return fmt.Errorf("update password wrapper: %w", err)
	}
	vault.setKey(dataKey)
	return nil
}

// Recover unwraps the vault with the recovery key, replaces the password, and rotates recovery material.
func (vault *Vault) Recover(ctx context.Context, recoveryDisplay, replacement string) (string, error) {
	if len(replacement) < 12 {
		return "", errors.New("new master password must contain at least 12 characters")
	}
	metadata, err := vault.loadMetadata(ctx)
	if err != nil {
		return "", err
	}
	recoveryRaw, err := parseRecovery(recoveryDisplay)
	if err != nil {
		return "", ErrUnauthorized
	}
	defer wipe(recoveryRaw)
	derived := recoveryKey(recoveryRaw, metadata.RecoverySalt)
	dataKey, err := openSealed(derived, metadata.RecoveryNonce, metadata.RecoveryWrapped, []byte("vault-key:recovery:v1"))
	wipe(derived)
	if err != nil {
		return "", ErrUnauthorized
	}
	defer wipe(dataKey)
	params, err := defaultKDF()
	if err != nil {
		return "", err
	}
	masterKey := passwordKey(replacement, params)
	masterNonce, masterWrapped, err := seal(masterKey, dataKey, []byte("vault-key:master:v1"))
	wipe(masterKey)
	if err != nil {
		return "", err
	}
	display, raw, salt, recoveryNonce, recoveryWrapped, err := wrapNewRecovery(dataKey)
	if err != nil {
		return "", err
	}
	defer wipe(raw)
	_, err = vault.db.ExecContext(ctx, `UPDATE vault_metadata SET
kdf_memory=?,kdf_time=?,kdf_threads=?,kdf_salt=?,master_nonce=?,master_wrapped=?,recovery_salt=?,recovery_nonce=?,recovery_wrapped=? WHERE id=1`,
		params.Memory, params.Time, params.Threads, params.Salt, masterNonce, masterWrapped, salt, recoveryNonce, recoveryWrapped)
	if err != nil {
		return "", fmt.Errorf("store recovery rotation: %w", err)
	}
	vault.setKey(dataKey)
	return display, nil
}

// RotateRecovery replaces the recovery wrapper and returns the new one-time key.
func (vault *Vault) RotateRecovery(ctx context.Context) (string, error) {
	key, err := vault.copyKey()
	if err != nil {
		return "", err
	}
	defer wipe(key)
	display, raw, salt, nonce, wrapped, err := wrapNewRecovery(key)
	if err != nil {
		return "", err
	}
	defer wipe(raw)
	if _, err := vault.db.ExecContext(ctx, `UPDATE vault_metadata SET recovery_salt=?,recovery_nonce=?,recovery_wrapped=? WHERE id=1`, salt, nonce, wrapped); err != nil {
		return "", fmt.Errorf("rotate recovery wrapper: %w", err)
	}
	return display, nil
}

func wrapNewRecovery(dataKey []byte) (string, []byte, []byte, []byte, []byte, error) {
	display, raw, err := generateRecovery()
	if err != nil {
		return "", nil, nil, nil, nil, err
	}
	salt, err := randomBytes(16)
	if err != nil {
		return "", raw, nil, nil, nil, err
	}
	derived := recoveryKey(raw, salt)
	nonce, wrapped, err := seal(derived, dataKey, []byte("vault-key:recovery:v1"))
	wipe(derived)
	return display, raw, salt, nonce, wrapped, err
}

// CreateCredential validates, encrypts, and stores a credential.
func (vault *Vault) CreateCredential(ctx context.Context, credential Credential) (Credential, error) {
	config, err := totp.Validate(credential.TOTPConfig())
	if err != nil {
		return Credential{}, err
	}
	credential.Issuer, credential.Account, credential.Secret = config.Issuer, config.Account, config.Secret
	credential.Algorithm, credential.Digits, credential.Period = config.Algorithm, config.Digits, config.Period
	credential.ID, err = newID()
	if err != nil {
		return Credential{}, err
	}
	now := time.Now().UTC()
	credential.CreatedAt, credential.UpdatedAt = now, now
	credential.Tags = normalizeTags(credential.Tags)
	if err := validateCredentialMetadata(credential); err != nil {
		return Credential{}, err
	}
	if err := vault.putEncrypted(ctx, "credentials", credential.ID, credential, now, now, false); err != nil {
		return Credential{}, err
	}
	return credential, nil
}

// ListCredentials decrypts all credentials in stable display order.
func (vault *Vault) ListCredentials(ctx context.Context) ([]Credential, error) {
	rows, err := vault.db.QueryContext(ctx, `SELECT id,nonce,payload FROM credentials ORDER BY created_at,id`)
	if err != nil {
		return nil, fmt.Errorf("list credentials: %w", err)
	}
	defer rows.Close()
	credentials := []Credential{}
	for rows.Next() {
		var id string
		var nonce, payload []byte
		if err := rows.Scan(&id, &nonce, &payload); err != nil {
			return nil, fmt.Errorf("scan credential: %w", err)
		}
		var credential Credential
		if err := vault.decryptRecord("credentials", id, nonce, payload, &credential); err != nil {
			return nil, err
		}
		credentials = append(credentials, credential)
	}
	return credentials, rows.Err()
}

// GetCredential decrypts one credential.
func (vault *Vault) GetCredential(ctx context.Context, id string) (Credential, error) {
	var nonce, payload []byte
	if err := vault.db.QueryRowContext(ctx, `SELECT nonce,payload FROM credentials WHERE id=?`, id).Scan(&nonce, &payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Credential{}, ErrNotFound
		}
		return Credential{}, fmt.Errorf("get credential: %w", err)
	}
	var credential Credential
	if err := vault.decryptRecord("credentials", id, nonce, payload, &credential); err != nil {
		return Credential{}, err
	}
	return credential, nil
}

// UpdateCredential replaces a credential while preserving identity and creation time.
func (vault *Vault) UpdateCredential(ctx context.Context, id string, replacement Credential) (Credential, error) {
	existing, err := vault.GetCredential(ctx, id)
	if err != nil {
		return Credential{}, err
	}
	if replacement.Secret == "" {
		replacement.Secret = existing.Secret
	}
	config, err := totp.Validate(replacement.TOTPConfig())
	if err != nil {
		return Credential{}, err
	}
	replacement.ID, replacement.CreatedAt, replacement.UpdatedAt = id, existing.CreatedAt, time.Now().UTC()
	replacement.Issuer, replacement.Account, replacement.Secret = config.Issuer, config.Account, config.Secret
	replacement.Algorithm, replacement.Digits, replacement.Period = config.Algorithm, config.Digits, config.Period
	replacement.Tags = normalizeTags(replacement.Tags)
	if err := validateCredentialMetadata(replacement); err != nil {
		return Credential{}, err
	}
	if err := vault.putEncrypted(ctx, "credentials", id, replacement, replacement.CreatedAt, replacement.UpdatedAt, true); err != nil {
		return Credential{}, err
	}
	return replacement, nil
}

// DeleteCredential permanently removes a credential.
func (vault *Vault) DeleteCredential(ctx context.Context, id string) error {
	result, err := vault.db.ExecContext(ctx, `DELETE FROM credentials WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("delete credential: %w", err)
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateGroup validates, encrypts, and stores a group.
func (vault *Vault) CreateGroup(ctx context.Context, group Group) (Group, error) {
	group.Name = strings.TrimSpace(group.Name)
	if group.Name == "" || len(group.Name) > 80 {
		return Group{}, errors.New("group name must contain 1 to 80 characters")
	}
	if group.Color == "" {
		group.Color = "#64748b"
	}
	var err error
	group.ID, err = newID()
	if err != nil {
		return Group{}, err
	}
	now := time.Now().UTC()
	group.CreatedAt, group.UpdatedAt = now, now
	if err := vault.ensureUniqueGroupName(ctx, group.Name, ""); err != nil {
		return Group{}, err
	}
	if err := vault.putEncrypted(ctx, "groups_data", group.ID, group, now, now, false); err != nil {
		return Group{}, err
	}
	return group, nil
}

// ListGroups decrypts all groups sorted by name.
func (vault *Vault) ListGroups(ctx context.Context) ([]Group, error) {
	rows, err := vault.db.QueryContext(ctx, `SELECT id,nonce,payload FROM groups_data`)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	defer rows.Close()
	groups := []Group{}
	for rows.Next() {
		var id string
		var nonce, payload []byte
		if err := rows.Scan(&id, &nonce, &payload); err != nil {
			return nil, err
		}
		var group Group
		if err := vault.decryptRecord("groups_data", id, nonce, payload, &group); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	sort.Slice(groups, func(i, j int) bool { return strings.ToLower(groups[i].Name) < strings.ToLower(groups[j].Name) })
	return groups, rows.Err()
}

// UpdateGroup replaces a group while preserving identity.
func (vault *Vault) UpdateGroup(ctx context.Context, id string, replacement Group) (Group, error) {
	groups, err := vault.ListGroups(ctx)
	if err != nil {
		return Group{}, err
	}
	var existing *Group
	for index := range groups {
		if groups[index].ID == id {
			existing = &groups[index]
			break
		}
	}
	if existing == nil {
		return Group{}, ErrNotFound
	}
	replacement.Name = strings.TrimSpace(replacement.Name)
	if replacement.Name == "" || len(replacement.Name) > 80 {
		return Group{}, errors.New("group name must contain 1 to 80 characters")
	}
	if replacement.Color == "" {
		replacement.Color = existing.Color
	}
	if err := vault.ensureUniqueGroupName(ctx, replacement.Name, id); err != nil {
		return Group{}, err
	}
	replacement.ID, replacement.CreatedAt, replacement.UpdatedAt = id, existing.CreatedAt, time.Now().UTC()
	if err := vault.putEncrypted(ctx, "groups_data", id, replacement, replacement.CreatedAt, replacement.UpdatedAt, true); err != nil {
		return Group{}, err
	}
	return replacement, nil
}

// DeleteGroup removes a group and clears references from credentials.
func (vault *Vault) DeleteGroup(ctx context.Context, id string) error {
	if _, err := vault.db.ExecContext(ctx, `DELETE FROM groups_data WHERE id=?`, id); err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	credentials, err := vault.ListCredentials(ctx)
	if err != nil {
		return err
	}
	for _, credential := range credentials {
		if credential.GroupID == id {
			credential.GroupID = ""
			if _, err := vault.UpdateCredential(ctx, credential.ID, credential); err != nil {
				return err
			}
		}
	}
	return nil
}

// CreateAPIKey creates a named read-only key and returns its plaintext once.
func (vault *Vault) CreateAPIKey(ctx context.Context, name string) (CreatedAPIKey, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 80 {
		return CreatedAPIKey{}, errors.New("API key name must contain 1 to 80 characters")
	}
	key, _, err := randomToken("ltotp_")
	if err != nil {
		return CreatedAPIKey{}, err
	}
	hash, _ := hashAPIKey(key)
	id, err := newID()
	if err != nil {
		return CreatedAPIKey{}, err
	}
	dataKey, err := vault.copyKey()
	if err != nil {
		return CreatedAPIKey{}, err
	}
	defer wipe(dataKey)
	nonce, payload, err := seal(dataKey, []byte(name), []byte("api-key:"+id+":"+schemaVersion))
	if err != nil {
		return CreatedAPIKey{}, err
	}
	now := time.Now().UTC()
	if _, err := vault.db.ExecContext(ctx, `INSERT INTO api_keys(id,key_hash,name_nonce,name_payload,created_at) VALUES(?,?,?,?,?)`, id, hash, nonce, payload, now.UnixNano()); err != nil {
		return CreatedAPIKey{}, fmt.Errorf("store API key: %w", err)
	}
	return CreatedAPIKey{APIKey: APIKey{ID: id, Name: name, CreatedAt: now}, Key: key}, nil
}

// ListAPIKeys returns API-key metadata without key material.
func (vault *Vault) ListAPIKeys(ctx context.Context) ([]APIKey, error) {
	dataKey, err := vault.copyKey()
	if err != nil {
		return nil, err
	}
	defer wipe(dataKey)
	rows, err := vault.db.QueryContext(ctx, `SELECT id,name_nonce,name_payload,created_at,last_used FROM api_keys ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := []APIKey{}
	for rows.Next() {
		var id string
		var nonce, payload []byte
		var created int64
		var last sql.NullInt64
		if err := rows.Scan(&id, &nonce, &payload, &created, &last); err != nil {
			return nil, err
		}
		name, err := openSealed(dataKey, nonce, payload, []byte("api-key:"+id+":"+schemaVersion))
		if err != nil {
			return nil, fmt.Errorf("decrypt API key name: %w", err)
		}
		item := APIKey{ID: id, Name: string(name), CreatedAt: time.Unix(0, created).UTC()}
		if last.Valid {
			value := time.Unix(0, last.Int64).UTC()
			item.LastUsed = &value
		}
		keys = append(keys, item)
	}
	return keys, rows.Err()
}

// AuthenticateAPIKey validates a high-entropy read-only key.
func (vault *Vault) AuthenticateAPIKey(ctx context.Context, value string) (bool, error) {
	if vault.IsLocked() {
		return false, ErrLocked
	}
	hash, err := hashAPIKey(value)
	if err != nil {
		return false, nil
	}
	var id string
	var stored []byte
	if err := vault.db.QueryRowContext(ctx, `SELECT id,key_hash FROM api_keys WHERE key_hash=?`, hash).Scan(&id, &stored); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	valid := subtle.ConstantTimeCompare(hash, stored) == 1
	if valid {
		_, _ = vault.db.ExecContext(ctx, `UPDATE api_keys SET last_used=? WHERE id=?`, time.Now().UTC().UnixNano(), id)
	}
	return valid, nil
}

// DeleteAPIKey revokes a named key.
func (vault *Vault) DeleteAPIKey(ctx context.Context, id string) error {
	result, err := vault.db.ExecContext(ctx, `DELETE FROM api_keys WHERE id=?`, id)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func (vault *Vault) putEncrypted(ctx context.Context, table, id string, value any, createdAt, updatedAt time.Time, update bool) error {
	key, err := vault.copyKey()
	if err != nil {
		return err
	}
	defer wipe(key)
	plaintext, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal encrypted record: %w", err)
	}
	nonce, payload, err := seal(key, plaintext, []byte(table+":"+id+":"+schemaVersion))
	if err != nil {
		return err
	}
	var result sql.Result
	if update {
		result, err = vault.db.ExecContext(ctx, `UPDATE `+table+` SET nonce=?,payload=?,updated_at=? WHERE id=?`, nonce, payload, updatedAt.UnixNano(), id)
	} else {
		result, err = vault.db.ExecContext(ctx, `INSERT INTO `+table+`(id,nonce,payload,created_at,updated_at) VALUES(?,?,?,?,?)`, id, nonce, payload, createdAt.UnixNano(), updatedAt.UnixNano())
	}
	if err != nil {
		return fmt.Errorf("store encrypted %s record: %w", table, err)
	}
	if update {
		count, _ := result.RowsAffected()
		if count == 0 {
			return ErrNotFound
		}
	}
	return nil
}

func (vault *Vault) decryptRecord(table, id string, nonce, payload []byte, target any) error {
	key, err := vault.copyKey()
	if err != nil {
		return err
	}
	defer wipe(key)
	plaintext, err := openSealed(key, nonce, payload, []byte(table+":"+id+":"+schemaVersion))
	if err != nil {
		return fmt.Errorf("decrypt %s record: %w", table, err)
	}
	if err := json.Unmarshal(plaintext, target); err != nil {
		return fmt.Errorf("decode %s record: %w", table, err)
	}
	return nil
}

func (vault *Vault) copyKey() ([]byte, error) {
	vault.mu.RLock()
	defer vault.mu.RUnlock()
	if len(vault.key) == 0 {
		return nil, ErrLocked
	}
	return append([]byte(nil), vault.key...), nil
}

func (vault *Vault) setKey(key []byte) {
	vault.mu.Lock()
	defer vault.mu.Unlock()
	wipe(vault.key)
	vault.key = append(vault.key[:0], key...)
}

func (vault *Vault) loadMetadata(ctx context.Context) (metadata, error) {
	var item metadata
	var memory, iterations, threads int64
	err := vault.db.QueryRowContext(ctx, `SELECT kdf_memory,kdf_time,kdf_threads,kdf_salt,master_nonce,master_wrapped,recovery_salt,recovery_nonce,recovery_wrapped FROM vault_metadata WHERE id=1`).Scan(
		&memory, &iterations, &threads, &item.KDF.Salt, &item.MasterNonce, &item.MasterWrapped, &item.RecoverySalt, &item.RecoveryNonce, &item.RecoveryWrapped)
	if errors.Is(err, sql.ErrNoRows) {
		return metadata{}, ErrNotSetup
	}
	if err != nil {
		return metadata{}, fmt.Errorf("load vault metadata: %w", err)
	}
	item.KDF.Memory, item.KDF.Time, item.KDF.Threads = uint32(memory), uint32(iterations), uint8(threads)
	return item, nil
}

func (vault *Vault) ensureUniqueGroupName(ctx context.Context, name, exceptID string) error {
	groups, err := vault.ListGroups(ctx)
	if err != nil && !errors.Is(err, ErrLocked) {
		return err
	}
	for _, group := range groups {
		if group.ID != exceptID && strings.EqualFold(group.Name, name) {
			return ErrConflict
		}
	}
	return nil
}

func validateCredentialMetadata(credential Credential) error {
	if len(credential.Issuer) > 120 || len(credential.Account) > 200 || len(credential.Notes) > 2000 {
		return errors.New("credential metadata exceeds allowed length")
	}
	if len(credential.Tags) > 32 {
		return errors.New("a credential may have at most 32 tags")
	}
	return nil
}

func normalizeTags(tags []string) []string {
	seen := map[string]bool{}
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || len(tag) > 40 || seen[strings.ToLower(tag)] {
			continue
		}
		seen[strings.ToLower(tag)] = true
		normalized = append(normalized, tag)
	}
	sort.Slice(normalized, func(i, j int) bool { return strings.ToLower(normalized[i]) < strings.ToLower(normalized[j]) })
	return normalized
}

func newID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate ID: %w", err)
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(value)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32], nil
}
