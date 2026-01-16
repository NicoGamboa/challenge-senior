package validator

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

var ErrInvalidJSON = errors.New("invalid json")

type JSON struct {
	MaxBytes int64
}

func NewJSON() *JSON {
	return &JSON{MaxBytes: 1 << 20}
}

func (v *JSON) Decode(w http.ResponseWriter, r *http.Request, dst any) error {
	body := http.MaxBytesReader(w, r.Body, v.MaxBytes)
	defer func() { _ = body.Close() }()

	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return ErrInvalidJSON
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return ErrInvalidJSON
	}
	return nil
}
