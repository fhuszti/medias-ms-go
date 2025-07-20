package storage

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/port"
	"github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/minio/minio-go/v7"
)

type mockMinio struct {
	bucketExistsFn       func(ctx context.Context, bucketName string) (bool, error)
	makeBucketFn         func(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) (err error)
	removeBucketFn       func(ctx context.Context, bucketName string) error
	listObjectsFn        func(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo
	removeObjectFn       func(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
	presignedGetObjectFn func(ctx context.Context, bucket, key string, expiry time.Duration) (*url.URL, error)
	presignedPutObjectFn func(ctx context.Context, bucket, key string, expiry time.Duration) (*url.URL, error)
	statObjectFn         func(ctx context.Context, bucket, key string, opts minio.StatObjectOptions) (minio.ObjectInfo, error)
	getObjectFn          func(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)
	putObjectFn          func(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	copyObjectFn         func(ctx context.Context, dst minio.CopyDestOptions, src minio.CopySrcOptions) (minio.UploadInfo, error)
}

func (m *mockMinio) GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error) {
	return m.getObjectFn(ctx, bucketName, objectName, opts)
}
func (m *mockMinio) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	return m.bucketExistsFn(ctx, bucketName)
}
func (m *mockMinio) MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) (err error) {
	return m.makeBucketFn(ctx, bucketName, opts)
}
func (m *mockMinio) RemoveBucket(ctx context.Context, bucketName string) error {
	return m.removeBucketFn(ctx, bucketName)
}
func (m *mockMinio) ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
	return m.listObjectsFn(ctx, bucketName, opts)
}
func (m *mockMinio) RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error {
	return m.removeObjectFn(ctx, bucketName, objectName, opts)
}
func (m *mockMinio) PresignedGetObject(ctx context.Context, bucket, key string, expiry time.Duration, reqParams url.Values) (*url.URL, error) {
	return m.presignedGetObjectFn(ctx, bucket, key, expiry)
}
func (m *mockMinio) PresignedPutObject(ctx context.Context, bucket, key string, expiry time.Duration) (*url.URL, error) {
	return m.presignedPutObjectFn(ctx, bucket, key, expiry)
}
func (m *mockMinio) StatObject(ctx context.Context, bucket, key string, opts minio.StatObjectOptions) (minio.ObjectInfo, error) {
	return m.statObjectFn(ctx, bucket, key, opts)
}
func (m *mockMinio) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	return m.putObjectFn(ctx, bucketName, objectName, reader, objectSize, opts)
}
func (m *mockMinio) CopyObject(ctx context.Context, dst minio.CopyDestOptions, src minio.CopySrcOptions) (minio.UploadInfo, error) {
	return m.copyObjectFn(ctx, dst, src)
}

func makeStorage(mockClient *mockMinio) port.Storage {
	return &Strg{
		Client: mockClient,
	}
}

func TestInitBucket(t *testing.T) {
	tests := []struct {
		name           string
		exists         bool
		existsErr      error
		makeErr        error
		wantMakeCalled bool
		wantErr        error
	}{
		{
			name:           "bucket exists, no create",
			exists:         true,
			wantMakeCalled: false,
		},
		{
			name:           "bucket does not exist, create succeeds",
			exists:         false,
			wantMakeCalled: true,
		},
		{
			name:      "BucketExists error bubbles up",
			existsErr: errors.New("exist fail"),
			wantErr:   media.ErrInternal,
		},
		{
			name:           "MakeBucket error bubbles up",
			exists:         false,
			makeErr:        errors.New("make fail"),
			wantMakeCalled: true,
			wantErr:        media.ErrInternal,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			makeCalled := false

			mock := &mockMinio{
				bucketExistsFn: func(ctx context.Context, bucketName string) (bool, error) {
					return tc.exists, tc.existsErr
				},
				makeBucketFn: func(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error {
					makeCalled = true
					return tc.makeErr
				},
			}

			strg := &Strg{Client: mock}
			err := strg.InitBucket("my-bucket")

			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tc.wantErr)
				}
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("error = %q; want %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if makeCalled != tc.wantMakeCalled {
				t.Errorf("MakeBucket called = %v; want %v", makeCalled, tc.wantMakeCalled)
			}
		})
	}
}

func TestGeneratePresignedDownloadURL(t *testing.T) {
	fake, _ := url.Parse("https://cdn.example.com/download")
	mock := &mockMinio{
		presignedGetObjectFn: func(_ context.Context, bucket, key string, expiry time.Duration) (*url.URL, error) {
			if bucket != "bucket" {
				t.Errorf("bucket = %q; want %q", bucket, "bucket")
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
	s := makeStorage(mock)

	out, err := s.GeneratePresignedDownloadURL(context.Background(), "bucket", "obj.bin", 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != fake.String() {
		t.Errorf("url = %q; want %q", out, fake.String())
	}
}

func TestGeneratePresignedDownloadURL_Error(t *testing.T) {
	mock := &mockMinio{
		presignedGetObjectFn: func(_ context.Context, _, _ string, _ time.Duration) (*url.URL, error) {
			return nil, errors.New("fail-put")
		},
	}
	s := makeStorage(mock)

	_, err := s.GeneratePresignedDownloadURL(context.Background(), "bucket", "k", time.Minute)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, media.ErrInternal) {
		t.Errorf("error = %q; want %q", err.Error(), "fail-put")
	}
}

func TestGeneratePresignedUploadURL(t *testing.T) {
	fake, _ := url.Parse("https://cdn.example.com/upload")
	mock := &mockMinio{
		presignedPutObjectFn: func(_ context.Context, bucket, key string, expiry time.Duration) (*url.URL, error) {
			if bucket != "bucket" {
				t.Errorf("bucket = %q; want %q", bucket, "bucket")
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
	s := makeStorage(mock)

	out, err := s.GeneratePresignedUploadURL(context.Background(), "bucket", "obj.bin", 5*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != fake.String() {
		t.Errorf("url = %q; want %q", out, fake.String())
	}
}

func TestGeneratePresignedUploadURL_Error(t *testing.T) {
	mock := &mockMinio{
		presignedPutObjectFn: func(_ context.Context, _, _ string, _ time.Duration) (*url.URL, error) {
			return nil, errors.New("fail-put")
		},
	}
	s := makeStorage(mock)

	_, err := s.GeneratePresignedUploadURL(context.Background(), "bucket", "k", time.Minute)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, media.ErrInternal) {
		t.Errorf("error = %q; want %q", err.Error(), "fail-put")
	}
}

func TestObjectExists(t *testing.T) {
	ctx := context.Background()

	// Case: object exists
	mock1 := &mockMinio{
		statObjectFn: func(_ context.Context, _, _ string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
			return minio.ObjectInfo{}, nil
		},
	}
	s1 := makeStorage(mock1)
	exists, err := s1.FileExists(ctx, "bucket", "foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("exists = false; want true")
	}

	// Case: NoSuchKey → does not exist
	mock2 := &mockMinio{
		statObjectFn: func(_ context.Context, _, _ string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
			e := minio.ToErrorResponse(errors.New("ignored"))
			e.Code = "NoSuchKey"
			return minio.ObjectInfo{}, e
		},
	}
	s2 := makeStorage(mock2)
	exists2, err2 := s2.FileExists(ctx, "bucket", "bar")
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}
	if exists2 {
		t.Error("exists = true; want false")
	}

	// Case: other error
	mock3 := &mockMinio{
		statObjectFn: func(_ context.Context, _, _ string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
			return minio.ObjectInfo{}, errors.New("boom")
		},
	}
	s3 := makeStorage(mock3)
	exists3, err3 := s3.FileExists(ctx, "bucket", "baz")
	if err3 == nil {
		t.Fatal("expected error, got nil")
	}
	if exists3 {
		t.Error("exists = true; want false")
	}
}

func TestStatFile_Success(t *testing.T) {
	ctx := context.Background()
	expected := minio.ObjectInfo{
		Size:        123,
		ContentType: "image/png",
	}
	mock := &mockMinio{
		statObjectFn: func(_ context.Context, bucket, key string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
			if bucket != "bucket" {
				t.Errorf("bucket = %q; want %q", bucket, "bucket")
			}
			if key != "f.txt" {
				t.Errorf("key = %q; want %q", key, "f.txt")
			}
			return expected, nil
		},
	}
	s := makeStorage(mock)
	fi, err := s.StatFile(ctx, "bucket", "f.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fi.SizeBytes != expected.Size {
		t.Errorf("SizeBytes = %d; want %d", fi.SizeBytes, expected.Size)
	}
	if fi.ContentType != expected.ContentType {
		t.Errorf("ContentType = %q; want %q", fi.ContentType, expected.ContentType)
	}
}

func TestStatFile_NotFound(t *testing.T) {
	ctx := context.Background()
	mock := &mockMinio{
		statObjectFn: func(_ context.Context, _, _ string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
			e := minio.ToErrorResponse(errors.New("ignored"))
			e.Code = "NoSuchKey"
			return minio.ObjectInfo{}, e
		},
	}
	s := makeStorage(mock)
	_, err := s.StatFile(ctx, "bucket", "k")
	if !errors.Is(err, media.ErrObjectNotFound) {
		t.Fatalf("err = %v; want ErrObjectNotFound", err)
	}
}

func TestStatFile_OtherError(t *testing.T) {
	ctx := context.Background()
	mock := &mockMinio{
		statObjectFn: func(_ context.Context, _, _ string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
			return minio.ObjectInfo{}, errors.New("some failure")
		},
	}
	s := makeStorage(mock)
	_, err := s.StatFile(ctx, "bucket", "k")
	if !errors.Is(err, media.ErrInternal) {
		t.Fatalf("err = %v; want ErrInternal", err)
	}
}

func TestRemoveFile_Success(t *testing.T) {
	ctx := context.Background()
	called := false
	mock := &mockMinio{
		removeObjectFn: func(_ context.Context, bucket, key string, _ minio.RemoveObjectOptions) error {
			called = true
			if bucket != "bucket" {
				t.Errorf("bucket = %q; want %q", bucket, "bucket")
			}
			if key != "file" {
				t.Errorf("key = %q; want %q", key, "file")
			}
			return nil
		},
	}
	s := makeStorage(mock)
	if err := s.RemoveFile(ctx, "bucket", "file"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("RemoveObject was not called")
	}
}

func TestRemoveFile_NotFound(t *testing.T) {
	ctx := context.Background()
	mock := &mockMinio{
		removeObjectFn: func(_ context.Context, _, _ string, _ minio.RemoveObjectOptions) error {
			e := minio.ToErrorResponse(errors.New("ignored"))
			e.Code = "NoSuchKey"
			return e
		},
	}
	s := makeStorage(mock)
	err := s.RemoveFile(ctx, "bucket", "f")
	if !errors.Is(err, media.ErrObjectNotFound) {
		t.Fatalf("err = %v; want ErrObjectNotFound", err)
	}
}

func TestRemoveFile_OtherError(t *testing.T) {
	ctx := context.Background()
	mock := &mockMinio{
		removeObjectFn: func(_ context.Context, _, _ string, _ minio.RemoveObjectOptions) error {
			return errors.New("boom")
		},
	}
	s := makeStorage(mock)
	err := s.RemoveFile(ctx, "bucket", "f")
	if !errors.Is(err, media.ErrInternal) {
		t.Fatalf("err = %v; want ErrInternal", err)
	}
}

func TestSaveFile_Success(t *testing.T) {
	ctx := context.Background()
	var (
		gotReader io.Reader
		gotSize   int64
		gotOpts   minio.PutObjectOptions
	)
	mock := &mockMinio{
		putObjectFn: func(_ context.Context, bucket, key string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
			if bucket != "dest" {
				t.Errorf("bucket = %q; want %q", bucket, "dest")
			}
			if key != "obj" {
				t.Errorf("key = %q; want %q", key, "obj")
			}
			gotReader = reader
			gotSize = objectSize
			gotOpts = opts
			return minio.UploadInfo{Size: objectSize}, nil
		},
	}
	s := makeStorage(mock)
	content := "hello"
	err := s.SaveFile(ctx, "dest", "obj", strings.NewReader(content), int64(len(content)), map[string]string{"Content-Type": "text/plain"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotSize != int64(len(content)) {
		t.Errorf("size = %d; want %d", gotSize, len(content))
	}
	if gotOpts.ContentType != "text/plain" {
		t.Errorf("ContentType = %q; want %q", gotOpts.ContentType, "text/plain")
	}
	// verify the reader was passed through
	buf := new(strings.Builder)
	if _, err := io.Copy(buf, gotReader); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if buf.String() != content {
		t.Errorf("reader content = %q; want %q", buf.String(), content)
	}
}

func TestSaveFile_NoContentType(t *testing.T) {
	ctx := context.Background()
	var gotOpts minio.PutObjectOptions
	mock := &mockMinio{
		putObjectFn: func(_ context.Context, _, _ string, _ io.Reader, _ int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
			gotOpts = opts
			return minio.UploadInfo{}, nil
		},
	}
	s := makeStorage(mock)
	if err := s.SaveFile(ctx, "bucket", "k", nil, 0, map[string]string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotOpts.ContentType != "" {
		t.Errorf("ContentType = %q; want empty", gotOpts.ContentType)
	}
}

func TestSaveFile_ErrorMapping(t *testing.T) {
	ctx := context.Background()
	mock := &mockMinio{
		putObjectFn: func(_ context.Context, _, _ string, _ io.Reader, _ int64, _ minio.PutObjectOptions) (minio.UploadInfo, error) {
			e := minio.ToErrorResponse(errors.New("denied"))
			e.Code = "AccessDenied"
			return minio.UploadInfo{}, e
		},
	}
	s := makeStorage(mock)
	err := s.SaveFile(ctx, "bucket", "k", nil, 0, map[string]string{})
	if !errors.Is(err, media.ErrUnauthorized) {
		t.Fatalf("err = %v; want ErrUnauthorized", err)
	}

	// another error case
	mock.putObjectFn = func(_ context.Context, _, _ string, _ io.Reader, _ int64, _ minio.PutObjectOptions) (minio.UploadInfo, error) {
		return minio.UploadInfo{}, errors.New("boom")
	}
	err2 := s.SaveFile(ctx, "bucket", "k", nil, 0, map[string]string{})
	if !errors.Is(err2, media.ErrInternal) {
		t.Fatalf("err = %v; want ErrInternal", err2)
	}
}

func TestGetFile_Success(t *testing.T) {
	ctx := context.Background()
	dummy := &minio.Object{} // zero-value is non-nil pointer
	mock := &mockMinio{
		getObjectFn: func(_ context.Context, bucket, key string, _ minio.GetObjectOptions) (*minio.Object, error) {
			if bucket != "bucket" {
				t.Errorf("bucket = %q; want %q", bucket, "bucket")
			}
			if key != "key" {
				t.Errorf("key = %q; want %q", key, "key")
			}
			return dummy, nil
		},
	}
	s := makeStorage(mock)
	rsc, err := s.GetFile(ctx, "bucket", "key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rsc != dummy {
		t.Errorf("reader = %v; want %v", rsc, dummy)
	}
}

func TestGetFile_NotFound(t *testing.T) {
	ctx := context.Background()
	mock := &mockMinio{
		getObjectFn: func(_ context.Context, _, _ string, _ minio.GetObjectOptions) (*minio.Object, error) {
			e := minio.ToErrorResponse(errors.New("missing"))
			e.Code = "NoSuchKey"
			return nil, e
		},
	}
	s := makeStorage(mock)
	_, err := s.GetFile(ctx, "bucket", "k")
	if !errors.Is(err, media.ErrObjectNotFound) {
		t.Fatalf("err = %v; want ErrObjectNotFound", err)
	}
}

func TestGetFile_OtherError(t *testing.T) {
	ctx := context.Background()
	mock := &mockMinio{
		getObjectFn: func(_ context.Context, _, _ string, _ minio.GetObjectOptions) (*minio.Object, error) {
			return nil, errors.New("boom")
		},
	}
	s := makeStorage(mock)
	_, err := s.GetFile(ctx, "bucket", "k")
	if !errors.Is(err, media.ErrInternal) {
		t.Fatalf("err = %v; want ErrInternal", err)
	}
}

func TestCopyFile_Success(t *testing.T) {
	ctx := context.Background()
	called := false
	mock := &mockMinio{
		copyObjectFn: func(_ context.Context, dst minio.CopyDestOptions, src minio.CopySrcOptions) (minio.UploadInfo, error) {
			called = true
			if dst.Bucket != "bucket" {
				t.Errorf("dst bucket = %q; want %q", dst.Bucket, "my-bucket")
			}
			if src.Bucket != "bucket" {
				t.Errorf("src bucket = %q; want %q", src.Bucket, "my-bucket")
			}

			if dst.Object != "destKey" {
				t.Errorf("dst key = %q; want %q", dst.Object, "file")
			}
			if src.Object != "srcKey" {
				t.Errorf("dst key = %q; want %q", src.Object, "file")
			}
			return minio.UploadInfo{}, nil
		},
	}
	s := makeStorage(mock)
	if err := s.CopyFile(ctx, "bucket", "srcKey", "destKey"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("CopyObject was not called")
	}
}

func TestCopyFile_NotFound(t *testing.T) {
	ctx := context.Background()
	mock := &mockMinio{
		copyObjectFn: func(_ context.Context, dst minio.CopyDestOptions, src minio.CopySrcOptions) (minio.UploadInfo, error) {
			e := minio.ToErrorResponse(errors.New("ignored"))
			e.Code = "NoSuchKey"
			return minio.UploadInfo{}, e
		},
	}
	s := makeStorage(mock)
	err := s.CopyFile(ctx, "bucket", "srcKey", "destKey")
	if !errors.Is(err, media.ErrObjectNotFound) {
		t.Fatalf("err = %v; want ErrObjectNotFound", err)
	}
}

func TestCopyFile_OtherError(t *testing.T) {
	ctx := context.Background()
	mock := &mockMinio{
		copyObjectFn: func(_ context.Context, dst minio.CopyDestOptions, src minio.CopySrcOptions) (minio.UploadInfo, error) {
			return minio.UploadInfo{}, errors.New("boom")
		},
	}
	s := makeStorage(mock)
	err := s.CopyFile(ctx, "bucket", "srcKey", "destKey")
	if !errors.Is(err, media.ErrInternal) {
		t.Fatalf("err = %v; want ErrInternal", err)
	}
}
