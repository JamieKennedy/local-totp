package vault

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/JamieKennedy/local-totp/internal/totp"
)

const testPassword = "synthetic-test-password"

func TestVaultLifecycleAndRecovery(t *testing.T) {
	ctx := context.Background()
	vault, err := Open(ctx, filepath.Join(t.TempDir(), "vault.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer vault.Close()
	recovery, err := vault.Setup(ctx, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	credential, err := vault.CreateCredential(ctx, Credential{
		Issuer: "Example", Account: "developer@example.test", Secret: "JBSWY3DPEHPK3PXP",
		Algorithm: totp.AlgorithmSHA1, Digits: 6, Period: 30, Tags: []string{"Test", "test", "staging"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(credential.Tags) != 2 {
		t.Fatalf("expected normalized tags, got %#v", credential.Tags)
	}
	vault.Lock()
	if _, err := vault.ListCredentials(ctx); !errors.Is(err, ErrLocked) {
		t.Fatalf("expected locked error, got %v", err)
	}
	if err := vault.Unlock(ctx, "incorrect-password"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
	if err := vault.Unlock(ctx, testPassword); err != nil {
		t.Fatal(err)
	}
	stored, err := vault.GetCredential(ctx, credential.ID)
	if err != nil || stored.Secret != credential.Secret {
		t.Fatalf("credential mismatch: %#v, %v", stored, err)
	}
	newRecovery, err := vault.Recover(ctx, recovery, "replacement-test-password")
	if err != nil || newRecovery == recovery {
		t.Fatalf("recovery failed or did not rotate: %v", err)
	}
	vault.Lock()
	if err := vault.Unlock(ctx, testPassword); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("old password should fail, got %v", err)
	}
	if err := vault.Unlock(ctx, "replacement-test-password"); err != nil {
		t.Fatal(err)
	}
}

func TestBackupMergeAndTamperRejection(t *testing.T) {
	ctx := context.Background()
	source, err := Open(ctx, filepath.Join(t.TempDir(), "source.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if _, err := source.Setup(ctx, testPassword); err != nil {
		t.Fatal(err)
	}
	if _, err := source.CreateCredential(ctx, Credential{Issuer: "Example", Account: "backup", Secret: "JBSWY3DPEHPK3PXP", Algorithm: totp.AlgorithmSHA1, Digits: 6, Period: 30}); err != nil {
		t.Fatal(err)
	}
	encoded, err := source.ExportBackup(ctx, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	destination, err := Open(ctx, filepath.Join(t.TempDir(), "destination.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer destination.Close()
	if _, err := destination.Setup(ctx, "destination-password"); err != nil {
		t.Fatal(err)
	}
	preview, err := destination.PreviewBackup(ctx, encoded, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if err := destination.ApplyBackup(ctx, preview, "merge"); err != nil {
		t.Fatal(err)
	}
	credentials, err := destination.ListCredentials(ctx)
	if err != nil || len(credentials) != 1 {
		t.Fatalf("unexpected imported credentials: %d, %v", len(credentials), err)
	}
	encoded[len(encoded)-2] ^= 1
	if _, err := destination.PreviewBackup(ctx, encoded, testPassword); err == nil {
		t.Fatal("tampered backup should be rejected")
	}
}

func TestCiphertextIsBoundToRecordID(t *testing.T) {
	ctx := context.Background()
	vault, err := Open(ctx, filepath.Join(t.TempDir(), "vault.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer vault.Close()
	if _, err := vault.Setup(ctx, testPassword); err != nil {
		t.Fatal(err)
	}
	first, _ := vault.CreateCredential(ctx, Credential{Account: "first", Secret: "JBSWY3DPEHPK3PXP", Algorithm: totp.AlgorithmSHA1, Digits: 6, Period: 30})
	second, _ := vault.CreateCredential(ctx, Credential{Account: "second", Secret: "KRUGS4ZANFZSAYJA", Algorithm: totp.AlgorithmSHA1, Digits: 6, Period: 30})
	_, err = vault.db.ExecContext(ctx, `UPDATE credentials SET nonce=(SELECT nonce FROM credentials WHERE id=?), payload=(SELECT payload FROM credentials WHERE id=?) WHERE id=?`, first.ID, first.ID, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := vault.GetCredential(ctx, second.ID); err == nil {
		t.Fatal("row-swapped ciphertext should fail authentication")
	}
}
