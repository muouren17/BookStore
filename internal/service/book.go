package service

import (
	"bs/internal/model"
	"bs/internal/pagination"
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrGetShelfFailed = errors.New("get shelf failed")
	ErrGetBookFailed  = errors.New("get book failed")
	ErrShelfOversize  = errors.New("shelf cant contain book out of capacity")
	ErrNotargetBook   = errors.New("find no target book")
	ErrNoBook         = errors.New("find no book exist")
	ErrInvalidParam   = errors.New("invalid param")
)

const (
	defaultAuthorName = "anonymous"
	defaultTitle      = "others"
	defaultPageSize   = 2
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

	if errors.Is(err, model.ErrShelfFull) {
		return nil, ErrShelfOversize
	}
	if errors.Is(err, model.ErrNotFound) {
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
		if errors.Is(err, model.ErrNotFound) {
			return nil, ErrNotargetBook
		}
		return nil, ErrGetBookFailed
	}
	return book, nil
}

func (svc *BookService) ListBooks(ctx context.Context, shelfid int64, pagetoken string) ([]*model.Book, string, error) {
	// books, err := svc.repo.ListBooks(ctx, shelfid)
	// if err != nil {
	// 	return nil, err
	// }
	// if len(books) == 0 {
	// 	return nil, ErrNoBook
	// }
	// return books, nil

	//参数check

	//没有分页token默认第一页
	pagetoken = strings.TrimSpace(string(pagetoken))
	page := pagination.Page{}
	if len(pagetoken) == 0 {
		page.PageSize = defaultPageSize
		page.NextTimeAtUTC = time.Now().Unix()
	} else {
		//有分页就先解析分页token
		page = pagination.Decode(pagetoken)
	}
	//判断解析结果是否有效 nextid time pagesize
	if page.Invalid() {
		return nil, "", ErrInvalidParam
	}

	books, err := svc.repo.ListBooks(ctx, shelfid, page.NextID, page.PageSize+1)
	if err != nil {
		return nil, "", err
	}
	if len(books) == 0 {
		return nil, "", ErrNoBook
	}
	if len(books) <= int(page.PageSize) {
		return books, "", nil
	}
	books = books[:page.PageSize]

	nextpage := pagination.Page{
		NextID:        books[page.PageSize-1].ID,
		NextTimeAtUTC: time.Now().Unix(),
		PageSize:      page.PageSize,
	}
	nextcursor := nextpage.Encode()

	return books, nextcursor, nil
}

func (svc *BookService) DeleteBook(ctx context.Context, shelfid int64, bookid int64) error {
	_, err := svc.repo.GetBook(ctx, shelfid, bookid)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return ErrNotargetBook
		}
		return err
	}
	err = svc.repo.DeleteBook(ctx, shelfid, bookid)
	if errors.Is(err, model.ErrNotFound) {
		return ErrNotargetBook
	}
	if err != nil {
		return err
	}
	return nil
}
