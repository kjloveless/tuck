package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

const PairingTokenPrefix = "tuck_pair_"

var (
	ErrInvalidDeviceName   = errors.New("device name must contain between 1 and 100 bytes")
	ErrInvalidPairingToken = errors.New("invalid or expired pairing token")
)

type PairingSession struct {
	Plaintext string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
}

type PairingModel struct {
	DB *sql.DB
}

func (m PairingModel) New(ttl time.Duration) (*PairingSession, error) {
	token, err := randomToken(PairingTokenPrefix, 32)
	if err != nil {
		return nil, err
	}
	session := &PairingSession{
		Plaintext: token,
		ExpiresAt: time.Now().Add(ttl),
	}
	tokenHash := sha256.Sum256([]byte(token))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = m.DB.ExecContext(ctx, `
		INSERT INTO pairing_sessions (token_hash, expires_at)
		VALUES (?1, ?2)`, tokenHash[:], session.ExpiresAt)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (m PairingModel) Consume(pairingToken, deviceName string) (*Device, string, error) {
	if !validPrefixedToken(pairingToken, PairingTokenPrefix) {
		return nil, "", ErrInvalidPairingToken
	}
	deviceName = strings.TrimSpace(deviceName)
	if deviceName == "" || len(deviceName) > 100 {
		return nil, "", ErrInvalidDeviceName
	}

	deviceID, err := randomToken("device_", 16)
	if err != nil {
		return nil, "", err
	}
	deviceToken, err := randomToken(DeviceTokenPrefix, 32)
	if err != nil {
		return nil, "", err
	}
	pairingHash := sha256.Sum256([]byte(pairingToken))
	deviceHash := sha256.Sum256([]byte(deviceToken))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, "", err
	}
	defer tx.Rollback()

	var consumed int
	err = tx.QueryRowContext(ctx, `
		DELETE FROM pairing_sessions
		WHERE token_hash = ?1 AND expires_at > ?2
		RETURNING 1`, pairingHash[:], time.Now()).Scan(&consumed)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", ErrInvalidPairingToken
	}
	if err != nil {
		return nil, "", err
	}

	device := &Device{ID: deviceID, Name: deviceName}
	err = tx.QueryRowContext(ctx, `
		INSERT INTO devices (id, name, token_hash)
		VALUES (?1, ?2, ?3)
		RETURNING created_at`, device.ID, device.Name, deviceHash[:]).Scan(&device.CreatedAt)
	if err != nil {
		return nil, "", err
	}

	if err := tx.Commit(); err != nil {
		return nil, "", err
	}
	return device, deviceToken, nil
}

func (m PairingModel) DeleteExpired() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := m.DB.ExecContext(ctx, `DELETE FROM pairing_sessions WHERE expires_at <= ?1`, time.Now())
	return err
}

func randomToken(prefix string, size int) (string, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generating random token: %w", err)
	}
	return prefix + base64.RawURLEncoding.EncodeToString(bytes), nil
}

func validPrefixedToken(token, prefix string) bool {
	if !strings.HasPrefix(token, prefix) {
		return false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(token, prefix))
	return err == nil && len(decoded) == 32
}
