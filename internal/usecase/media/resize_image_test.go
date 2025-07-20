package media

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/mock"
	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/port"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	"github.com/google/uuid"
)

// errSeekReader forces Seek to fail
type errSeekReader struct{ io.Reader }

func (errSeekReader) Seek(int64, int) (int64, error) { return 0, errors.New("seek fail") }
func (errSeekReader) Close() error                   { return nil }

func TestResizeImage_GetByIDNotFound(t *testing.T) {
	repo := &mock.MediaRepo{GetByIDErr: sql.ErrNoRows}
	svc := NewImageResizer(repo, &mock.FileOptimiser{}, &mock.MockStorage{}, &mock.Cache{})

	id := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	err := svc.ResizeImage(context.Background(), port.ResizeImageInput{ID: id})
	if !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound, got %v", err)
	}
}

func TestResizeImage_GetByIDError(t *testing.T) {
	repo := &mock.MediaRepo{GetByIDErr: errors.New("db fail")}
	svc := NewImageResizer(repo, &mock.FileOptimiser{}, &mock.MockStorage{}, &mock.Cache{})

	id := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	err := svc.ResizeImage(context.Background(), port.ResizeImageInput{ID: id})
	if err == nil || err.Error() != "db fail" {
		t.Fatalf("expected db fail, got %v", err)
	}
}

func TestResizeImage_WrongStatus(t *testing.T) {
	mt := "image/png"
	m := &model.Media{Status: model.MediaStatusPending, MimeType: &mt}
	repo := &mock.MediaRepo{MediaOut: m}
	svc := NewImageResizer(repo, &mock.FileOptimiser{}, &mock.MockStorage{}, &mock.Cache{})

	id := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	err := svc.ResizeImage(context.Background(), port.ResizeImageInput{ID: id})
	want := "media status should be 'completed' to be resized"
	if err == nil || err.Error() != want {
		t.Fatalf("expected %q, got %v", want, err)
	}
}

func TestResizeImage_NotImage(t *testing.T) {
	mt := "application/pdf"
	m := &model.Media{Status: model.MediaStatusCompleted, MimeType: &mt}
	repo := &mock.MediaRepo{MediaOut: m}
	svc := NewImageResizer(repo, &mock.FileOptimiser{}, &mock.MockStorage{}, &mock.Cache{})

	id := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	err := svc.ResizeImage(context.Background(), port.ResizeImageInput{ID: id})
	if err == nil || err.Error() != "media is not an image" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResizeImage_GetFileError(t *testing.T) {
	mt := "image/png"
	m := &model.Media{Status: model.MediaStatusCompleted, MimeType: &mt}
	repo := &mock.MediaRepo{MediaOut: m}
	stg := &mock.MockStorage{GetErr: errors.New("get fail")}
	svc := NewImageResizer(repo, &mock.FileOptimiser{}, stg, &mock.Cache{})

	id := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	err := svc.ResizeImage(context.Background(), port.ResizeImageInput{ID: id})
	if err == nil || err.Error() != "get fail" {
		t.Fatalf("expected get fail, got %v", err)
	}
}

func TestResizeImage_SeekError(t *testing.T) {
	mt := "image/png"
	m := &model.Media{Status: model.MediaStatusCompleted, MimeType: &mt, Metadata: model.Metadata{Width: 100, Height: 50}}
	repo := &mock.MediaRepo{MediaOut: m}
	stg := &mock.MockStorage{Reader: errSeekReader{bytes.NewReader([]byte("a"))}}
	svc := NewImageResizer(repo, &mock.FileOptimiser{}, stg, &mock.Cache{})

	id := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	err := svc.ResizeImage(context.Background(), port.ResizeImageInput{ID: id, Sizes: []int{10}})
	if err == nil || !strings.Contains(err.Error(), "seek fail") {
		t.Fatalf("expected seek fail, got %v", err)
	}
}

func TestResizeImage_ResizeError(t *testing.T) {
	mt := "image/png"
	m := &model.Media{Status: model.MediaStatusCompleted, MimeType: &mt, Metadata: model.Metadata{Width: 100, Height: 50}}
	repo := &mock.MediaRepo{MediaOut: m}
	stg := &mock.MockStorage{Reader: bytes.NewReader([]byte("a"))}
	fo := &mock.FileOptimiser{ResizeErr: errors.New("resize fail")}
	svc := NewImageResizer(repo, fo, stg, &mock.Cache{})

	id := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	err := svc.ResizeImage(context.Background(), port.ResizeImageInput{ID: id, Sizes: []int{10}})
	if err == nil || err.Error() != "resize fail" {
		t.Fatalf("expected resize fail, got %v", err)
	}
}

func TestResizeImage_SaveFileError(t *testing.T) {
	mt := "image/png"
	m := &model.Media{Status: model.MediaStatusCompleted, MimeType: &mt, Metadata: model.Metadata{Width: 100, Height: 50}}
	repo := &mock.MediaRepo{MediaOut: m}
	stg := &mock.MockStorage{SaveErr: errors.New("save fail"), Reader: bytes.NewReader([]byte("a"))}
	fo := &mock.FileOptimiser{ResizeOut: []byte("r")}
	svc := NewImageResizer(repo, fo, stg, &mock.Cache{})

	id := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	err := svc.ResizeImage(context.Background(), port.ResizeImageInput{ID: id, Sizes: []int{10}})
	if err == nil || !strings.Contains(err.Error(), "save fail") {
		t.Fatalf("expected save fail, got %v", err)
	}
}

func TestResizeImage_StatError(t *testing.T) {
	mt := "image/png"
	m := &model.Media{Status: model.MediaStatusCompleted, MimeType: &mt, Metadata: model.Metadata{Width: 100, Height: 50}}
	repo := &mock.MediaRepo{MediaOut: m}
	stg := &mock.MockStorage{StatErr: errors.New("stat fail"), Reader: bytes.NewReader([]byte("a"))}
	fo := &mock.FileOptimiser{ResizeOut: []byte("r")}
	svc := NewImageResizer(repo, fo, stg, &mock.Cache{})

	id := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	err := svc.ResizeImage(context.Background(), port.ResizeImageInput{ID: id, Sizes: []int{10}})
	if err == nil || !strings.Contains(err.Error(), "stat fail") {
		t.Fatalf("expected stat fail, got %v", err)
	}
}

func TestResizeImage_UpdateError(t *testing.T) {
	mt := "image/png"
	m := &model.Media{Status: model.MediaStatusCompleted, MimeType: &mt, Metadata: model.Metadata{Width: 100, Height: 50}}
	repo := &mock.MediaRepo{MediaOut: m, UpdateErr: errors.New("update fail")}
	stg := &mock.MockStorage{Reader: bytes.NewReader([]byte("a")), StatInfo: port.FileInfo{SizeBytes: 1}}
	fo := &mock.FileOptimiser{ResizeOut: []byte("r")}
	svc := NewImageResizer(repo, fo, stg, &mock.Cache{})

	id := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	err := svc.ResizeImage(context.Background(), port.ResizeImageInput{ID: id, Sizes: []int{10}})
	if err == nil || !strings.Contains(err.Error(), "update fail") {
		t.Fatalf("expected update fail, got %v", err)
	}
}

func TestResizeImage_Success(t *testing.T) {
	idStr := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	mt := "image/png"
	size := int64(0)
	m := &model.Media{
		ID:        msuuid.UUID(uuid.MustParse(idStr)),
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
	repo := &mock.MediaRepo{MediaOut: m}
	stg := &mock.MockStorage{Reader: bytes.NewReader([]byte("abc")), StatInfo: port.FileInfo{SizeBytes: 123}}
	fo := &mock.FileOptimiser{ResizeOut: []byte("resized")}
	svc := NewImageResizer(repo, fo, stg, &mock.Cache{})

	err := svc.ResizeImage(context.Background(), port.ResizeImageInput{ID: m.ID, Sizes: []int{20, 0, -1, 40}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.GotUpdated == nil {
		t.Fatal("expected repo.Update to be called")
	}
	if len(repo.GotUpdated.Variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(repo.GotUpdated.Variants))
	}
	v := repo.GotUpdated.Variants[0]
	if v.ObjectKey != fmt.Sprintf("variants/%s/foo_20.webp", idStr) || v.Width != 20 || v.Height != 10 || v.SizeBytes != 123 {
		t.Errorf("first variant unexpected: %+v", v)
	}
	v2 := repo.GotUpdated.Variants[1]
	if v2.ObjectKey != fmt.Sprintf("variants/%s/foo_40.webp", idStr) || v2.Width != 40 || v2.Height != 20 {
		t.Errorf("second variant unexpected: %+v", v2)
	}
}

func TestResizeImage_CopyWhenWidthTooLarge(t *testing.T) {
	idStr := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	mt := "image/png"
	size := int64(0)
	m := &model.Media{
		ID:        msuuid.UUID(uuid.MustParse(idStr)),
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
	repo := &mock.MediaRepo{MediaOut: m}
	stg := &mock.MockStorage{Reader: bytes.NewReader([]byte("abc")), StatInfo: port.FileInfo{SizeBytes: 456}}
	fo := &mock.FileOptimiser{ResizeOut: []byte("resized")}
	svc := NewImageResizer(repo, fo, stg, &mock.Cache{})

	err := svc.ResizeImage(context.Background(), port.ResizeImageInput{ID: m.ID, Sizes: []int{200}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !stg.CopyCalled {
		t.Error("expected CopyFile to be called")
	}
	if stg.SaveCalled {
		t.Error("SaveFile should not be called when copying original")
	}
	if fo.ResizeCalled {
		t.Error("Resize should not be called when width is larger than original")
	}
	if len(repo.GotUpdated.Variants) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(repo.GotUpdated.Variants))
	}
	v := repo.GotUpdated.Variants[0]
	if v.ObjectKey != fmt.Sprintf("variants/%s/foo_200.webp", idStr) || v.Width != 100 || v.Height != 50 || v.SizeBytes != 456 {
		t.Errorf("variant unexpected: %+v", v)
	}
}
