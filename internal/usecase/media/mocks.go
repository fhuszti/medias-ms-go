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

type mockStorage struct {
	reader     io.Reader
	statInfo   FileInfo
	objectKey  string
	ttl        time.Duration
	fileExists bool

	generateDownloadLinkError error
	generateUploadLinkError   error
	statErr                   error
	getErr                    error
	saveErr                   error
	copyErr                   error
	fileExistsErr             error

	generateDownloadLinkCalled bool
	generateUploadLinkCalled   bool
	statCalled                 bool
	getCalled                  bool
	saveCalled                 bool
	removeCalled               bool
	copyCalled                 bool
	fileExistsCalled           bool
}

func (m *mockStorage) FileExists(ctx context.Context, fileKey string) (bool, error) {
	m.fileExistsCalled = true
	if m.fileExistsErr != nil {
		return false, m.fileExistsErr
	}
	return m.fileExists, nil
}
func (m *mockStorage) GeneratePresignedDownloadURL(ctx context.Context, fileKey string, expiry time.Duration) (string, error) {
	m.generateDownloadLinkCalled = true
	m.objectKey = fileKey
	m.ttl = expiry
	if m.generateDownloadLinkError != nil {
		return "", m.generateDownloadLinkError
	}
	return "https://example.com/upload", nil
}
func (m *mockStorage) GeneratePresignedUploadURL(ctx context.Context, fileKey string, expiry time.Duration) (string, error) {
	m.generateUploadLinkCalled = true
	m.objectKey = fileKey
	m.ttl = expiry
	if m.generateUploadLinkError != nil {
		return "", m.generateUploadLinkError
	}
	return "https://example.com/upload", nil
}
func (m *mockStorage) StatFile(ctx context.Context, fileKey string) (FileInfo, error) {
	m.statCalled = true
	if m.statErr != nil {
		return FileInfo{}, m.statErr
	}
	return m.statInfo, nil
}
func (m *mockStorage) RemoveFile(ctx context.Context, fileKey string) error {
	m.removeCalled = true
	return nil
}
func (m *mockStorage) GetFile(ctx context.Context, fileKey string) (io.ReadCloser, error) {
	m.getCalled = true
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.reader != nil {
		return io.NopCloser(m.reader), nil
	}
	return io.NopCloser(bytes.NewReader([]byte("dummy"))), nil
}
func (m *mockStorage) SaveFile(ctx context.Context, fileKey string, reader io.Reader, fileSize int64, opts map[string]string) error {
	m.saveCalled = true
	return m.saveErr
}
func (m *mockStorage) CopyFile(ctx context.Context, srcKey, destKey string) error {
	m.copyCalled = true
	return m.copyErr
}

type mockStorageGetter struct {
	strg *mockStorage
	err  error
}

func (m *mockStorageGetter) Get(bucket string) (Storage, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.strg, nil
}

type mockCache struct {
	out *GetMediaOutput

	getMediaErr error
	setMediaErr error
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

func (c *mockCache) SetMediaDetails(ctx context.Context, id db.UUID, value *GetMediaOutput) error {
	c.setMediaCalled = true
	c.out = value
	return c.setMediaErr
}

func (c *mockCache) DeleteMediaDetails(ctx context.Context, id db.UUID) error {
	c.delMediaCalled = true
	return c.delMediaErr
}
