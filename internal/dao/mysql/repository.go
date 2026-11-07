package mysql

import "gorm.io/gorm"

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
