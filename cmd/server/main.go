package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"url-shortener/internal/grpc_handler"
	"url-shortener/internal/handler"
	"url-shortener/internal/metrics"
	"url-shortener/internal/middleware"
	pb "url-shortener/internal/proto"
	"url-shortener/internal/service"
	"url-shortener/internal/storage"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	metrics.Init()
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	db := storage.SetupPostgres()
	redis := storage.SetupRedis()

	baseStorage := storage.NewSQLStorage(db)
	cachedStorage := storage.NewCachedStorage(baseStorage, redis)
	svc := service.NewURLService(cachedStorage)

	// User

	userStorage := storage.NewSQLUserStorage(db)
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	authSvc := service.NewAuthService(userStorage,jwtSecret)
	authHandler := handler.NewAuthHandler(authSvc, logger)



	// gRPC
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		logger.Fatalf("gRPC listen error: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterURLServiceServer(grpcServer, grpc_handler.NewURLGRPCHandler(svc))
	reflection.Register(grpcServer)

	go func() {
		logger.Println("gRPC server listening on :50051")
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatalf("gRPC server error: %v", err)
		}
	}()

	// HTTP
	h := handler.NewURLHandler(svc, logger)

	r := chi.NewRouter()
	r.Get("/{code}", h.Get)
	r.Handle("/metrics", promhttp.Handler())

	r.Post("/api/v1/register", authHandler.Register)
	r.Post("/api/v1/login", authHandler.Login)

	r.With(middleware.Auth(authSvc)).Post("/api/v1/shorten",h.Post)
	logger.Println("Server listening on :8080")
	
	srv := &http.Server{Addr: ":8080", Handler: r}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func () {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.Fatalf("Server error: %v", err)
		}
	}()

	<-ctx.Done()
	logger.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	srv.Shutdown(shutdownCtx)
	grpcServer.GracefulStop()

}
