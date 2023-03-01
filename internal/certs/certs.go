package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

func GenerateSelfSignedCert(expiration time.Time) ([]byte, *rsa.PrivateKey, error) {
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(2019),
		Subject:               *getSubject(),
		NotBefore:             time.Now(),
		NotAfter:              expiration,
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	caPrivateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return []byte{}, nil, err
	}

	caCert, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caPrivateKey.PublicKey, caPrivateKey)
	if err != nil {
		return []byte{}, nil, err
	}

	return caCert, caPrivateKey, nil
}

func GenerateSignedCert(expiration time.Time, rootCert []byte, rootPrivateKey *rsa.PrivateKey) ([]byte, *rsa.PrivateKey, error) {
	certTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject:      *getSubject(),
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     expiration,
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	certPrivateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return []byte{}, nil, err
	}

	structuredCaCert, err := x509.ParseCertificate(rootCert)
	if err != nil {
		return []byte{}, nil, err
	}

	cert, err := x509.CreateCertificate(rand.Reader, certTemplate, structuredCaCert, &certPrivateKey.PublicKey, rootPrivateKey)
	if err != nil {
		return []byte{}, nil, err
	}

	return cert, certPrivateKey, nil
}

func VerifyIfSecondIsSignedByFirst(first, second []byte) (bool, error) {
	firstTemplate, err := x509.ParseCertificate(first)
	if err != nil {
		return false, err
	}
	if !firstTemplate.IsCA {
		return false, fmt.Errorf("secret %s is not CA", "")
	}

	roots := x509.NewCertPool()
	firstPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: firstTemplate.Raw})
	ok := roots.AppendCertsFromPEM(firstPem)
	if !ok {
		return false, fmt.Errorf("pem fail")
	}
	verifyOpts := x509.VerifyOptions{
		Roots: roots,
	}

	secondTemplate, err := x509.ParseCertificate(second)
	if err != nil {
		return false, err
	}

	if secondTemplate.IsCA {
		return false, fmt.Errorf("secret %s is CA", "")
	}

	_, err = secondTemplate.Verify(verifyOpts)
	if err != nil {
		return false, fmt.Errorf("verify error: %w", err)
	}

	return true, nil
}

func getSubject() *pkix.Name {
	return &pkix.Name{
		Organization:  []string{"SAP, INC."},
		Country:       []string{"US"},
		Province:      []string{""},
		Locality:      []string{"San Francisco"},
		StreetAddress: []string{"Golden Gate Bridge"},
		PostalCode:    []string{"94016"},
	}
}
