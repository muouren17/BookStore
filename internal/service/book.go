package service

import (
	"bs/internal/dao/mysql"
	"bs/internal/model"
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrGetShelfFailed = errors.New("get shelf failed")
	ErrGetBookFailed  = errors.New("get book failed")
	ErrShelfOversize  = errors.New("shelf cant contain book out of capacity")
	ErrNotargetBook   = errors.New("find no target book")
	ErrNoBook         = errors.New("find no book exist")
)

const (
	defaultAuthorName = "anonymous"
	defaultTitle      = "others"
)

type BookService struct {
	repo BookRepository
}

func NewBookService(repo BookRepository) *BookService {
	return &BookService{
		repo: repo,
	}
}

func (svc *BookService) CreateBook(ctx context.Context, b *model.Book) (*model.Book, error) {
	if strings.TrimSpace(b.Author) == "" {
		b.Author = defaultAuthorName
	}
	if strings.TrimSpace(b.Title) == "" {
		b.Title = defaultTitle
	}
	book, err := svc.repo.CreateBook(ctx, b)

	if errors.Is(err, mysql.ErrShelfFull) {
		return nil, ErrShelfOversize
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotargetShelf
	}
	if err != nil {
		return nil, err
	}
	return book, nil
}

func (svc *BookService) GetBook(ctx context.Context, shelfid int64, bookid int64) (*model.Book, error) {
	book, err := svc.repo.GetBook(ctx, shelfid, bookid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotargetBook
		}
		return nil, ErrGetBookFailed
	}
	return book, nil
}

func (svc *BookService) ListBooks(ctx context.Context, shelfid int64) ([]*model.Book, error) {
	books, err := svc.repo.ListBooks(ctx, shelfid)
	if err != nil {
		return nil, err
	}
	if len(books) == 0 {
		return nil, ErrNoBook
	}
	return books, nil
}

func (svc *BookService) DeleteBook(ctx context.Context, shelfid int64, bookid int64) error {
	_, err := svc.repo.GetBook(ctx, shelfid, bookid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotargetBook
		}
		return err
	}
	err = svc.repo.DeleteBook(ctx, shelfid, bookid)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotargetBook
	}
	if err != nil {
		return err
	}
	return nil
}
