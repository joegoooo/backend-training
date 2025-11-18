package form

import (
	"awesomeProject/internal/jwt"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Request struct {
	Title       string `json:"title" validate:"required"`
	Description string `json:"description" validate:"required"`
}

type UpdateRequest struct {
	ID          string `json:"id" validate:"required"`
	Title       string `json:"title" validate:"required"`
	Description string `json:"description" validate:"required"`
}

type DeleteRequest struct {
	ID string `json:"id" validate:"required"`
}

type Response struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	IsBookmarked bool      `json:"is_bookmarked"`
	CreatedAt    time.Time `json:"createdAt"`
}

//go:generate mockery --name=Store
type Store interface {
	Create(ctx context.Context, name, description string, authorId uuid.UUID) (Form, error)
	List(ctx context.Context) ([]Form, error)
	Update(ctx context.Context, id uuid.UUID, name, description string) (Form, error)
	Delete(ctx context.Context, id uuid.UUID) error
	IsBookmarked(ctx context.Context, formId, userId uuid.UUID) (bool, error)
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

	var req Request
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

	token, ok := ctx.Value(jwt.UserContextKey).(uuid.UUID)
	if !ok {
		h.logger.Error("Failed to get user ID from context")
		http.Error(w, "Invalid user context", http.StatusInternalServerError)
		return
	}

	newForm, err := h.store.Create(ctx, req.Title, req.Description, token)
	if err != nil {
		h.logger.Error("Failed to create form", zap.Error(err))
		http.Error(w, "Failed to create form", http.StatusInternalServerError)
		return
	}

	resp := Response{
		ID:          newForm.ID.String(),
		Title:       newForm.Title,
		Description: newForm.Description.String,
		CreatedAt:   newForm.CreatedAt.Time,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	forms, err := h.store.List(ctx)

	if err != nil {
		h.logger.Error("Failed to list forms", zap.Error(err))
		http.Error(w, "Failed to list forms", http.StatusInternalServerError)
		return
	}
	userId := ctx.Value(jwt.UserContextKey).(uuid.UUID)
	var resp []Response
	for _, form := range forms {
		bookmarked, err := h.store.IsBookmarked(ctx, form.ID, userId)
		if err != nil {
			h.logger.Error("Failed to check bookmark", zap.Error(err))
			http.Error(w, "Failed to check bookmark", http.StatusInternalServerError)
			return
		}
		resp = append(resp, Response{
			ID:           form.ID.String(),
			Title:        form.Title,
			Description:  form.Description.String,
			IsBookmarked: bookmarked,
			CreatedAt:    form.CreatedAt.Time,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req UpdateRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.logger.Error("Failed to decode request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	// validate request
	err = h.validator.Struct(req)
	if err != nil {
		h.logger.Error("Validation failed", zap.Error(err))
		http.Error(w, "Validation failed", http.StatusBadRequest)
		return
	}
	id, err := uuid.Parse(req.ID)
	updateForm, err := h.store.Update(ctx, id, req.Title, req.Description)

	if err != nil {
		h.logger.Error("Failed to update form", zap.Error(err))
		http.Error(w, "Failed to update form", http.StatusInternalServerError)
		return
	}

	resp := Response{
		ID:          updateForm.ID.String(),
		Title:       updateForm.Title,
		Description: updateForm.Description.String,
		CreatedAt:   updateForm.CreatedAt.Time,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req DeleteRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.logger.Error("Failed to decode request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	// validate request
	err = h.validator.Struct(req)
	if err != nil {
		h.logger.Error("Validation failed", zap.Error(err))
		http.Error(w, "Validation failed", http.StatusBadRequest)
		return
	}
	id, err := uuid.Parse(req.ID)
	err = h.store.Delete(ctx, id)
	if err != nil {
		h.logger.Error("Failed to delete form", zap.Error(err))
		http.Error(w, "Failed to delete form", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
