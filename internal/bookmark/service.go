package bookmark

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

//go:generate mockery --name=Querier
type Querier interface {
	Create(ctx context.Context, arg CreateParams) (Bookmark, error)
	Delete(ctx context.Context, arg DeleteParams) error
	Exist(ctx context.Context, arg ExistParams) (bool, error)
	GetFormsByUserID(ctx context.Context, userID uuid.UUID) ([]GetFormsByUserIDRow, error)
	CountBookmark(ctx context.Context, userID uuid.UUID) (int64, error)
	FormCount(ctx context.Context, formID uuid.UUID) (int64, error)
}

type Service struct {
	logger  *zap.Logger
	queries Querier
}

func NewService(logger *zap.Logger, queries Querier) *Service {
	return &Service{
		logger:  logger,
		queries: queries,
	}
}

func (s Service) GetFormsByUserID(ctx context.Context, userID uuid.UUID) ([]GetFormsByUserIDRow, error) {
	forms, err := s.queries.GetFormsByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get bookmarked forms", zap.Error(err))
		return nil, err
	}
	return forms, nil
}

func (s Service) ToggleBookmark(ctx context.Context, userID, formID uuid.UUID) (bool, error) {
	exists, err := s.queries.Exist(ctx, ExistParams{
		UserID: userID,
		FormID: formID,
	})
	if err != nil {
		s.logger.Error("Failed to check bookmark existence", zap.Error(err))
		return false, err
	}
	if exists {
		err = s.queries.Delete(ctx, DeleteParams{
			UserID: userID,
			FormID: formID,
		})
		if err != nil {
			s.logger.Error("Failed to remove bookmark", zap.Error(err))
			return false, err
		}
		s.logger.Info("Removed bookmark", zap.String("user_id", userID.String()), zap.String("form_id", formID.String()))
		return false, nil
	} else {
		_, err = s.queries.Create(ctx, CreateParams{
			UserID: userID,
			FormID: formID,
		})
		if err != nil {
			s.logger.Error("Failed to add bookmark", zap.Error(err))
			return false, err
		}
		s.logger.Info("Added bookmark", zap.String("user_id", userID.String()), zap.String("form_id", formID.String()))
		return true, nil
	}
}

func (s Service) SpecificForm(ctx context.Context, userID, formID uuid.UUID) (bool, error) {
	exists, err := s.queries.Exist(ctx, ExistParams{
		UserID: userID,
		FormID: formID,
	})

	if err != nil {
		s.logger.Error("Failed to check specific bookmark", zap.Error(err))
		return false, err
	}

	return exists, err
}

func (s Service) CountBookmark(ctx context.Context, userID uuid.UUID) (int64, error) {
	formCount, err := s.queries.CountBookmark(ctx, userID)
	if err != nil {
		s.logger.Error("Fail to count bookmarks")
		return 0, err
	}
	return formCount, err
}

func (s Service) FormCount(ctx context.Context, formID uuid.UUID) (int64, error) {
	userCount, err := s.queries.FormCount(ctx, formID)
	if err != nil {
		s.logger.Error("Fail to count form bookmarked by how many users")
		return 0, err
	}
	return userCount, err
}
