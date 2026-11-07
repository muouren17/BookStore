package handler

import (
	"bs/internal/model"
	"bs/internal/service"
	"bs/pb"
	"context"
	"errors"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func booksToProto(books []*model.Book) []*pb.Book {
	pbbooks := make([]*pb.Book, 0, len(books))
	for _, m := range books {
		pbbooks = append(pbbooks, &pb.Book{
			Id:     m.ID,
			Author: m.Author,
			Title:  m.Title,
		})
	}
	return pbbooks
}

func (bs *BookstoreServer) ListBooks(ctx context.Context, s *pb.ListBooksRequest) (*pb.ListBooksResponse, error) {
	if s == nil || s.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid shelf id")
	}
	books, err := bs.bookService.ListBooks(ctx, s.Id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrGetBookFailed):
			return &pb.ListBooksResponse{Books: nil}, status.Error(codes.Internal, err.Error())
		case errors.Is(err, service.ErrNoBook):
			return &pb.ListBooksResponse{Books: nil}, nil
		default:
			return &pb.ListBooksResponse{Books: nil}, status.Error(codes.Internal, "service.ListBooks failed")
		}
	}
	pbbooks := booksToProto(books)
	return &pb.ListBooksResponse{
		Books: pbbooks,
	}, nil
}

func (bs *BookstoreServer) CreateBook(ctx context.Context, s *pb.CreateBookRequest) (*pb.CreateBookResponse, error) {
	if s == nil || s.Book == nil {
		return nil, status.Error(codes.InvalidArgument, "missing book param")
	}
	if s.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid shelf id")
	}
	book, err := bs.bookService.CreateBook(ctx, &model.Book{
		ShelfID: s.Id,
		Author:  s.Book.Author,
		Title:   s.Book.Title,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrGetBookFailed) || errors.Is(err, service.ErrGetShelfFailed):
			return nil, status.Error(codes.Internal, err.Error())
		case errors.Is(err, service.ErrShelfOversize):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, service.ErrNotargetShelf):
			return nil, status.Error(codes.NotFound, err.Error())
		default:
			log.Printf("service.CreateBook failed , err:%v\n", err)
			return nil, status.Error(codes.Internal, "service.CreateBook failed")
		}
	}
	return &pb.CreateBookResponse{
		Book: &pb.Book{
			Id:     book.ID,
			Author: book.Author,
			Title:  book.Title,
		},
	}, nil
}

func (bs *BookstoreServer) GetBook(ctx context.Context, s *pb.GetBookRequest) (*pb.GetBookResponse, error) {
	if s == nil || s.Shelfid <= 0 || s.Bookid <= 0 {
		return &pb.GetBookResponse{Book: nil}, status.Error(codes.InvalidArgument, "invalid shelf id or invalid book id")
	}
	book, err := bs.bookService.GetBook(ctx, s.Shelfid, s.Bookid)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotargetBook):
			return &pb.GetBookResponse{Book: nil}, status.Error(codes.NotFound, err.Error())
		default:
			return &pb.GetBookResponse{Book: nil}, status.Error(codes.Internal, "service.GetBook failed")
		}
	}
	return &pb.GetBookResponse{
		Book: &pb.Book{
			Id:     book.ID,
			Author: book.Author,
			Title:  book.Title,
		},
	}, nil

}

func (bs *BookstoreServer) DeleteBook(ctx context.Context, s *pb.DeleteBookRequest) (*emptypb.Empty, error) {
	if s == nil || s.Shelfid <= 0 || s.Bookid <= 0 {
		return &emptypb.Empty{}, status.Error(codes.InvalidArgument, "invalid shelf id or invalid book id")
	}
	err := bs.bookService.DeleteBook(ctx, s.Shelfid, s.Bookid)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotargetBook):
			return &emptypb.Empty{}, status.Error(codes.NotFound, err.Error())
		default:
			return &emptypb.Empty{}, status.Error(codes.Internal, "service.DeleteBook failed")
		}
	}
	return &emptypb.Empty{}, nil
}
