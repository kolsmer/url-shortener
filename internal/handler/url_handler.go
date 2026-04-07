package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"url-shortener/internal/service"
	"url-shortener/internal/storage"
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
	if r.URL.Path == "/" {
		http.NotFound(w, r)
		return
	}
	code := strings.TrimPrefix(r.URL.Path, "/")
	if code == "" {
		http.Error(w, "Code not provided", http.StatusBadRequest)
		return
	}

	originalURL, err := h.service.GetOriginalURL(ctx, code)
	if errors.Is(err, storage.ErrURLNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)

}
