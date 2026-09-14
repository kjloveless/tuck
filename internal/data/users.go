package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"time"

	"tuck.loveless.dev/internal/validator"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrDuplicateUsername 	= errors.New("duplicate username")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type User struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	Password  password  `json:"-"`
	Activated bool      `json:"activated"`
	Version   int       `json:"-"`
}

type password struct {
	plaintext *string
	hash      []byte
}

func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return err
	}

	p.plaintext = &plaintextPassword
	p.hash = hash

	return nil
}

func (p *password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}

func ValidateUsername(v *validator.Validator, username string) {
	v.Check(username != "", "username", "must be provided")
	v.Check(validator.Matches(username, validator.UsernameRX), "username", "must be a valid username")
}

func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}

func ValidateUser(v *validator.Validator, user *User) {
	ValidateUsername(v, user.Username)

	if user.Password.plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.plaintext)
	}

	if user.Password.hash == nil {
		panic("missing password hash for user")
	}
}

type UserModel struct {
	DB *sql.DB
}

func (m UserModel) Insert(user *User) error {
	query := `
		INSERT INTO users (username, password_hash, activated)
		VALUES (?1, ?2, ?3)
		RETURNING id, created_at, version`

	args := []any{user.Username, user.Password.hash, user.Activated}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&user.ID, &user.CreatedAt, &user.Version)
	if err != nil {
		switch {
		case err.Error() == `constraint failed: UNIQUE constraint failed: users.username (2067)`:
			return ErrDuplicateUsername
		default:
			return err
		}
	}

	return nil
}

func (m UserModel) Authenticate(username, plaintextPassword string) (*User, error) {
	user, err := m.GetByUsername(username)
	if err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	match, err := user.Password.Matches(plaintextPassword)
	if err != nil {
		return nil, err
	}
	if !match {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

func (m UserModel) GetByUsername(username string) (*User, error) {
	query := `
		SELECT id, created_at, username, password_hash, activated, version
		FROM users
		WHERE username = ?`

	var user User

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Username,
		&user.Password.hash,
		&user.Activated,
		&user.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (m UserModel) GetByID(id int) (*User, error) {
	query := `
		SELECT id, created_at, username, password_hash, activated, version
		FROM users
		WHERE id = ?`

	var user User

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Username,
		&user.Password.hash,
		&user.Activated,
		&user.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (m UserModel) Update(user *User) error {
	query := `
		UPDATE users
		SET username = ?1, password_hash = ?2, activated = ?3, version = version + 1
		WHERE id = ?4 AND version = ?5
		RETURNING version`

	args := []any{
		user.Username,
		user.Password.hash,
		user.Activated,
		user.ID,
		user.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&user.Version)
	if err != nil {
		switch {
		case err.Error() == `constraint failed: UNIQUE constraint failed: users.username (2067)`:
			return ErrDuplicateUsername
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (m UserModel) GetForToken(tokenScope, tokenPlaintext string) (*User, error) {
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))

	query := `
		SELECT 
			users.id,
			users.created_at,
			users.username,
			users.password_hash,
			users.activated,
			users.version
		FROM users
		INNER JOIN tokens
			ON users.id = tokens.user_id
		WHERE tokens.hash = ?1
		AND tokens.scope = ?2
		AND tokens.expiry > ?3`

	args := []any{tokenHash[:], tokenScope, time.Now()}

	var user User

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Username,
		&user.Password.hash,
		&user.Activated,
		&user.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}
