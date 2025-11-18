package jwt

import (
	"awesomeProject/internal/user"
	// "awesomeProject/internal/auth" // [REMOVED]
	"context"
	"encoding/json"
	"net/http"
	//"time" // [ADDED]

	"github.com/go-playground/validator/v10" // [ADDED]
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

type jwtService interface {
	// 接口來自 service.go
	Create(ctx context.Context, userID uuid.UUID, time pgtype.Timestamptz) (Jwt, error)
	Update(ctx context.Context, id uuid.UUID, isAvailable bool) (Jwt, error)
	// 'IsAvailable' 實現了輪換邏輯
	IsAvailable(ctx context.Context, id uuid.UUID) (Jwt, error)
	// 'New' 用於創建新的 Access Token
	New(ctx context.Context, id uuid.UUID, email string) (string, error)
}

type userService interface {
	GetByID(ctx context.Context, id uuid.UUID) (user.User, error)
}

// [REMOVED] type Request struct {}

type Handler struct {
	logger      *zap.Logger
	validator   *validator.Validate // [ADDED]
	jwtService  jwtService
	userService userService
}

// [MODIFIED] 注入 validator
func NewHandler(logger *zap.Logger, validator *validator.Validate, jwtService jwtService, userService userService) *Handler {
	return &Handler{
		logger:      logger,
		validator:   validator,
		jwtService:  jwtService,
		userService: userService,
	}
}

// [ADDED] Request 和 Response 結構
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required,uuid"`
}

type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// [MODIFIED] 實現 Refresh 函數
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. 解碼並驗證請求
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		h.logger.Error("Validation failed", zap.Error(err))
		http.Error(w, "Validation failed", http.StatusBadRequest)
		return
	}

	// 2. 解析 UUID
	tokenID, err := uuid.Parse(req.RefreshToken)
	if err != nil {
		h.logger.Error("Failed to parse UUID from refresh token", zap.Error(err))
		http.Error(w, "Invalid refresh token format", http.StatusBadRequest)
		return
	}

	// 3. 調用服務以輪換 token
	// 'IsAvailable' 會驗證舊 token、使其失效，並創建一個新 token
	newRefreshToken, err := h.jwtService.IsAvailable(ctx, tokenID)
	if err != nil {
		h.logger.Warn("Refresh token rotation failed", zap.Error(err))
		// 錯誤可能是 "already used", "expired", 或 "not found"
		http.Error(w, "Invalid or expired refresh token", http.StatusUnauthorized)
		return
	}

	// 4. 創建新的 Access Token
	// newRefreshToken 包含 UserEmail
	User, err := h.userService.GetByID(ctx, newRefreshToken.UserID)
	if err != nil {
		h.logger.Error("Failed to find User for new access token", zap.String("user_id", newRefreshToken.UserID.String()), zap.Error(err))
		http.Error(w, "Failed to create new access token", http.StatusInternalServerError)
		return
	}

	newAccessToken, err := h.jwtService.New(ctx, newRefreshToken.UserID, User.Email)
	if err != nil {
		h.logger.Error("Failed to create new access token after refresh", zap.Error(err))
		http.Error(w, "Failed to create new access token", http.StatusInternalServerError)
		return
	}

	// 5. 發送包含新 tokens 的響應
	resp := RefreshResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken.ID.String(), // 發送新 Refresh Token 的 ID
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
