package db

import "errors"

var (
	ErrNotFound  = errors.New("db: not found")
	ErrConflict  = errors.New("db: conflict")
	ErrInvalid   = errors.New("db: invalid")
	ErrInternal  = errors.New("db: internal")
)

func IsNotFound(err error) bool  { return errors.Is(err, ErrNotFound) }
func IsConflict(err error) bool  { return errors.Is(err, ErrConflict) }
func IsInvalid(err error) bool   { return errors.Is(err, ErrInvalid) }
func IsInternal(err error) bool  { return errors.Is(err, ErrInternal) }
