package signing

import (
	"testing"
)

func BenchmarkGenerateKeyPair(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, _ = GenerateKeyPair()
	}
}

func BenchmarkSign(b *testing.B) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 0)
	data := []byte("benchmark test data for signing operations")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = signer.Sign(data)
	}
}

func BenchmarkVerify(b *testing.B) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 0)
	data := []byte("benchmark test data for signing operations")
	sig, _ := signer.Sign(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = signer.Verify(data, sig)
	}
}

func BenchmarkFingerprint(b *testing.B) {
	pub, _, _ := GenerateKeyPair()
	signer := NewSigner(pub, nil, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = signer.Fingerprint()
	}
}

func BenchmarkSignLarge(b *testing.B) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 0)
	data := make([]byte, 1024*1024) // 1MB
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = signer.Sign(data)
	}
}

func BenchmarkVerifyLarge(b *testing.B) {
	pub, priv, _ := GenerateKeyPair()
	signer := NewSigner(pub, priv, 0)
	data := make([]byte, 1024*1024) // 1MB
	sig, _ := signer.Sign(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = signer.Verify(data, sig)
	}
}
