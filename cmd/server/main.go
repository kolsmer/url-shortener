package main

import (
	""
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
	"url-shortener/internal/handler"
	"url-shortener/internal/metrics"
	"url-shortener/internal/service"
	"url-shortener/internal/storage"
)

func main() {
	metrics.Init()
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	db := storage.SetupPostgres()
	redis := storage.SetupRedis()

	baseStorage := storage.NewSQLStorage(db)
	cachedStorage := storage.NewCachedStorage(redis)

	service := service.NewURLService(cachedStorage)
	h := handler.NewURLHandler(service, logger)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/create", handler.URLHandler{}.Post)
	mux.HandleFunc("/", handler.URLHandler{}.Get)

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		return
	}

}
