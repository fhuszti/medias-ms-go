package media

import (
	"bytes"
	"context"
	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"io"
	"time"
)

type mockRepo struct {
	mediaRecord *model.Media

	getErr    error
	createErr error
	updateErr error

	getCalled bool
	created   *model.Media
	updated   *model.Media
}

func (m *mockRepo) GetByID(ctx context.Context, id db.UUID) (*model.Media, error) {
	m.getCalled = true
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.mediaRecord, nil
}
func (m *mockRepo) Update(ctx context.Context, media *model.Media) error {
	m.updated = media
	return m.updateErr
}
func (m *mockRepo) Create(ctx context.Context, media *model.Media) error {
	m.created = media
	return m.createErr
}

type nopRSC struct{ io.ReadSeeker }

func (nopRSC) Close() error { return nil }

type mockStorage struct {
	reader     io.ReadSeeker
	statInfo   FileInfo
	objectKey  string
	ttl        time.Duration
	fileExists bool

	initBucketErr           error
	generateDownloadLinkErr error
	generateUploadLinkErr   error
	statErr                 error
	getErr                  error
	saveErr                 error
	copyErr                 error
	fileExistsErr           error

	initBucketCalled           bool
	generateDownloadLinkCalled bool
	generateUploadLinkCalled   bool
	statCalled                 bool
	getCalled                  bool
	saveCalled                 bool
	removeCalled               bool
	copyCalled                 bool
	fileExistsCalled           bool
}

func (m *mockStorage) InitBucket(bucket string) error {
	m.initBucketCalled = true
	return m.initBucketErr
}
func (m *mockStorage) GeneratePresignedDownloadURL(ctx context.Context, bucket, fileKey string, expiry time.Duration) (string, error) {
	m.generateDownloadLinkCalled = true
	m.objectKey = fileKey
	m.ttl = expiry
	if m.generateDownloadLinkErr != nil {
		return "", m.generateDownloadLinkErr
	}
	return "https://example.com/upload", nil
}
func (m *mockStorage) GeneratePresignedUploadURL(ctx context.Context, bucket, fileKey string, expiry time.Duration) (string, error) {
	m.generateUploadLinkCalled = true
	m.objectKey = fileKey
	m.ttl = expiry
	if m.generateUploadLinkErr != nil {
		return "", m.generateUploadLinkErr
	}
	return "https://example.com/upload", nil
}
func (m *mockStorage) StatFile(ctx context.Context, bucket, fileKey string) (FileInfo, error) {
	m.statCalled = true
	if m.statErr != nil {
		return FileInfo{}, m.statErr
	}
	return m.statInfo, nil
}
func (m *mockStorage) RemoveFile(ctx context.Context, bucket, fileKey string) error {
	m.removeCalled = true
	return nil
}
func (m *mockStorage) GetFile(ctx context.Context, bucket, fileKey string) (io.ReadSeekCloser, error) {
	m.getCalled = true
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.reader != nil {
		return nopRSC{m.reader}, nil
	}
	return nopRSC{bytes.NewReader([]byte("dummy"))}, nil
}
func (m *mockStorage) SaveFile(ctx context.Context, bucket, fileKey string, reader io.Reader, fileSize int64, opts map[string]string) error {
	m.saveCalled = true
	return m.saveErr
}
func (m *mockStorage) CopyFile(ctx context.Context, bucket, srcKey, destKey string) error {
	m.copyCalled = true
	return m.copyErr
}
func (m *mockStorage) FileExists(ctx context.Context, bucket, fileKey string) (bool, error) {
	m.fileExistsCalled = true
	if m.fileExistsErr != nil {
		return false, m.fileExistsErr
	}
	return m.fileExists, nil
}

type mockCache struct {
	out *GetMediaOutput

	getMediaErr error
	delMediaErr error

	getMediaCalled bool
	setMediaCalled bool
	delMediaCalled bool
}

func (c *mockCache) GetMediaDetails(ctx context.Context, id db.UUID) (*GetMediaOutput, error) {
	c.getMediaCalled = true
	if c.getMediaErr != nil {
		return nil, c.getMediaErr
	}
	return c.out, nil
}

func (c *mockCache) SetMediaDetails(ctx context.Context, id db.UUID, value *GetMediaOutput) {
	c.setMediaCalled = true
	c.out = value
}

func (c *mockCache) DeleteMediaDetails(ctx context.Context, id db.UUID) error {
	c.delMediaCalled = true
	return c.delMediaErr
}

type mockFileOptimiser struct {
	compressOut []byte
	resizeOut   []byte
	mimeOut     string

	compressErr error
	resizeErr   error

	resizeCalled bool
}

func (m *mockFileOptimiser) Compress(mimeType string, r io.Reader) (io.ReadCloser, string, error) {
	if m.compressErr != nil {
		return nil, "", m.compressErr
	}
	return io.NopCloser(bytes.NewReader(m.compressOut)), m.mimeOut, nil
}

func (m *mockFileOptimiser) Resize(mimeType string, r io.Reader, width, height int) (io.ReadCloser, error) {
	m.resizeCalled = true
	if m.resizeErr != nil {
		return nil, m.resizeErr
	}
	return io.NopCloser(bytes.NewReader(m.resizeOut)), nil
}

type mockDispatcher struct {
	optimiseCalled bool
	id             db.UUID
	optimiseErr    error
}

func (m *mockDispatcher) EnqueueOptimiseMedia(ctx context.Context, id db.UUID) error {
	m.optimiseCalled = true
	m.id = id
	return m.optimiseErr
}
