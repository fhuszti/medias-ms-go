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

type Strg struct {
	client minioClient
	useSSL bool
}

// compile-time check: *MinioStorage must satisfy media.Storage
var _ media.Storage = (*Strg)(nil)

func NewMinioClient(endpoint, accessKey, secretKey string, useSSL bool) (*Strg, error) {
	log.Println("initialising minio client...")
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, mapMinioErr(err)
	}
	return &Strg{client, useSSL}, nil
}

func (s *Strg) InitBucket(bucket string) error {
	ok, err := s.client.BucketExists(context.Background(), bucket)
	if err != nil {
		return mapMinioErr(err)
	}
	if !ok {
		log.Printf("bucket %q does not exist, creating it...", bucket)
		if err := s.client.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{}); err != nil {
			return mapMinioErr(err)
		}
	}
	return nil
}

func (s *Strg) GeneratePresignedDownloadURL(ctx context.Context, bucket, fileKey string, expiry time.Duration) (string, error) {
	log.Printf("generating a presigned download link for file %q in bucket %q...", fileKey, bucket)

	presignedURL, err := s.client.PresignedGetObject(ctx, bucket, fileKey, expiry, url.Values{})
	if err != nil {
		return "", mapMinioErr(err)
	}

	return presignedURL.String(), nil
}

func (s *Strg) GeneratePresignedUploadURL(ctx context.Context, bucket, fileKey string, expiry time.Duration) (string, error) {
	log.Printf("generating a presigned upload link for file %q in bucket %q...", fileKey, bucket)

	presignedURL, err := s.client.PresignedPutObject(ctx, bucket, fileKey, expiry)
	if err != nil {
		return "", mapMinioErr(err)
	}

	return presignedURL.String(), nil
}

func (s *Strg) FileExists(ctx context.Context, bucket, fileKey string) (bool, error) {
	log.Printf("checking if file %q exists in bucket %q...", fileKey, bucket)

	_, err := s.StatFile(ctx, bucket, fileKey)
	if errors.Is(err, media.ErrObjectNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Strg) StatFile(ctx context.Context, bucket, fileKey string) (media.FileInfo, error) {
	log.Printf("getting stats on file %q in bucket %q...", fileKey, bucket)

	info, err := s.client.StatObject(ctx, bucket, fileKey, minio.StatObjectOptions{})
	if err != nil {
		return media.FileInfo{}, mapMinioErr(err)
	}
	return media.FileInfo{
		SizeBytes:   info.Size,
		ContentType: info.ContentType,
	}, nil
}

func (s *Strg) RemoveFile(ctx context.Context, bucket, fileKey string) error {
	log.Printf("removing file %q from bucket %q...", fileKey, bucket)

	err := s.client.RemoveObject(ctx, bucket, fileKey, minio.RemoveObjectOptions{})
	return mapMinioErr(err)
}

func (s *Strg) GetFile(ctx context.Context, bucket, fileKey string) (io.ReadSeekCloser, error) {
	log.Printf("getting file %q from bucket %q...", fileKey, bucket)

	obj, err := s.client.GetObject(ctx, bucket, fileKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, mapMinioErr(err)
	}
	return obj, nil
}

func (s *Strg) SaveFile(ctx context.Context, bucket, fileKey string, reader io.Reader, fileSize int64, opts map[string]string) error {
	log.Printf("saving file %q into bucket %q...", fileKey, bucket)

	putOpts := minio.PutObjectOptions{}
	if ct := opts["Content-Type"]; ct != "" {
		putOpts.ContentType = ct
	}

	_, err := s.client.PutObject(ctx, bucket, fileKey, reader, fileSize, putOpts)
	if err != nil {
		return mapMinioErr(err)
	}
	return nil
}

func (s *Strg) CopyFile(ctx context.Context, bucket, srcKey, destKey string) error {
	log.Printf("copying file %q to %q inside bucket %q...", srcKey, destKey, bucket)

	destOpts := minio.CopyDestOptions{
		Bucket: bucket,
		Object: destKey,
	}
	srcOpts := minio.CopySrcOptions{
		Bucket: bucket,
		Object: srcKey,
	}

	_, err := s.client.CopyObject(ctx, destOpts, srcOpts)
	if err != nil {
		return mapMinioErr(err)
	}
	return nil
}
