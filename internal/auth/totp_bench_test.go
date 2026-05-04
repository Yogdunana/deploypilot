package auth

import (
	"encoding/json"
	"testing"
)

func BenchmarkTOTPSecret(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = TOTPSecret()
	}
}

func BenchmarkGenerateBackupCodes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, _ = GenerateBackupCodes()
	}
}

func BenchmarkVerifyBackupCode(b *testing.B) {
	plaintext, hashes, _ := GenerateBackupCodes()
	// Pre-marshal JSON (done once)
	hashesJSON := func() string {
		data, _ := json.Marshal(hashes)
		return string(data)
	}()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = VerifyBackupCode(hashesJSON, plaintext[0])
	}
}
