package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TOTPSecret generates a new TOTP secret key.
func TOTPSecret() (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "DeployPilot",
		AccountName: "user",
		Period:      30,
		SecretSize:  32,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA256,
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate TOTP secret: %w", err)
	}
	return key.Secret(), nil
}

// TOTPQRCodeURL generates the otpauth:// URI for QR code scanning.
func TOTPQRCodeURL(secret, username string) string {
	return fmt.Sprintf("otpauth://totp/DeployPilot:%s?secret=%s&issuer=DeployPilot&algorithm=SHA256&digits=6&period=30",
		username, secret)
}

// TOTPValidate checks if a 6-digit code is valid for the given secret.
func TOTPValidate(secret, code string) bool {
	return totp.Validate(code, secret)
}

// GenerateBackupCodes creates 10 random 8-character backup codes.
// Returns the plaintext codes for one-time display and their SHA-256 hashes for storage.
func GenerateBackupCodes() (plaintext []string, hashes []string, err error) {
	plaintext = make([]string, 10)
	hashes = make([]string, 10)

	for i := 0; i < 10; i++ {
		b := make([]byte, 4)
		if _, err = rand.Read(b); err != nil {
			return nil, nil, fmt.Errorf("failed to generate backup code: %w", err)
		}
		code := hex.EncodeToString(b) // 8 hex chars
		plaintext[i] = strings.ToUpper(code)

		hash := sha256.Sum256([]byte(strings.ToLower(code)))
		hashes[i] = hex.EncodeToString(hash[:])
	}

	return plaintext, hashes, nil
}

// VerifyBackupCode checks if a plaintext code matches any of the stored hashes.
// Returns the index of the matched code, or -1 if no match.
func VerifyBackupCode(storedHashesJSON, code string) int {
	var hashes []string
	if err := json.Unmarshal([]byte(storedHashesJSON), &hashes); err != nil {
		return -1
	}

	inputHash := sha256.Sum256([]byte(strings.ToLower(code)))
	inputHashStr := hex.EncodeToString(inputHash[:])

	for i, h := range hashes {
		if h == inputHashStr {
			return i
		}
	}
	return -1
}

// RemoveBackupCode removes a used backup code from the stored hashes.
func RemoveBackupCode(storedHashesJSON string, index int) string {
	var hashes []string
	if err := json.Unmarshal([]byte(storedHashesJSON), &hashes); err != nil {
		return storedHashesJSON
	}

	if index < 0 || index >= len(hashes) {
		return storedHashesJSON
	}

	// Remove the used code
	hashes = append(hashes[:index], hashes[index+1:]...)
	updated, _ := json.Marshal(hashes)
	return string(updated)
}
