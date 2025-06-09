package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/task"
	mediaSvc "github.com/fhuszti/medias-ms-go/internal/usecase/media"
	"github.com/google/uuid"
)

type mockOptimiser struct {
	in     mediaSvc.OptimiseMediaInput
	called bool
	err    error
}

func (m *mockOptimiser) OptimiseMedia(ctx context.Context, in mediaSvc.OptimiseMediaInput) error {
	m.called = true
	m.in = in
	return m.err
}

func TestOptimiseMediaHandler_InvalidID(t *testing.T) {
	svc := &mockOptimiser{}
	err := OptimiseMediaHandler(context.Background(), task.OptimiseMediaPayload{ID: "invalid"}, svc)
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
	if svc.called {
		t.Error("service should not be called on invalid id")
	}
}

func TestOptimiseMediaHandler_ServiceError(t *testing.T) {
	id := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	svcErr := errors.New("svc fail")
	svc := &mockOptimiser{err: svcErr}

	err := OptimiseMediaHandler(context.Background(), task.OptimiseMediaPayload{ID: id.String()}, svc)
	if !errors.Is(err, svcErr) {
		t.Fatalf("got error %v; want %v", err, svcErr)
	}
	if !svc.called {
		t.Error("service not called")
	}
	if svc.in.ID != id {
		t.Errorf("service got id %s; want %s", svc.in.ID, id)
	}
}

func TestOptimiseMediaHandler_Success(t *testing.T) {
	id := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	svc := &mockOptimiser{}

	err := OptimiseMediaHandler(context.Background(), task.OptimiseMediaPayload{ID: id.String()}, svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !svc.called {
		t.Error("service not called")
	}
	if svc.in.ID != id {
		t.Errorf("service got id %s; want %s", svc.in.ID, id)
	}
}
