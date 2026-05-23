package store

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrAmbiguous     = errors.New("ambiguous prefix")
)
