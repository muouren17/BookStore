package handler

import (
	"bs/internal/service"
	"bs/pb"
)

type BookstoreServer struct {
	pb.UnimplementedBookstoreServer

	shelfService *service.ShelfService
	bookService  *service.BookService
}

func NewBookstoreServer(
	shelfService *service.ShelfService,
	bookService *service.BookService,
) *BookstoreServer {
	return &BookstoreServer{
		shelfService: shelfService,
		bookService:  bookService,
	}
}
