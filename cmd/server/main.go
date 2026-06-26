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

	"url-shortener/internal/telemetry"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	metrics.Init()
	shutdown , err := telemetry.InitTracer(context.Background())
	if err != nil {
		log.Fatalf("failed to initialize tracer: %v", err)
	}
	defer shutdown(context.Background())
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
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			http.Error(w, "db unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})


	r.Post("/api/v1/register", authHandler.Register)
	r.Post("/api/v1/login", authHandler.Login)

	r.With(middleware.Auth(authSvc)).Post("/api/v1/shorten",h.Post)

	r.With(middleware.Auth(authSvc)).Get("/api/v1/my-urls", h.GetUserURLs)
	logger.Println("Server listening on :8080")

	srv := &http.Server{Addr: ":8080", Handler: otelhttp.NewHandler(r, "url-shortener")}
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
