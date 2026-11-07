package mysql

import (
	"bs/internal/model"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type ShelfRepository struct {
	db *gorm.DB
}

func NewShelfRepository(db *gorm.DB) *ShelfRepository {
	return &ShelfRepository{
		db: db,
	}
}

type BookRepository struct {
	db *gorm.DB
}

func NewBookRepository(db *gorm.DB) *BookRepository {
	return &BookRepository{
		db: db,
	}
}

func normalizeError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("%w: %w", model.ErrNotFound, err)
	}
	return err
}
