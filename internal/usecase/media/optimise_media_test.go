package media

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/google/uuid"
)

func newCompletedMedia() *model.Media {
	mt := "image/png"
	size := int64(123)
	return &model.Media{
		ID:        db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")),
		ObjectKey: "foo.png",
		Bucket:    "images",
		MimeType:  &mt,
		SizeBytes: &size,
		Status:    model.MediaStatusCompleted,
	}
}

func TestOptimiseMedia_GetByIDNotFound(t *testing.T) {
	repo := &mockRepo{getErr: sql.ErrNoRows}
	svc := NewMediaOptimiser(repo, &mockFileOptimiser{}, (&mockStorageGetter{strg: &mockStorage{}}).Get)

	err := svc.OptimiseMedia(context.Background(), OptimiseMediaInput{ID: db.NewUUID()})
	if !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound, got %v", err)
	}
}

func TestOptimiseMedia_GetByIDError(t *testing.T) {
	repo := &mockRepo{getErr: errors.New("db fail")}
	svc := NewMediaOptimiser(repo, &mockFileOptimiser{}, (&mockStorageGetter{strg: &mockStorage{}}).Get)

	err := svc.OptimiseMedia(context.Background(), OptimiseMediaInput{ID: db.NewUUID()})
	if err == nil || err.Error() != "db fail" {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestOptimiseMedia_WrongStatus(t *testing.T) {
	m := newCompletedMedia()
	m.Status = model.MediaStatusPending
	repo := &mockRepo{mediaRecord: m}
	svc := NewMediaOptimiser(repo, &mockFileOptimiser{}, (&mockStorageGetter{strg: &mockStorage{}}).Get)

	err := svc.OptimiseMedia(context.Background(), OptimiseMediaInput{ID: m.ID})
	if err == nil || !strings.Contains(err.Error(), "completed") {
		t.Fatalf("expected status error, got %v", err)
	}
}

func TestOptimiseMedia_GetTargetError(t *testing.T) {
	m := newCompletedMedia()
	repo := &mockRepo{mediaRecord: m}
	svc := NewMediaOptimiser(repo, &mockFileOptimiser{}, (&mockStorageGetter{err: errors.New("no bucket")}).Get)

	err := svc.OptimiseMedia(context.Background(), OptimiseMediaInput{ID: m.ID})
	if err == nil || err.Error() != "no bucket" {
		t.Fatalf("expected bucket error, got %v", err)
	}
}

func TestOptimiseMedia_GetFileError(t *testing.T) {
	m := newCompletedMedia()
	repo := &mockRepo{mediaRecord: m}
	strg := &mockStorage{getErr: errors.New("get fail")}
	svc := NewMediaOptimiser(repo, &mockFileOptimiser{}, (&mockStorageGetter{strg: strg}).Get)

	err := svc.OptimiseMedia(context.Background(), OptimiseMediaInput{ID: m.ID})
	if err == nil || err.Error() != "get fail" {
		t.Fatalf("expected get error, got %v", err)
	}
}

func TestOptimiseMedia_CompressError(t *testing.T) {
	m := newCompletedMedia()
	repo := &mockRepo{mediaRecord: m}
	strg := &mockStorage{}
	fo := &mockFileOptimiser{compressErr: errors.New("compress fail")}
	svc := NewMediaOptimiser(repo, fo, (&mockStorageGetter{strg: strg}).Get)

	err := svc.OptimiseMedia(context.Background(), OptimiseMediaInput{ID: m.ID})
	if err == nil || err.Error() != "compress fail" {
		t.Fatalf("expected compress error, got %v", err)
	}
}

func TestOptimiseMedia_ExtensionError(t *testing.T) {
	m := newCompletedMedia()
	repo := &mockRepo{mediaRecord: m}
	strg := &mockStorage{}
	fo := &mockFileOptimiser{mimeOut: "application/unknown"}
	svc := NewMediaOptimiser(repo, fo, (&mockStorageGetter{strg: strg}).Get)

	err := svc.OptimiseMedia(context.Background(), OptimiseMediaInput{ID: m.ID})
	if err == nil || !strings.Contains(err.Error(), "unsupported mime type") {
		t.Fatalf("expected mime type error, got %v", err)
	}
}

func TestOptimiseMedia_SaveFileError(t *testing.T) {
	m := newCompletedMedia()
	repo := &mockRepo{mediaRecord: m}
	strg := &mockStorage{saveErr: errors.New("save fail")}
	fo := &mockFileOptimiser{mimeOut: *m.MimeType}
	svc := NewMediaOptimiser(repo, fo, (&mockStorageGetter{strg: strg}).Get)

	err := svc.OptimiseMedia(context.Background(), OptimiseMediaInput{ID: m.ID})
	if err == nil || !strings.Contains(err.Error(), "save fail") {
		t.Fatalf("expected save error, got %v", err)
	}
}

func TestOptimiseMedia_CopyFileError(t *testing.T) {
	m := newCompletedMedia()
	repo := &mockRepo{mediaRecord: m}
	strg := &mockStorage{copyErr: errors.New("copy fail")}
	fo := &mockFileOptimiser{mimeOut: *m.MimeType}
	svc := NewMediaOptimiser(repo, fo, (&mockStorageGetter{strg: strg}).Get)

	err := svc.OptimiseMedia(context.Background(), OptimiseMediaInput{ID: m.ID})
	if err == nil || !strings.Contains(err.Error(), "copy fail") {
		t.Fatalf("expected copy error, got %v", err)
	}
}

func TestOptimiseMedia_StatError(t *testing.T) {
	m := newCompletedMedia()
	repo := &mockRepo{mediaRecord: m}
	strg := &mockStorage{statErr: errors.New("stat fail")}
	fo := &mockFileOptimiser{mimeOut: *m.MimeType}
	svc := NewMediaOptimiser(repo, fo, (&mockStorageGetter{strg: strg}).Get)

	err := svc.OptimiseMedia(context.Background(), OptimiseMediaInput{ID: m.ID})
	if err == nil || !strings.Contains(err.Error(), "stat fail") {
		t.Fatalf("expected stat error, got %v", err)
	}
}

func TestOptimiseMedia_UpdateError(t *testing.T) {
	m := newCompletedMedia()
	repo := &mockRepo{mediaRecord: m, updateErr: errors.New("update fail")}
	strg := &mockStorage{}
	strg.statInfo = FileInfo{SizeBytes: 200}
	fo := &mockFileOptimiser{mimeOut: *m.MimeType}
	svc := NewMediaOptimiser(repo, fo, (&mockStorageGetter{strg: strg}).Get)

	err := svc.OptimiseMedia(context.Background(), OptimiseMediaInput{ID: m.ID})
	if err == nil || !strings.Contains(err.Error(), "update fail") {
		t.Fatalf("expected update error, got %v", err)
	}
}

func TestOptimiseMedia_SuccessSameMime(t *testing.T) {
	m := newCompletedMedia()
	repo := &mockRepo{mediaRecord: m}
	strg := &mockStorage{}
	strg.statInfo = FileInfo{SizeBytes: 456}
	fo := &mockFileOptimiser{mimeOut: *m.MimeType, out: []byte("comp")}
	svc := NewMediaOptimiser(repo, fo, (&mockStorageGetter{strg: strg}).Get)

	err := svc.OptimiseMedia(context.Background(), OptimiseMediaInput{ID: m.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !repo.updated.Optimised {
		t.Error("media should be marked optimised")
	}
	if repo.updated.SizeBytes == nil || *repo.updated.SizeBytes != strg.statInfo.SizeBytes {
		t.Error("size not updated")
	}
	if repo.updated.ObjectKey != m.ObjectKey {
		t.Errorf("object key changed: %q", repo.updated.ObjectKey)
	}
	if !strg.saveCalled || !strg.copyCalled || !strg.removeCalled || !strg.getCalled || !strg.statCalled {
		t.Error("storage methods not fully called")
	}
}

func TestOptimiseMedia_SuccessMimeChange(t *testing.T) {
	m := newCompletedMedia()
	repo := &mockRepo{mediaRecord: m}
	strg := &mockStorage{}
	strg.statInfo = FileInfo{SizeBytes: 789}
	fo := &mockFileOptimiser{mimeOut: "image/webp", out: []byte("webp")}
	svc := NewMediaOptimiser(repo, fo, (&mockStorageGetter{strg: strg}).Get)

	err := svc.OptimiseMedia(context.Background(), OptimiseMediaInput{ID: m.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updated.ObjectKey != "foo.webp" {
		t.Errorf("expected new object key foo.webp, got %s", repo.updated.ObjectKey)
	}
	if repo.updated.MimeType == nil || *repo.updated.MimeType != "image/webp" {
		t.Errorf("mime type not updated")
	}
	if !strg.saveCalled || !strg.copyCalled {
		t.Error("expected save and copy calls")
	}
}
