package data

import (
	"database/sql"
	"errors"
	"testing"
	"time"
)

func TestPairingSessionIssuesOneUsableDeviceToken(t *testing.T) {
	db := newPairingTestDB(t)
	pairings := PairingModel{DB: db}
	devices := DeviceModel{DB: db}

	session, err := pairings.New(time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	device, token, err := pairings.Consume(session.Plaintext, "  Test phone  ")
	if err != nil {
		t.Fatal(err)
	}
	if device.Name != "Test phone" {
		t.Fatalf("got device name %q", device.Name)
	}
	if !IsDeviceToken(token) {
		t.Fatalf("issued token %q does not have the device-token prefix", token)
	}

	authenticated, err := devices.GetForToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if authenticated.ID != device.ID {
		t.Fatalf("authenticated device %q; want %q", authenticated.ID, device.ID)
	}
	if _, _, err := pairings.Consume(session.Plaintext, "Replay"); !errors.Is(err, ErrInvalidPairingToken) {
		t.Fatalf("reusing pairing token returned %v", err)
	}

	if err := devices.Revoke(device.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := devices.GetForToken(token); !errors.Is(err, ErrRecordNotFound) {
		t.Fatalf("using revoked token returned %v", err)
	}
}

func TestPairingSessionRejectsExpiredToken(t *testing.T) {
	db := newPairingTestDB(t)
	pairings := PairingModel{DB: db}

	session, err := pairings.New(-time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := pairings.Consume(session.Plaintext, "Test phone"); !errors.Is(err, ErrInvalidPairingToken) {
		t.Fatalf("consuming expired pairing token returned %v", err)
	}
	if err := pairings.DeleteExpired(); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pairing_sessions`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("got %d expired pairing sessions; want 0", count)
	}
}

func newPairingTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`
		CREATE TABLE pairing_sessions (
			token_hash BLOB PRIMARY KEY NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL
		);
		CREATE TABLE devices (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			token_hash BLOB NOT NULL UNIQUE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			revoked_at TIMESTAMP
		);`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}
