package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/db"
	mediaSvc "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/google/uuid"
)

type mockResizer struct {
	in     mediaSvc.ResizeImageInput
	called bool
	err    error
}

func (m *mockResizer) ResizeImage(ctx context.Context, in mediaSvc.ResizeImageInput) error {
	m.called = true
	m.in = in
	return m.err
}

func TestResizeImageHandler_InvalidID(t *testing.T) {
	svc := &mockResizer{}
	err := ResizeImageHandler(context.Background(), ResizeImagePayload{MediaID: "invalid"}, svc)
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
	if svc.called {
		t.Error("service should not be called on invalid id")
	}
}

func TestResizeImageHandler_ServiceError(t *testing.T) {
	id := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	svcErr := errors.New("svc fail")
	svc := &mockResizer{err: svcErr}

	sizes := []int{100, 200}
	err := ResizeImageHandler(context.Background(), ResizeImagePayload{MediaID: id.String(), Sizes: sizes}, svc)
	if !errors.Is(err, svcErr) {
		t.Fatalf("got error %v; want %v", err, svcErr)
	}
	if !svc.called {
		t.Error("service not called")
	}
	if svc.in.ID != id {
		t.Errorf("service got id %s; want %s", svc.in.ID, id)
	}
	if len(svc.in.Sizes) != len(sizes) {
		t.Errorf("service got sizes %v; want %v", svc.in.Sizes, sizes)
	}
}

func TestResizeImageHandler_Success(t *testing.T) {
	id := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	svc := &mockResizer{}
	sizes := []int{100, 200}

	err := ResizeImageHandler(context.Background(), ResizeImagePayload{MediaID: id.String(), Sizes: sizes}, svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !svc.called {
		t.Error("service not called")
	}
	if svc.in.ID != id {
		t.Errorf("service got id %s; want %s", svc.in.ID, id)
	}
	if len(svc.in.Sizes) != len(sizes) {
		t.Errorf("service got sizes %v; want %v", svc.in.Sizes, sizes)
	}
}
