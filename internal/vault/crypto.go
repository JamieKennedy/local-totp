package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	keySize        = 32
	defaultMemory  = 64 * 1024
	defaultTime    = 3
	defaultThreads = 4
	recoveryPrefix = "ltotp-recovery-"
)

type kdfParams struct {
	Memory  uint32 `json:"memory"`
	Time    uint32 `json:"time"`
	Threads uint8  `json:"threads"`
	Salt    []byte `json:"salt"`
}

func defaultKDF() (kdfParams, error) {
	salt, err := randomBytes(16)
	if err != nil {
		return kdfParams{}, err
	}
	return kdfParams{Memory: defaultMemory, Time: defaultTime, Threads: defaultThreads, Salt: salt}, nil
}

func passwordKey(password string, params kdfParams) []byte {
	return argon2.IDKey([]byte(password), params.Salt, params.Time, params.Memory, params.Threads, keySize)
}

func recoveryKey(raw, salt []byte) []byte {
	mac := hmac.New(sha256.New, salt)
	_, _ = mac.Write([]byte("local-totp recovery wrapper v1\x00"))
	_, _ = mac.Write(raw)
	return mac.Sum(nil)
}

func seal(key, plaintext, additionalData []byte) (nonce, ciphertext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("create AES cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("create GCM: %w", err)
	}
	nonce, err = randomBytes(aead.NonceSize())
	if err != nil {
		return nil, nil, err
	}
	return nonce, aead.Seal(nil, nonce, plaintext, additionalData), nil
}

func openSealed(key, nonce, ciphertext, additionalData []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	plaintext, err := aead.Open(nil, nonce, ciphertext, additionalData)
	if err != nil {
		return nil, ErrUnauthorized
	}
	return plaintext, nil
}

func generateRecovery() (display string, raw []byte, err error) {
	raw, err = randomBytes(32)
	if err != nil {
		return "", nil, err
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
	parts := make([]string, 0, 7)
	for len(encoded) > 0 {
		length := min(8, len(encoded))
		parts = append(parts, encoded[:length])
		encoded = encoded[length:]
	}
	return recoveryPrefix + strings.Join(parts, "-"), raw, nil
}

func parseRecovery(display string) ([]byte, error) {
	value := strings.TrimSpace(strings.ToLower(display))
	if !strings.HasPrefix(value, recoveryPrefix) {
		return nil, ErrUnauthorized
	}
	value = strings.ToUpper(strings.ReplaceAll(strings.TrimPrefix(value, recoveryPrefix), "-", ""))
	raw, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(value)
	if err != nil || len(raw) != 32 {
		return nil, ErrUnauthorized
	}
	return raw, nil
}

func randomToken(prefix string) (string, []byte, error) {
	raw, err := randomBytes(32)
	if err != nil {
		return "", nil, err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(raw), raw, nil
}

func randomBytes(size int) ([]byte, error) {
	value := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, value); err != nil {
		return nil, fmt.Errorf("read secure randomness: %w", err)
	}
	return value, nil
}

func wipe(value []byte) {
	for index := range value {
		value[index] = 0
	}
}

func hashAPIKey(value string) ([]byte, error) {
	if !strings.HasPrefix(value, "ltotp_") {
		return nil, errors.New("invalid API key prefix")
	}
	digest := sha256.Sum256([]byte(value))
	return digest[:], nil
}
