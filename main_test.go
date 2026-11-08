package main

import (
	"bs/pb"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestShutdownFinishesAcceptedGatewayRequest(t *testing.T) {
	f := newShutdownFixture(t, true, false)
	response := f.requestHTTP("/v1/shelves")
	awaitShutdownValue(t, f.gatewayEntered)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	shutdown := make(chan error, 1)
	go func() { shutdown <- f.shutdown(ctx) }()

	if code := f.waitUntilDraining(t); code != http.StatusServiceUnavailable {
		t.Errorf("new REST request status = %d, want 503 while accepted request drains", code)
	}
	f.releaseGateway()
	result := awaitShutdownValue(t, response)
	if result.err != nil || result.status != http.StatusOK {
		t.Errorf("accepted Gateway request failed: status=%d, err=%v, body=%s", result.status, result.err, result.body)
	}
	if err := awaitShutdownValue(t, shutdown); err != nil {
		t.Errorf("shutdown failed: %v", err)
	}
}

func TestShutdownFinishesInFlightRPC(t *testing.T) {
	for _, gateway := range []bool{false, true} {
		name := "direct_grpc"
		if gateway {
			name = "http_gateway"
		}
		t.Run(name, func(t *testing.T) {
			f := newShutdownFixture(t, false, true)
			var response <-chan shutdownResponse
			if gateway {
				response = f.requestHTTP("/v1/shelves")
			} else {
				response = f.requestRPC()
			}
			awaitShutdownValue(t, f.rpcEntered)

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			shutdown := make(chan error, 1)
			go func() { shutdown <- f.shutdown(ctx) }()
			f.waitUntilDraining(t)
			f.releaseRPC()

			result := awaitShutdownValue(t, response)
			if result.err != nil || result.status != http.StatusOK {
				t.Errorf("in-flight RPC failed: status=%d, err=%v", result.status, result.err)
			}
			if err := awaitShutdownValue(t, shutdown); err != nil {
				t.Errorf("shutdown failed: %v", err)
			}
		})
	}
}

func TestShutdownDeadlineClosesBlockedRequests(t *testing.T) {
	for _, gateway := range []bool{true, false} {
		name := "direct_grpc"
		if gateway {
			name = "gateway_before_rpc"
		}
		t.Run(name, func(t *testing.T) {
			f := newShutdownFixture(t, gateway, !gateway)
			var response <-chan shutdownResponse
			if gateway {
				response = f.requestHTTP("/v1/shelves")
				awaitShutdownValue(t, f.gatewayEntered)
			} else {
				response = f.requestRPC()
				awaitShutdownValue(t, f.rpcEntered)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			if err := f.shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("shutdown error = %v, want deadline exceeded", err)
			}
			if result := awaitShutdownValue(t, response); result.err == nil {
				t.Errorf("blocked request completed successfully after forced shutdown: %+v", result)
			}
		})
	}
}

type shutdownResponse struct {
	status int
	body   string
	err    error
}

type shutdownFixture struct {
	client         *http.Client
	grpcClient     pb.BookstoreClient
	gatewayEntered chan struct{}
	rpcEntered     chan struct{}
	releaseGateway func()
	releaseRPC     func()
	shutdown       func(context.Context) error
}

func newShutdownFixture(t *testing.T, blockGateway, blockRPC bool) *shutdownFixture {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	gatewayRelease, rpcRelease := make(chan struct{}), make(chan struct{})
	var gatewayOnce, rpcOnce sync.Once
	f := &shutdownFixture{
		gatewayEntered: make(chan struct{}),
		rpcEntered:     make(chan struct{}),
		releaseGateway: func() { gatewayOnce.Do(func() { close(gatewayRelease) }) },
		releaseRPC:     func() { rpcOnce.Do(func() { close(rpcRelease) }) },
	}
	business := &shutdownBookstoreServer{entered: f.rpcEntered}
	if blockRPC {
		business.release = rpcRelease
	}
	pb.RegisterBookstoreServer(grpcServer, business)
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	gateway := runtime.NewServeMux()
	if err := pb.RegisterBookstoreHandlerFromEndpoint(ctx, gateway, "passthrough:///shutdown-test", options); err != nil {
		t.Fatal(err)
	}
	var enteredOnce sync.Once
	wrappedGateway := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The probe observes admission without creating an internal gRPC connection.
		if r.URL.Path == "/shutdown-probe" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		enteredOnce.Do(func() { close(f.gatewayEntered) })
		if blockGateway {
			select {
			case <-gatewayRelease:
			case <-r.Context().Done():
				return
			}
		}
		gateway.ServeHTTP(w, r)
	})
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)
	drain := &gatewayDrain{}
	server := &http.Server{
		Protocols: protocols,
		Handler:   grpcGatewayHandler(grpcServer, wrappedGateway, drain),
	}
	f.shutdown = func(ctx context.Context) error { return shutdownServers(ctx, server, grpcServer, drain) }
	go func() { _ = server.Serve(listener) }()
	transport := &http.Transport{
		DisableKeepAlives: true,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		},
	}
	f.client = &http.Client{Transport: transport, Timeout: 3 * time.Second}
	connection, err := grpc.NewClient("passthrough:///shutdown-test", options...)
	if err != nil {
		t.Fatal(err)
	}
	f.grpcClient = pb.NewBookstoreClient(connection)
	t.Cleanup(func() {
		f.releaseGateway()
		f.releaseRPC()
		_ = server.Close()
		grpcServer.Stop()
		_ = connection.Close()
		transport.CloseIdleConnections()
		_ = listener.Close()
	})
	return f
}

func (f *shutdownFixture) requestHTTP(path string) <-chan shutdownResponse {
	done := make(chan shutdownResponse, 1)
	go func() { done <- f.getHTTP(path) }()
	return done
}

func (f *shutdownFixture) getHTTP(path string) shutdownResponse {
	response, err := f.client.Get("http://shutdown-test" + path)
	if err != nil {
		return shutdownResponse{err: err}
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	return shutdownResponse{status: response.StatusCode, body: string(body), err: err}
}

func (f *shutdownFixture) requestRPC() <-chan shutdownResponse {
	done := make(chan shutdownResponse, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err := f.grpcClient.ListShelves(ctx, &emptypb.Empty{})
		done <- shutdownResponse{status: http.StatusOK, err: err}
	}()
	return done
}

func (f *shutdownFixture) waitUntilDraining(t *testing.T) int {
	t.Helper()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(3 * time.Second)
	defer timeout.Stop()
	for {
		response := f.getHTTP("/shutdown-probe")
		if response.err != nil || response.status == http.StatusServiceUnavailable {
			return response.status
		}
		if response.status != http.StatusNoContent {
			t.Fatalf("unexpected shutdown probe response: %+v", response)
		}
		select {
		case <-ticker.C:
		case <-timeout.C:
			t.Fatal("server did not begin shutdown")
		}
	}
}

func awaitShutdownValue[T any](t *testing.T, values <-chan T) T {
	t.Helper()
	select {
	case value := <-values:
		return value
	case <-time.After(4 * time.Second):
		t.Fatal("timed out waiting for shutdown test")
		var zero T
		return zero
	}
}

type shutdownBookstoreServer struct {
	pb.UnimplementedBookstoreServer
	entered chan struct{}
	release <-chan struct{}
	once    sync.Once
}

func (s *shutdownBookstoreServer) ListShelves(ctx context.Context, _ *emptypb.Empty) (*pb.ListShelvesResponse, error) {
	s.once.Do(func() { close(s.entered) })
	if s.release != nil {
		select {
		case <-s.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return &pb.ListShelvesResponse{Shelf: []*pb.Shelf{{Id: 1, Theme: "shutdown test", Size: 10}}}, nil
}
