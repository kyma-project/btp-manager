package certs

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"time"
)

var (
	rsaKeyBits = 4096
	randMax    = 10000
)

func RsaKeyBits() int {
	return rsaKeyBits
}

func SetRsaKeyBits(newValue int) {
	rsaKeyBits = newValue
}

func getRandomInt() *big.Int {
	return big.NewInt(int64(mathrand.Intn(randMax)))
}

func GenerateSelfSignedCertificate(expiration time.Time) ([]byte, []byte, error) {
	newCertificateTemplate := &x509.Certificate{
		SerialNumber:          getRandomInt(),
		DNSNames:              getDns(),
		NotBefore:             time.Now(),
		NotAfter:              expiration,
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	newCertificatePrivateKey, err := rsa.GenerateKey(rand.Reader, RsaKeyBits())
	if err != nil {
		return []byte{}, nil, err
	}

	newCertificate, err := x509.CreateCertificate(rand.Reader, newCertificateTemplate, newCertificateTemplate, &newCertificatePrivateKey.PublicKey, newCertificatePrivateKey)
	if err != nil {
		return []byte{}, nil, err
	}

	newCertificatePem := new(bytes.Buffer)
	pem.Encode(newCertificatePem, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: newCertificate,
	})

	newCertificatePrivateKeyPem := new(bytes.Buffer)
	pem.Encode(newCertificatePrivateKeyPem, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(newCertificatePrivateKey),
	})

	return newCertificatePem.Bytes(), newCertificatePrivateKeyPem.Bytes(), nil
}

func GenerateSignedCertificate(expiration time.Time, sourceCertificate, sourcePrivateKey []byte) ([]byte, []byte, error) {
	newCertificateTemplate := &x509.Certificate{
		SerialNumber: getRandomInt(),
		DNSNames:     getDns(),
		NotBefore:    time.Now(),
		NotAfter:     expiration,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	newCertificatePrivateKey, err := rsa.GenerateKey(rand.Reader, RsaKeyBits())
	if err != nil {
		return []byte{}, nil, err
	}

	decodedSourceCertificate, _ := pem.Decode(sourceCertificate)
	parsedSourceCertificate, err := x509.ParseCertificate(decodedSourceCertificate.Bytes)
	if err != nil {
		return []byte{}, nil, err
	}
	decodedSourcePrivateKey, _ := pem.Decode(sourcePrivateKey)
	parsedSourcePrivateKey, err := x509.ParsePKCS1PrivateKey(decodedSourcePrivateKey.Bytes)
	if err != nil {
		return []byte{}, nil, err
	}

	newCertificate, err := x509.CreateCertificate(rand.Reader, newCertificateTemplate, parsedSourceCertificate, &newCertificatePrivateKey.PublicKey, parsedSourcePrivateKey)
	if err != nil {
		return []byte{}, nil, err
	}

	newCertificatePem := new(bytes.Buffer)
	pem.Encode(newCertificatePem, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: newCertificate,
	})

	newCertificatePrivateKeyPem := new(bytes.Buffer)
	pem.Encode(newCertificatePrivateKeyPem, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(newCertificatePrivateKey),
	})

	return newCertificatePem.Bytes(), newCertificatePrivateKeyPem.Bytes(), nil
}

func VerifyIfLeafSignedByGivenCA(caCertificate, leafCertificate []byte) (bool, error) {
	caCertificateDecoded, _ := pem.Decode(caCertificate)
	leafCertificateDecoded, _ := pem.Decode(leafCertificate)

	caCertificateTemplate, err := x509.ParseCertificate(caCertificateDecoded.Bytes)
	if err != nil {
		return false, err
	}
	if !caCertificateTemplate.IsCA {
		return false, fmt.Errorf("CA certificate is not CA")
	}

	roots := x509.NewCertPool()
	caCertificatePem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertificateTemplate.Raw})
	ok := roots.AppendCertsFromPEM(caCertificatePem)
	if !ok {
		return false, fmt.Errorf("appending first pem to root fail")
	}
	verifyOpts := x509.VerifyOptions{
		Roots: roots,
	}

	leafCertificateTemplate, err := x509.ParseCertificate(leafCertificateDecoded.Bytes)
	if err != nil {
		return false, err
	}

	if leafCertificateTemplate.IsCA {
		return false, fmt.Errorf("leaf certificate is a CA one but it is expected to be leaf")
	}

	_, err = leafCertificateTemplate.Verify(verifyOpts)
	if err != nil {
		return false, fmt.Errorf("while verifying certificate: %w", err)
	}

	return true, nil
}

func getDns() []string {
	return []string{"sap-btp-operator-webhook-service.kyma-system.svc", "sap-btp-operator-webhook-service.kyma-system"}
}
