package media

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/mock"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/port"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	"github.com/google/uuid"
)

func newCompletedMedia() *model.Media {
	mt := "image/png"
	size := int64(123)
	return &model.Media{
		ID:        msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")),
		ObjectKey: "foo.png",
		Bucket:    "images",
		MimeType:  &mt,
		SizeBytes: &size,
		Status:    model.MediaStatusCompleted,
	}
}

func TestOptimiseMedia_GetByIDNotFound(t *testing.T) {
	repo := &mock.MockMediaRepo{GetErr: sql.ErrNoRows}
	strg := &mock.MockStorage{}
	svc := NewMediaOptimiser(repo, &mock.FileOptimiser{}, strg, &mock.MockDispatcher{}, &mock.Cache{})

	err := svc.OptimiseMedia(context.Background(), msuuid.NewUUID())
	if !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound, got %v", err)
	}
}

func TestOptimiseMedia_GetByIDError(t *testing.T) {
	repo := &mock.MockMediaRepo{GetErr: errors.New("db fail")}
	strg := &mock.MockStorage{}
	svc := NewMediaOptimiser(repo, &mock.FileOptimiser{}, strg, &mock.MockDispatcher{}, &mock.Cache{})

	err := svc.OptimiseMedia(context.Background(), msuuid.NewUUID())
	if err == nil || err.Error() != "db fail" {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestOptimiseMedia_WrongStatus(t *testing.T) {
	m := newCompletedMedia()
	m.Status = model.MediaStatusPending
	repo := &mock.MockMediaRepo{MediaRecord: m}
	strg := &mock.MockStorage{}
	svc := NewMediaOptimiser(repo, &mock.FileOptimiser{}, strg, &mock.MockDispatcher{}, &mock.Cache{})

	err := svc.OptimiseMedia(context.Background(), m.ID)
	if err == nil || !strings.Contains(err.Error(), "completed") {
		t.Fatalf("expected status error, got %v", err)
	}
}

func TestOptimiseMedia_GetFileError(t *testing.T) {
	m := newCompletedMedia()
	repo := &mock.MockMediaRepo{MediaRecord: m}
	strg := &mock.MockStorage{GetErr: errors.New("get fail")}
	svc := NewMediaOptimiser(repo, &mock.FileOptimiser{}, strg, &mock.MockDispatcher{}, &mock.Cache{})

	err := svc.OptimiseMedia(context.Background(), m.ID)
	if err == nil || err.Error() != "get fail" {
		t.Fatalf("expected get error, got %v", err)
	}
}

func TestOptimiseMedia_CompressError(t *testing.T) {
	m := newCompletedMedia()
	repo := &mock.MockMediaRepo{MediaRecord: m}
	strg := &mock.MockStorage{}
	fo := &mock.FileOptimiser{CompressErr: errors.New("compress fail")}
	svc := NewMediaOptimiser(repo, fo, strg, &mock.MockDispatcher{}, &mock.Cache{})

	err := svc.OptimiseMedia(context.Background(), m.ID)
	if err == nil || err.Error() != "compress fail" {
		t.Fatalf("expected compress error, got %v", err)
	}
}

func TestOptimiseMedia_ExtensionError(t *testing.T) {
	m := newCompletedMedia()
	repo := &mock.MockMediaRepo{MediaRecord: m}
	strg := &mock.MockStorage{}
	fo := &mock.FileOptimiser{MimeOut: "application/unknown"}
	svc := NewMediaOptimiser(repo, fo, strg, &mock.MockDispatcher{}, &mock.Cache{})

	err := svc.OptimiseMedia(context.Background(), m.ID)
	if err == nil || !strings.Contains(err.Error(), "unsupported mime type") {
		t.Fatalf("expected mime type error, got %v", err)
	}
}

func TestOptimiseMedia_SaveFileError(t *testing.T) {
	m := newCompletedMedia()
	repo := &mock.MockMediaRepo{MediaRecord: m}
	strg := &mock.MockStorage{SaveErr: errors.New("save fail")}
	fo := &mock.FileOptimiser{MimeOut: *m.MimeType}
	svc := NewMediaOptimiser(repo, fo, strg, &mock.MockDispatcher{}, &mock.Cache{})

	err := svc.OptimiseMedia(context.Background(), m.ID)
	if err == nil || !strings.Contains(err.Error(), "save fail") {
		t.Fatalf("expected save error, got %v", err)
	}
}

func TestOptimiseMedia_CopyFileError(t *testing.T) {
	m := newCompletedMedia()
	repo := &mock.MockMediaRepo{MediaRecord: m}
	strg := &mock.MockStorage{CopyErr: errors.New("copy fail")}
	fo := &mock.FileOptimiser{MimeOut: *m.MimeType}
	svc := NewMediaOptimiser(repo, fo, strg, &mock.MockDispatcher{}, &mock.Cache{})

	err := svc.OptimiseMedia(context.Background(), m.ID)
	if err == nil || !strings.Contains(err.Error(), "copy fail") {
		t.Fatalf("expected copy error, got %v", err)
	}
}

func TestOptimiseMedia_StatError(t *testing.T) {
	m := newCompletedMedia()
	repo := &mock.MockMediaRepo{MediaRecord: m}
	strg := &mock.MockStorage{StatErr: errors.New("stat fail")}
	fo := &mock.FileOptimiser{MimeOut: *m.MimeType}
	svc := NewMediaOptimiser(repo, fo, strg, &mock.MockDispatcher{}, &mock.Cache{})

	err := svc.OptimiseMedia(context.Background(), m.ID)
	if err == nil || !strings.Contains(err.Error(), "stat fail") {
		t.Fatalf("expected stat error, got %v", err)
	}
}

func TestOptimiseMedia_UpdateError(t *testing.T) {
	m := newCompletedMedia()
	repo := &mock.MockMediaRepo{MediaRecord: m, UpdateErr: errors.New("update fail")}
	strg := &mock.MockStorage{}
	strg.StatInfo = port.FileInfo{SizeBytes: 200}
	fo := &mock.FileOptimiser{MimeOut: *m.MimeType}
	svc := NewMediaOptimiser(repo, fo, strg, &mock.MockDispatcher{}, &mock.Cache{})

	err := svc.OptimiseMedia(context.Background(), m.ID)
	if err == nil || !strings.Contains(err.Error(), "update fail") {
		t.Fatalf("expected update error, got %v", err)
	}
}

func TestOptimiseMedia_SuccessSameMime(t *testing.T) {
	m := newCompletedMedia()
	repo := &mock.MockMediaRepo{MediaRecord: m}
	strg := &mock.MockStorage{}
	strg.StatInfo = port.FileInfo{SizeBytes: 456}
	fo := &mock.FileOptimiser{MimeOut: *m.MimeType, CompressOut: []byte("comp")}
	dispatcher := &mock.MockDispatcher{}
	svc := NewMediaOptimiser(repo, fo, strg, dispatcher, &mock.Cache{})

	err := svc.OptimiseMedia(context.Background(), m.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !repo.Updated.Optimised {
		t.Error("media should be marked optimised")
	}
	if repo.Updated.SizeBytes == nil || *repo.Updated.SizeBytes != strg.StatInfo.SizeBytes {
		t.Error("size not updated")
	}
	if repo.Updated.ObjectKey != m.ObjectKey {
		t.Errorf("object key changed: %q", repo.Updated.ObjectKey)
	}
	if !strg.SaveCalled || !strg.CopyCalled || !strg.RemoveCalled || !strg.GetCalled || !strg.StatCalled {
		t.Error("storage methods not fully called")
	}
	if !dispatcher.ResizeCalled || len(dispatcher.ResizeIDs) != 1 || dispatcher.ResizeIDs[0] != m.ID {
		t.Error("resize task not enqueued")
	}
}

func TestOptimiseMedia_SuccessMimeChange(t *testing.T) {
	m := newCompletedMedia()
	repo := &mock.MockMediaRepo{MediaRecord: m}
	strg := &mock.MockStorage{}
	strg.StatInfo = port.FileInfo{SizeBytes: 789}
	fo := &mock.FileOptimiser{MimeOut: "image/webp", CompressOut: []byte("webp")}
	dispatcher := &mock.MockDispatcher{}
	svc := NewMediaOptimiser(repo, fo, strg, dispatcher, &mock.Cache{})

	err := svc.OptimiseMedia(context.Background(), m.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.Updated.ObjectKey != "foo.webp" {
		t.Errorf("expected new object key foo.webp, got %s", repo.Updated.ObjectKey)
	}
	if repo.Updated.MimeType == nil || *repo.Updated.MimeType != "image/webp" {
		t.Errorf("mime type not updated")
	}
	if !strg.SaveCalled || !strg.CopyCalled {
		t.Error("expected save and copy calls")
	}
	if !dispatcher.ResizeCalled || len(dispatcher.ResizeIDs) != 1 || dispatcher.ResizeIDs[0] != m.ID {
		t.Error("resize task not enqueued")
	}
}
