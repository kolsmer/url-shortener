package main

import (
	"log"
	"net/http"
	"os"
	"url-shortener/internal/handler"
	"url-shortener/internal/metrics"
	"url-shortener/internal/service"
	"url-shortener/internal/storage"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	metrics.Init()
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	db := storage.SetupPostgres()
	redis := storage.SetupRedis()

	baseStorage := storage.NewSQLStorage(db)
	cachedStorage := storage.NewCachedStorage(baseStorage, redis)
	svc := service.NewURLService(cachedStorage)
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
