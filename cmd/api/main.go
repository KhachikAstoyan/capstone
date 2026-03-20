package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/KhachikAstoyan/capstone/internal/api"
	"github.com/KhachikAstoyan/capstone/internal/api/auth"
	authhttp "github.com/KhachikAstoyan/capstone/internal/api/auth/http"
	authrepo "github.com/KhachikAstoyan/capstone/internal/api/auth/repository"
	authservice "github.com/KhachikAstoyan/capstone/internal/api/auth/service"
	problemshttp "github.com/KhachikAstoyan/capstone/internal/api/problems/http"
	problemsrepo "github.com/KhachikAstoyan/capstone/internal/api/problems/repository"
	problemsservice "github.com/KhachikAstoyan/capstone/internal/api/problems/service"
	"github.com/KhachikAstoyan/capstone/internal/api/rbac"
	rbachttp "github.com/KhachikAstoyan/capstone/internal/api/rbac/http"
	rbacrepo "github.com/KhachikAstoyan/capstone/internal/api/rbac/repository"
	rbacservice "github.com/KhachikAstoyan/capstone/internal/api/rbac/service"
	"github.com/KhachikAstoyan/capstone/pkg/database"
	"github.com/KhachikAstoyan/capstone/pkg/logger"
	"github.com/KhachikAstoyan/capstone/pkg/migrations"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.uber.org/zap"
)

func main() {
	// Load configuration first
	cfg, err := api.LoadConfig()
	if err != nil {
		tempLog := logger.Init("production")
		tempLog.Fatal("Failed to load config", zap.Error(err))
	}

	// Initialize logger based on environment
	log := logger.Init(cfg.Environment)
	defer log.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Add logger to context
	ctx = logger.WithLogger(ctx, log)

	log.Info("Starting API server", zap.String("environment", cfg.Environment))

	// Connect to API database
	db := database.MustConnect(ctx, database.Config{
		URL: cfg.DatabaseURL,
	})
	defer db.Close()

	// Convert to absolute path
	absPath, err := filepath.Abs(cfg.MigrationsPath)
	if err != nil {
		log.Fatal("Failed to get absolute path for migrations", zap.Error(err))
	}

	// Run migrations
	log.Info("Running migrations", zap.String("path", absPath))
	if err := migrations.RunMigrations(db, absPath); err != nil {
		log.Fatal("Failed to run migrations", zap.Error(err))
	}

	log.Info("Migrations completed successfully")

	// Seed core RBAC permissions and roles
	log.Info("Seeding core RBAC data")
	if err := rbac.SeedCoreRBAC(ctx, db); err != nil {
		log.Fatal("Failed to seed RBAC data", zap.Error(err))
	}
	log.Info("RBAC seeding completed successfully")

	jwtManager := auth.NewJWTManager(
		cfg.JWTSecret,
		time.Duration(cfg.JWTAccessTokenDuration)*time.Second,
		time.Duration(cfg.JWTRefreshTokenDuration)*time.Second,
	)

	// Auth repositories
	userRepo := authrepo.NewUserRepository(db)
	identityRepo := authrepo.NewAuthIdentityRepository(db)
	refreshTokenRepo := authrepo.NewRefreshTokenRepository(db)

	// RBAC repositories
	roleRepo := rbacrepo.NewRoleRepository(db)
	permRepo := rbacrepo.NewPermissionRepository(db)
	userRoleRepo := rbacrepo.NewUserRoleRepository(db)

	// Problems repositories
	problemsRepo := problemsrepo.NewRepository(db)

	// Services
	rbacService := rbacservice.NewService(roleRepo, permRepo, userRoleRepo)
	authService := authservice.NewService(userRepo, identityRepo, refreshTokenRepo, jwtManager, rbacService)
	problemsService := problemsservice.NewService(problemsRepo)

	// Managers
	rbacManager := rbac.NewManager(rbacService)

	// Handlers
	authHandler := authhttp.NewHandler(authService, cfg.SecureCookies)
	rbacHandler := rbachttp.NewHandler(rbacService)
	problemsHandler := problemshttp.NewHandler(problemsService)

	handler := setupRoutes(authHandler, rbacHandler, problemsHandler, jwtManager, rbacManager)

	r := chi.NewRouter()
	
	// CORS middleware
	allowedOrigins := []string{"*"}
	if cfg.AllowedOrigins != "" && cfg.AllowedOrigins != "*" {
		allowedOrigins = strings.Split(cfg.AllowedOrigins, ",")
		// Trim whitespace from each origin
		for i := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
		}
	}
	
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Mount("/", handler)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort),
		Handler: r,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Server error", zap.Error(err))
		}
	}()

	log.Info("Server started successfully",
		zap.String("host", cfg.ServerHost),
		zap.Int("port", cfg.ServerPort),
		zap.String("address", fmt.Sprintf("http://%s:%d", cfg.ServerHost, cfg.ServerPort)))

	// Wait for interrupt signal
	<-ctx.Done()
	log.Info("Shutting down server gracefully...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal("Server shutdown failed", zap.Error(err))
	}

	log.Info("Server stopped")
}
