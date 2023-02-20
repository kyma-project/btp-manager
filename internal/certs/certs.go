package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"time"
)

func GenerateSelfSignedCert(expiration time.Time) ([]byte, *rsa.PrivateKey, error) {
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization: []string{"SAP"},
		},
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
		Subject: pkix.Name{
			Organization: []string{"SAP"},
		},
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
