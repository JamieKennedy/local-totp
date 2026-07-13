package totp

import (
	"encoding/base32"
	"testing"
	"time"
)

func TestRFC6238Vectors(t *testing.T) {
	t.Parallel()
	vectors := []struct {
		unix                 int64
		sha1, sha256, sha512 string
	}{
		{59, "94287082", "46119246", "90693936"},
		{1111111109, "07081804", "68084774", "25091201"},
		{1111111111, "14050471", "67062674", "99943326"},
		{1234567890, "89005924", "91819424", "93441116"},
		{2000000000, "69279037", "90698825", "38618901"},
		{20000000000, "65353130", "77737706", "47863826"},
	}
	secrets := map[Algorithm]string{
		AlgorithmSHA1:   "12345678901234567890",
		AlgorithmSHA256: "12345678901234567890123456789012",
		AlgorithmSHA512: "1234567890123456789012345678901234567890123456789012345678901234",
	}
	for _, vector := range vectors {
		want := map[Algorithm]string{AlgorithmSHA1: vector.sha1, AlgorithmSHA256: vector.sha256, AlgorithmSHA512: vector.sha512}
		for algorithm, rawSecret := range secrets {
			algorithm := algorithm
			t.Run(string(algorithm), func(t *testing.T) {
				secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte(rawSecret))
				code, err := Generate(Config{Secret: secret, Account: "rfc", Algorithm: algorithm, Digits: 8, Period: 30}, time.Unix(vector.unix, 0))
				if err != nil {
					t.Fatal(err)
				}
				if code.Value != want[algorithm] {
					t.Fatalf("got %s, want %s", code.Value, want[algorithm])
				}
			})
		}
	}
}

func TestURIRoundTrip(t *testing.T) {
	t.Parallel()
	input := Config{Secret: "JBSWY3DPEHPK3PXP", Issuer: "Example Dev", Account: "dev@example.test", Algorithm: AlgorithmSHA256, Digits: 7, Period: 45}
	raw, err := BuildURI(input)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseURI(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed != input {
		t.Fatalf("round trip mismatch: %#v", parsed)
	}
}

func TestValidationRejectsInvalidValues(t *testing.T) {
	t.Parallel()
	tests := []Config{
		{Secret: "not base32", Account: "dev"},
		{Secret: "JBSWY3DPEHPK3PXP", Account: "", Digits: 6, Period: 30},
		{Secret: "JBSWY3DPEHPK3PXP", Account: "dev", Digits: 9, Period: 30},
		{Secret: "JBSWY3DPEHPK3PXP", Account: "dev", Digits: 6, Period: 4},
	}
	for _, test := range tests {
		if _, err := Validate(test); err == nil {
			t.Fatalf("expected validation error for %#v", test)
		}
	}
}
