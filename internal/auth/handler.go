package auth

import (
	"awesomeProject/internal/auth/oauthprovider"
	"awesomeProject/internal/jwt" // [ADDED]
	"awesomeProject/internal/user"
	"context"
	"fmt"
	"net/http"
	"os"
	"time" // [ADDED]

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype" // [ADDED]
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type jwtService interface {
	New(ctx context.Context, userID uuid.UUID, email string) (string, error)
}
type userService interface {
	Create(ctx context.Context, email string) (user.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	GetByEmail(ctx context.Context, email string) (user.User, error)
}
type OAuthProvider interface {
	Name() string
	Config() *oauth2.Config
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
	GetUserInfo(ctx context.Context, token *oauth2.Token) (oauthprovider.UserInfo, error)
}

// [ADDED] Refresh Token 服務的接口
type refreshTokenService interface {
	Create(ctx context.Context, userID uuid.UUID, time pgtype.Timestamptz) (jwt.Jwt, error)
}

type Handler struct {
	logger      *zap.Logger
	baseURL     string
	jwtService  jwtService
	userService userService
	provider    map[string]OAuthProvider
	rtService   refreshTokenService // [ADDED]
}

// [MODIFIED] 注入 rtService
func NewHandler(logger *zap.Logger, baseURL string, jwtService jwtService, userService userService, rtService refreshTokenService) *Handler {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

	return &Handler{
		logger:      logger,
		baseURL:     baseURL,
		jwtService:  jwtService,
		userService: userService,
		rtService:   rtService, // [ADDED]
		provider: map[string]OAuthProvider{
			"google": oauthprovider.NewGoogleConfig(
				clientID,
				clientSecret,
				fmt.Sprintf("%s/api/oauth/google/callback", baseURL)),
		},
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	// ... (此函數未更改)
	providerName := r.PathValue("provider")
	provider := h.provider[providerName]
	if provider == nil {
		h.logger.Warn("No such provider", zap.String("provider", providerName))
		http.Error(w, "Unsupported OAuth2 provider", http.StatusBadRequest)
		return
	}

	redirectTo := r.URL.Query().Get("c")
	frontendRedirectTo := r.URL.Query().Get("r")
	if redirectTo == "" {
		redirectTo = fmt.Sprintf("%s/api/oauth/debug/token", h.baseURL)
	}
	if frontendRedirectTo != "" {
		redirectTo = fmt.Sprintf("%s?r=%s", redirectTo, frontendRedirectTo)
	}

	authURL := provider.Config().AuthCodeURL(redirectTo, oauth2.AccessTypeOffline)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
	h.logger.Info("Redirecting to Google OAuth2", zap.String("url", authURL))
}

func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {

	providerName := r.PathValue("provider")
	provider := h.provider[providerName]
	if provider == nil {
		h.logger.Warn("No such provider", zap.String("provider", providerName))
		http.Error(w, "Unsupported OAuth2 provider", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	redirectTo := state
	if redirectTo == "" {
		redirectTo = fmt.Sprintf("%s/api/oauth/debug/token", h.baseURL)
	}

	authError := r.URL.Query().Get("error")
	if authError != "" {
		redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, authError)
		h.logger.Warn("OAuth2 callback returned error", zap.String("error", authError))
		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, "missing_code")
		h.logger.Warn("Missing code in callback")
		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		return
	}
	token, err := provider.Exchange(r.Context(), code)
	if err != nil {
		redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, err)
		h.logger.Error("Failed to exchange code for token", zap.Error(err))
		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		return
	}

	userInfo, err := provider.GetUserInfo(r.Context(), token)
	if err != nil {
		redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, err)
		h.logger.Error("Failed to get user info", zap.Error(err))
		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		return
	}
	dbUser, err := h.userService.GetByEmail(r.Context(), userInfo.Email)
	if err != nil {
		// Assuming pgx.ErrNoRows or similar error, create new user
		h.logger.Warn("User not found, creating new user", zap.String("email", userInfo.Email))
		dbUser, err = h.userService.Create(r.Context(), userInfo.Email)
		if err != nil {
			redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, err)
			h.logger.Error("Failed to create user", zap.Error(err))
			http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
			return
		}
	}

	// 創建 Access Token
	jwtToken, err := h.jwtService.New(r.Context(), dbUser.ID, dbUser.Email)
	if err != nil {
		redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, err)
		h.logger.Error("Failed to create JWT token", zap.Error(err))
		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		return
	}

	// 檢查用戶是否存在，若不存在則創建
	exist, err := h.userService.ExistsByEmail(r.Context(), userInfo.Email)
	if err != nil {
		redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, err)
		h.logger.Error("Failed to check user existence", zap.Error(err)) // [MODIFIED] 錯誤日誌
		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		return
	}
	if !exist {
		_, err := h.userService.Create(r.Context(), userInfo.Email)
		if err != nil {
			h.logger.Error("Failed to create user", zap.Error(err))
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}
	}

	// [ADDED] 創建 Refresh Token（30 分鐘過期）
	newRefreshToken, err := h.rtService.Create(r.Context(), dbUser.ID, pgtype.Timestamptz{
		Time:  time.Now().Add(30 * time.Minute),
		Valid: true,
	})
	if err != nil {
		redirectTo = fmt.Sprintf("%s?error=%s", redirectTo, err)
		h.logger.Error("Failed to create refresh token", zap.Error(err))
		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
		return
	}

	// [MODIFIED] 在重定向 URL 中同時包含 access_token 和 refresh_token
	redirectTo = fmt.Sprintf("%s?access_token=%s&refresh_token=%s", redirectTo, jwtToken, newRefreshToken.ID.String())

	http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
	h.logger.Info("OAuth2 callback successful", zap.String("user_email", userInfo.Email))
}

func (h *Handler) DebugToken(w http.ResponseWriter, r *http.Request) {
	// ... (此函數未更改)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, err := w.Write([]byte(`{"message":"Login successful"}`))
	if err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
