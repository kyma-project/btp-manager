package certs

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
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
	newCertificatePrivateKey, err := rsa.GenerateKey(rand.Reader, RsaKeyBits())
	if err != nil {
		return nil, nil, err
	}

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&newCertificatePrivateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	subjectKeyId := sha256.Sum256(pubKeyBytes)

	newCertificateTemplate := &x509.Certificate{
		SerialNumber:          getRandomInt(),
		DNSNames:              getDns(),
		NotBefore:             time.Now().UTC(),
		NotAfter:              expiration,
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		SignatureAlgorithm:    x509.SHA256WithRSA,
		PublicKeyAlgorithm:    x509.RSA,
		SubjectKeyId:          subjectKeyId[:],
	}

	newCertificate, err := x509.CreateCertificate(rand.Reader, newCertificateTemplate, newCertificateTemplate, &newCertificatePrivateKey.PublicKey, newCertificatePrivateKey)
	if err != nil {
		return nil, nil, err
	}

	newCertificatePem := new(bytes.Buffer)
	if err = pem.Encode(newCertificatePem, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: newCertificate,
	}); err != nil {
		return nil, nil, err
	}

	newCertificatePrivateKeyPem := new(bytes.Buffer)
	if err = pem.Encode(newCertificatePrivateKeyPem, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(newCertificatePrivateKey),
	}); err != nil {
		return nil, nil, err
	}

	return newCertificatePem.Bytes(), newCertificatePrivateKeyPem.Bytes(), nil
}

func GenerateSignedCertificate(expiration time.Time, sourceCertificate, sourcePrivateKey []byte) ([]byte, []byte, error) {
	newCertificatePrivateKey, err := rsa.GenerateKey(rand.Reader, RsaKeyBits())
	if err != nil {
		return nil, nil, err
	}

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&newCertificatePrivateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	subjectKeyId := sha256.Sum256(pubKeyBytes)

	newCertificateTemplate := &x509.Certificate{
		SerialNumber:       getRandomInt(),
		DNSNames:           getDns(),
		NotBefore:          time.Now().UTC(),
		NotAfter:           expiration,
		ExtKeyUsage:        []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:           x509.KeyUsageDigitalSignature,
		SignatureAlgorithm: x509.SHA256WithRSA,
		PublicKeyAlgorithm: x509.RSA,
		SubjectKeyId:       subjectKeyId[:],
	}

	decodedSourceCertificate, err := DecodeCertificate(sourceCertificate)
	if err != nil {
		return nil, nil, err
	}
	parsedSourceCertificate, err := x509.ParseCertificate(decodedSourceCertificate.Bytes)
	if err != nil {
		return nil, nil, err
	}
	decodedSourcePrivateKey, err := DecodeCertificate(sourcePrivateKey)
	if err != nil {
		return nil, nil, err
	}
	parsedSourcePrivateKey, err := x509.ParsePKCS1PrivateKey(decodedSourcePrivateKey.Bytes)
	if err != nil {
		return nil, nil, err
	}

	newCertificate, err := x509.CreateCertificate(rand.Reader, newCertificateTemplate, parsedSourceCertificate, &newCertificatePrivateKey.PublicKey, parsedSourcePrivateKey)
	if err != nil {
		return nil, nil, err
	}

	newCertificatePem := new(bytes.Buffer)
	if err := pem.Encode(newCertificatePem, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: newCertificate,
	}); err != nil {
		return nil, nil, err
	}

	newCertificatePrivateKeyPem := new(bytes.Buffer)
	if err := pem.Encode(newCertificatePrivateKeyPem, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(newCertificatePrivateKey),
	}); err != nil {
		return nil, nil, err
	}

	return newCertificatePem.Bytes(), newCertificatePrivateKeyPem.Bytes(), nil
}

func VerifyIfLeafIsSignedByGivenCA(caCertificate, leafCertificate []byte) (bool, error) {
	caCertificateDecoded, err := DecodeCertificate(caCertificate)
	if err != nil {
		return true, fmt.Errorf("CA certificate: %w", err)
	}
	leafCertificateDecoded, err := DecodeCertificate(leafCertificate)
	if err != nil {
		return true, fmt.Errorf("leaf certificate: %w", err)
	}
	caCertificateTemplate, err := x509.ParseCertificate(caCertificateDecoded.Bytes)
	if err != nil {
		return false, fmt.Errorf("CA certificate: %w", err)
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
		return false, fmt.Errorf("leaf certificate: %w", err)
	}

	if leafCertificateTemplate.IsCA {
		return false, fmt.Errorf("leaf certificate is a CA one but it is expected to be leaf")
	}

	_, err = leafCertificateTemplate.Verify(verifyOpts)
	if err != nil {
		return false, fmt.Errorf("failed to verify certificate: %w", err)
	}

	return true, nil
}

func getDns() []string {
	return []string{"sap-btp-operator-webhook-service.kyma-system.svc", "sap-btp-operator-webhook-service.kyma-system"}
}

func DecodeCertificate(cert []byte) (*pem.Block, error) {
	decoded, _ := pem.Decode(cert)
	if decoded == nil {
		return nil, fmt.Errorf("while decoding cert to pem")
	}
	return decoded, nil
}

func CertificateExpires(cert *x509.Certificate, expirationBoundary time.Duration) bool {
	expirationTriggerBoundary := cert.NotAfter.UTC().Add(expirationBoundary)
	return time.Now().UTC().After(expirationTriggerBoundary)
}
