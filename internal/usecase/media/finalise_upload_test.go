package media

import (
	"bytes"
	"context"
	"errors"
	"github.com/fhuszti/medias-ms-go/internal/mock"
	"github.com/google/uuid"
	"image"
	"image/png"
	"io"
	"strings"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/port"
)

func TestFinaliseUpload_ErrGetByID(t *testing.T) {
	repo := &mock.MockMediaRepo{GetErr: errors.New("db fail")}
	svc := NewUploadFinaliser(repo, &mock.MockStorage{}, &mock.MockDispatcher{})

	err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || err.Error() != "db fail" {
		t.Errorf("expected getByID error, got %v", err)
	}
}

func TestFinaliseUpload_AlreadyCompleted(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusCompleted}
	repo := &mock.MockMediaRepo{MediaRecord: mrec}
	svc := NewUploadFinaliser(repo, &mock.MockStorage{}, &mock.MockDispatcher{})

	if err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFinaliseUpload_WrongStatus(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusFailed}
	repo := &mock.MockMediaRepo{MediaRecord: mrec}
	svc := NewUploadFinaliser(repo, &mock.MockStorage{}, &mock.MockDispatcher{})

	err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "media status should be 'pending'") {
		t.Errorf("expected status error, got %v", err)
	}
}

func TestFinaliseUpload_StatNotFound(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusPending, ObjectKey: "k"}
	stg := &mock.MockStorage{StatErr: ErrObjectNotFound}
	repo := &mock.MockMediaRepo{MediaRecord: mrec}
	svc := NewUploadFinaliser(repo, stg, &mock.MockDispatcher{})

	err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "staging file \"k\" not found") {
		t.Errorf("expected not found error, got %v", err)
	}
	if !stg.RemoveCalled {
		t.Error("expected cleanupFile to be called")
	}
	if repo.Updated == nil || repo.Updated.Status != model.MediaStatusFailed {
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
		stg := &mock.MockStorage{StatInfo: port.FileInfo{SizeBytes: tc.size, ContentType: "image/png"}}
		repo := &mock.MockMediaRepo{MediaRecord: mrec}
		svc := NewUploadFinaliser(repo, stg, &mock.MockDispatcher{})
		err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("size %d: expected error containing %q, got %v", tc.size, tc.wantErr, err)
		}
	}
}

func TestFinaliseUpload_UnsupportedMime(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusPending, ObjectKey: "k"}
	stg := &mock.MockStorage{StatInfo: port.FileInfo{SizeBytes: MinFileSize, ContentType: "application/zip"}}
	repo := &mock.MockMediaRepo{MediaRecord: mrec}
	svc := NewUploadFinaliser(repo, stg, &mock.MockDispatcher{})

	err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "unsupported mime-type") {
		t.Errorf("expected unsupported mime-type error, got %v", err)
	}
}

func TestFinaliseUpload_MoveGetFileError(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusPending, ObjectKey: "k"}
	stg := &mock.MockStorage{StatInfo: port.FileInfo{SizeBytes: MinFileSize, ContentType: "image/png"}, GetErr: errors.New("can't read file")}
	repo := &mock.MockMediaRepo{MediaRecord: mrec}
	svc := NewUploadFinaliser(repo, stg, &mock.MockDispatcher{})

	err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "can't read file") {
		t.Errorf("expected getfile error, got %v", err)
	}
}

func TestFinaliseUpload_MoveExtensionError(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusPending, ObjectKey: "k"}
	stg := &mock.MockStorage{StatInfo: port.FileInfo{SizeBytes: MinFileSize, ContentType: "application/unknown"}}
	repo := &mock.MockMediaRepo{MediaRecord: mrec}
	svc := NewUploadFinaliser(repo, stg, &mock.MockDispatcher{})

	err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "unsupported mime-type") {
		t.Errorf("expected extension error, got %v", err)
	}
}

func TestFinaliseUpload_MoveMetadataError(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusPending, ObjectKey: "k"}
	repo := &mock.MockMediaRepo{MediaRecord: mrec}
	stg := &mock.MockStorage{StatInfo: port.FileInfo{SizeBytes: MinFileSize, ContentType: "image/png"}, Reader: strings.NewReader("not-a-png")}
	svc := NewUploadFinaliser(repo, stg, &mock.MockDispatcher{})

	err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "error decoding") {
		t.Errorf("expected metadata error, got %v", err)
	}
}

func TestFinaliseUpload_MoveSaveFileError(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusPending, ObjectKey: "k"}
	repo := &mock.MockMediaRepo{MediaRecord: mrec}
	stg := &mock.MockStorage{SaveErr: errors.New("save fail"), StatInfo: port.FileInfo{SizeBytes: MinFileSize, ContentType: "image/png"}, Reader: getPNGReader(t)}
	svc := NewUploadFinaliser(repo, stg, &mock.MockDispatcher{})

	err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "save fail") {
		t.Errorf("expected savefile error, got %v", err)
	}
}

func TestFinaliseUpload_MoveUpdateMediaError(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusPending, ObjectKey: "k"}
	repo := &mock.MockMediaRepo{MediaRecord: mrec, UpdateErr: errors.New("update fail")}
	stg := &mock.MockStorage{StatInfo: port.FileInfo{SizeBytes: MinFileSize, ContentType: "image/png"}, Reader: getPNGReader(t)}
	svc := NewUploadFinaliser(repo, stg, &mock.MockDispatcher{})

	err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "update fail") {
		t.Errorf("expected repo update error, got %v", err)
	}
}

func TestFinaliseUpload_Success(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusPending, ObjectKey: "name"}
	repo := &mock.MockMediaRepo{MediaRecord: mrec}
	stg := &mock.MockStorage{StatInfo: port.FileInfo{SizeBytes: MinFileSize, ContentType: "image/png"}, Reader: getPNGReader(t)}
	dispatcher := &mock.MockDispatcher{}
	svc := NewUploadFinaliser(repo, stg, dispatcher)

	if err := svc.FinaliseUpload(context.Background(), FinaliseUploadInput{ID: db.UUID(uuid.Nil), DestBucket: "images"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mrec.Bucket != "images" {
		t.Errorf("bucket should be 'images', got %q", mrec.Bucket)
	}
	if mrec.Status != model.MediaStatusCompleted {
		t.Errorf("Status = %q; want Completed", mrec.Status)
	}
	if !stg.SaveCalled {
		t.Error("expected SaveFile on destination to be called")
	}
	if !stg.RemoveCalled {
		t.Error("expected RemoveFile on staging to be called")
	}
	if repo.Updated == nil || repo.Updated.Status != model.MediaStatusCompleted {
		t.Error("expected repo.Update to set status Completed")
	}
	if !dispatcher.OptimiseCalled {
		t.Error("expected optimise task to be enqueued")
	}
}

func getPNGReader(t *testing.T) io.ReadSeeker {
	// build a 1x1 PNG in memory
	buf := &bytes.Buffer{}
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	if err := png.Encode(buf, img); err != nil {
		t.Fatalf("failed to encode test PNG: %v", err)
	}

	return bytes.NewReader(buf.Bytes())
}
