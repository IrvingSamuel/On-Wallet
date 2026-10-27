package collections

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, ownerID, name string) (Collection, error) {
	name, err := normalizeName(name)
	if err != nil {
		return Collection{}, err
	}
	c := Collection{
		ID:      uuid.New(),
		OwnerID: ownerID,
		Name:    name,
	}
	if err := s.repo.Create(ctx, &c); err != nil {
		return Collection{}, err
	}
	return c, nil
}

func (s *Service) List(ctx context.Context, ownerID string) ([]ListItem, error) {
	return s.repo.List(ctx, ownerID)
}

func (s *Service) Get(ctx context.Context, ownerID string, id uuid.UUID) (Detail, error) {
	return s.repo.Get(ctx, ownerID, id)
}

func (s *Service) Rename(ctx context.Context, ownerID string, id uuid.UUID, name string) (Collection, error) {
	name, err := normalizeName(name)
	if err != nil {
		return Collection{}, err
	}
	return s.repo.Rename(ctx, ownerID, id, name)
}

func (s *Service) Delete(ctx context.Context, ownerID string, id uuid.UUID) error {
	return s.repo.Delete(ctx, ownerID, id)
}

func (s *Service) AddDocuments(ctx context.Context, ownerID string, collectionID uuid.UUID, docIDs []uuid.UUID) (Detail, error) {
	if err := s.repo.OwnedCollection(ctx, ownerID, collectionID); err != nil {
		return Detail{}, err
	}
	if len(docIDs) == 0 {
		return Detail{}, fmt.Errorf("%w: informe ao menos um document_id", ErrInvalidInput)
	}
	owned, err := s.repo.OwnedDocumentIDs(ctx, ownerID, docIDs)
	if err != nil {
		return Detail{}, err
	}
	if len(owned) != len(uniqueUUIDs(docIDs)) {
		return Detail{}, ErrNotFound
	}
	if err := s.repo.AddDocuments(ctx, collectionID, owned); err != nil {
		return Detail{}, err
	}
	return s.repo.Get(ctx, ownerID, collectionID)
}

func (s *Service) RemoveDocument(ctx context.Context, ownerID string, collectionID, docID uuid.UUID) error {
	return s.repo.RemoveDocument(ctx, ownerID, collectionID, docID)
}

func normalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("%w: name obrigatório", ErrInvalidInput)
	}
	if len([]rune(name)) > 120 {
		return "", fmt.Errorf("%w: name com no máximo 120 caracteres", ErrInvalidInput)
	}
	return name, nil
}

func uniqueUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
