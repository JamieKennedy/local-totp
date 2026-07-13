// Package totp implements RFC 6238 parsing, validation, and code generation.
package totp

import (
	"crypto/hmac"
	"crypto/sha1" // #nosec G505 -- SHA-1 is required for RFC 6238 interoperability, not collision resistance.
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Algorithm names an RFC 6238 HMAC algorithm.
type Algorithm string

const (
	AlgorithmSHA1   Algorithm = "SHA1"
	AlgorithmSHA256 Algorithm = "SHA256"
	AlgorithmSHA512 Algorithm = "SHA512"
)

// Config is the complete configuration needed to generate a TOTP code.
type Config struct {
	Secret    string    `json:"secret"`
	Issuer    string    `json:"issuer"`
	Account   string    `json:"account"`
	Algorithm Algorithm `json:"algorithm"`
	Digits    int       `json:"digits"`
	Period    int       `json:"period"`
}

// Code is a generated value and its validity interval.
type Code struct {
	Value      string    `json:"code"`
	ValidFrom  time.Time `json:"validFrom"`
	ValidUntil time.Time `json:"validUntil"`
}

// NormalizeSecret canonicalizes and validates a Base32 TOTP seed.
func NormalizeSecret(value string) (string, error) {
	replacer := strings.NewReplacer(" ", "", "-", "", "\t", "", "\r", "", "\n", "")
	value = strings.ToUpper(replacer.Replace(strings.TrimSpace(value)))
	if value == "" {
		return "", errors.New("secret is required")
	}
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.TrimRight(value, "="))
	if err != nil {
		return "", fmt.Errorf("secret must be valid Base32: %w", err)
	}
	if len(decoded) < 10 {
		return "", errors.New("secret must contain at least 80 bits")
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(decoded), nil
}

// Validate normalizes a configuration and applies supported parameter limits.
func Validate(config Config) (Config, error) {
	secret, err := NormalizeSecret(config.Secret)
	if err != nil {
		return Config{}, err
	}
	config.Secret = secret
	config.Issuer = strings.TrimSpace(config.Issuer)
	config.Account = strings.TrimSpace(config.Account)
	if config.Account == "" {
		return Config{}, errors.New("account is required")
	}
	if config.Algorithm == "" {
		config.Algorithm = AlgorithmSHA1
	}
	switch config.Algorithm {
	case AlgorithmSHA1, AlgorithmSHA256, AlgorithmSHA512:
	default:
		return Config{}, fmt.Errorf("unsupported algorithm %q", config.Algorithm)
	}
	if config.Digits == 0 {
		config.Digits = 6
	}
	if config.Digits < 6 || config.Digits > 8 {
		return Config{}, errors.New("digits must be between 6 and 8")
	}
	if config.Period == 0 {
		config.Period = 30
	}
	if config.Period < 5 || config.Period > 300 {
		return Config{}, errors.New("period must be between 5 and 300 seconds")
	}
	return config, nil
}

// ParseURI parses an otpauth://totp URI into a validated configuration.
func ParseURI(raw string) (Config, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return Config{}, fmt.Errorf("parse otpauth URI: %w", err)
	}
	if parsed.Scheme != "otpauth" || parsed.Host != "totp" {
		return Config{}, errors.New("URI must use otpauth://totp")
	}
	label := strings.TrimPrefix(parsed.Path, "/")
	if label == "" {
		return Config{}, errors.New("otpauth label is required")
	}
	issuerFromLabel := ""
	account := label
	if before, after, ok := strings.Cut(label, ":"); ok {
		issuerFromLabel = strings.TrimSpace(before)
		account = strings.TrimSpace(after)
	}
	query := parsed.Query()
	issuer := strings.TrimSpace(query.Get("issuer"))
	if issuer == "" {
		issuer = issuerFromLabel
	} else if issuerFromLabel != "" && !strings.EqualFold(issuer, issuerFromLabel) {
		return Config{}, errors.New("issuer query parameter does not match label")
	}
	digits, err := parseIntDefault(query.Get("digits"), 6)
	if err != nil {
		return Config{}, fmt.Errorf("digits: %w", err)
	}
	period, err := parseIntDefault(query.Get("period"), 30)
	if err != nil {
		return Config{}, fmt.Errorf("period: %w", err)
	}
	return Validate(Config{
		Secret:    query.Get("secret"),
		Issuer:    issuer,
		Account:   account,
		Algorithm: Algorithm(strings.ToUpper(query.Get("algorithm"))),
		Digits:    digits,
		Period:    period,
	})
}

// BuildURI returns the canonical otpauth URI for a configuration.
func BuildURI(config Config) (string, error) {
	config, err := Validate(config)
	if err != nil {
		return "", err
	}
	label := config.Account
	if config.Issuer != "" {
		label = config.Issuer + ":" + config.Account
	}
	query := url.Values{}
	query.Set("secret", config.Secret)
	if config.Issuer != "" {
		query.Set("issuer", config.Issuer)
	}
	query.Set("algorithm", string(config.Algorithm))
	query.Set("digits", strconv.Itoa(config.Digits))
	query.Set("period", strconv.Itoa(config.Period))
	return (&url.URL{Scheme: "otpauth", Host: "totp", Path: label, RawQuery: query.Encode()}).String(), nil
}

// Generate returns the code valid at the supplied instant.
func Generate(config Config, now time.Time) (Code, error) {
	config, err := Validate(config)
	if err != nil {
		return Code{}, err
	}
	secret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(config.Secret)
	if err != nil {
		return Code{}, fmt.Errorf("decode normalized secret: %w", err)
	}
	period := int64(config.Period)
	unix := now.UTC().Unix()
	counter := uint64(unix / period) // #nosec G115 -- negative Unix times are not accepted by the application.
	message := make([]byte, 8)
	binary.BigEndian.PutUint64(message, counter)

	var hashFactory func() hash.Hash
	switch config.Algorithm {
	case AlgorithmSHA1:
		hashFactory = sha1.New // #nosec G505 -- mandated interoperability option.
	case AlgorithmSHA256:
		hashFactory = sha256.New
	case AlgorithmSHA512:
		hashFactory = sha512.New
	}
	mac := hmac.New(hashFactory, secret)
	_, _ = mac.Write(message)
	digest := mac.Sum(nil)
	offset := digest[len(digest)-1] & 0x0f
	binaryCode := binary.BigEndian.Uint32(digest[offset:offset+4]) & 0x7fffffff
	modulus := uint32(1)
	for range config.Digits {
		modulus *= 10
	}
	value := fmt.Sprintf("%0*d", config.Digits, binaryCode%modulus)
	validFrom := time.Unix((unix/period)*period, 0).UTC()
	return Code{Value: value, ValidFrom: validFrom, ValidUntil: validFrom.Add(time.Duration(period) * time.Second)}, nil
}

func parseIntDefault(value string, fallback int) (int, error) {
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New("must be an integer")
	}
	return parsed, nil
}
