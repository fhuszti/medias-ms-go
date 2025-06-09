package media

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/google/uuid"
)

// errSeekReader forces Seek to fail
type errSeekReader struct{ io.Reader }

func (errSeekReader) Seek(int64, int) (int64, error) { return 0, errors.New("seek fail") }
func (errSeekReader) Close() error                   { return nil }

func TestResizeImage_GetByIDNotFound(t *testing.T) {
	repo := &mockRepo{getErr: sql.ErrNoRows}
	svc := NewImageResizer(repo, &mockFileOptimiser{}, &mockStorage{})

	err := svc.ResizeImage(context.Background(), ResizeImageInput{ID: db.NewUUID()})
	if !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound, got %v", err)
	}
}

func TestResizeImage_GetByIDError(t *testing.T) {
	repo := &mockRepo{getErr: errors.New("db fail")}
	svc := NewImageResizer(repo, &mockFileOptimiser{}, &mockStorage{})

	err := svc.ResizeImage(context.Background(), ResizeImageInput{ID: db.NewUUID()})
	if err == nil || err.Error() != "db fail" {
		t.Fatalf("expected db fail, got %v", err)
	}
}

func TestResizeImage_WrongStatus(t *testing.T) {
	mt := "image/png"
	m := &model.Media{Status: model.MediaStatusPending, MimeType: &mt}
	repo := &mockRepo{mediaRecord: m}
	svc := NewImageResizer(repo, &mockFileOptimiser{}, &mockStorage{})

	err := svc.ResizeImage(context.Background(), ResizeImageInput{ID: db.NewUUID()})
	want := "media status should be 'completed' to be resized"
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

func TestResizeImage_NotImage(t *testing.T) {
	mt := "application/pdf"
	m := &model.Media{Status: model.MediaStatusCompleted, MimeType: &mt}
	repo := &mockRepo{mediaRecord: m}
	svc := NewImageResizer(repo, &mockFileOptimiser{}, &mockStorage{})

	err := svc.ResizeImage(context.Background(), ResizeImageInput{ID: db.NewUUID()})
	if err == nil || err.Error() != "media is not an image" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResizeImage_GetFileError(t *testing.T) {
	mt := "image/png"
	m := &model.Media{Status: model.MediaStatusCompleted, MimeType: &mt}
	repo := &mockRepo{mediaRecord: m}
	stg := &mockStorage{getErr: errors.New("get fail")}
	svc := NewImageResizer(repo, &mockFileOptimiser{}, stg)

	err := svc.ResizeImage(context.Background(), ResizeImageInput{ID: db.NewUUID()})
	if err == nil || err.Error() != "get fail" {
		t.Fatalf("expected get fail, got %v", err)
	}
}

func TestResizeImage_SeekError(t *testing.T) {
	mt := "image/png"
	m := &model.Media{Status: model.MediaStatusCompleted, MimeType: &mt}
	repo := &mockRepo{mediaRecord: m}
	stg := &mockStorage{reader: errSeekReader{bytes.NewReader([]byte("a"))}}
	svc := NewImageResizer(repo, &mockFileOptimiser{}, stg)

	err := svc.ResizeImage(context.Background(), ResizeImageInput{ID: db.NewUUID(), Sizes: []int{10}})
	if err == nil || !strings.Contains(err.Error(), "seek fail") {
		t.Fatalf("expected seek fail, got %v", err)
	}
}

func TestResizeImage_ResizeError(t *testing.T) {
	mt := "image/png"
	m := &model.Media{Status: model.MediaStatusCompleted, MimeType: &mt}
	repo := &mockRepo{mediaRecord: m}
	stg := &mockStorage{reader: bytes.NewReader([]byte("a"))}
	fo := &mockFileOptimiser{resizeErr: errors.New("resize fail")}
	svc := NewImageResizer(repo, fo, stg)

	err := svc.ResizeImage(context.Background(), ResizeImageInput{ID: db.NewUUID(), Sizes: []int{10}})
	if err == nil || err.Error() != "resize fail" {
		t.Fatalf("expected resize fail, got %v", err)
	}
}

func TestResizeImage_SaveFileError(t *testing.T) {
	mt := "image/png"
	m := &model.Media{Status: model.MediaStatusCompleted, MimeType: &mt}
	repo := &mockRepo{mediaRecord: m}
	stg := &mockStorage{saveErr: errors.New("save fail"), reader: bytes.NewReader([]byte("a"))}
	fo := &mockFileOptimiser{resizeOut: []byte("r")}
	svc := NewImageResizer(repo, fo, stg)

	err := svc.ResizeImage(context.Background(), ResizeImageInput{ID: db.NewUUID(), Sizes: []int{10}})
	if err == nil || !strings.Contains(err.Error(), "save fail") {
		t.Fatalf("expected save fail, got %v", err)
	}
}

func TestResizeImage_StatError(t *testing.T) {
	mt := "image/png"
	m := &model.Media{Status: model.MediaStatusCompleted, MimeType: &mt}
	repo := &mockRepo{mediaRecord: m}
	stg := &mockStorage{statErr: errors.New("stat fail"), reader: bytes.NewReader([]byte("a"))}
	fo := &mockFileOptimiser{resizeOut: []byte("r")}
	svc := NewImageResizer(repo, fo, stg)

	err := svc.ResizeImage(context.Background(), ResizeImageInput{ID: db.NewUUID(), Sizes: []int{10}})
	if err == nil || !strings.Contains(err.Error(), "stat fail") {
		t.Fatalf("expected stat fail, got %v", err)
	}
}

func TestResizeImage_UpdateError(t *testing.T) {
	mt := "image/png"
	m := &model.Media{Status: model.MediaStatusCompleted, MimeType: &mt}
	repo := &mockRepo{mediaRecord: m, updateErr: errors.New("update fail")}
	stg := &mockStorage{reader: bytes.NewReader([]byte("a")), statInfo: FileInfo{SizeBytes: 1}}
	fo := &mockFileOptimiser{resizeOut: []byte("r")}
	svc := NewImageResizer(repo, fo, stg)

	err := svc.ResizeImage(context.Background(), ResizeImageInput{ID: db.NewUUID(), Sizes: []int{10}})
	if err == nil || !strings.Contains(err.Error(), "update fail") {
		t.Fatalf("expected update fail, got %v", err)
	}
}

func TestResizeImage_Success(t *testing.T) {
	mt := "image/png"
	size := int64(0)
	m := &model.Media{
		ID:        db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")),
		Status:    model.MediaStatusCompleted,
		MimeType:  &mt,
		Bucket:    "images",
		ObjectKey: "foo.png",
		Metadata: model.Metadata{
			Width:  100,
			Height: 50,
		},
		SizeBytes: &size,
	}
	repo := &mockRepo{mediaRecord: m}
	stg := &mockStorage{reader: bytes.NewReader([]byte("abc")), statInfo: FileInfo{SizeBytes: 123}}
	fo := &mockFileOptimiser{resizeOut: []byte("resized")}
	svc := NewImageResizer(repo, fo, stg)

	err := svc.ResizeImage(context.Background(), ResizeImageInput{ID: m.ID, Sizes: []int{20, 0, -1, 40}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.updated == nil {
		t.Fatal("expected repo.Update to be called")
	}
	if len(repo.updated.Variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(repo.updated.Variants))
	}
	v := repo.updated.Variants[0]
	if v.ObjectKey != "variants/foo.png_20.webp" || v.Width != 20 || v.Height != 10 || v.SizeBytes != 123 {
		t.Errorf("first variant unexpected: %+v", v)
	}
	v2 := repo.updated.Variants[1]
	if v2.ObjectKey != "variants/foo.png_40.webp" || v2.Width != 40 || v2.Height != 20 {
		t.Errorf("second variant unexpected: %+v", v2)
	}
}
