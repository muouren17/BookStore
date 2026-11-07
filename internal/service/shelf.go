package service

import (
	"bs/internal/model"
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrEmptyTheme    = errors.New("theme cant be nil")
	ErrNoShelf       = errors.New("find no shelf exist")
	ErrNotargetShelf = errors.New("find no target shelf")
)

const (
	defaultSize = 500
	maxSize     = 5000
)

type ShelfService struct {
	repo ShelfRepository
}

func NewShelfService(repo ShelfRepository) *ShelfService {
	return &ShelfService{
		repo: repo,
	}
}

func (svc *ShelfService) CreateShelf(ctx context.Context, s *model.Shelf) (*model.Shelf, error) {
	s.Theme = strings.TrimSpace(s.Theme)
	if s.Theme == "" {
		return nil, ErrEmptyTheme
	}
	if s.Size <= 0 || s.Size > maxSize {
		s.Size = defaultSize
	}
	return svc.repo.CreateShelf(ctx, s)
}

func (svc *ShelfService) ListShelves(ctx context.Context) ([]*model.Shelf, error) {
	shelves, err := svc.repo.ListShelves(ctx)
	if err != nil {
		return nil, err
	}
	if len(shelves) == 0 {
		return nil, ErrNoShelf
	}
	return shelves, nil
}

func (svc *ShelfService) GetShelf(ctx context.Context, shelfid int64) (*model.Shelf, error) {
	shelf, err := svc.repo.GetShelf(ctx, shelfid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotargetShelf
		}
		return nil, ErrGetShelfFailed
	}
	return shelf, nil
}

func (svc *ShelfService) DeleteShelf(ctx context.Context, shelfid int64) error {
	err := svc.repo.DeleteShelf(ctx, shelfid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotargetShelf
		}
		return err
	}
	return nil
}
