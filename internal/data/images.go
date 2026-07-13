package data

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
	"encoding/json"

	"tuck.loveless.dev/internal/validator"

	_ "modernc.org/sqlite"
)

type Image struct {
	ID				int					`json:"id"`
	CreatedAt	time.Time		`json:"-"`
	Year			int					`json:"year,omitzero"`
	Location	string			`json:"location,omitzero"`
	People		[]string		`json:"people,omitempty"`
	Version		int					`json:"version"`
}

// define a imagemodel struct type which wraps a sql.DB connection pool
type ImageModel struct {
	DB *sql.DB
}

func (i ImageModel) Insert(image Image) (Image, error) {
	// define a sql query which inserts a new record in the images table, and
	// returns the system-generated data
	query := `
		INSERT INTO images (location, year, people)
		VALUES ($1, ?, ?)
		RETURNING id, created_at, version`

	people, err := json.Marshal(image.People)
	if err != nil {
		return Image{}, err
	}

	// create an args slice containing the values for the placeholder parameters.
	// declaring this slice immediately next to our sql query helps to make it
	// nice and clear *what values are being used where* in the query
	args :=[]any{image.Location, image.Year, string(people)}

	err = i.DB.QueryRow(query, args...).Scan(&image.ID, &image.CreatedAt, &image.Version)

	return image, err
}

func (i ImageModel) Get(id int) (Image, error) {
	if id < 1 {
		return Image{}, ErrRecordNotFound
	}

	query := `
		SELECT id, created_at, location, year, people, version
		FROM images
		WHERE id = ?`

	var image Image
	var peopleJSON []byte

	err := i.DB.QueryRow(query, id).Scan(
		&image.ID,
		&image.CreatedAt,
		&image.Location,
		&image.Year,
		&peopleJSON,
		&image.Version)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return Image{}, ErrRecordNotFound
		default:
			return Image{}, err
		}
	}

	err = json.Unmarshal(peopleJSON, &image.People)
	if err != nil {
		return Image{}, fmt.Errorf("decoding people for image: %d: %w", image.ID, err)
	}

	return image, nil
}

func (i ImageModel) Update(image Image) (Image, error) {
	query := `
		UPDATE images
		SET location = ?, year = ?, people = ?, version = version + 1
		WHERE id = ? AND version = ?
		RETURNING version`

	people, err := json.Marshal(image.People)
	if err != nil {
		return Image{}, fmt.Errorf("marshaling people for imagee: %d: %w", image.ID, err)
	}

	args := []any{
		image.Location,
		image.Year,
		string(people),
		image.ID,
		image.Version,
	}

	err = i.DB.QueryRow(query, args...).Scan(&image.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return Image{}, ErrEditConflict
		default:
			return Image{}, err
		}
	}

	return image, nil
}

func (i ImageModel) Delete(id int) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	query := `
		DELETE FROM images
		WHERE id = ?`

	result, err := i.DB.Exec(query, id)
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

func ValidateImage(v *validator.Validator, image Image) {
	v.Check(image.Year != 0, "year", "must be provdided")
	v.Check(image.Year >= 1888, "year", "must be greater than 1888")
	v.Check(image.Year <= time.Now().Year(), "year", "must not be in the future")

	v.Check(validator.Unique(image.People), "people", "must not contain duplicate values")
}

