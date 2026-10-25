package nativetls

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

func loadOrCreateAuthority(directory, commonName string) (*x509.Certificate, crypto.Signer, []byte, error) {
	certificatePath, keyPath := filepath.Join(directory, "ca.crt"), filepath.Join(directory, "ca.key")
	exists, existenceError := certificatePairExists(certificatePath, keyPath)
	if existenceError != nil {
		return nil, nil, nil, existenceError
	}
	if exists {
		return readAuthority(directory)
	}
	for _, name := range []string{"server.crt", "server.key"} {
		if _, fileError := os.Lstat(filepath.Join(directory, name)); !os.IsNotExist(fileError) {
			return nil, nil, nil, fmt.Errorf("%w: %s", ErrInvalidTLS, invalidAuthorityMessage)
		}
	}
	_, privateKey, generationError := ed25519.GenerateKey(rand.Reader)
	if generationError != nil {
		return nil, nil, nil, generationError
	}
	serial, serialError := certificateSerial()
	if serialError != nil {
		return nil, nil, nil, serialError
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial, Subject: pkix.Name{CommonName: commonName},
		NotBefore: now.Add(-5 * time.Minute), NotAfter: now.Add(CertificateAuthorityValidity),
		IsCA: true, BasicConstraintsValid: true, MaxPathLen: 0, MaxPathLenZero: true,
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	encoded, creationError := x509.CreateCertificate(rand.Reader, template, template, privateKey.Public(), privateKey)
	if creationError != nil {
		return nil, nil, nil, creationError
	}
	publicCertificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: encoded})
	encodedKey, keyError := marshalCertificateKey(privateKey)
	if keyError != nil {
		return nil, nil, nil, keyError
	}
	defer clear(encodedKey)
	if writeError := writeCertificateFile(keyPath, encodedKey, 0600); writeError != nil {
		return nil, nil, nil, writeError
	}
	if writeError := writeCertificateFile(certificatePath, publicCertificate, 0644); writeError != nil {
		return nil, nil, nil, writeError
	}
	return readAuthority(directory)
}

func readAuthority(directory string) (*x509.Certificate, crypto.Signer, []byte, error) {
	publicCertificate, certificateError := readCertificateFile(filepath.Join(directory, "ca.crt"))
	privateKeyBytes, keyError := readCertificateFile(filepath.Join(directory, "ca.key"))
	defer clear(privateKeyBytes)
	if certificateError != nil || keyError != nil {
		return nil, nil, nil, fmt.Errorf("%w: %s", ErrInvalidTLS, invalidAuthorityMessage)
	}
	block, remaining := pem.Decode(publicCertificate)
	if block == nil || block.Type != "CERTIFICATE" || len(bytes.TrimSpace(remaining)) != 0 {
		return nil, nil, nil, fmt.Errorf("%w: %s", ErrInvalidTLS, invalidAuthorityMessage)
	}
	authority, parseError := x509.ParseCertificate(block.Bytes)
	privateKey, privateKeyError := parseCertificateKey(privateKeyBytes)
	now := time.Now()
	if parseError != nil || privateKeyError != nil || !authority.IsCA || !authority.BasicConstraintsValid || authority.KeyUsage&x509.KeyUsageCertSign == 0 || now.Before(authority.NotBefore) || !now.Before(authority.NotAfter) || authority.CheckSignatureFrom(authority) != nil {
		return nil, nil, nil, fmt.Errorf("%w: %s", ErrInvalidTLS, invalidAuthorityMessage)
	}
	certificatePublicKey, publicKeyError := x509.MarshalPKIXPublicKey(authority.PublicKey)
	privatePublicKey, privatePublicKeyError := x509.MarshalPKIXPublicKey(privateKey.Public())
	if publicKeyError != nil || privatePublicKeyError != nil || !bytes.Equal(certificatePublicKey, privatePublicKey) {
		return nil, nil, nil, fmt.Errorf("%w: %s", ErrInvalidTLS, invalidAuthorityMessage)
	}
	return authority, privateKey, publicCertificate, nil
}

func loadOrCreateLeaf(options Options, host string, authority *x509.Certificate, authorityKey crypto.Signer) (*tls.Certificate, error) {
	certificatePath := filepath.Join(options.ConfigurationDirectory, "server.crt")
	keyPath := filepath.Join(options.ConfigurationDirectory, "server.key")
	if recoveryError := recoverLeafCertificate(certificatePath, keyPath, authority); recoveryError != nil {
		return nil, recoveryError
	}
	exists, existenceError := certificatePairExists(certificatePath, keyPath)
	if existenceError != nil {
		return nil, existenceError
	}
	var privateKey crypto.Signer
	replaceKey := !exists
	now := time.Now()
	if exists {
		publicCertificate, certificateError := readCertificateFile(certificatePath)
		privateKeyBytes, keyError := readCertificateFile(keyPath)
		defer clear(privateKeyBytes)
		if certificateError != nil || keyError != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidTLS, invalidCertificateMessage)
		}
		certificate, pairError := tls.X509KeyPair(publicCertificate, privateKeyBytes)
		if pairError != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidTLS, invalidCertificateMessage)
		}
		leaf, validationError := validateManagedLeaf(&certificate, authority)
		if validationError != nil {
			return nil, validationError
		}
		privateKey, keyError = parseCertificateKey(privateKeyBytes)
		if keyError != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidTLS, invalidCertificateMessage)
		}
		replaceKey = leaf.PublicKeyAlgorithm == x509.Ed25519
		if !replaceKey && validLeaf(&certificate, host, now) && leaf.NotAfter.Sub(now) > options.RenewBefore {
			return &certificate, nil
		}
	}
	if replaceKey {
		generatedKey, generationError := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if generationError != nil {
			return nil, generationError
		}
		privateKey = generatedKey
	}
	serial, serialError := certificateSerial()
	if serialError != nil {
		return nil, serialError
	}
	expires := now.Add(options.LeafValidity)
	if authority.NotAfter.Before(expires) {
		expires = authority.NotAfter
	}
	if expires.Sub(now) <= options.RenewBefore {
		return nil, fmt.Errorf("%w: %s", ErrInvalidTLS, invalidAuthorityMessage)
	}
	template := &x509.Certificate{
		SerialNumber: serial, Subject: pkix.Name{CommonName: options.LeafName},
		NotBefore: now.Add(-5 * time.Minute), NotAfter: expires,
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}
	if address := net.ParseIP(host); address != nil {
		if !address.Equal(template.IPAddresses[0]) && !address.Equal(template.IPAddresses[1]) {
			template.IPAddresses = append(template.IPAddresses, address)
		}
	} else if host != "localhost" {
		template.DNSNames = append(template.DNSNames, host)
	}
	encoded, creationError := x509.CreateCertificate(rand.Reader, template, authority, privateKey.Public(), authorityKey)
	if creationError != nil {
		return nil, creationError
	}
	publicCertificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: encoded})
	privateKeyBytes, keyError := marshalCertificateKey(privateKey)
	if keyError != nil {
		return nil, keyError
	}
	defer clear(privateKeyBytes)
	if replaceKey {
		if writeError := writeCertificatePair(certificatePath, keyPath, publicCertificate, privateKeyBytes); writeError != nil {
			return nil, writeError
		}
	} else if writeError := writeCertificateFile(certificatePath, publicCertificate, 0644); writeError != nil {
		return nil, writeError
	}
	certificate, pairError := tls.X509KeyPair(publicCertificate, privateKeyBytes)
	if pairError != nil || !validLeaf(&certificate, host, now) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidTLS, invalidCertificateMessage)
	}
	return &certificate, nil
}

func recoverLeafCertificate(certificatePath, keyPath string, authority *x509.Certificate) error {
	pendingPath := certificatePath + ".pending"
	pendingCertificate, pendingError := readCertificateFile(pendingPath)
	if os.IsNotExist(pendingError) {
		return nil
	}
	if pendingError != nil {
		return pendingError
	}
	privateKeyBytes, keyError := readCertificateFile(keyPath)
	defer clear(privateKeyBytes)
	if os.IsNotExist(keyError) {
		return nil
	}
	if keyError != nil {
		return keyError
	}
	currentCertificate, currentError := readCertificateFile(certificatePath)
	if currentError != nil && !os.IsNotExist(currentError) {
		return currentError
	}
	if currentError == nil {
		if _, pairError := tls.X509KeyPair(currentCertificate, privateKeyBytes); pairError == nil {
			return nil
		}
	}
	certificate, pairError := tls.X509KeyPair(pendingCertificate, privateKeyBytes)
	if pairError != nil {
		return fmt.Errorf("%w: %s", ErrInvalidTLS, invalidCertificateMessage)
	}
	leaf, validationError := validateManagedLeaf(&certificate, authority)
	if validationError != nil {
		return validationError
	}
	publicKey, compatible := leaf.PublicKey.(*ecdsa.PublicKey)
	if !compatible || publicKey.Curve != elliptic.P256() {
		return fmt.Errorf("%w: %s", ErrInvalidTLS, invalidCertificateMessage)
	}
	return os.Rename(pendingPath, certificatePath)
}

func validateManagedLeaf(certificate *tls.Certificate, authority *x509.Certificate) (*x509.Certificate, error) {
	if len(certificate.Certificate) != 1 {
		return nil, fmt.Errorf("%w: %s", ErrInvalidTLS, invalidCertificateMessage)
	}
	leaf, parseError := x509.ParseCertificate(certificate.Certificate[0])
	if parseError != nil || leaf.CheckSignatureFrom(authority) != nil || leaf.IsCA {
		return nil, fmt.Errorf("%w: %s", ErrInvalidTLS, invalidCertificateMessage)
	}
	certificate.Leaf = leaf
	return leaf, nil
}

func parseCertificateKey(contents []byte) (crypto.Signer, error) {
	block, remaining := pem.Decode(contents)
	if block == nil || block.Type != "PRIVATE KEY" || len(bytes.TrimSpace(remaining)) != 0 {
		return nil, ErrInvalidTLS
	}
	parsed, parseError := x509.ParsePKCS8PrivateKey(block.Bytes)
	if parseError != nil {
		return nil, ErrInvalidTLS
	}
	privateKey, valid := parsed.(crypto.Signer)
	if !valid {
		return nil, ErrInvalidTLS
	}
	return privateKey, nil
}

func marshalCertificateKey(privateKey crypto.Signer) ([]byte, error) {
	encoded, marshalError := x509.MarshalPKCS8PrivateKey(privateKey)
	if marshalError != nil {
		return nil, marshalError
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encoded}), nil
}

func certificateSerial() (*big.Int, error) {
	serial, generationError := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if generationError != nil {
		return nil, generationError
	}
	if serial.Sign() == 0 {
		serial.SetInt64(1)
	}
	return serial, nil
}
