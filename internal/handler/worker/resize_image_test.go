package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/mock"
	"github.com/fhuszti/medias-ms-go/internal/task"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	"github.com/google/uuid"
)

func TestResizeImageHandler_InvalidID(t *testing.T) {
	svc := &mock.ImageResizer{}
	err := ResizeImageHandler(context.Background(), task.ResizeImagePayload{ID: "invalid"}, nil, svc)
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
	if svc.Called {
		t.Error("service should not be called on invalid id")
	}
}

func TestResizeImageHandler_ServiceError(t *testing.T) {
	id := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	svcErr := errors.New("svc fail")
	svc := &mock.ImageResizer{Err: svcErr}

	sizes := []int{100, 200}
	err := ResizeImageHandler(context.Background(), task.ResizeImagePayload{ID: id.String()}, sizes, svc)
	if !errors.Is(err, svcErr) {
		t.Fatalf("got error %v; want %v", err, svcErr)
	}
	if !svc.Called {
		t.Error("service not called")
	}
	if svc.In.ID != id {
		t.Errorf("service got id %s; want %s", svc.In.ID, id)
	}
	if len(svc.In.Sizes) != len(sizes) {
		t.Errorf("service got sizes %v; want %v", svc.In.Sizes, sizes)
	}
}

func TestResizeImageHandler_Success(t *testing.T) {
	id := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	svc := &mock.ImageResizer{}
	sizes := []int{100, 200}

	err := ResizeImageHandler(context.Background(), task.ResizeImagePayload{ID: id.String()}, sizes, svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !svc.Called {
		t.Error("service not called")
	}
	if svc.In.ID != id {
		t.Errorf("service got id %s; want %s", svc.In.ID, id)
	}
	if len(svc.In.Sizes) != len(sizes) {
		t.Errorf("service got sizes %v; want %v", svc.In.Sizes, sizes)
	}
}
