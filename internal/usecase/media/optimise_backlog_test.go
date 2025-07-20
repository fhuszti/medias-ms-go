package media

import (
	"context"
	"errors"
	"testing"

	"github.com/fhuszti/medias-ms-go/internal/mock"
	msuuid "github.com/fhuszti/medias-ms-go/internal/uuid"
	"github.com/google/uuid"
)

func TestBacklogOptimiser_RepoError(t *testing.T) {
	repo := &mock.MediaRepo{ListUnoptimisedCompletedBeforeErr: errors.New("db fail")}
	dispatcher := &mock.Dispatcher{}
	svc := NewBacklogOptimiser(repo, dispatcher)

	err := svc.OptimiseBacklog(context.Background())
	if err == nil || err.Error() != "db fail" {
		t.Fatalf("expected db fail, got %v", err)
	}
	if !repo.ListUnoptimisedCompletedBeforeCalled {
		t.Error("expected list to be called")
	}
}

func TestBacklogOptimiser_Success(t *testing.T) {
	id1 := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	id2 := msuuid.UUID(uuid.MustParse("ffffffff-1111-2222-3333-444444444444"))
	resize1 := msuuid.UUID(uuid.MustParse("11111111-2222-3333-4444-555555555555"))
	resize2 := msuuid.UUID(uuid.MustParse("66666666-7777-8888-9999-000000000000"))
	repo := &mock.MediaRepo{ListOut: []msuuid.UUID{id1, id2}, ListVariantsOut: []msuuid.UUID{resize1, resize2}}
	dispatcher := &mock.Dispatcher{}
	svc := NewBacklogOptimiser(repo, dispatcher)

	if err := svc.OptimiseBacklog(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dispatcher.OptimiseIDs) != 2 {
		t.Fatalf("expected 2 optimise calls, got %d", len(dispatcher.OptimiseIDs))
	}
	if dispatcher.OptimiseIDs[0] != id1 || dispatcher.OptimiseIDs[1] != id2 {
		t.Errorf("optimise IDs mismatch: %+v", dispatcher.OptimiseIDs)
	}
	if len(dispatcher.ResizeIDs) != 2 {
		t.Fatalf("expected 2 resize calls, got %d", len(dispatcher.ResizeIDs))
	}
	if dispatcher.ResizeIDs[0] != resize1 || dispatcher.ResizeIDs[1] != resize2 {
		t.Errorf("resize IDs mismatch: %+v", dispatcher.ResizeIDs)
	}
	if !repo.ListOptimisedImagesNoVariantsBeforeCalled {
		t.Error("expected list variants to be called")
	}
}

func TestBacklogOptimiser_DispatcherError(t *testing.T) {
	id1 := msuuid.UUID(uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"))
	id2 := msuuid.UUID(uuid.MustParse("ffffffff-1111-2222-3333-444444444444"))
	repo := &mock.MediaRepo{ListOut: []msuuid.UUID{id1, id2}}
	dispatcher := &mock.Dispatcher{OptimiseErr: errors.New("queue fail")}
	svc := NewBacklogOptimiser(repo, dispatcher)

	if err := svc.OptimiseBacklog(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dispatcher.OptimiseIDs) != 2 {
		t.Fatalf("expected 2 optimise calls, got %d", len(dispatcher.OptimiseIDs))
	}
}

func TestBacklogOptimiser_ListVariantsError(t *testing.T) {
	repo := &mock.MediaRepo{ListOptimisedImagesNoVariantsBeforeErr: errors.New("variants fail")}
	dispatcher := &mock.Dispatcher{}
	svc := NewBacklogOptimiser(repo, dispatcher)

	err := svc.OptimiseBacklog(context.Background())
	if err == nil || err.Error() != "variants fail" {
		t.Fatalf("expected variants fail, got %v", err)
	}
	if !repo.ListOptimisedImagesNoVariantsBeforeCalled {
		t.Error("expected list variants to be called")
	}
}

func TestBacklogOptimiser_ResizeDispatcherError(t *testing.T) {
	resize1 := msuuid.UUID(uuid.MustParse("11111111-2222-3333-4444-555555555555"))
	repo := &mock.MediaRepo{ListVariantsOut: []msuuid.UUID{resize1}}
	dispatcher := &mock.Dispatcher{ResizeErr: errors.New("queue fail")}
	svc := NewBacklogOptimiser(repo, dispatcher)

	if err := svc.OptimiseBacklog(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dispatcher.ResizeIDs) != 1 {
		t.Fatalf("expected 1 resize call, got %d", len(dispatcher.ResizeIDs))
	}
}
