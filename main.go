package main

import (
	"bs/internal/dao/mysql"
	"bs/internal/handler"
	"bs/internal/service"
	"bs/pb"
	"bs/settings"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}

}

func run() error {

	signalCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	if err := settings.Init(); err != nil {
		return fmt.Errorf("settings.Init() failed , err:%v\n", err)
	}
	db, err := mysql.Init(settings.Conf.MySQLConfig)
	if err != nil {
		return fmt.Errorf("mysql.Init(settings.Conf.MySQLConfig) failed , err:%v\n", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("db.DB() failed , err:%v\n", err)
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
		return fmt.Errorf("net.Listen failed , err:%v\n", err)
	}
	defer listener.Close()

	grpcServer := grpc.NewServer()
	pb.RegisterBookstoreServer(grpcServer, bookstoreServer)
	defer grpcServer.Stop()

	serveErr := make(chan error, 2)

	go func() {
		serveErr <- grpcServer.Serve(listener)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gatewayMux := runtime.NewServeMux()
	err = pb.RegisterBookstoreHandlerFromEndpoint(
		ctx,
		gatewayMux,
		fmt.Sprintf("127.0.0.1:%d", settings.Conf.Port),
		[]grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		},
	)
	if err != nil {
		return fmt.Errorf("注册 Gateway 失败: %w", err)
	}
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", settings.Conf.HTTPPort),
		Handler: gatewayMux,
	}
	defer httpServer.Close()

	go func() {
		serveErr <- httpServer.ListenAndServe()
	}()

	log.Printf(
		"gRPC 端口: %d，HTTP Gateway 端口: %d",
		settings.Conf.Port,
		settings.Conf.HTTPPort,
	)

	var runErr error
	select {
	case <-signalCtx.Done():
		log.Println("收到退出信号，开始关闭服务")
	case runErr = <-serveErr:
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second*10)
	defer shutdownCancel()
	shutdownErr := shutdownServers(shutdownCtx, httpServer, grpcServer)
	return errors.Join(runErr, shutdownErr)
}

func shutdownServers(ctx context.Context, httpServer *http.Server, grpcServer *grpc.Server) error {
	httpErr := httpServer.Shutdown(ctx)
	if httpErr != nil {
		_ = httpServer.Close()
	}
	grpcDone := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(grpcDone)
	}()
	select {
	case <-grpcDone:
		return httpErr
	case <-ctx.Done():
		grpcServer.Stop()
		return errors.Join(httpErr, ctx.Err())
	}
}
