package jwt

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

const secret = "default_secret"

type Querier interface {
	Create(ctx context.Context, arg CreateParams) (Jwt, error)
	Update(ctx context.Context, arg UpdateParams) (Jwt, error)
	IsAvailable(ctx context.Context, id uuid.UUID) (Jwt, error)
}
type Service struct {
	logger     *zap.Logger
	expiration time.Duration
	queries    Querier
}

func NewService(logger *zap.Logger, expiration time.Duration, querier Querier) *Service {
	return &Service{
		logger:     logger,
		expiration: expiration,
		queries:    querier,
	}
}

type claims struct {
	Message string
	Id      uuid.UUID
	Email   string
	jwt.RegisteredClaims
}

func (s Service) New(ctx context.Context, id uuid.UUID, email string) (string, error) {
	jwtID := uuid.New()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Message: "This is a Backend-Training JWT token",
		Id:      id,
		Email:   email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "Backend-Training",
			Subject:   "Backend-Training Token",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expiration)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        jwtID.String(),
		},
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		s.logger.Error("Failed to sign token", zap.Error(err))
		return "", err
	}

	s.logger.Debug("Generated new JWT token")

	return tokenString, nil
}

func (s Service) Parse(ctx context.Context, tokenString string) (uuid.UUID, error) {
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	token, err := jwt.ParseWithClaims(tokenString, &claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenMalformed):
			s.logger.Warn("Failed to parse JWT token due to malformed structure, this is not a JWT token", zap.String("error", err.Error()))
			return uuid.UUID{}, err
		case errors.Is(err, jwt.ErrSignatureInvalid):
			s.logger.Warn("Failed to parse JWT token due to invalid signature", zap.String("error", err.Error()))
			return uuid.UUID{}, err
		case errors.Is(err, jwt.ErrTokenExpired):
			expiredTime, getErr := token.Claims.GetExpirationTime()
			if getErr != nil {
				s.logger.Warn("Failed to parse JWT token due to expired timestamp", zap.String("error", err.Error()))
			} else {
				s.logger.Warn("Failed to parse JWT token due to expired timestamp", zap.String("error", err.Error()), zap.Time("expired_at", expiredTime.Time))
			}

			return uuid.UUID{}, err
		case errors.Is(err, jwt.ErrTokenNotValidYet):
			notBeforeTime, getErr := token.Claims.GetNotBefore()
			if getErr != nil {
				s.logger.Warn("Failed to parse JWT token due to not valid yet timestamp", zap.String("error", err.Error()))
			} else {
				s.logger.Warn("Failed to parse JWT token due to not valid yet timestamp", zap.String("error", err.Error()), zap.Time("not_valid_yet", notBeforeTime.Time))
			}

			return uuid.UUID{}, err
		default:
			s.logger.Error("Failed to parse or validate JWT token", zap.Error(err))
			return uuid.UUID{}, err
		}
	}

	c, ok := token.Claims.(*claims)
	if !ok {
		s.logger.Warn("Invalid JWT token claims")
		return uuid.UUID{}, errors.New("invalid token claims")
	}

	s.logger.Debug("Parsed JWT token successfully")

	// [MODIFIED] 返回 Email 而不是 Message
	return c.Id, nil
}

func (s Service) Create(ctx context.Context, userID uuid.UUID, time pgtype.Timestamptz) (Jwt, error) {
	result, err := s.queries.Create(ctx, CreateParams{
		UserID:         userID,
		ExpirationTime: time,
	})
	if err != nil {
		s.logger.Error("Failed to create JWT", zap.Error(err))
		return Jwt{}, err
	}
	s.logger.Info("Create JWT", zap.String("jwt_id", result.ID.String()), zap.String("user_id", result.UserID.String()))
	return result, err
}

func (s Service) Update(ctx context.Context, id uuid.UUID, isAvailable bool) (Jwt, error) {
	result, err := s.queries.Update(ctx, UpdateParams{
		ID:          id,
		IsAvailable: isAvailable,
	})
	if err != nil {
		s.logger.Error("Failed to update jwt", zap.Error(err))
		return Jwt{}, err
	}
	return result, nil
}

// IsAvailable [MODIFIED] 此函數現在實現了 Refresh Token 的輪換邏輯
func (s Service) IsAvailable(ctx context.Context, id uuid.UUID) (Jwt, error) {
	result, err := s.queries.IsAvailable(ctx, id)
	if err != nil {
		s.logger.Error("Failed to query jwt", zap.String("jwt_id", id.String()), zap.Error(err))
		return Jwt{}, err // 可能是 pgx.ErrNoRows，表示 token 不存在
	}

	// 檢查1：是否已被使用
	if !result.IsAvailable {
		s.logger.Warn("Refresh token already used", zap.String("jwt_id", id.String()))
		return Jwt{}, errors.New("refresh token already used")
	}

	// 檢查2：是否已過期
	if result.ExpirationTime.Time.Before(time.Now()) {
		s.logger.Warn("Refresh token expired", zap.String("jwt_id", id.String()), zap.Time("expiration_time", result.ExpirationTime.Time))
		// 將過期的 token 標記為不可用
		_, updateErr := s.Update(ctx, id, false)
		if updateErr != nil {
			s.logger.Error("Failed to invalidate expired token", zap.Error(updateErr))
		}
		return Jwt{}, errors.New("refresh token expired")
	}

	// Token 有效，將其標記為不可用（僅限一次使用）
	_, err = s.Update(ctx, id, false)
	if err != nil {
		s.logger.Error("Failed to invalidate token during rotation", zap.String("jwt_id", id.String()), zap.Error(err))
		return Jwt{}, err
	}

	// 建立一個新的 Refresh Token（30 分鐘後過期）
	newJwt, err := s.Create(ctx, result.UserID, pgtype.Timestamptz{Time: time.Now().Add(30 * time.Minute), Valid: true})
	if err != nil {
		s.logger.Error("Failed to create new refresh token during rotation", zap.Error(err))
		return Jwt{}, err
	}

	// 返回 *新的* Refresh Token
	return newJwt, nil
}
