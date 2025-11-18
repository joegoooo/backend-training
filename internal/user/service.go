package user

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Querier interface {
	Create(ctx context.Context, email string) (User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	GetByID(ctx context.Context, ID uuid.UUID) (User, error)
}

type Service struct {
	logger  *zap.Logger
	queries Querier
}

func NewService(logger *zap.Logger, querier Querier) *Service {
	return &Service{
		logger:  logger,
		queries: querier,
	}
}

func (s *Service) Create(ctx context.Context, email string) (User, error) {
	result, err := s.queries.Create(ctx, email)
	if err != nil {
		s.logger.Error("Failed to create user", zap.Error(err))
		return User{}, err
	}

	s.logger.Info("Created user", zap.String("email", result.Email))

	return result, nil
}

func (s *Service) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	result, err := s.queries.ExistsByEmail(ctx, email)
	if err != nil {
		s.logger.Error("Failed to find the email", zap.Error(err))
		return false, err
	}
	return result, nil
}

func (s *Service) GetByEmail(ctx context.Context, email string) (User, error) {
	result, err := s.queries.GetByEmail(ctx, email)
	if err != nil {
		s.logger.Warn("Failed to find user by email", zap.String("email", email), zap.Error(err))
		return User{}, err
	}
	return result, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	result, err := s.queries.GetByID(ctx, id)
	if err != nil {
		s.logger.Warn("Failed to find user by ID", zap.String("ID", id.String()), zap.Error(err))
		return User{}, err
	}
	return result, err
}
