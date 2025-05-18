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

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
)

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
	stg := &mockStorage{statInfo: FileInfo{SizeBytes: MinFileSize, ContentType: "image/png"}, reader: strings.NewReader("not-a-png")}
	svc := NewUploadFinaliser(repo, stg, (&mockStorageGetter{}).Get)

	_, err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "error decoding") {
		t.Errorf("expected metadata error, got %v", err)
	}
}

func TestFinaliseUpload_MoveSaveFileError(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusPending, ObjectKey: "k"}
	repo := &mockRepo{mediaRecord: mrec}
	stg := &mockStorage{statInfo: FileInfo{SizeBytes: MinFileSize, ContentType: "image/png"}, reader: getPNGReader(t)}
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
	stg := &mockStorage{statInfo: FileInfo{SizeBytes: MinFileSize, ContentType: "image/png"}, reader: getPNGReader(t)}
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
	stg := &mockStorage{statInfo: FileInfo{SizeBytes: MinFileSize, ContentType: "image/png"}, reader: getPNGReader(t)}
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
