package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"slices"
	"time"
)

type Permissions []string

func (p Permissions) Include(code string) bool {
	return slices.Contains(p, code)
}

type PermissionModel struct {
	DB *sql.DB
}

func (m PermissionModel) GetAllForUser(userID int) (Permissions, error) {
	query := `
		SELECT permissions.code
		FROM permissions
		INNER JOIN users_permissions 
			ON users_permissions.permission_id = permissions.id
		INNER JOIN users
			ON users_permissions.user_id = users.id
		WHERE users.id = ?1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions Permissions

	for rows.Next() {
		var permission string

		err := rows.Scan(&permission)
		if err != nil {
			return nil, err
		}

		permissions = append(permissions, permission)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}

func (m PermissionModel) AddForUser(userID int, codes ...string) error {
	if len(codes) < 1 {
		return nil
	}

	codesJSON, err := json.Marshal(codes)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO users_permissions
			SELECT ?1, permissions.id
			FROM permissions
			WHERE permissions.code IN (
				SELECT value
				FROM json_each(?2))`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = m.DB.ExecContext(ctx, query, userID, string(codesJSON))
	return err
}
