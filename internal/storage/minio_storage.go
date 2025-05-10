package storage

import (
	"context"
	"fmt"
	"log"
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
	GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)
}

type MinioStorage struct {
	client     minioClient
	bucketName string
	useSSL     bool
}

type Strg struct {
	Client minioClient
	useSSL bool
}

// compile-time check: *MinioStorage must satisfy media.Storage
var _ media.Storage = (*MinioStorage)(nil)

func NewMinioClient(endpoint, accessKey, secretKey string, useSSL bool) (*Strg, error) {
	log.Println("initialising minio client...")
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	return &Strg{Client: client, useSSL: useSSL}, nil
}

func (c *Strg) WithBucket(bucket string) (media.Storage, error) {
	ok, err := c.Client.BucketExists(context.Background(), bucket)
	if err != nil {
		return nil, err
	}
	if !ok {
		log.Printf("bucket '%s' does not exist, creating it...", bucket)
		if err := c.Client.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}
	return &MinioStorage{client: c.Client, bucketName: bucket, useSSL: c.useSSL}, nil
}

func (s *MinioStorage) GeneratePresignedDownloadURL(ctx context.Context, objectKey string, expiry time.Duration, downloadName string, inline bool) (string, error) {
	log.Printf("generating a presigned download link for media '%s' in bucket '%s'...", objectKey, s.bucketName)

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
	log.Printf("generating a presigned upload link for media '%s' in bucket '%s'...", objectKey, s.bucketName)

	presignedURL, err := s.client.PresignedPutObject(ctx, s.bucketName, objectKey, expiry)
	if err != nil {
		return "", err
	}

	return presignedURL.String(), nil
}

func (s *MinioStorage) ObjectExists(ctx context.Context, objectKey string) (bool, error) {
	log.Printf("checking if media '%s' exists in bucket '%s'...", objectKey, s.bucketName)

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
