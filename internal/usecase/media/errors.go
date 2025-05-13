package media

import "errors"

var (
	ErrObjectNotFound = errors.New("storage: object not found")
	ErrBucketNotFound = errors.New("storage: bucket not found")
	ErrUnauthorized   = errors.New("storage: unauthorized")
	ErrInternal       = errors.New("storage: internal error")
)
