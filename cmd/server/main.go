package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"url-shortener/internal/grpc_handler"
	"url-shortener/internal/handler"
	"url-shortener/internal/metrics"
	"url-shortener/internal/service"
	"url-shortener/internal/storage"
	pb "url-shortener/internal/proto"

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
	// gRPC

	go func() {
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			logger.Fatalf("gRPC listen error: %v", err)
		}
		grpcServer := grpc.NewServer()
		pb.RegisterURLServiceServer(grpcServer, grpc_handler.NewURLGRPCHandler(svc))
		reflection.Register(grpcServer)
		logger.Println("gRPC server listening on :50051")
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatalf("gRPC server error: %v", err)
		}
	}()

	// HTTP
	h := handler.NewURLHandler(svc, logger)
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	r := chi.NewRouter()
	r.Post("/api/v1/shorten", h.Post)
	r.Get("/{code}", h.Get)

	logger.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		logger.Fatalf("Server error: %v", err)
	}

}
