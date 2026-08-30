package identity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base32"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	caCertificateFilename = "ca-cert.pem"
	caKeyFilename         = "ca-key.pem"
)

type Options struct {
	Directory    string
	AdvertiseURL string
}

type Identity struct {
	serverID         string
	caCertificateSHA string
	certificate      tls.Certificate
}

func LoadOrCreate(opts Options) (*Identity, error) {
	if opts.Directory == "" {
		return nil, errors.New("identity directory must not be empty")
	}

	if err := os.MkdirAll(opts.Directory, 0o700); err != nil {
		return nil, fmt.Errorf("creating identity directory: %w", err)
	}
	if err := os.Chmod(opts.Directory, 0o700); err != nil {
		return nil, fmt.Errorf("securing identity directory: %w", err)
	}

	caCertificate, caKey, err := loadOrCreateCA(
		filepath.Join(opts.Directory, caCertificateFilename),
		filepath.Join(opts.Directory, caKeyFilename),
		time.Now(),
	)
	if err != nil {
		return nil, err
	}

	dnsNames, ipAddresses, err := certificateNames(opts.AdvertiseURL)
	if err != nil {
		return nil, err
	}
	certificate, err := issueServerCertificate(caCertificate, caKey, dnsNames, ipAddresses, time.Now())
	if err != nil {
		return nil, err
	}

	fingerprint := sha256.Sum256(caCertificate.Raw)
	serverIDEncoding := base32.StdEncoding.WithPadding(base32.NoPadding)

	return &Identity{
		serverID:         "tuck-" + strings.ToLower(serverIDEncoding.EncodeToString(fingerprint[:16])),
		caCertificateSHA: base64.RawURLEncoding.EncodeToString(fingerprint[:]),
		certificate:      certificate,
	}, nil
}

func (i *Identity) ServerID() string {
	return i.serverID
}

func (i *Identity) CACertificateSHA256() string {
	return i.caCertificateSHA
}

func (i *Identity) TLSCertificate() tls.Certificate {
	return i.certificate
}

func loadOrCreateCA(certificatePath, keyPath string, now time.Time) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	certificateExists, err := fileExists(certificatePath)
	if err != nil {
		return nil, nil, err
	}
	keyExists, err := fileExists(keyPath)
	if err != nil {
		return nil, nil, err
	}

	if certificateExists != keyExists {
		return nil, nil, errors.New("incomplete server identity: CA certificate and private key must both exist")
	}
	if !certificateExists {
		return createCA(certificatePath, keyPath, now)
	}
	if err := os.Chmod(keyPath, 0o600); err != nil {
		return nil, nil, fmt.Errorf("securing CA private key: %w", err)
	}

	certificatePEM, err := os.ReadFile(certificatePath)
	if err != nil {
		return nil, nil, fmt.Errorf("reading CA certificate: %w", err)
	}
	certificateBlock, rest := pem.Decode(certificatePEM)
	if certificateBlock == nil || certificateBlock.Type != "CERTIFICATE" || len(rest) != 0 {
		return nil, nil, errors.New("decoding CA certificate: expected one PEM certificate")
	}
	certificate, err := x509.ParseCertificate(certificateBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing CA certificate: %w", err)
	}
	if !certificate.IsCA || certificate.KeyUsage&x509.KeyUsageCertSign == 0 {
		return nil, nil, errors.New("stored CA certificate is not a certificate authority")
	}

	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("reading CA private key: %w", err)
	}
	keyBlock, rest := pem.Decode(keyPEM)
	if keyBlock == nil || keyBlock.Type != "PRIVATE KEY" || len(rest) != 0 {
		return nil, nil, errors.New("decoding CA private key: expected one PKCS#8 PEM key")
	}
	parsedKey, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing CA private key: %w", err)
	}
	key, ok := parsedKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, nil, errors.New("stored CA private key is not ECDSA")
	}
	certificatePublicKey, ok := certificate.PublicKey.(*ecdsa.PublicKey)
	if !ok || !certificatePublicKey.Equal(&key.PublicKey) {
		return nil, nil, errors.New("stored CA certificate and private key do not match")
	}
	if now.Before(certificate.NotBefore) || !now.Before(certificate.NotAfter) {
		return nil, nil, errors.New("stored CA certificate is not currently valid")
	}

	return certificate, key, nil
}

func createCA(certificatePath, keyPath string, now time.Time) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generating CA private key: %w", err)
	}
	serial, err := randomSerial()
	if err != nil {
		return nil, nil, err
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"Tuck"},
			CommonName:   "Tuck Local Root",
		},
		NotBefore:             now.Add(-24 * time.Hour),
		NotAfter:              now.AddDate(20, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}

	certificateDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, fmt.Errorf("creating CA certificate: %w", err)
	}
	certificate, err := x509.ParseCertificate(certificateDER)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing generated CA certificate: %w", err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling CA private key: %w", err)
	}

	if err := writeNewFile(certificatePath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificateDER}), 0o644); err != nil {
		return nil, nil, fmt.Errorf("writing CA certificate: %w", err)
	}
	if err := writeNewFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		return nil, nil, fmt.Errorf("writing CA private key: %w", err)
	}

	return certificate, key, nil
}

func issueServerCertificate(
	caCertificate *x509.Certificate,
	caKey *ecdsa.PrivateKey,
	dnsNames []string,
	ipAddresses []net.IP,
	now time.Time,
) (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generating TLS private key: %w", err)
	}
	serial, err := randomSerial()
	if err != nil {
		return tls.Certificate{}, err
	}
	notAfter := now.AddDate(1, 0, 0)
	if !notAfter.Before(caCertificate.NotAfter) {
		notAfter = caCertificate.NotAfter.Add(-time.Minute)
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"Tuck"},
			CommonName:   "Tuck Server",
		},
		NotBefore:   now.Add(-24 * time.Hour),
		NotAfter:    notAfter,
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    dnsNames,
		IPAddresses: ipAddresses,
	}

	leafDER, err := x509.CreateCertificate(rand.Reader, template, caCertificate, &key.PublicKey, caKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("creating TLS certificate: %w", err)
	}
	leaf, err := x509.ParseCertificate(leafDER)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("parsing generated TLS certificate: %w", err)
	}

	return tls.Certificate{
		Certificate: [][]byte{leafDER, caCertificate.Raw},
		PrivateKey:  key,
		Leaf:        leaf,
	}, nil
}

func certificateNames(advertiseURL string) ([]string, []net.IP, error) {
	dnsNames := []string{"localhost"}
	ipAddresses := []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}

	if advertiseURL != "" {
		parsedURL, err := url.Parse(advertiseURL)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing advertise URL: %w", err)
		}
		if parsedURL.Scheme != "https" || parsedURL.Hostname() == "" {
			return nil, nil, errors.New("advertise URL must be an absolute https URL")
		}
		if ip := net.ParseIP(parsedURL.Hostname()); ip != nil {
			ipAddresses = append(ipAddresses, ip)
		} else {
			dnsNames = append(dnsNames, parsedURL.Hostname())
		}
	}

	if interfaceAddresses, err := net.InterfaceAddrs(); err == nil {
		for _, address := range interfaceAddresses {
			ip, _, err := net.ParseCIDR(address.String())
			if err == nil && ip.IsGlobalUnicast() {
				ipAddresses = append(ipAddresses, ip)
			}
		}
	}

	slices.Sort(dnsNames)
	dnsNames = slices.Compact(dnsNames)
	slices.SortFunc(ipAddresses, func(a, b net.IP) int {
		return strings.Compare(a.String(), b.String())
	})
	ipAddresses = slices.CompactFunc(ipAddresses, func(a, b net.IP) bool {
		return a.Equal(b)
	})

	return dnsNames, ipAddresses, nil
}

func randomSerial() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return nil, fmt.Errorf("generating certificate serial number: %w", err)
	}
	return serial, nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("checking identity file %q: %w", path, err)
}

func writeNewFile(path string, contents []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	removeFile := true
	defer func() {
		file.Close()
		if removeFile {
			os.Remove(path)
		}
	}()

	if _, err := file.Write(contents); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	removeFile = false
	return nil
}
