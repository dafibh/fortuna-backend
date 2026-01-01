package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dafibh/fortuna/fortuna-backend/internal/config"
	"github.com/dafibh/fortuna/fortuna-backend/internal/handler"
	"github.com/dafibh/fortuna/fortuna-backend/internal/middleware"
	"github.com/dafibh/fortuna/fortuna-backend/internal/repository/postgres"
	"github.com/dafibh/fortuna/fortuna-backend/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Initialize zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("ENV") != "production" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Connect to database
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer pool.Close()

	// Verify database connection
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("Failed to ping database")
	}
	log.Info().Msg("Connected to database")

	// Initialize repositories
	userRepo := postgres.NewUserRepository(pool)
	workspaceRepo := postgres.NewWorkspaceRepository(pool)
	accountRepo := postgres.NewAccountRepository(pool)
	transactionRepo := postgres.NewTransactionRepository(pool)
	monthRepo := postgres.NewMonthRepository(pool)
	budgetCategoryRepo := postgres.NewBudgetCategoryRepository(pool)

	// Initialize services
	authService := service.NewAuthService(userRepo, workspaceRepo)
	profileService := service.NewProfileService(userRepo)
	accountService := service.NewAccountService(accountRepo)
	transactionService := service.NewTransactionService(transactionRepo, accountRepo, budgetCategoryRepo)
	calculationService := service.NewCalculationService(accountRepo, transactionRepo)
	monthService := service.NewMonthService(monthRepo, transactionRepo, calculationService)
	dashboardService := service.NewDashboardService(accountRepo, transactionRepo, monthService, calculationService)
	budgetCategoryService := service.NewBudgetCategoryService(budgetCategoryRepo)

	// Create workspace provider adapter for auth middleware
	workspaceProvider := &workspaceProviderAdapter{authService: authService}

	// Initialize auth middleware
	authMiddleware, err := middleware.NewAuthMiddleware(cfg.Auth0Domain, cfg.Auth0Audience, workspaceProvider)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create auth middleware")
	}

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	profileHandler := handler.NewProfileHandler(profileService)
	accountHandler := handler.NewAccountHandler(accountService, calculationService)
	transactionHandler := handler.NewTransactionHandler(transactionService)
	monthHandler := handler.NewMonthHandler(monthService)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)
	budgetCategoryHandler := handler.NewBudgetCategoryHandler(budgetCategoryService)

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Request ID middleware
	e.Use(echomiddleware.RequestID())

	// CORS middleware
	e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowCredentials: true,
		MaxAge:           86400,
	}))

	// Security headers middleware (helmet-like)
	e.Use(echomiddleware.SecureWithConfig(echomiddleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		HSTSMaxAge:            31536000,
		ContentSecurityPolicy: "default-src 'self'",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
	}))

	// Request logging middleware with zerolog
	e.Use(zerologMiddleware())

	// Recovery middleware
	e.Use(echomiddleware.Recover())

	// Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Register API routes
	handler.RegisterRoutes(e, authMiddleware, authHandler, profileHandler, accountHandler, transactionHandler, monthHandler, dashboardHandler, budgetCategoryHandler)

	// Start server in goroutine
	go func() {
		log.Info().Str("port", cfg.Port).Msg("Starting server")
		if err := e.Start(":" + cfg.Port); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited")
}

// workspaceProviderAdapter adapts AuthService to middleware.WorkspaceProvider
type workspaceProviderAdapter struct {
	authService *service.AuthService
}

// GetWorkspaceByAuth0ID implements middleware.WorkspaceProvider
func (a *workspaceProviderAdapter) GetWorkspaceByAuth0ID(auth0ID string) (int32, error) {
	workspace, err := a.authService.GetWorkspaceByAuth0ID(auth0ID)
	if err != nil {
		return 0, err
	}
	return workspace.ID, nil
}

// zerologMiddleware returns a middleware that logs requests using zerolog
func zerologMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			req := c.Request()
			res := c.Response()

			log.Info().
				Str("method", req.Method).
				Str("path", req.URL.Path).
				Int("status", res.Status).
				Dur("latency", time.Since(start)).
				Str("request_id", res.Header().Get(echo.HeaderXRequestID)).
				Msg("request")

			return nil
		}
	}
}
