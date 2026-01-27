package media

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"io"
	"strings"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/mock"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/port"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	"github.com/google/uuid"
)

func TestFinaliseUpdate_ErrGetByID(t *testing.T) {
	repo := &mock.MediaRepo{GetByIDErr: errors.New("db fail")}
	svc := NewUpdateFinaliser(repo, &mock.Storage{}, &mock.Dispatcher{})

	err := svc.FinaliseUpdate(context.Background(), port.FinaliseUpdateInput{ID: msuuid.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || err.Error() != "db fail" {
		t.Errorf("expected getByID error, got %v", err)
	}
}

func TestFinaliseUpdate_AlreadyCompleted_NoStaging(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusCompleted, ID: msuuid.UUID(uuid.Nil), Version: 1}
	repo := &mock.MediaRepo{MediaOut: mrec}
	stg := &mock.Storage{StatErr: ErrObjectNotFound}
	svc := NewUpdateFinaliser(repo, stg, &mock.Dispatcher{})

	err := svc.FinaliseUpdate(context.Background(), port.FinaliseUpdateInput{ID: msuuid.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "staging file") {
		t.Fatalf("expected staging file error, got %v", err)
	}
}

func TestFinaliseUpdate_WrongStatus(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusFailed, ID: msuuid.UUID(uuid.Nil), Version: 1}
	repo := &mock.MediaRepo{MediaOut: mrec}
	svc := NewUpdateFinaliser(repo, &mock.Storage{}, &mock.Dispatcher{})

	err := svc.FinaliseUpdate(context.Background(), port.FinaliseUpdateInput{ID: msuuid.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "media status should be 'completed'") {
		t.Errorf("expected status error, got %v", err)
	}
}

func TestFinaliseUpdate_StatNotFound(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusCompleted, ID: msuuid.UUID(uuid.Nil), Version: 1}
	stg := &mock.Storage{StatErr: ErrObjectNotFound}
	repo := &mock.MediaRepo{MediaOut: mrec}
	svc := NewUpdateFinaliser(repo, stg, &mock.Dispatcher{})

	err := svc.FinaliseUpdate(context.Background(), port.FinaliseUpdateInput{ID: msuuid.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "staging file") {
		t.Errorf("expected not found error, got %v", err)
	}
	if !stg.RemoveCalled {
		t.Error("expected cleanupStagingFile to be called")
	}
	if repo.GotUpdated == nil || repo.GotUpdated.Status != model.MediaStatusFailed {
		t.Error("expected markMediaFailed to update status to Failed")
	}
}

func TestFinaliseUpdate_SizeValidation(t *testing.T) {
	tests := []struct {
		size    int64
		wantErr string
	}{
		{MinFileSize - 1, "too small"},
		{MaxFileSize + 1, "too large"},
	}
	for _, tc := range tests {
		mrec := &model.Media{Status: model.MediaStatusCompleted, ID: msuuid.UUID(uuid.Nil), Version: 1}
		stg := &mock.Storage{StatInfoOut: port.FileInfo{SizeBytes: tc.size, ContentType: "image/png"}}
		repo := &mock.MediaRepo{MediaOut: mrec}
		svc := NewUpdateFinaliser(repo, stg, &mock.Dispatcher{})
		err := svc.FinaliseUpdate(context.Background(), port.FinaliseUpdateInput{ID: msuuid.UUID(uuid.Nil), DestBucket: "images"})
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("size %d: expected error containing %q, got %v", tc.size, tc.wantErr, err)
		}
	}
}

func TestFinaliseUpdate_UnsupportedMime(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusCompleted, ID: msuuid.UUID(uuid.Nil), Version: 1}
	stg := &mock.Storage{StatInfoOut: port.FileInfo{SizeBytes: MinFileSize, ContentType: "application/zip"}}
	repo := &mock.MediaRepo{MediaOut: mrec}
	svc := NewUpdateFinaliser(repo, stg, &mock.Dispatcher{})

	err := svc.FinaliseUpdate(context.Background(), port.FinaliseUpdateInput{ID: msuuid.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "unsupported mime-type") {
		t.Errorf("expected unsupported mime-type error, got %v", err)
	}
}

func TestFinaliseUpdate_MoveGetFileError(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusCompleted, ID: msuuid.UUID(uuid.Nil), Version: 1}
	stg := &mock.Storage{StatInfoOut: port.FileInfo{SizeBytes: MinFileSize, ContentType: "image/png"}, GetErr: errors.New("can't read file")}
	repo := &mock.MediaRepo{MediaOut: mrec}
	svc := NewUpdateFinaliser(repo, stg, &mock.Dispatcher{})

	err := svc.FinaliseUpdate(context.Background(), port.FinaliseUpdateInput{ID: msuuid.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "can't read file") {
		t.Errorf("expected getfile error, got %v", err)
	}
}

func TestFinaliseUpdate_MoveMetadataError(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusCompleted, ID: msuuid.UUID(uuid.Nil), Version: 1}
	repo := &mock.MediaRepo{MediaOut: mrec}
	stg := &mock.Storage{StatInfoOut: port.FileInfo{SizeBytes: MinFileSize, ContentType: "image/png"}, GetOut: strings.NewReader("not-a-png")}
	svc := NewUpdateFinaliser(repo, stg, &mock.Dispatcher{})

	err := svc.FinaliseUpdate(context.Background(), port.FinaliseUpdateInput{ID: msuuid.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "error decoding") {
		t.Errorf("expected metadata error, got %v", err)
	}
}

func TestFinaliseUpdate_MoveSaveFileError(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusCompleted, ID: msuuid.UUID(uuid.Nil), Version: 1}
	repo := &mock.MediaRepo{MediaOut: mrec}
	stg := &mock.Storage{SaveErr: errors.New("save fail"), StatInfoOut: port.FileInfo{SizeBytes: MinFileSize, ContentType: "image/png"}, GetOut: getUpdatePNGReader(t)}
	svc := NewUpdateFinaliser(repo, stg, &mock.Dispatcher{})

	err := svc.FinaliseUpdate(context.Background(), port.FinaliseUpdateInput{ID: msuuid.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "save fail") {
		t.Errorf("expected savefile error, got %v", err)
	}
}

func TestFinaliseUpdate_MoveUpdateMediaError(t *testing.T) {
	mrec := &model.Media{Status: model.MediaStatusCompleted, ID: msuuid.UUID(uuid.Nil), Version: 1}
	repo := &mock.MediaRepo{MediaOut: mrec, UpdateErr: errors.New("update fail")}
	stg := &mock.Storage{StatInfoOut: port.FileInfo{SizeBytes: MinFileSize, ContentType: "image/png"}, GetOut: getUpdatePNGReader(t)}
	svc := NewUpdateFinaliser(repo, stg, &mock.Dispatcher{})

	err := svc.FinaliseUpdate(context.Background(), port.FinaliseUpdateInput{ID: msuuid.UUID(uuid.Nil), DestBucket: "images"})
	if err == nil || !strings.Contains(err.Error(), "update fail") {
		t.Errorf("expected repo update error, got %v", err)
	}
}

func TestFinaliseUpdate_Success(t *testing.T) {
	mrec := &model.Media{
		Status:    model.MediaStatusCompleted,
		ID:        msuuid.UUID(uuid.Nil),
		Version:   1,
		ObjectKey: "old.webp",
		Bucket:    "images",
		Variants: model.Variants{
			{ObjectKey: "variants/x/foo_50.webp"},
			{ObjectKey: "variants/x/foo_300.webp"},
		},
	}
	repo := &mock.MediaRepo{MediaOut: mrec}
	stg := &mock.Storage{StatInfoOut: port.FileInfo{SizeBytes: MinFileSize, ContentType: "image/png"}, GetOut: getUpdatePNGReader(t)}
	dispatcher := &mock.Dispatcher{}
	svc := NewUpdateFinaliser(repo, stg, dispatcher)

	if err := svc.FinaliseUpdate(context.Background(), port.FinaliseUpdateInput{ID: msuuid.UUID(uuid.Nil), DestBucket: "images"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mrec.Bucket != "images" {
		t.Errorf("bucket should be 'images', got %q", mrec.Bucket)
	}
	if mrec.Status != model.MediaStatusCompleted {
		t.Errorf("Status = %q; want Completed", mrec.Status)
	}
	if mrec.Version != 2 {
		t.Errorf("Version = %d; want 2", mrec.Version)
	}
	if !stg.SaveCalled {
		t.Error("expected SaveFile on destination to be called")
	}
	if !stg.RemoveCalled {
		t.Error("expected RemoveFile on staging/previous to be called")
	}
	if repo.GotUpdated == nil || repo.GotUpdated.Status != model.MediaStatusCompleted {
		t.Error("expected repo.Update to set status Completed")
	}
	if !dispatcher.OptimiseCalled {
		t.Error("expected optimise task to be enqueued")
	}
	if mrec.Optimised {
		t.Error("expected Optimised to be false after update")
	}
	if len(mrec.Variants) != 0 {
		t.Errorf("expected Variants to be cleared, got %d", len(mrec.Variants))
	}
}

func getUpdatePNGReader(t *testing.T) io.ReadSeeker {
	buf := &bytes.Buffer{}
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	if err := png.Encode(buf, img); err != nil {
		t.Fatalf("failed to encode test PNG: %v", err)
	}

	return bytes.NewReader(buf.Bytes())
}
