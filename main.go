package main

import (
	"bs/internal/dao/mysql"
	"bs/internal/handler"
	"bs/internal/service"
	"bs/pb"
	"bs/settings"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}

}

func run() error {
	if err := settings.Init(); err != nil {
		log.Fatalf("settings.Init() failed , err:%v\n", err)
	}
	db, err := mysql.Init(settings.Conf.MySQLConfig)
	if err != nil {
		log.Fatalf("mysql.Init(settings.Conf.MySQLConfig) failed , err:%v\n", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("db.DB() failed , err:%v\n", err)
	}
	defer sqlDB.Close()
	shelfRepo := mysql.NewShelfRepository(db)
	bookRepo := mysql.NewBookRepository(db)

	shelfSvc := service.NewShelfService(shelfRepo)
	bookSvc := service.NewBookService(bookRepo)

	bookstoreServer := handler.NewBookstoreServer(shelfSvc, bookSvc)

	listener, err := net.Listen(
		"tcp",
		fmt.Sprintf(":%d", settings.Conf.Port),
	)
	if err != nil {
		log.Fatalf("net.Listen failed , err:%v\n", err)
	}
	defer listener.Close()

	grpcServer := grpc.NewServer()
	pb.RegisterBookstoreServer(grpcServer, bookstoreServer)
	log.Printf("gRPC listening %s", listener.Addr())
	return grpcServer.Serve(listener)
}
