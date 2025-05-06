package storage

import (
	"context"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/service"
	"net/url"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioStorage struct {
	client     *minio.Client
	bucketName string
	useSSL     bool
}

// compile-time check: *MinioStorage must satisfy service.Storage
var _ service.Storage = (*MinioStorage)(nil)

func NewMinioStorage(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinioStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	s := &MinioStorage{client: client, bucketName: bucket, useSSL: useSSL}
	return s, nil
}

func (s *MinioStorage) GeneratePresignedDownloadURL(ctx context.Context, objectKey string, expiry time.Duration, downloadName string, inline bool) (string, error) {
	dispositionType := "attachment"
	if inline {
		dispositionType = "inline"
	}

	filename := filepath.Base(objectKey)
	if downloadName != "" {
		filename = downloadName
	}

	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", fmt.Sprintf("%s; filename=%q", dispositionType, filename))

	presignedURL, err := s.client.PresignedGetObject(ctx, s.bucketName, objectKey, expiry, reqParams)
	if err != nil {
		return "", err
	}

	return presignedURL.String(), nil
}

func (s *MinioStorage) GeneratePresignedUploadURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	presignedURL, err := s.client.PresignedPutObject(ctx, s.bucketName, objectKey, expiry)
	if err != nil {
		return "", err
	}

	return presignedURL.String(), nil
}

func (s *MinioStorage) ObjectExists(ctx context.Context, objectKey string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucketName, objectKey, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *MinioStorage) PublicURL(objectKey string) string {
	scheme := "https"
	if !s.useSSL {
		scheme = "http"
	}
	return scheme + "://" + s.client.EndpointURL().Host + "/" + s.bucketName + "/" + objectKey
}
