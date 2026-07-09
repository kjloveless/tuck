package data

import (
	"database/sql"
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
	return Image{}, nil
}

func (i ImageModel) Update(image Image) (Image, error) {
	return Image{}, nil
}

func (i ImageModel) Delete(id int) error {
	return nil
}

func ValidateImage(v *validator.Validator, image Image) {
	v.Check(image.Year != 0, "year", "must be provdided")
	v.Check(image.Year >= 1888, "year", "must be greater than 1888")
	v.Check(image.Year <= time.Now().Year(), "year", "must not be in the future")

	v.Check(validator.Unique(image.People), "people", "must not contain duplicate values")
}
