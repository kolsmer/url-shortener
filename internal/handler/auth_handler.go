package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"url-shortener/internal/service"
	"url-shortener/internal/storage"
)



type AuthHandler struct {
	svc *service.AuthService
	logger *log.Logger
}

func NewAuthHandler(svc *service.AuthService, logger *log.Logger) *AuthHandler {
	return &AuthHandler{
		svc: svc,
		logger: logger,
	}
}
	
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Email	string `json:"email"`
		Password string `json:"password"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	err = h.svc.RegisterUser(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, storage.ErrEmailAlreadyExists) {
			http.Error(w, "User already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to register user", http.StatusInternalServerError)
		h.logger.Printf("Error registering user: %v", err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Email	string `json:"email"`
		Password string `json:"password"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	token, err := h.svc.LoginUser(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, "Failed to login user", http.StatusUnauthorized)
		h.logger.Printf("Error logging in user: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	type loginResponse struct {
    Token string `json:"token"`
	}
	err = json.NewEncoder(w).Encode(loginResponse{Token: token})
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}