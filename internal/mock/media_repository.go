package mock

import (
	"context"
	"time"

	"github.com/fhuszti/medias-ms-go/internal/db"
	"github.com/fhuszti/medias-ms-go/internal/model"
)

// MockMediaRepo implements repository operations for tests.
type MockMediaRepo struct {
	MediaRecord *model.Media

	GetErr     error
	CreateErr  error
	UpdateErr  error
	DeleteErr  error
	ListErr    error
	ListOut    []db.UUID
	ListBefore time.Time

	ListVariantsErr    error
	ListVariantsOut    []db.UUID
	ListVariantsBefore time.Time

	ListVariantsCalled bool

	GetCalled    bool
	Created      *model.Media
	Updated      *model.Media
	DeleteCalled bool
	DeletedID    db.UUID
	ListCalled   bool
}

func (m *MockMediaRepo) GetByID(ctx context.Context, id db.UUID) (*model.Media, error) {
	m.GetCalled = true
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	return m.MediaRecord, nil
}

func (m *MockMediaRepo) Update(ctx context.Context, media *model.Media) error {
	m.Updated = media
	return m.UpdateErr
}

func (m *MockMediaRepo) Create(ctx context.Context, media *model.Media) error {
	m.Created = media
	return m.CreateErr
}

func (m *MockMediaRepo) Delete(ctx context.Context, id db.UUID) error {
	m.DeleteCalled = true
	m.DeletedID = id
	return m.DeleteErr
}

func (m *MockMediaRepo) ListUnoptimisedCompletedBefore(ctx context.Context, before time.Time) ([]db.UUID, error) {
	m.ListCalled = true
	m.ListBefore = before
	if m.ListErr != nil {
		return nil, m.ListErr
	}
	return m.ListOut, nil
}

func (m *MockMediaRepo) ListOptimisedImagesNoVariantsBefore(ctx context.Context, before time.Time) ([]db.UUID, error) {
	m.ListVariantsCalled = true
	m.ListVariantsBefore = before
	if m.ListVariantsErr != nil {
		return nil, m.ListVariantsErr
	}
	return m.ListVariantsOut, nil
}
