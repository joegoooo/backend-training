package auth

import (
	"awesomeProject/internal/user"
	//"awesomeProject/internal/user"
	"context"
	"net/http"

	//"github.com/google/uuid"
	"go.uber.org/zap"
)

const mockToken = "9ffe73a2-28c7-48fd-b049-0dc3a5d09135"
const UserContextKey = "user-id"

type Middleware struct {
	logger *zap.Logger
	store  userStore
}

type userStore interface {
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	GetByEmail(ctx context.Context, email string) (user.User, error)
}

func NewMiddleware(logger *zap.Logger, store userStore) *Middleware {
	return &Middleware{
		logger: logger,
		store:  store,
	}
}

func (m Middleware) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")

		if token == "" {
			m.logger.Warn("Missing Authorization header")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		exist, err := m.store.ExistsByEmail(r.Context(), token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if !exist {
			m.logger.Warn("Email doesn't exist")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		user, err := m.store.GetByEmail(r.Context(), token)
		if err != nil {
			m.logger.Warn("Failed to get user")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := r.Context()

		ctx = context.WithValue(ctx, UserContextKey, user.ID)

		next(w, r.WithContext(ctx))
	}
}
