package mock

import (
	"context"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/model"
	"github.com/fhuszti/medias-ms-go/internal/uuid"
)

// MediaRepo implements repository operations for tests.
type MediaRepo struct {
	// stored values
	MediaOut        *model.Media
	ListOut         []uuid.UUID
	ListVariantsOut []uuid.UUID

	// captured inputs
	GotCreated                             *model.Media
	GotUpdated                             *model.Media
	GotDeletedID                           uuid.UUID
	GotListUnoptimisedCompletedBefore      time.Time
	GotListOptimisedImagesNoVariantsBefore time.Time

	// errors
	GetByIDErr                             error
	CreateErr                              error
	UpdateErr                              error
	DeleteErr                              error
	ListUnoptimisedCompletedBeforeErr      error
	ListOptimisedImagesNoVariantsBeforeErr error

	// call flags
	GetByIDCalled                             bool
	CreateCalled                              bool
	UpdateCalled                              bool
	DeleteCalled                              bool
	ListUnoptimisedCompletedBeforeCalled      bool
	ListOptimisedImagesNoVariantsBeforeCalled bool
}

func (m *MediaRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Media, error) {
	m.GetByIDCalled = true
	if m.GetByIDErr != nil {
		return nil, m.GetByIDErr
	}
	return m.MediaOut, nil
}

func (m *MediaRepo) Create(ctx context.Context, media *model.Media) error {
	m.CreateCalled = true
	m.GotCreated = media
	return m.CreateErr
}

func (m *MediaRepo) Update(ctx context.Context, media *model.Media) error {
	m.UpdateCalled = true
	m.GotUpdated = media
	return m.UpdateErr
}

func (m *MediaRepo) Delete(ctx context.Context, id uuid.UUID) error {
	m.DeleteCalled = true
	m.GotDeletedID = id
	return m.DeleteErr
}

func (m *MediaRepo) ListUnoptimisedCompletedBefore(ctx context.Context, before time.Time) ([]uuid.UUID, error) {
	m.ListUnoptimisedCompletedBeforeCalled = true
	m.GotListUnoptimisedCompletedBefore = before
	if m.ListUnoptimisedCompletedBeforeErr != nil {
		return nil, m.ListUnoptimisedCompletedBeforeErr
	}
	return m.ListOut, nil
}

func (m *MediaRepo) ListOptimisedImagesNoVariantsBefore(ctx context.Context, before time.Time) ([]uuid.UUID, error) {
	m.ListOptimisedImagesNoVariantsBeforeCalled = true
	m.GotListOptimisedImagesNoVariantsBefore = before
	if m.ListOptimisedImagesNoVariantsBeforeErr != nil {
		return nil, m.ListOptimisedImagesNoVariantsBeforeErr
	}
	return m.ListVariantsOut, nil
}
