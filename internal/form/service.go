package form

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

type Querier interface {
	Create(ctx context.Context, arg CreateParams) (Form, error)
	List(ctx context.Context) ([]Form, error)
	Update(ctx context.Context, arg UpdateParams) (Form, error)
	Delete(ctx context.Context, id uuid.UUID) error
	IsBookmarked(ctx context.Context, arg IsBookmarkedParams) (bool, error)
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

func (s *Service) Create(ctx context.Context, name, description string, authorID uuid.UUID) (Form, error) {

	var result, err = s.queries.Create(ctx, CreateParams{
		Title:       name,
		Description: pgtype.Text{String: description, Valid: true},
		AuthorID:    pgtype.Text{String: authorID.String(), Valid: true}, // Renamed from AuthorEmail
	})
	if err != nil {
		s.logger.Error("Failed to create form", zap.Error(err))
		return Form{}, err
	}

	s.logger.Info("Created form", zap.String("form_id", result.ID.String()), zap.String("author_id", result.AuthorID.String))

	return result, nil
}

func (s *Service) List(ctx context.Context) ([]Form, error) {
	result, err := s.queries.List(ctx)
	if err != nil {
		s.logger.Error("Failed to list form", zap.Error(err))
		return []Form{}, err
	}

	return result, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, name, description string) (Form, error) {

	result, err := s.queries.Update(ctx, UpdateParams{
		ID:          id,
		Title:       name,
		Description: pgtype.Text{String: description, Valid: true},
	})
	if err != nil {
		s.logger.Error("Failed to update form", zap.Error(err))
		return Form{}, err
	}

	return result, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {

	err := s.queries.Delete(ctx, id)
	if err != nil {
		s.logger.Error("Failed to delete form", zap.Error(err))
		return err
	}
	return nil
}

func (s *Service) IsBookmarked(ctx context.Context, form_id, user_id uuid.UUID) (bool, error) {
	exists, err := s.queries.IsBookmarked(ctx, IsBookmarkedParams{
		FormID: form_id,
		UserID: user_id,
	})
	if err != nil {
		s.logger.Error("Failed to query bookmark", zap.Error(err))
		return false, err
	}

	return exists, err
}
