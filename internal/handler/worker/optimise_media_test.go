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

func TestOptimiseMediaHandler_InvalidID(t *testing.T) {
	svc := &mock.MediaOptimiser{}
	err := OptimiseMediaHandler(context.Background(), task.OptimiseMediaPayload{ID: "invalid"}, svc)
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
	if svc.Called {
		t.Error("service should not be called on invalid id")
	}
}

func TestOptimiseMediaHandler_ServiceError(t *testing.T) {
	id := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	svcErr := errors.New("svc fail")
	svc := &mock.MediaOptimiser{Err: svcErr}

	err := OptimiseMediaHandler(context.Background(), task.OptimiseMediaPayload{ID: id.String()}, svc)
	if !errors.Is(err, svcErr) {
		t.Fatalf("got error %v; want %v", err, svcErr)
	}
	if !svc.Called {
		t.Error("service not called")
	}
	if svc.ID != id {
		t.Errorf("service got id %s; want %s", svc.ID, id)
	}
}

func TestOptimiseMediaHandler_Success(t *testing.T) {
	id := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	svc := &mock.MediaOptimiser{}

	err := OptimiseMediaHandler(context.Background(), task.OptimiseMediaPayload{ID: id.String()}, svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !svc.Called {
		t.Error("service not called")
	}
	if svc.ID != id {
		t.Errorf("service got id %s; want %s", svc.ID, id)
	}
}
