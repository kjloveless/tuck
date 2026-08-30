package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"strings"
	"time"
)

const DeviceTokenPrefix = "tuck_device_"

type Device struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	RevokedAt *time.Time `json:"-"`
}

type DeviceModel struct {
	DB *sql.DB
}

func (m DeviceModel) GetForToken(token string) (*Device, error) {
	if !validPrefixedToken(token, DeviceTokenPrefix) {
		return nil, ErrRecordNotFound
	}

	tokenHash := sha256.Sum256([]byte(token))
	query := `
		SELECT id, name, created_at, revoked_at
		FROM devices
		WHERE token_hash = ?1 AND revoked_at IS NULL`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var device Device
	err := m.DB.QueryRowContext(ctx, query, tokenHash[:]).Scan(
		&device.ID,
		&device.Name,
		&device.CreatedAt,
		&device.RevokedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRecordNotFound
	}
	if err != nil {
		return nil, err
	}

	return &device, nil
}

func (m DeviceModel) Revoke(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, `
		UPDATE devices
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE id = ?1 AND revoked_at IS NULL`, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}

func IsDeviceToken(token string) bool {
	return strings.HasPrefix(token, DeviceTokenPrefix)
}
