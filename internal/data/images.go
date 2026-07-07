package data

import (
	"time"

	"tuck.loveless.dev/internal/validator"
)

type Image struct {
	ID				int					`json:"id"`
	CreatedAt	time.Time		`json:"-"`
	Year			int					`json:"year,omitzero"`
	Location	string			`json:"location,omitzero"`
	People		[]string		`json:"people,omitempty"`
	Version		int					`json:"version"`
}

func ValidateImage(v *validator.Validator, image Image) {
	v.Check(image.Year != 0, "year", "must be provdided")
	v.Check(image.Year >= 1888, "year", "must be greater than 1888")
	v.Check(image.Year <= time.Now().Year(), "year", "must not be in the future")

	v.Check(validator.Unique(image.People), "people", "must not contain duplicate values")
}
