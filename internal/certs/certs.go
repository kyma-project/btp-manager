package certs

import (
	"bytes"
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

func GenerateSelfSignedCert(expiration time.Time) ([]byte, []byte, error) {
	DNS := []string{"sap-btp-operator-webhook-service.kyma-system.svc", "sap-btp-operator-webhook-service.kyma-system"}

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(2019),
		DNSNames:              DNS,
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

	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCert,
	})

	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivateKey),
	})
	return caPEM.Bytes(), caPrivKeyPEM.Bytes(), nil
}

func GenerateSignedCert(expiration time.Time, rootCert, rootPrivateKey []byte) ([]byte, []byte, error) {
	DNS := []string{"sap-btp-operator-webhook-service.kyma-system.svc", "sap-btp-operator-webhook-service.kyma-system"}

	certTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		DNSNames:     DNS,
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

	p, _ := pem.Decode(rootCert)
	structuredCaCert, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		return []byte{}, nil, err
	}
	pk, _ := pem.Decode(rootPrivateKey)
	priv, err := x509.ParsePKCS1PrivateKey(pk.Bytes)
	cert, err := x509.CreateCertificate(rand.Reader, certTemplate, structuredCaCert, &certPrivateKey.PublicKey, priv)
	if err != nil {
		return []byte{}, nil, err
	}

	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})

	certPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivateKey),
	})

	return certPEM.Bytes(), certPrivKeyPEM.Bytes(), nil
}

func VerifyIfSecondIsSignedByFirst(first, second []byte) (bool, error) {
	fd, _ := pem.Decode(first)
	sd, _ := pem.Decode(second)

	firstTemplate, err := x509.ParseCertificate(fd.Bytes)
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

	fmt.Errorf("Aaaa")
	secondTemplate, err := x509.ParseCertificate(sd.Bytes)
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
		CommonName: "sap-btp-operator-webhook-service-ca",
	}
}
