package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"
	"url-shortener/internal/service"
	"url-shortener/internal/storage"
	"github.com/go-chi/chi/v5"
)

type URLHandler struct {
	service *service.URLService
	logger  *log.Logger
}

func NewURLHandler(service *service.URLService, logger *log.Logger) *URLHandler {
	return &URLHandler{service: service, logger: logger}
}

func (h *URLHandler) Post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	defer r.Body.Close()
	var req shortenRequest
	err := d.Decode(&req)
	if err != nil {
		h.logger.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if d.More() {
		http.Error(w, "only one JSON object allowed", http.StatusBadRequest)
		return
	}
	code, err := h.service.ShortenURL(ctx, req.URL)
	if errors.Is(err, service.ErrEmptyURL) || errors.Is(err, service.ErrInvalidURL) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		h.logger.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	shortURL := "http://" + r.Host + "/" + code
	resp := shortenResponse{Code: code, URL: shortURL}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		h.logger.Println(err)
		return
	}
}

func (h *URLHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	code := chi.URLParam(r, "code")
	if code == "" {
		http.Error(w, "Code not provided", http.StatusBadRequest)
		return
	}

	const maxRetries = 3
	var originalURL string
	var lastErr error
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		var err error
		originalURL, err = h.service.GetOriginalURL(ctx, code)
		if err == nil {
			http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
			return
		}
		lastErr = err
		
		if errors.Is(err, storage.ErrURLNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		
		if attempt < maxRetries-1 {
			time.Sleep(time.Duration((attempt+1)*10) * time.Millisecond)
		}
	}
	
	h.logger.Println(lastErr)
	w.WriteHeader(http.StatusInternalServerError)
}
