package storage

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
)

type mockMinio struct {
	getFn       func(ctx context.Context, bucket, key string, expiry time.Duration, params url.Values) (*url.URL, error)
	putFn       func(ctx context.Context, bucket, key string, expiry time.Duration) (*url.URL, error)
	statFn      func(ctx context.Context, bucket, key string, opts minio.StatObjectOptions) (minio.ObjectInfo, error)
	endpointURL *url.URL
}

func (m *mockMinio) PresignedGetObject(ctx context.Context, bucket, key string, expiry time.Duration, params url.Values) (*url.URL, error) {
	return m.getFn(ctx, bucket, key, expiry, params)
}
func (m *mockMinio) PresignedPutObject(ctx context.Context, bucket, key string, expiry time.Duration) (*url.URL, error) {
	return m.putFn(ctx, bucket, key, expiry)
}
func (m *mockMinio) StatObject(ctx context.Context, bucket, key string, opts minio.StatObjectOptions) (minio.ObjectInfo, error) {
	return m.statFn(ctx, bucket, key, opts)
}
func (m *mockMinio) EndpointURL() *url.URL {
	return m.endpointURL
}

func makeStorage(mockClient *mockMinio, bucket string, useSSL bool) *MinioStorage {
	return &MinioStorage{
		client:     mockClient,
		bucketName: bucket,
		useSSL:     useSSL,
	}
}

func TestGeneratePresignedDownloadURL(t *testing.T) {
	fake, _ := url.Parse("https://cdn.example.com/download?x=1")
	mock := &mockMinio{
		getFn: func(_ context.Context, bucket, key string, expiry time.Duration, params url.Values) (*url.URL, error) {
			// bucket and key should be forwarded
			if bucket != "my-bucket" {
				t.Errorf("bucket = %q; want %q", bucket, "my-bucket")
			}
			if key != "path/to/asset.png" {
				t.Errorf("key = %q; want %q", key, "path/to/asset.png")
			}
			// expiry should be preserved
			if expiry != 15*time.Minute {
				t.Errorf("expiry = %v; want %v", expiry, 15*time.Minute)
			}
			// default disposition: attachment + basename
			disp := params.Get("response-content-disposition")
			expected := `attachment; filename="asset.png"`
			if disp != expected {
				t.Errorf("disposition = %q; want %q", disp, expected)
			}
			return fake, nil
		},
		endpointURL: &url.URL{Scheme: "https", Host: "cdn.example.com"},
	}
	s := makeStorage(mock, "my-bucket", true)

	out, err := s.GeneratePresignedDownloadURL(
		context.Background(),
		"path/to/asset.png",
		15*time.Minute,
		"",
		false,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != fake.String() {
		t.Errorf("url = %q; want %q", out, fake.String())
	}
}

func TestGeneratePresignedDownloadURL_InlineAndName(t *testing.T) {
	fake, _ := url.Parse("http://localhost/get")
	mock := &mockMinio{
		getFn: func(_ context.Context, _, _ string, _ time.Duration, params url.Values) (*url.URL, error) {
			disp := params.Get("response-content-disposition")
			// inline + custom filename
			expected := `inline; filename="download.dat"`
			if disp != expected {
				t.Errorf("disposition = %q; want %q", disp, expected)
			}
			return fake, nil
		},
		endpointURL: &url.URL{Scheme: "http", Host: "localhost:9000"},
	}
	s := makeStorage(mock, "bucket", false)

	out, err := s.GeneratePresignedDownloadURL(
		context.Background(),
		"any/key.txt",
		1*time.Minute,
		"download.dat",
		true, // inline
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != fake.String() {
		t.Errorf("url = %q; want %q", out, fake.String())
	}
}

func TestGeneratePresignedDownloadURL_Error(t *testing.T) {
	mock := &mockMinio{
		getFn: func(_ context.Context, _, _ string, _ time.Duration, _ url.Values) (*url.URL, error) {
			return nil, errors.New("fail-get")
		},
		endpointURL: &url.URL{Scheme: "https", Host: "x"},
	}
	s := makeStorage(mock, "b", true)

	_, err := s.GeneratePresignedDownloadURL(context.Background(), "k", 5*time.Minute, "", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "fail-get" {
		t.Errorf("error = %q; want %q", err.Error(), "fail-get")
	}
}

func TestGeneratePresignedUploadURL(t *testing.T) {
	fake, _ := url.Parse("https://cdn.example.com/upload")
	mock := &mockMinio{
		putFn: func(_ context.Context, bucket, key string, expiry time.Duration) (*url.URL, error) {
			if bucket != "u-bucket" {
				t.Errorf("bucket = %q; want %q", bucket, "u-bucket")
			}
			if key != "obj.bin" {
				t.Errorf("key = %q; want %q", key, "obj.bin")
			}
			if expiry != 5*time.Minute {
				t.Errorf("expiry = %v; want %v", expiry, 5*time.Minute)
			}
			return fake, nil
		},
	}
	s := makeStorage(mock, "u-bucket", true)

	out, err := s.GeneratePresignedUploadURL(context.Background(), "obj.bin", 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != fake.String() {
		t.Errorf("url = %q; want %q", out, fake.String())
	}
}

func TestGeneratePresignedUploadURL_Error(t *testing.T) {
	mock := &mockMinio{
		putFn: func(_ context.Context, _, _ string, _ time.Duration) (*url.URL, error) {
			return nil, errors.New("fail-put")
		},
	}
	s := makeStorage(mock, "any", false)

	_, err := s.GeneratePresignedUploadURL(context.Background(), "k", time.Minute)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "fail-put" {
		t.Errorf("error = %q; want %q", err.Error(), "fail-put")
	}
}

func TestObjectExists(t *testing.T) {
	ctx := context.Background()

	// Case: object exists
	mock1 := &mockMinio{
		statFn: func(_ context.Context, _, _ string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
			return minio.ObjectInfo{}, nil
		},
	}
	s1 := makeStorage(mock1, "b", false)
	exists, err := s1.ObjectExists(ctx, "foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("exists = false; want true")
	}

	// Case: NoSuchKey → does not exist
	mock2 := &mockMinio{
		statFn: func(_ context.Context, _, _ string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
			e := minio.ToErrorResponse(errors.New("ignored"))
			e.Code = "NoSuchKey"
			return minio.ObjectInfo{}, e
		},
	}
	s2 := makeStorage(mock2, "b", false)
	exists2, err2 := s2.ObjectExists(ctx, "bar")
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}
	if exists2 {
		t.Error("exists = true; want false")
	}

	// Case: other error
	mock3 := &mockMinio{
		statFn: func(_ context.Context, _, _ string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
			return minio.ObjectInfo{}, errors.New("boom")
		},
	}
	s3 := makeStorage(mock3, "b", true)
	exists3, err3 := s3.ObjectExists(ctx, "baz")
	if err3 == nil {
		t.Fatal("expected error, got nil")
	}
	if exists3 {
		t.Error("exists = true; want false")
	}
}

func TestPublicURL(t *testing.T) {
	endp, _ := url.Parse("https://files.example")
	mock := &mockMinio{endpointURL: endp}

	s1 := makeStorage(mock, "buck", false)
	got1 := s1.PublicURL("f.txt")
	want1 := "http://files.example/buck/f.txt"
	if got1 != want1 {
		t.Errorf("PublicURL = %q; want %q", got1, want1)
	}

	s2 := makeStorage(mock, "buck", true)
	got2 := s2.PublicURL("dir/x.jpg")
	want2 := "https://files.example/buck/dir/x.jpg"
	if got2 != want2 {
		t.Errorf("PublicURL = %q; want %q", got2, want2)
	}
}
