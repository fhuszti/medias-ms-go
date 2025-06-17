package media

import (
	"context"
	"errors"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/google/uuid"
)

func TestBacklogOptimiser_RepoError(t *testing.T) {
	repo := &mockRepo{listErr: errors.New("db fail")}
	dispatcher := &mockDispatcher{}
	svc := NewBacklogOptimiser(repo, dispatcher)

	err := svc.OptimiseBacklog(context.Background())
	if err == nil || err.Error() != "db fail" {
		t.Fatalf("expected db fail, got %v", err)
	}
	if !repo.listCalled {
		t.Error("expected list to be called")
	}
}

func TestBacklogOptimiser_Success(t *testing.T) {
	id1 := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	id2 := db.UUID(uuid.MustParse("ffffffff-1111-2222-3333-444444444444"))
	resize1 := db.UUID(uuid.MustParse("11111111-2222-3333-4444-555555555555"))
	resize2 := db.UUID(uuid.MustParse("66666666-7777-8888-9999-000000000000"))
	repo := &mockRepo{listOut: []db.UUID{id1, id2}, listVariantsOut: []db.UUID{resize1, resize2}}
	dispatcher := &mockDispatcher{}
	svc := NewBacklogOptimiser(repo, dispatcher)

	if err := svc.OptimiseBacklog(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dispatcher.optimiseIDs) != 2 {
		t.Fatalf("expected 2 optimise calls, got %d", len(dispatcher.optimiseIDs))
	}
	if dispatcher.optimiseIDs[0] != id1 || dispatcher.optimiseIDs[1] != id2 {
		t.Errorf("optimise IDs mismatch: %+v", dispatcher.optimiseIDs)
	}
	if len(dispatcher.resizeIDs) != 2 {
		t.Fatalf("expected 2 resize calls, got %d", len(dispatcher.resizeIDs))
	}
	if dispatcher.resizeIDs[0] != resize1 || dispatcher.resizeIDs[1] != resize2 {
		t.Errorf("resize IDs mismatch: %+v", dispatcher.resizeIDs)
	}
	if !repo.listVariantsCalled {
		t.Error("expected list variants to be called")
	}
}

func TestBacklogOptimiser_DispatcherError(t *testing.T) {
	id1 := db.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	id2 := db.UUID(uuid.MustParse("ffffffff-1111-2222-3333-444444444444"))
	repo := &mockRepo{listOut: []db.UUID{id1, id2}}
	dispatcher := &mockDispatcher{optimiseErr: errors.New("queue fail")}
	svc := NewBacklogOptimiser(repo, dispatcher)

	if err := svc.OptimiseBacklog(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dispatcher.optimiseIDs) != 2 {
		t.Fatalf("expected 2 optimise calls, got %d", len(dispatcher.optimiseIDs))
	}
}

func TestBacklogOptimiser_ListVariantsError(t *testing.T) {
	repo := &mockRepo{listVariantsErr: errors.New("variants fail")}
	dispatcher := &mockDispatcher{}
	svc := NewBacklogOptimiser(repo, dispatcher)

	err := svc.OptimiseBacklog(context.Background())
	if err == nil || err.Error() != "variants fail" {
		t.Fatalf("expected variants fail, got %v", err)
	}
	if !repo.listVariantsCalled {
		t.Error("expected list variants to be called")
	}
}

func TestBacklogOptimiser_ResizeDispatcherError(t *testing.T) {
	resize1 := db.UUID(uuid.MustParse("11111111-2222-3333-4444-555555555555"))
	repo := &mockRepo{listVariantsOut: []db.UUID{resize1}}
	dispatcher := &mockDispatcher{resizeErr: errors.New("queue fail")}
	svc := NewBacklogOptimiser(repo, dispatcher)

	if err := svc.OptimiseBacklog(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dispatcher.resizeIDs) != 1 {
		t.Fatalf("expected 1 resize call, got %d", len(dispatcher.resizeIDs))
	}
}
