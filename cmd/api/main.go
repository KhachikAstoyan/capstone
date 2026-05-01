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
	airepo "github.com/KhachikAstoyan/capstone/internal/api/ai/repository"
	aiservice "github.com/KhachikAstoyan/capstone/internal/api/ai/service"
	aiapi "go.jetify.com/ai/api"
	aimodel "github.com/KhachikAstoyan/capstone/internal/api/ai"
	"go.jetify.com/ai/provider/anthropic"
	"go.jetify.com/ai/provider/openai"
	"github.com/KhachikAstoyan/capstone/internal/api/auth"
	authhttp "github.com/KhachikAstoyan/capstone/internal/api/auth/http"
	authrepo "github.com/KhachikAstoyan/capstone/internal/api/auth/repository"
	authservice "github.com/KhachikAstoyan/capstone/internal/api/auth/service"
	languageshttp "github.com/KhachikAstoyan/capstone/internal/api/languages/http"
	languagesrepo "github.com/KhachikAstoyan/capstone/internal/api/languages/repository"
	languagesservice "github.com/KhachikAstoyan/capstone/internal/api/languages/service"
	problemshttp "github.com/KhachikAstoyan/capstone/internal/api/problems/http"
	problemsrepo "github.com/KhachikAstoyan/capstone/internal/api/problems/repository"
	problemsservice "github.com/KhachikAstoyan/capstone/internal/api/problems/service"
	"github.com/KhachikAstoyan/capstone/internal/api/rbac"
	rbachttp "github.com/KhachikAstoyan/capstone/internal/api/rbac/http"
	rbacrepo "github.com/KhachikAstoyan/capstone/internal/api/rbac/repository"
	rbacservice "github.com/KhachikAstoyan/capstone/internal/api/rbac/service"
	submissionsclient "github.com/KhachikAstoyan/capstone/internal/api/submissions/client"
	submissionshttp "github.com/KhachikAstoyan/capstone/internal/api/submissions/http"
	submissionsrepo "github.com/KhachikAstoyan/capstone/internal/api/submissions/repository"
	submissionsservice "github.com/KhachikAstoyan/capstone/internal/api/submissions/service"
	tagshttp "github.com/KhachikAstoyan/capstone/internal/api/tags/http"
	tagsrepo "github.com/KhachikAstoyan/capstone/internal/api/tags/repository"
	tagsservice "github.com/KhachikAstoyan/capstone/internal/api/tags/service"
	"github.com/KhachikAstoyan/capstone/pkg/database"
	"github.com/KhachikAstoyan/capstone/pkg/logger"
	"github.com/KhachikAstoyan/capstone/pkg/migrations"
	"github.com/KhachikAstoyan/capstone/pkg/rabbitmq"
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
	emailVerificationRepo := authrepo.NewEmailVerificationRepository(db)
	statsRepo := authrepo.NewStatsRepository(db)

	// RBAC repositories
	roleRepo := rbacrepo.NewRoleRepository(db)
	permRepo := rbacrepo.NewPermissionRepository(db)
	userRoleRepo := rbacrepo.NewUserRoleRepository(db)

	// Problems repositories
	problemsRepo := problemsrepo.NewRepository(db)
	tagsRepo := tagsrepo.New(db)
	languagesRepo := languagesrepo.New(db)

	// Services
	rbacService := rbacservice.NewService(roleRepo, permRepo, userRoleRepo)

	emailVerificationPub := rabbitmq.NewNoopEmailVerificationPublisher()
	if cfg.RabbitMQURL != "" {
		pub, err := rabbitmq.NewPublisher(cfg.RabbitMQURL, cfg.RabbitMQExchange)
		if err != nil {
			log.Fatal("Failed to connect to RabbitMQ", zap.Error(err))
		}
		defer func() {
			if err := pub.Close(); err != nil {
				log.Error("RabbitMQ publisher close", zap.Error(err))
			}
		}()
		emailVerificationPub = rabbitmq.NewEmailVerificationPublisher(pub, cfg.RabbitMQEmailVerificationRoutingKey)
		log.Info("RabbitMQ publisher ready",
			zap.String("exchange", cfg.RabbitMQExchange),
			zap.String("email_verification_routing_key", cfg.RabbitMQEmailVerificationRoutingKey))
	}

	authService := authservice.NewService(userRepo, identityRepo, refreshTokenRepo, emailVerificationRepo, statsRepo, jwtManager, rbacService, cfg.FrontendURL, emailVerificationPub)
	problemsService := problemsservice.NewService(problemsRepo)
	tagsService := tagsservice.New(tagsRepo)
	languagesService := languagesservice.New(languagesRepo)

	// Managers
	rbacManager := rbac.NewManager(rbacService)

	// Handlers
	authHandler := authhttp.NewHandler(authService, cfg.SecureCookies)
	rbacHandler := rbachttp.NewHandler(rbacService)
	problemsHandler := problemshttp.NewHandler(problemsService)
	tagsHandler := tagshttp.NewHandler(tagsService, problemsService)
	languagesHandler := languageshttp.NewHandler(languagesService, problemsService)

	cpClient := submissionsclient.NewCPClient(cfg.ControlPlaneURL, cfg.ControlPlaneKey)
	submissionsRepo := submissionsrepo.NewRepository(db)

	// AI validation service
	aiRepo := airepo.New(db)
	var aiModel aiapi.LanguageModel
	switch cfg.AIProvider {
	case "openai":
		if cfg.AIAPIBaseURL != "" {
			log.Info("using ollama with openai provider", zap.String("base_url", cfg.AIAPIBaseURL))
			aiModel = aimodel.NewOllamaModel(cfg.AIAPIBaseURL, cfg.AIModel)
		} else {
			aiModel = openai.NewLanguageModel(cfg.AIModel)
		}
	case "anthropic":
		fallthrough
	default:
		aiModel = anthropic.NewLanguageModel(cfg.AIModel)
	}
	aiSvc := aiservice.New(aiRepo, aiModel, log)

	submissionsService := submissionsservice.NewService(submissionsRepo, cpClient, problemsRepo, aiSvc)
	submissionsHandler := submissionshttp.NewHandler(submissionsService, rbacManager)

	handler := setupRoutes(authHandler, rbacHandler, problemsHandler, tagsHandler, languagesHandler, submissionsHandler, jwtManager, rbacManager)

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
