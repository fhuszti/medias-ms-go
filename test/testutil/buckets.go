package testutil

import (
	"context"
	"fmt"
	"github.com/fhuszti/medias-ms-go/internal/storage"
	"github.com/minio/minio-go/v7"
)

type TestBuckets struct {
	Client  *storage.Client
	Cleanup func() error
}

func SetupTestBuckets(strg *storage.Client) (*TestBuckets, error) {
	buckets := []string{"staging", "images", "docs"}
	ctx := context.Background()
	client := strg.Client

	// (re)create each bucket
	for _, b := range buckets {
		// if it already exists, drop it
		if err := client.RemoveBucket(ctx, b); err != nil {
			// ignore any errors here (e.g., bucket not found)
		}
		// now make a fresh one
		if err := client.MakeBucket(ctx, b, minio.MakeBucketOptions{}); err != nil {
			// if it already exists, skip; otherwise fail
			exists, err2 := client.BucketExists(ctx, b)
			if err2 != nil || !exists {
				return nil, fmt.Errorf("could not create bucket %q: %w", b, err)
			}
		}
	}

	cleanup := func() error {
		// remove all objects and then the buckets themselves
		for _, b := range buckets {
			// list and remove every object
			for obj := range client.ListObjects(ctx, b, minio.ListObjectsOptions{Recursive: true}) {
				if obj.Err != nil {
					continue
				}
				_ = client.RemoveObject(ctx, b, obj.Key, minio.RemoveObjectOptions{})
			}
			// delete the bucket
			if err := client.RemoveBucket(ctx, b); err != nil {
				return fmt.Errorf("could not remove bucket %q: %w", b, err)
			}
		}
		return nil
	}

	return &TestBuckets{
		Client:  strg,
		Cleanup: cleanup,
	}, nil
}
