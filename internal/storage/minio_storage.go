package storage

import (
	"context"
	"errors"
	"io"
	"net/url"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/fhuszti/medias-ms-go/internal/logger"
)

type Strg struct {
	Client minioClient
}

// compile-time check: *Strg must satisfy port.Storage
var _ port.Storage = (*Strg)(nil)

func NewStorage(endpoint, accessKey, secretKey string, useSSL bool) (*Strg, error) {
	logger.Info(context.Background(), "initialising minio client...")
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, mapMinioErr(err)
	}
	return &Strg{client}, nil
}

func (s *Strg) InitBucket(bucket string) error {
	ok, err := s.Client.BucketExists(context.Background(), bucket)
	if err != nil {
		return mapMinioErr(err)
	}
	if !ok {
		logger.Infof(context.Background(), "bucket %q does not exist, creating it...", bucket)
		if err := s.Client.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{}); err != nil {
			return mapMinioErr(err)
		}
	}
	return nil
}

func (s *Strg) GeneratePresignedDownloadURL(ctx context.Context, bucket, fileKey string, expiry time.Duration) (string, error) {
	logger.Debugf(ctx, "generating a presigned download link for file %q in bucket %q...", fileKey, bucket)

	presignedURL, err := s.Client.PresignedGetObject(ctx, bucket, fileKey, expiry, url.Values{})
	if err != nil {
		return "", mapMinioErr(err)
	}

	return presignedURL.String(), nil
}

func (s *Strg) GeneratePresignedUploadURL(ctx context.Context, bucket, fileKey string, expiry time.Duration) (string, error) {
	logger.Debugf(ctx, "generating a presigned upload link for file %q in bucket %q...", fileKey, bucket)

	presignedURL, err := s.Client.PresignedPutObject(ctx, bucket, fileKey, expiry)
	if err != nil {
		return "", mapMinioErr(err)
	}

	return presignedURL.String(), nil
}

func (s *Strg) FileExists(ctx context.Context, bucket, fileKey string) (bool, error) {
	logger.Debugf(ctx, "checking if file %q exists in bucket %q...", fileKey, bucket)

	_, err := s.StatFile(ctx, bucket, fileKey)
	if errors.Is(err, media.ErrObjectNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Strg) StatFile(ctx context.Context, bucket, fileKey string) (port.FileInfo, error) {
	logger.Debugf(ctx, "getting stats on file %q in bucket %q...", fileKey, bucket)

	info, err := s.Client.StatObject(ctx, bucket, fileKey, minio.StatObjectOptions{})
	if err != nil {
		return port.FileInfo{}, mapMinioErr(err)
	}
	return port.FileInfo{
		SizeBytes:   info.Size,
		ContentType: info.ContentType,
	}, nil
}

func (s *Strg) RemoveFile(ctx context.Context, bucket, fileKey string) error {
	logger.Debugf(ctx, "removing file %q from bucket %q...", fileKey, bucket)

	err := s.Client.RemoveObject(ctx, bucket, fileKey, minio.RemoveObjectOptions{})
	return mapMinioErr(err)
}

func (s *Strg) GetFile(ctx context.Context, bucket, fileKey string) (io.ReadSeekCloser, error) {
	logger.Debugf(ctx, "getting file %q from bucket %q...", fileKey, bucket)

	obj, err := s.Client.GetObject(ctx, bucket, fileKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, mapMinioErr(err)
	}
	return obj, nil
}

func (s *Strg) SaveFile(ctx context.Context, bucket, fileKey string, reader io.Reader, fileSize int64, opts map[string]string) error {
	logger.Debugf(ctx, "saving file %q into bucket %q...", fileKey, bucket)

	putOpts := minio.PutObjectOptions{}
	if ct := opts["Content-Type"]; ct != "" {
		putOpts.ContentType = ct
	}

	_, err := s.Client.PutObject(ctx, bucket, fileKey, reader, fileSize, putOpts)
	if err != nil {
		return mapMinioErr(err)
	}
	return nil
}

func (s *Strg) CopyFile(ctx context.Context, bucket, srcKey, destKey string) error {
	logger.Debugf(ctx, "copying file %q to %q inside bucket %q...", srcKey, destKey, bucket)

	destOpts := minio.CopyDestOptions{
		Bucket: bucket,
		Object: destKey,
	}
	srcOpts := minio.CopySrcOptions{
		Bucket: bucket,
		Object: srcKey,
	}

	_, err := s.Client.CopyObject(ctx, destOpts, srcOpts)
	if err != nil {
		return mapMinioErr(err)
	}
	return nil
}
