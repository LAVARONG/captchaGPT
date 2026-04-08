package service

import "errors"

var (
	ErrEmptyModelResponse = errors.New("model returned empty content")
	ErrInvalidRequest     = errors.New("invalid request")
)
