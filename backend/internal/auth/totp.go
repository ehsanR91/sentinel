package auth

import (
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// GenerateSecret creates a new TOTP key for the given username.
// Returns the base32 secret (store in DB) and the otpauth:// URL (render as QR).
func GenerateSecret(username string) (secret, otpauthURL string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "SentinelCore",
		AccountName: username,
		SecretSize:  20,
		Algorithm:   otp.AlgorithmSHA1,
		Digits:      otp.DigitsSix,
		Period:      30,
	})
	if err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

// ValidateCode checks a 6-digit TOTP code against the stored base32 secret.
func ValidateCode(secret, code string) bool {
	return totp.Validate(code, secret)
}
