package identity

import (
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreatePersistsCAIdentity(t *testing.T) {
	directory := t.TempDir()
	first, err := LoadOrCreate(Options{Directory: directory, AdvertiseURL: "https://192.0.2.10:5375"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := LoadOrCreate(Options{Directory: directory, AdvertiseURL: "https://192.0.2.11:5375"})
	if err != nil {
		t.Fatal(err)
	}

	if first.ServerID() != second.ServerID() {
		t.Fatalf("server ID changed: %q != %q", first.ServerID(), second.ServerID())
	}
	if first.CACertificateSHA256() != second.CACertificateSHA256() {
		t.Fatal("CA fingerprint changed")
	}
	if first.TLSCertificate().Leaf.SerialNumber.Cmp(second.TLSCertificate().Leaf.SerialNumber) == 0 {
		t.Fatal("expected a newly issued leaf certificate")
	}
}

func TestLoadOrCreateIssuesVerifiableChain(t *testing.T) {
	serverIdentity, err := LoadOrCreate(Options{Directory: t.TempDir(), AdvertiseURL: "https://192.0.2.10:5375"})
	if err != nil {
		t.Fatal(err)
	}
	chain := serverIdentity.TLSCertificate().Certificate
	if len(chain) != 2 {
		t.Fatalf("got certificate chain length %d; want 2", len(chain))
	}
	leaf, err := x509.ParseCertificate(chain[0])
	if err != nil {
		t.Fatal(err)
	}
	root, err := x509.ParseCertificate(chain[1])
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(root)
	if _, err := leaf.Verify(x509.VerifyOptions{Roots: roots, DNSName: "192.0.2.10"}); err != nil {
		t.Fatalf("verifying leaf: %v", err)
	}

	foundAdvertisedIP := false
	for _, ip := range leaf.IPAddresses {
		if ip.Equal(net.ParseIP("192.0.2.10")) {
			foundAdvertisedIP = true
		}
	}
	if !foundAdvertisedIP {
		t.Fatal("leaf certificate does not contain advertised IP")
	}
}

func TestLoadOrCreateRejectsIncompleteIdentity(t *testing.T) {
	directory := t.TempDir()
	certificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("not a certificate")})
	if err := os.WriteFile(filepath.Join(directory, caCertificateFilename), certificatePEM, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrCreate(Options{Directory: directory}); err == nil {
		t.Fatal("expected incomplete identity error")
	}
}
