package data

import (
	"database/sql"
	"errors"
)

// define a custom ErrRecordNotFound error. we'll return this from our Get()
// method when looking up an image that doesn't exist in our databse
var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict		= erros.New("edit conflict")
)

// create a models struct which wraps the ImageModel. we'll add other models to
// this, like a UserModel and PermissionModel
type Models struct {
	Images ImageModel
}

// for ease of use, we also add a New() method which returns a Models struct
// containing the initialized ImageModel
func NewModels(db *sql.DB) Models {
	return Models{
		Images: ImageModel{DB: db},
	}
}
