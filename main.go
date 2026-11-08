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
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
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

	//统一端口

	grpcServer := grpc.NewServer()
	pb.RegisterBookstoreServer(grpcServer, bookstoreServer)
	defer grpcServer.Stop()

	serveErr := make(chan error, 1)

	//

	// listener, err := net.Listen(
	// 	"tcp",
	// 	fmt.Sprintf(":%d", settings.Conf.Port),
	// )
	// if err != nil {
	// 	return fmt.Errorf("net.Listen failed , err:%v\n", err)
	// }
	// defer listener.Close()

	// grpcServer := grpc.NewServer()
	// pb.RegisterBookstoreServer(grpcServer, bookstoreServer)
	// defer grpcServer.Stop()

	// serveErr := make(chan error, 2)

	// go func() {
	// 	serveErr <- grpcServer.Serve(listener)
	// }()

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

	//统一端口

	drain := &gatewayDrain{}
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	httpServer := &http.Server{
		Addr:      fmt.Sprintf(":%d", settings.Conf.Port),
		Protocols: protocols,
		// Handler: grpcGatewayHandler(grpcServer, gatewayMux),
		Handler: grpcGatewayHandler(grpcServer, gatewayMux, drain),
	}
	defer httpServer.Close()

	log.Printf("gRPC 和 HTTP API 共用端口: %d", settings.Conf.Port)
	//

	// httpServer := &http.Server{
	// 	Addr:    fmt.Sprintf(":%d", settings.Conf.HTTPPort),
	// 	Handler: gatewayMux,
	// }
	// defer httpServer.Close()

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
	// shutdownErr := shutdownServers(shutdownCtx, httpServer, grpcServer)
	shutdownErr := shutdownServers(shutdownCtx, httpServer, grpcServer, drain)
	return errors.Join(runErr, shutdownErr)
}

// 原有退出逻辑，保留供对照。
/*
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
*/

// 原有请求分流逻辑，保留供对照。
/*
func grpcGatewayHandler(
	grpcServer *grpc.Server,
	gatewayMux http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 &&
			strings.HasPrefix(
				r.Header.Get("Content-Type"),
				"application/grpc",
			) {
			grpcServer.ServeHTTP(w, r)
			return
		}

		gatewayMux.ServeHTTP(w, r)
	})
}
*/

func shutdownServers(
	ctx context.Context,
	httpServer *http.Server,
	grpcServer *grpc.Server,
	drain *gatewayDrain,
) error {
	defer grpcServer.Stop()

	// Gateway 还需要回连共用端口，先等待已有 REST 请求完成。
	select {
	case <-drain.startDrain():
	case <-ctx.Done():
		return errors.Join(ctx.Err(), httpServer.Close())
	}

	// 此时再关闭监听端口，等待剩余的 HTTP/2 gRPC 请求完成。
	if err := httpServer.Shutdown(ctx); err != nil {
		return errors.Join(err, httpServer.Close())
	}
	return nil
}

func grpcGatewayHandler(
	grpcServer *grpc.Server,
	gatewayMux http.Handler,
	drain *gatewayDrain,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 &&
			strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			// 退出期间继续允许 Gateway 发起内部 gRPC 调用。
			grpcServer.ServeHTTP(w, r)
			return
		}

		if !drain.begin() {
			http.Error(w, "服务正在关闭，请稍后重试", http.StatusServiceUnavailable)
			return
		}
		defer drain.active.Done()
		gatewayMux.ServeHTTP(w, r)
	})
}

type gatewayDrain struct {
	mu      sync.Mutex
	closing bool
	active  sync.WaitGroup
}

func (d *gatewayDrain) begin() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closing {
		return false
	}
	d.active.Add(1)
	return true
}

func (d *gatewayDrain) startDrain() <-chan struct{} {
	d.mu.Lock()
	d.closing = true
	d.mu.Unlock()

	done := make(chan struct{})
	go func() {
		d.active.Wait()
		close(done)
	}()
	return done
}
