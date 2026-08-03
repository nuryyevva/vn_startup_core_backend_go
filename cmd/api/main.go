// Command api is the entrypoint for the visual-novel core backend service.
// It loads configuration once, wires every module's dependencies together
// explicitly, and starts the HTTP server, dialog scheduler, and realtime
// NATS subscriber.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"vn_startup_core_backend_go/internal/auth"
	"vn_startup_core_backend_go/internal/config"
	"vn_startup_core_backend_go/internal/dialog"
	"vn_startup_core_backend_go/internal/realtime"
	"vn_startup_core_backend_go/internal/story"
	"vn_startup_core_backend_go/internal/user"
	"vn_startup_core_backend_go/internal/wallet"
	"vn_startup_core_backend_go/pkg/events"
	"vn_startup_core_backend_go/pkg/httpserver"
	"vn_startup_core_backend_go/pkg/logger"
	"vn_startup_core_backend_go/pkg/middleware"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal startup error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.New(cfg.Server.Env)

	pool, err := connectPostgres(context.Background(), cfg.Postgres)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer func() { _ = redisClient.Close() }()

	natsConn, js, err := events.Connect(cfg.NATS.URL)
	if err != nil {
		return fmt.Errorf("connect nats: %w", err)
	}
	defer natsConn.Close()

	publisher := events.NewPublisher(js)
	tokenIssuer := auth.NewTokenIssuer(cfg.JWT.Secret, cfg.JWT.AccessTokenTTL, cfg.JWT.RefreshTokenTTL)
	hub := realtime.NewHub()

	// --- repositories ---
	authRepo := auth.NewPostgresRepository(pool)
	userRepo := user.NewPostgresRepository(pool)
	storyRepo := story.NewPostgresRepository(pool)
	walletRepo := wallet.NewPostgresRepository(pool)
	dialogRepo := dialog.NewPostgresRepository(pool)

	// --- services (wired in dependency order) ---
	authService := auth.NewService(authRepo, tokenIssuer)
	userService := user.NewService(userRepo)
	walletService := wallet.NewService(walletRepo)
	storyService := story.NewService(storyRepo, walletService, publisher, log)
	dialogService := dialog.NewService(
		dialogRepo,
		storySceneAdapter{storyService},
		walletService,
		publisher,
		hub,
		int32(cfg.Dialog.DefaultMessageCost),
		log,
	)

	// --- handlers ---
	authHandler := auth.NewHandler(authService)
	userHandler := user.NewHandler(userService)
	storyHandler := story.NewHandler(storyService)
	walletHandler := wallet.NewHandler(walletService)
	dialogHandler := dialog.NewHandler(dialogService)
	realtimeHandler := realtime.NewHandler(hub, log)

	app := httpserver.New(log)

	app.Use("/auth/login", middleware.RateLimitByIP(redisClient, "auth_login", 20, time.Minute))
	app.Use("/auth/register", middleware.RateLimitByIP(redisClient, "auth_register", 10, time.Minute))
	authHandler.RegisterRoutes(app)

	// Auth is applied per-route (not via an empty-prefix app.Group, which
	// Fiber treats as a catch-all "Use" matching every path — including
	// undefined ones, turning what should be 404s into 401s).
	authMiddleware := middleware.Auth(tokenIssuer)
	userHandler.RegisterRoutes(app, authMiddleware)
	storyHandler.RegisterRoutes(app, authMiddleware)
	walletHandler.RegisterRoutes(app, authMiddleware)
	dialogHandler.RegisterRoutes(app, authMiddleware)

	if cfg.Dev.EnableDevEndpoints {
		devWalletHandler := wallet.NewDevHandler(walletService)
		devWalletHandler.RegisterRoutes(app, authMiddleware)
		log.Warn("dev endpoints enabled: /dev/wallet/grant is exposed")
	}

	realtimeHandler.RegisterRoutes(app, authMiddleware)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	subscriber := realtime.NewSubscriber(js, hub, dialogService, log)
	if err := subscriber.Start(ctx); err != nil {
		return fmt.Errorf("start realtime subscriber: %w", err)
	}

	scheduler := dialog.NewSessionScheduler(dialogService, cfg.Dialog.SessionCheckInterval, log)
	go scheduler.Run(ctx)

	serverErr := make(chan error, 1)
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Server.Port)
		log.Info("http server listening", slog.String("addr", addr), slog.String("env", cfg.Server.Env))
		serverErr <- app.Listen(addr)
	}()

	select {
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return app.ShutdownWithContext(shutdownCtx)
	case err := <-serverErr:
		return err
	}
}

func connectPostgres(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse postgres dsn: %w", err)
	}
	poolCfg.MaxConns = int32(cfg.MaxConns)

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}

// storySceneAdapter satisfies dialog.SceneProvider using the story
// module's service, so the dialog module never needs to import story
// directly.
type storySceneAdapter struct {
	service *story.Service
}

func (a storySceneAdapter) GetSceneInfo(ctx context.Context, sceneID uuid.UUID) (dialog.SceneInfo, error) {
	scene, _, err := a.service.GetScene(ctx, sceneID)
	if err != nil {
		return dialog.SceneInfo{}, err
	}
	return dialog.SceneInfo{
		ID:                scene.ID,
		FreeDialogEnabled: scene.FreeDialogEnabled,
		DialogLimitType:   scene.DialogLimitType,
		DialogLimitValue:  scene.DialogLimitValue,
	}, nil
}
