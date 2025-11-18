package main

import (
	"awesomeProject/databaseutil"
	"awesomeProject/handlerutil"
	"awesomeProject/internal"
	"awesomeProject/internal/auth"
	"awesomeProject/internal/bookmark"
	"awesomeProject/internal/form"
	"awesomeProject/internal/jwt"
	"awesomeProject/internal/user"
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

const baseURL = "http://localhost:8080"

func main() {
	_ = godotenv.Load()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer func(logger *zap.Logger) {
		_ = logger.Sync()
	}(logger)

	logger.Info("Starting backend service")

	err = databaseutil.MigrationUp("file://internal/database/migrations", "postgresql://postgres:password@localhost:5432/postgres?sslmode=disable", logger)
	if err != nil {
		logger.Fatal("Failed to run database migration", zap.Error(err))
	}

	poolConfig, err := pgxpool.ParseConfig("postgresql://postgres:password@localhost:5432/postgres?sslmode=disable")
	if err != nil {
		logger.Fatal("Failed to parse database URL", zap.Error(err))
	}

	dbPool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		logger.Fatal("Failed to create database connection pool", zap.Error(err))
	}
	defer dbPool.Close()

	validator := internal.NewValidator()

	formQuerier := form.New(dbPool)
	userQuerier := user.New(dbPool)
	jwtQuerier := jwt.New(dbPool)
	bookmarkQuerier := bookmark.New(dbPool)

	formService := form.NewService(logger, formQuerier)
	userService := user.NewService(logger, userQuerier)
	// [MODIFIED] Add dbPool argument, as required by the new service definition
	jwtService := jwt.NewService(logger, 15*time.Minute, jwtQuerier)
	bookmarkService := bookmark.NewService(logger, bookmarkQuerier)

	formHandler := form.NewHandler(logger, validator, formService)
	userHandler := user.NewHandler(logger, validator, userService)
	authHandler := auth.NewHandler(logger, baseURL, jwtService, userService, jwtService)
	jwtHandler := jwt.NewHandler(logger, validator, jwtService, userService)
	bookmarkHandler := bookmark.NewHandler(logger, validator, bookmarkService)

	basicMiddleware := handlerutil.NewMiddleware(logger, true)
	jwtMiddleware := jwt.NewMiddleware(logger, jwtService)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/forms", basicMiddleware.RecoverMiddleware(jwtMiddleware.HandlerFunc(formHandler.Create)))
	mux.HandleFunc("GET /api/forms", basicMiddleware.RecoverMiddleware(jwtMiddleware.HandlerFunc(formHandler.List)))
	mux.HandleFunc("PUT /api/forms", basicMiddleware.RecoverMiddleware(jwtMiddleware.HandlerFunc(formHandler.Update)))
	mux.HandleFunc("DELETE /api/forms", basicMiddleware.RecoverMiddleware(jwtMiddleware.HandlerFunc(formHandler.Delete)))
	mux.HandleFunc("POST /api/users", basicMiddleware.RecoverMiddleware(userHandler.Create))

	mux.HandleFunc("GET /api/oauth/{provider}", basicMiddleware.RecoverMiddleware(authHandler.Login))
	mux.HandleFunc("GET /api/oauth/{provider}/callback", basicMiddleware.RecoverMiddleware(authHandler.Callback))
	mux.HandleFunc("GET /api/oauth/debug/token", basicMiddleware.RecoverMiddleware(authHandler.DebugToken))

	// [ADDED] Add the new refresh token endpoint
	mux.HandleFunc("POST /api/auth/refresh", basicMiddleware.RecoverMiddleware(jwtHandler.Refresh))

	mux.HandleFunc("GET /api/bookmarks", basicMiddleware.RecoverMiddleware(jwtMiddleware.HandlerFunc(bookmarkHandler.Toggle)))
	//mux.HandleFunc("POST /api/bookmarks", basicMiddleware.RecoverMiddleware(jwtMiddleware.HandlerFunc(bookmarkHandler.UserBookmarksCount)))
	mux.HandleFunc("POST /api/bookmarks", basicMiddleware.RecoverMiddleware(jwtMiddleware.HandlerFunc(bookmarkHandler.FormBookmarksCount)))

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	logger.Info("Backend started on :8080")

	err = server.ListenAndServe()
	if err != nil {
		logger.Fatal("Failed to start HTTP server", zap.Error(err))
	}
}
