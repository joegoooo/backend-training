package user

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	//"github.com/google/uuid"
	"go.uber.org/zap"
)

type CreateRequest struct {
	Email string `json:"email" validate:"required"`
}
type Response struct {
	Id        string    `json:"id"` // Added
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

type Store interface {
	Create(ctx context.Context, email string) (User, error)
}

type Handler struct {
	logger    *zap.Logger
	validator *validator.Validate
	store     Store
}

func NewHandler(logger *zap.Logger, validator *validator.Validate, store Store) *Handler {
	return &Handler{
		logger:    logger,
		validator: validator,
		store:     store,
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.logger.Error("Failed to decode request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
	}
	// validate request
	err = h.validator.Struct(req)
	if err != nil {
		h.logger.Error("Validation failed", zap.Error(err))
		http.Error(w, "Validation failed", http.StatusBadRequest)
		return
	}

	newUser, err := h.store.Create(ctx, req.Email)
	if err != nil {
		h.logger.Error("Failed to create user", zap.Error(err))
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	resp := Response{
		Id:        newUser.ID.String(), // Added
		Email:     newUser.Email,
		CreatedAt: newUser.CreatedAt.Time,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
