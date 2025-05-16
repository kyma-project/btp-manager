package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func BenchmarkGenerateKey(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_, err := rsa.GenerateKey(rand.Reader, RsaKeyBits())
		assert.NoError(b, err)
	}
}

func BenchmarkGenerateSelfSignedCertificate(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_, _, err := GenerateSelfSignedCertificate(time.Now())
		assert.NoError(b, err)
	}
}
