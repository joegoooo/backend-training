package bookmark_test

import (
	"awesomeProject/internal/bookmark"
	"awesomeProject/internal/bookmark/mocks"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

func TestService_Toggle(t *testing.T) {
	testUserID := uuid.New()
	testFormID := uuid.New()
	tests := []struct {
		name         string
		formID       uuid.UUID
		userID       uuid.UUID // ID of the user making the request
		customBody   []byte    // Optional raw body for more complex cases
		setMock      func(querier *mocks.Querier, formID, userID uuid.UUID)
		expectStatus bool
		//expectErr    bool
	}{
		{
			name:       "Exist a bookmark",
			formID:     testFormID,
			userID:     testUserID,
			customBody: nil,
			setMock: func(querier *mocks.Querier, formID, userID uuid.UUID) {
				querier.On("Exist", mock.Anything, bookmark.ExistParams{
					FormID: testFormID,
					UserID: testUserID,
				}).Return(false, nil)

				querier.On("Create", mock.Anything, bookmark.CreateParams{
					FormID: testFormID,
					UserID: testUserID,
				}).Return(bookmark.Bookmark{}, nil)
			},
			expectStatus: true,
		},
		{
			name:   "User doesn't bookmark the form",
			formID: testFormID,
			userID: testUserID,
			setMock: func(querier *mocks.Querier, formID, userID uuid.UUID) {
				querier.On("Exist", mock.Anything, bookmark.ExistParams{
					FormID: testFormID,
					UserID: testUserID,
				}).Return(true, nil)
				querier.On("Delete", mock.Anything, bookmark.DeleteParams{
					FormID: testFormID,
					UserID: testUserID,
				}).Return(nil)
			},
			expectStatus: false,
		},
		{
			name:   "Fail on Create call",
			userID: testUserID,
			formID: testFormID,
			setMock: func(querier *mocks.Querier, formID, userID uuid.UUID) {
				// 1. Mock Exist
				querier.On("Exist", mock.Anything, mock.Anything).
					Return(false, nil)
				// 2. Mock Create to return an error
				querier.On("Create", mock.Anything, mock.Anything).
					Return(bookmark.Bookmark{}, errors.New("database error"))
			},
			expectStatus: false, // Should return false on error
		},
		{
			name:   "Fail on Delete call",
			userID: testUserID,
			formID: testFormID,
			setMock: func(querier *mocks.Querier, formID, userID uuid.UUID) {
				// 1. Mock Exist
				querier.On("Exist", mock.Anything, mock.Anything).
					Return(true, nil)
				// 2. Mock Delete to return an error
				querier.On("Delete", mock.Anything, mock.Anything).
					Return(errors.New("database error"))
			},
			expectStatus: false, // Error expected
		},
	}
	logger := zaptest.NewLogger(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier := mocks.NewQuerier(t)
			tt.setMock(querier, tt.formID, tt.userID)
			service := bookmark.NewService(logger, querier)

			created, _ := service.ToggleBookmark(context.Background(), tt.userID, tt.formID)
			assert.Equalf(t, tt.expectStatus, created, "Expected status code to match, Expected %d, got %d", tt.expectStatus, created)
		})
	}
}
