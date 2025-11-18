package form_test

import (
	"awesomeProject/internal/form"
	"awesomeProject/internal/form/mocks"
	"awesomeProject/internal/jwt"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

func TestHandler_Create(t *testing.T) {
	tests := []struct {
		name         string
		formID       uuid.UUID
		userID       uuid.UUID    // ID of the user making the request
		reqBody      form.Request // Customize based on actual request structure
		customBody   []byte       // Optional raw body for more complex cases
		setMock      func(store *mocks.Store, formID uuid.UUID)
		expectStatus int
	}{
		{
			name:   "Successful form creation",
			formID: uuid.New(),
			userID: uuid.New(),
			reqBody: form.Request{
				Title:       "Test Form",
				Description: "This is a test form",
			},
			setMock: func(store *mocks.Store, formID uuid.UUID) {
				store.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(form.Form{
					ID:          formID,
					Title:       "Test Form",
					Description: pgtype.Text{String: "This is a test form", Valid: true},
				}, nil)
			},
			expectStatus: 201,
		},
		{
			name:   "Missing title",
			formID: uuid.New(),
			userID: uuid.New(),
			reqBody: form.Request{
				Description: "This is a test form without title",
			},
			setMock:      func(store *mocks.Store, formID uuid.UUID) {},
			expectStatus: 400,
		},
		{
			name:   "Database error on creation",
			formID: uuid.New(),
			userID: uuid.New(),
			reqBody: form.Request{
				Title:       "Test Form",
				Description: "This is a test form",
			},
			setMock: func(store *mocks.Store, formID uuid.UUID) {
				store.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(form.Form{}, errors.New("database error"))
			},
			expectStatus: 500,
		},
		{
			name:   "Invalid description type",
			formID: uuid.New(),
			userID: uuid.New(),
			customBody: []byte(`{
				"title": "Test Form",
				"description": 12345
			}`),
			setMock:      func(store *mocks.Store, formID uuid.UUID) {},
			expectStatus: 400,
		},
		{
			name:         "Empty request body",
			formID:       uuid.New(),
			userID:       uuid.New(),
			customBody:   []byte(`{}`),
			setMock:      func(store *mocks.Store, formID uuid.UUID) {},
			expectStatus: 400,
		},
	}

	logger := zaptest.NewLogger(t)
	v := validator.New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := mocks.NewStore(t)
			tt.setMock(store, tt.formID)

			handler := form.NewHandler(logger, v, store)
			var rawBody []byte
			if tt.customBody != nil {
				rawBody = tt.customBody
			} else {
				rawBody, _ = json.Marshal(tt.reqBody)
			}

			r := httptest.NewRequest(http.MethodPost, "/api/forms", bytes.NewBuffer(rawBody))
			w := httptest.NewRecorder()

			r = r.WithContext(context.WithValue(r.Context(), jwt.UserContextKey, tt.userID))
			handler.Create(w, r)

			assert.Equalf(t, tt.expectStatus, w.Result().StatusCode, "Expected status code to match, Expected %d, got %d", tt.expectStatus, w.Result().StatusCode)
		})
	}
}

func TestHandler_List(t *testing.T) {
	forms := []form.Form{
		{
			ID:          uuid.New(),
			Title:       "title1",
			Description: pgtype.Text{String: "this is description 1", Valid: true},
			AuthorID:    pgtype.Text{String: "author_id", Valid: true},
		},
		{
			ID:          uuid.New(),
			Title:       "title2",
			Description: pgtype.Text{String: "this is description 2", Valid: true},
			AuthorID:    pgtype.Text{String: "author_id", Valid: true},
		},
		{
			ID:          uuid.New(),
			Title:       "title3",
			Description: pgtype.Text{String: "this is description 3", Valid: true},
			AuthorID:    pgtype.Text{String: "author_id", Valid: true},
		},
	}
	tests := []struct {
		name         string
		formID       uuid.UUID
		userID       uuid.UUID // ID of the user making the request
		setMock      func(store *mocks.Store, formID uuid.UUID)
		expectStatus int
	}{
		{
			name:   "Successful list forms",
			formID: uuid.New(),
			userID: uuid.New(),
			setMock: func(store *mocks.Store, formID uuid.UUID) {
				store.On("IsBookmarked", mock.Anything, mock.Anything, mock.Anything).Return(
					true, nil)
				store.On("List", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
					forms, nil)
			},
			expectStatus: 200,
		},
		{
			name:   "database error",
			formID: uuid.New(),
			userID: uuid.New(),
			setMock: func(store *mocks.Store, formID uuid.UUID) {
				store.On("List", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
					[]form.Form{}, errors.New("database error"))
			},
			expectStatus: 500,
		},
	}
	logger := zaptest.NewLogger(t)
	v := validator.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := mocks.NewStore(t)
			tt.setMock(store, tt.formID)

			handler := form.NewHandler(logger, v, store)
			var rawBody []byte
			r := httptest.NewRequest(http.MethodGet, "/api/forms", bytes.NewBuffer(rawBody))
			w := httptest.NewRecorder()

			r = r.WithContext(context.WithValue(r.Context(), jwt.UserContextKey, tt.userID))
			handler.List(w, r)

			assert.Equalf(t, tt.expectStatus, w.Result().StatusCode, "Expected status code to match, Expected %d, got %d", tt.expectStatus, w.Result().StatusCode)
		})
	}

}

func TestHandler_Update(t *testing.T) {
	testformID := uuid.New()
	tests := []struct {
		name         string
		formID       uuid.UUID
		userID       uuid.UUID          // ID of the user making the request
		reqBody      form.UpdateRequest // Customize based on actual request structure
		customBody   []byte             // Optional raw body for more complex cases
		setMock      func(store *mocks.Store, formID uuid.UUID)
		expectStatus int
	}{
		{
			name:   "Successful form update",
			formID: uuid.New(),
			userID: uuid.New(),
			reqBody: form.UpdateRequest{
				ID:          testformID.String(),
				Title:       "Update Form",
				Description: "This is an updated form",
			},
			setMock: func(store *mocks.Store, formID uuid.UUID) {
				store.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(form.Form{
					ID:          testformID,
					Title:       "Update Form",
					Description: pgtype.Text{String: "This is an updated form", Valid: true},
				}, nil)
			},
			expectStatus: 200,
		},
		{
			name:   "Missing title",
			formID: uuid.New(),
			userID: uuid.New(),
			reqBody: form.UpdateRequest{
				Description: "This is a test form without title",
			},
			setMock:      func(store *mocks.Store, formID uuid.UUID) {},
			expectStatus: 400,
		},
		{
			name:   "Database error on creation",
			formID: uuid.New(),
			userID: uuid.New(),
			reqBody: form.UpdateRequest{
				ID:          testformID.String(),
				Title:       "Test Form",
				Description: "This is a test form",
			},
			setMock: func(store *mocks.Store, formID uuid.UUID) {
				store.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(form.Form{}, errors.New("database error"))
			},
			expectStatus: 500,
		},
		{
			name:   "Invalid description type",
			formID: uuid.New(),
			userID: uuid.New(),
			customBody: []byte(`{
				"title": "Test Form",
				"description": 12345
			}`),
			setMock:      func(store *mocks.Store, formID uuid.UUID) {},
			expectStatus: 400,
		},
		{
			name:         "Empty request body",
			formID:       uuid.New(),
			userID:       uuid.New(),
			customBody:   []byte(`{}`),
			setMock:      func(store *mocks.Store, formID uuid.UUID) {},
			expectStatus: 400,
		},
	}

	logger := zaptest.NewLogger(t)
	v := validator.New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := mocks.NewStore(t)
			tt.setMock(store, tt.formID)

			handler := form.NewHandler(logger, v, store)
			var rawBody []byte
			if tt.customBody != nil {
				rawBody = tt.customBody
			} else {
				rawBody, _ = json.Marshal(tt.reqBody)
			}

			r := httptest.NewRequest(http.MethodPost, "/api/forms", bytes.NewBuffer(rawBody))
			w := httptest.NewRecorder()

			r = r.WithContext(context.WithValue(r.Context(), jwt.UserContextKey, tt.userID))
			handler.Update(w, r)

			assert.Equalf(t, tt.expectStatus, w.Result().StatusCode, "Expected status code to match, Expected %d, got %d", tt.expectStatus, w.Result().StatusCode)
		})
	}
}
