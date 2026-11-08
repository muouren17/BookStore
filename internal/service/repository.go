package service

import (
	"bs/internal/model"
	"context"
)

type ShelfRepository interface {
	CreateShelf(ctx context.Context, shelf *model.Shelf) (*model.Shelf, error)
	GetShelf(ctx context.Context, shelfid int64) (*model.Shelf, error)
	DeleteShelf(ctx context.Context, shelfid int64) error
	ListShelves(ctx context.Context) ([]*model.Shelf, error)
}

type BookRepository interface {
	CreateBook(ctx context.Context, b *model.Book) (*model.Book, error)
	GetBook(ctx context.Context, shelfid int64, bookid int64) (*model.Book, error)
	ListBooks(ctx context.Context, shelfid int64, cursor int64, pagesize int64) ([]*model.Book, error)
	DeleteBook(ctx context.Context, shelfid int64, bookid int64) error
}
