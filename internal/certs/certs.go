package certs

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

func GenerateSelfSignedCertificate(expiration time.Time) ([]byte, []byte, error) {
	newCertificateTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(2019),
		DNSNames:              getDns(),
		NotBefore:             time.Now(),
		NotAfter:              expiration,
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	newCertificatePrivateKey, err := rsa.GenerateKey(rand.Reader, 4096)
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
		SerialNumber: big.NewInt(1658),
		DNSNames:     getDns(),
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     expiration,
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	newCertificatePrivateKey, err := rsa.GenerateKey(rand.Reader, 4096)
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

func VerifyIfSecondCertificateIsSignedByFirstCertificate(firstCertificate, secondCertificate []byte) (bool, error) {
	firstCertificateDecoded, _ := pem.Decode(firstCertificate)
	secondCertificateDecoded, _ := pem.Decode(secondCertificate)

	firstCertificateTemplate, err := x509.ParseCertificate(firstCertificateDecoded.Bytes)
	if err != nil {
		return false, err
	}
	if !firstCertificateTemplate.IsCA {
		return false, fmt.Errorf("certificate given as first is not CA")
	}

	roots := x509.NewCertPool()
	firstCertificatePem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: firstCertificateTemplate.Raw})
	ok := roots.AppendCertsFromPEM(firstCertificatePem)
	if !ok {
		return false, fmt.Errorf("appending first pem to root fail")
	}
	verifyOpts := x509.VerifyOptions{
		Roots: roots,
	}

	secondCertificateTemplate, err := x509.ParseCertificate(secondCertificateDecoded.Bytes)
	if err != nil {
		return false, err
	}

	if secondCertificateTemplate.IsCA {
		return false, fmt.Errorf("certificate given as second is CA")
	}

	_, err = secondCertificateTemplate.Verify(verifyOpts)
	if err != nil {
		return false, fmt.Errorf("verify of second certificate from first certificate error: %w", err)
	}

	return true, nil
}

func getDns() []string {
	return []string{"sap-btp-operator-webhook-service.kyma-system.svc", "sap-btp-operator-webhook-service.kyma-system"}
}
