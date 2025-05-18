package media

import (
	"bytes"
	"context"
	"errors"
	"github.com/google/uuid"
	"image"
	"image/png"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
)

type mockRepo struct {
	mediaRecord *model.Media
	getErr      error
	updateErr   error
	updated     *model.Media
}

func (m *mockRepo) GetByID(ctx context.Context, id db.UUID) (*model.Media, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.mediaRecord, nil
}
func (m *mockRepo) Update(ctx context.Context, media *model.Media) error {
	m.updated = media
	return m.updateErr
}
func (m *mockRepo) Create(ctx context.Context, media *model.Media) error { panic("not used") }

type mockStorage struct {
	statInfo     FileInfo
	statErr      error
	getErr       error
	saveErr      error
	getCalled    bool
	saveCalled   bool
	removeCalled bool
}

func (m *mockStorage) GeneratePresignedUploadURL(ctx context.Context, fileKey string, expiry time.Duration) (string, error) {
	panic("not used")
}
func (m *mockStorage) FileExists(ctx context.Context, fileKey string) (bool, error) {
	panic("not used")
}
func (m *mockStorage) StatFile(ctx context.Context, fileKey string) (FileInfo, error) {
	return m.statInfo, m.statErr
}
func (m *mockStorage) RemoveFile(ctx context.Context, fileKey string) error {
	m.removeCalled = true
	return nil
}
func (m *mockStorage) GetFile(ctx context.Context, fileKey string) (io.ReadCloser, error) {
	m.getCalled = true
	return io.NopCloser(bytes.NewReader([]byte("dummy"))), m.getErr
}
func (m *mockStorage) SaveFile(ctx context.Context, fileKey string, reader io.Reader, fileSize int64, opts map[string]string) error {
	m.saveCalled = true
	return m.saveErr
}

type mockStorageGetter struct {
	dest *mockStorage
	err  error
}

func (m *mockStorageGetter) Get(bucket string) (Storage, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.dest, nil
}

// pngStorage wraps mockStorage to return a fixed reader
type pngStorage struct {
	*mockStorage
	reader io.ReadCloser
}

func (p *pngStorage) GetFile(ctx context.Context, key string) (io.ReadCloser, error) {
	p.getCalled = true
	return p.reader, nil
}

func TestFinaliseUpload_ErrGetByID(t *testing.T) {
	repo := &mockRepo{getErr: errors.New("db fail")}
	svc := NewUploadFinaliser(repo, &mockStorage{}, (&mockStorageGetter{}).Get)

	_, err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || err.Error() != "db fail" {
		t.Errorf("expected getByID error, got %v", err)
	}
}

func TestFinaliseUpload_AlreadyCompleted(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusCompleted}
	repo := &mockRepo{mediaRecord: mrec}
	svc := NewUploadFinaliser(repo, &mockStorage{}, (&mockStorageGetter{}).Get)

	out, err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != mrec {
		t.Errorf("expected returned media unchanged, got %v", out)
	}
}

func TestFinaliseUpload_WrongStatus(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusFailed}
	repo := &mockRepo{mediaRecord: mrec}
	svc := NewUploadFinaliser(repo, &mockStorage{}, (&mockStorageGetter{}).Get)

	_, err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "media status should be 'pending'") {
		t.Errorf("expected status error, got %v", err)
	}
}

func TestFinaliseUpload_StatNotFound(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusPending, ObjectKey: "k"}
	stg := &mockStorage{statErr: ErrObjectNotFound}
	repo := &mockRepo{mediaRecord: mrec}
	svc := NewUploadFinaliser(repo, stg, (&mockStorageGetter{}).Get)

	_, err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "staging file \"k\" not found") {
		t.Errorf("expected not found error, got %v", err)
	}
	if !stg.removeCalled {
		t.Error("expected cleanupFile to be called")
	}
	if repo.updated == nil || repo.updated.Status != model.MediaStatusFailed {
		t.Error("expected markAsFailed to update status to Failed")
	}
}

func TestFinaliseUpload_SizeValidation(t *testing.T) {
	tests := []struct {
		size    int64
		wantErr string
	}{
		{MinFileSize - 1, "too small"},
		{MaxFileSize + 1, "too large"},
	}
	for _, tc := range tests {
		mrec := &model.Media{Status: model.MediaStatusPending, ObjectKey: "k"}
		stg := &mockStorage{statInfo: FileInfo{SizeBytes: tc.size, ContentType: "image/png"}}
		repo := &mockRepo{mediaRecord: mrec}
		svc := NewUploadFinaliser(repo, stg, (&mockStorageGetter{}).Get)
		_, err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("size %d: expected error containing %q, got %v", tc.size, tc.wantErr, err)
		}
	}
}

func TestFinaliseUpload_UnsupportedMime(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusPending, ObjectKey: "k"}
	stg := &mockStorage{statInfo: FileInfo{SizeBytes: MinFileSize, ContentType: "application/zip"}}
	repo := &mockRepo{mediaRecord: mrec}
	svc := NewUploadFinaliser(repo, stg, (&mockStorageGetter{}).Get)

	_, err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "unsupported mime-type") {
		t.Errorf("expected unsupported mime-type error, got %v", err)
	}
}

func TestFinaliseUpload_MoveBucketError(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusPending, ObjectKey: "k"}
	stg := &mockStorage{statInfo: FileInfo{SizeBytes: MinFileSize, ContentType: "image/png"}}
	destGetter := func(bucket string) (Storage, error) { return nil, errors.New("no bucket") }
	repo := &mockRepo{mediaRecord: mrec}
	svc := NewUploadFinaliser(repo, stg, destGetter)

	_, err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "unknown destination bucket") {
		t.Errorf("expected bucket error, got %v", err)
	}
}

func TestFinaliseUpload_MoveGetFileError(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusPending, ObjectKey: "k"}
	stg := &mockStorage{statInfo: FileInfo{SizeBytes: MinFileSize, ContentType: "image/png"}, getErr: errors.New("can't read file")}
	repo := &mockRepo{mediaRecord: mrec}
	svc := NewUploadFinaliser(repo, stg, (&mockStorageGetter{}).Get)

	_, err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "can't read file") {
		t.Errorf("expected getfile error, got %v", err)
	}
}

func TestFinaliseUpload_MoveExtensionError(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusPending, ObjectKey: "k"}
	stg := &mockStorage{statInfo: FileInfo{SizeBytes: MinFileSize, ContentType: "application/unknown"}}
	repo := &mockRepo{mediaRecord: mrec}
	svc := NewUploadFinaliser(repo, stg, (&mockStorageGetter{}).Get)

	_, err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "unsupported mime-type") {
		t.Errorf("expected extension error, got %v", err)
	}
}

func TestFinaliseUpload_MoveMetadataError(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusPending, ObjectKey: "k"}
	repo := &mockRepo{mediaRecord: mrec}
	reader := io.NopCloser(strings.NewReader("not-a-png"))
	stgBase := &mockStorage{statInfo: FileInfo{SizeBytes: MinFileSize, ContentType: "image/png"}}
	stg := &pngStorage{mockStorage: stgBase, reader: reader}
	svc := NewUploadFinaliser(repo, stg, (&mockStorageGetter{}).Get)

	_, err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "error decoding") {
		t.Errorf("expected metadata error, got %v", err)
	}
}

func TestFinaliseUpload_MoveSaveFileError(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusPending, ObjectKey: "k"}
	repo := &mockRepo{mediaRecord: mrec}
	reader := io.NopCloser(getPNGReader(t))
	stgBase := &mockStorage{statInfo: FileInfo{SizeBytes: MinFileSize, ContentType: "image/png"}}
	stg := &pngStorage{mockStorage: stgBase, reader: reader}
	dest := &mockStorage{saveErr: errors.New("save fail")}
	svc := NewUploadFinaliser(repo, stg, (&mockStorageGetter{dest: dest}).Get)

	_, err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "save fail") {
		t.Errorf("expected savefile error, got %v", err)
	}
}

func TestFinaliseUpload_MoveUpdateMediaError(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusPending, ObjectKey: "k"}
	repo := &mockRepo{mediaRecord: mrec, updateErr: errors.New("update fail")}
	reader := io.NopCloser(getPNGReader(t))
	stgBase := &mockStorage{statInfo: FileInfo{SizeBytes: MinFileSize, ContentType: "image/png"}}
	stg := &pngStorage{mockStorage: stgBase, reader: reader}
	dest := &mockStorage{}
	svc := NewUploadFinaliser(repo, stg, (&mockStorageGetter{dest: dest}).Get)

	_, err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "update fail") {
		t.Errorf("expected repo update error, got %v", err)
	}
}

func TestFinaliseUpload_Success(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusPending, ObjectKey: "name"}
	repo := &mockRepo{mediaRecord: mrec}
	reader := io.NopCloser(getPNGReader(t))
	stgBase := &mockStorage{statInfo: FileInfo{SizeBytes: MinFileSize, ContentType: "image/png"}}
	stg := &pngStorage{mockStorage: stgBase, reader: reader}
	dest := &mockStorage{}
	svc := NewUploadFinaliser(repo, stg, (&mockStorageGetter{dest: dest}).Get)

	out, err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Status != model.MediaStatusCompleted {
		t.Errorf("Status = %q; want Completed", out.Status)
	}
	if !dest.saveCalled {
		t.Error("expected SaveFile on destination to be called")
	}
	if !stg.removeCalled {
		t.Error("expected RemoveFile on staging to be called")
	}
	if repo.updated == nil || repo.updated.Status != model.MediaStatusCompleted {
		t.Error("expected repo.Update to set status Completed")
	}
}

func getPNGReader(t *testing.T) io.Reader {
	// build a 1x1 PNG in memory
	buf := &bytes.Buffer{}
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	if err := png.Encode(buf, img); err != nil {
		t.Fatalf("failed to encode test PNG: %v", err)
	}

	return bytes.NewReader(buf.Bytes())
}
