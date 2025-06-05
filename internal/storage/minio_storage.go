package storage

import (
	"context"
	"errors"
	"io"
	"log"
	"net/url"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/usecase/media"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

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
		return nil, mapMinioErr(err)
	}
	return &Strg{Client: client, useSSL: useSSL}, nil
}

func (c *Strg) WithBucket(bucket string) (media.Storage, error) {
	ok, err := c.Client.BucketExists(context.Background(), bucket)
	if err != nil {
		return nil, mapMinioErr(err)
	}
	if !ok {
		log.Printf("bucket %q does not exist, creating it...", bucket)
		if err := c.Client.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, mapMinioErr(err)
		}
	}
	return &MinioStorage{client: c.Client, bucketName: bucket, useSSL: c.useSSL}, nil
}

func (s *MinioStorage) GeneratePresignedDownloadURL(ctx context.Context, fileKey string, expiry time.Duration) (string, error) {
	log.Printf("generating a presigned download link for file %q in bucket %q...", fileKey, s.bucketName)

	presignedURL, err := s.client.PresignedGetObject(ctx, s.bucketName, fileKey, expiry, url.Values{})
	if err != nil {
		return "", mapMinioErr(err)
	}

	return presignedURL.String(), nil
}

func (s *MinioStorage) GeneratePresignedUploadURL(ctx context.Context, fileKey string, expiry time.Duration) (string, error) {
	log.Printf("generating a presigned upload link for file %q in bucket %q...", fileKey, s.bucketName)

	presignedURL, err := s.client.PresignedPutObject(ctx, s.bucketName, fileKey, expiry)
	if err != nil {
		return "", mapMinioErr(err)
	}

	return presignedURL.String(), nil
}

func (s *MinioStorage) FileExists(ctx context.Context, fileKey string) (bool, error) {
	log.Printf("checking if file %q exists in bucket %q...", fileKey, s.bucketName)

	_, err := s.StatFile(ctx, fileKey)
	if errors.Is(err, media.ErrObjectNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *MinioStorage) StatFile(ctx context.Context, fileKey string) (media.FileInfo, error) {
	log.Printf("getting stats on file %q in bucket %q...", fileKey, s.bucketName)

	info, err := s.client.StatObject(ctx, s.bucketName, fileKey, minio.StatObjectOptions{})
	if err != nil {
		return media.FileInfo{}, mapMinioErr(err)
	}
	return media.FileInfo{
		SizeBytes:   info.Size,
		ContentType: info.ContentType,
	}, nil
}

func (s *MinioStorage) RemoveFile(ctx context.Context, fileKey string) error {
	log.Printf("removing file %q from bucket %q...", fileKey, s.bucketName)

	err := s.client.RemoveObject(ctx, s.bucketName, fileKey, minio.RemoveObjectOptions{})
	return mapMinioErr(err)
}

func (s *MinioStorage) GetFile(ctx context.Context, fileKey string) (io.ReadSeekCloser, error) {
	log.Printf("getting file %q from bucket %q...", fileKey, s.bucketName)

	obj, err := s.client.GetObject(ctx, s.bucketName, fileKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, mapMinioErr(err)
	}
	return obj, nil
}

func (s *MinioStorage) SaveFile(ctx context.Context, fileKey string, reader io.Reader, fileSize int64, opts map[string]string) error {
	log.Printf("saving file %q into bucket %q...", fileKey, s.bucketName)

	putOpts := minio.PutObjectOptions{}
	if ct := opts["Content-Type"]; ct != "" {
		putOpts.ContentType = ct
	}

	_, err := s.client.PutObject(ctx, s.bucketName, fileKey, reader, fileSize, putOpts)
	if err != nil {
		return mapMinioErr(err)
	}
	return nil
}

func (s *MinioStorage) CopyFile(ctx context.Context, srcKey, destKey string) error {
	log.Printf("copying file %q to %q inside bucket %q...", srcKey, destKey, s.bucketName)

	destOpts := minio.CopyDestOptions{
		Bucket: s.bucketName,
		Object: destKey,
	}
	srcOpts := minio.CopySrcOptions{
		Bucket: s.bucketName,
		Object: srcKey,
	}

	_, err := s.client.CopyObject(ctx, destOpts, srcOpts)
	if err != nil {
		return mapMinioErr(err)
	}
	return nil
}
