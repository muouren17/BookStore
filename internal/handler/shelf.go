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

func shelvesToProto(shelves []*model.Shelf) []*pb.Shelf {
	pbshelves := make([]*pb.Shelf, 0, len(shelves))
	for _, m := range shelves {
		pbshelves = append(pbshelves, &pb.Shelf{
			Id:    m.ID,
			Theme: m.Theme,
			Size:  m.Size,
		})
	}
	return pbshelves
}

func (bs *BookstoreServer) CreateShelf(ctx context.Context, s *pb.CreateShelfRequest) (*pb.CreateShelfResponse, error) {
	if s == nil || s.Shelf == nil {
		return &pb.CreateShelfResponse{Shelf: nil}, status.Error(codes.InvalidArgument, "missing shelf param")
	}
	shelf, err := bs.shelfService.CreateShelf(ctx, &model.Shelf{
		Theme: s.Shelf.Theme,
		Size:  s.Shelf.Size,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmptyTheme):
			return &pb.CreateShelfResponse{Shelf: nil}, status.Error(codes.InvalidArgument, err.Error())
		default:
			log.Printf("service.CreateShelf failed , err:%v\n", err)
			return &pb.CreateShelfResponse{Shelf: nil}, status.Error(codes.Internal, "service.CreateShelf failed")
		}
	}
	return &pb.CreateShelfResponse{
		Shelf: &pb.Shelf{
			Id:    shelf.ID,
			Theme: shelf.Theme,
			Size:  shelf.Size,
		},
	}, nil

}

func (bs *BookstoreServer) ListShelves(ctx context.Context, s *emptypb.Empty) (*pb.ListShelvesResponse, error) {
	shelves, err := bs.shelfService.ListShelves(ctx)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrGetShelfFailed):
			return &pb.ListShelvesResponse{Shelf: nil}, status.Error(codes.Internal, err.Error())
		case errors.Is(err, service.ErrNoShelf):
			return &pb.ListShelvesResponse{Shelf: nil}, nil
		default:
			return &pb.ListShelvesResponse{Shelf: nil}, status.Error(codes.Internal, "service.ListShelves failed")
		}
	}
	pbshelves := shelvesToProto(shelves)
	return &pb.ListShelvesResponse{
		Shelf: pbshelves,
	}, nil
}

func (bs *BookstoreServer) GetShelf(ctx context.Context, s *pb.GetShelfRequest) (*pb.GetShelfResponse, error) {
	if s == nil || s.Id <= 0 {
		return &pb.GetShelfResponse{Shelf: nil}, status.Error(codes.InvalidArgument, "invalid shelf id")
	}
	shelf, err := bs.shelfService.GetShelf(ctx, s.Id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotargetShelf):
			return &pb.GetShelfResponse{Shelf: nil}, status.Error(codes.NotFound, err.Error())
		default:
			return &pb.GetShelfResponse{Shelf: nil}, status.Error(codes.Internal, "service.GetShelf failed")
		}
	}
	return &pb.GetShelfResponse{
		Shelf: &pb.Shelf{
			Id:    shelf.ID,
			Theme: shelf.Theme,
			Size:  shelf.Size,
		},
	}, nil
}

func (bs *BookstoreServer) DeleteShelf(ctx context.Context, s *pb.DeleteShelfRequest) (*emptypb.Empty, error) {
	if s == nil || s.Id <= 0 {
		return &emptypb.Empty{}, status.Error(codes.InvalidArgument, "invalid shelf id")
	}
	err := bs.shelfService.DeleteShelf(ctx, s.Id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotargetShelf):
			return &emptypb.Empty{}, status.Error(codes.NotFound, err.Error())
		default:
			return &emptypb.Empty{}, status.Error(codes.Internal, "service.DeleteShelf failed")
		}
	}
	return &emptypb.Empty{}, nil
}
