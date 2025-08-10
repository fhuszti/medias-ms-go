package storage

import (
	"fmt"

	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/minio/minio-go/v7"
)

func mapMinioErr(err error) error {
	if err == nil {
		return nil
	}
	resp := minio.ToErrorResponse(err)
	switch resp.Code {
	case "NoSuchKey":
		return media.ErrObjectNotFound
	case "NoSuchBucket":
		return media.ErrBucketNotFound
	case "AccessDenied", "InvalidAccessKeyId", "SignatureDoesNotMatch":
		return media.ErrUnauthorized
	default:
		// catch everything else
		return fmt.Errorf("%w: %v", media.ErrInternal, err)
	}
}
