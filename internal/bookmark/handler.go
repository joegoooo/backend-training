package bookmark

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ToggleRequest struct {
	UserID string `json:"user_id"`
	FormID string `json:"form_id"`
}

type GetFormRequest struct {
	UserID string `json:"user_id"`
}
type FormCountRequest struct {
	FormID string `json:"form_id"`
}

type ToggleResponse struct {
	UserID string `json:"user_id"`
	FormID string `json:"form_id"`
}

type GetFormResponse struct {
	FormID    string    `json:"form_id"`
	CreatedAt time.Time `json:"createdAt"`
}

type ExistResponse struct {
	UserID string `json:"user_id"`
	FormID string `json:"form_id"`
	Exists bool   `json:"exist_or_not"`
}

type UserBookmarksCount struct {
	UserID string `json:"user_id"`
	Count  int64  `json:"number_of_bookmarks"`
}

type FormBookmarksCount struct {
	FormID string `json:"form_id"`
	Count  int64  `json:"number_of_bookmarks"`
}

type Store interface {
	ToggleBookmark(ctx context.Context, userID, formID uuid.UUID) (bool, error)
	GetFormsByUserID(ctx context.Context, userID uuid.UUID) ([]GetFormsByUserIDRow, error)
	SpecificForm(ctx context.Context, userID, formID uuid.UUID) (bool, error)
	CountBookmark(ctx context.Context, userID uuid.UUID) (int64, error)
	FormCount(ctx context.Context, formID uuid.UUID) (int64, error)
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

func (h *Handler) Toggle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ToggleRequest
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

	//token, ok := ctx.Value(jwt.UserContextKey).(uuid.UUID)
	//if !ok {
	//	h.logger.Error("Failed to get user ID from context")
	//	http.Error(w, "Invalid user context", http.StatusInternalServerError)
	//	return
	//}
	userID, err := uuid.Parse(req.UserID)
	formID, err := uuid.Parse(req.FormID)
	_, err = h.store.ToggleBookmark(ctx, userID, formID)

	if err != nil {
		h.logger.Error("Failed to create or delete form", zap.Error(err))
		http.Error(w, "Failed to create or delete form", http.StatusInternalServerError)
		return
	}

	resp := ToggleResponse{
		UserID: userID.String(),
		FormID: formID.String(),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}

}

func (h *Handler) GetFormsByUserID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req GetFormRequest
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
	userID, err := uuid.Parse(req.UserID)
	forms, err := h.store.GetFormsByUserID(ctx, userID)
	if err != nil {
		h.logger.Error("Fail to get forms", zap.Error(err))
		http.Error(w, "Fail to get forms", http.StatusBadRequest)
		return
	}
	var resp []GetFormResponse
	for _, form := range forms {
		resp = append(resp, GetFormResponse{
			FormID:    form.FormID.String(),
			CreatedAt: form.CreatedAt.Time,
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

func (h *Handler) SpecificForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ToggleRequest
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
	userID, err := uuid.Parse(req.UserID)
	formID, err := uuid.Parse(req.FormID)
	exists, err := h.store.SpecificForm(ctx, userID, formID)

	if err != nil {
		h.logger.Error("Fail to check specific form", zap.Error(err))
		http.Error(w, "Fail to check specifc form", http.StatusBadRequest)
		return
	}
	resp := ExistResponse{
		UserID: userID.String(),
		FormID: formID.String(),
		Exists: exists,
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

func (h *Handler) UserBookmarksCount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req GetFormRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.logger.Error("Failed to decode request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	userID, err := uuid.Parse(req.UserID)
	count, err := h.store.CountBookmark(ctx, userID)
	if err != nil {
		h.logger.Error("Fail to count bookmarks", zap.Error(err))
		http.Error(w, "Fail to count bookmarks", http.StatusBadRequest)
		return
	}

	resp := UserBookmarksCount{
		UserID: userID.String(),
		Count:  count,
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

func (h *Handler) FormBookmarksCount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req FormCountRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.logger.Error("Failed to decode request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	formID, err := uuid.Parse(req.FormID)
	count, err := h.store.FormCount(ctx, formID)
	if err != nil {
		h.logger.Error("Fail to count bookmarks of form", zap.Error(err))
		http.Error(w, "Fail to count bookmarks of form", http.StatusBadRequest)
		return
	}
	resp := FormBookmarksCount{
		FormID: formID.String(),
		Count:  count,
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
