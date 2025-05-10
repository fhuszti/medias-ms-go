package storage

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/usecase/media"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type minioClient interface {
	PresignedGetObject(ctx context.Context, bucketName, objectKey string, expiry time.Duration, reqParams url.Values) (*url.URL, error)
	PresignedPutObject(ctx context.Context, bucketName, objectKey string, expiry time.Duration) (*url.URL, error)
	StatObject(ctx context.Context, bucketName, objectKey string, opts minio.StatObjectOptions) (minio.ObjectInfo, error)
	EndpointURL() *url.URL
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) (err error)
	RemoveBucket(ctx context.Context, bucketName string) error
	ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo
	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
}

type MinioStorage struct {
	client     minioClient
	bucketName string
	useSSL     bool
}

type Client struct {
	Client minioClient
	useSSL bool
}

// compile-time check: *MinioStorage must satisfy media.Storage
var _ media.Storage = (*MinioStorage)(nil)

func NewMinioClient(endpoint, accessKey, secretKey string, useSSL bool) (*Client, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	return &Client{Client: client, useSSL: useSSL}, nil
}

func (c *Client) WithBucket(bucket string) media.Storage {
	return &MinioStorage{client: c.Client, bucketName: bucket, useSSL: c.useSSL}
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
