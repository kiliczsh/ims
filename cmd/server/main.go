package main

// @title           IMS API
// @version         1.0
// @description     A message scheduling and sending service with audit logging
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"ims/internal/config"
	"ims/internal/repository"
	"ims/internal/repository/postgres"
	redisRepo "ims/internal/repository/redis"
	"ims/internal/scheduler"
	"ims/internal/server"
	"ims/internal/service"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// Version information (set by build flags)
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	// Parse command line flags
	var showVersion = flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("IMS (Insider Message Sender)\n")
		fmt.Printf("Version: %s\n", version)
		fmt.Printf("Build Time: %s\n", buildTime)
		fmt.Printf("Git Commit: %s\n", gitCommit)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting IMS (Insider Message Sender) v%s on port %s", version, cfg.Server.Port)

	// Initialize database
	sqlDB, err := postgres.NewDB(cfg.Database.URL, cfg.Database.MaxConnections, cfg.Database.MaxIdleConnections)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer sqlDB.Close()

	// Wrap with sqlx for audit repository
	db := sqlx.NewDb(sqlDB, "postgres")

	log.Println("Connected to PostgreSQL database")

	// Initialize Redis (optional)
	var redisClient *redis.Client
	if cfg.Redis.URL != "" {
		client, err := redisRepo.NewRedisClient(cfg.Redis.URL)
		if err != nil {
			log.Printf("Failed to connect to Redis (continuing without cache): %v", err)
		} else {
			redisClient = client
			log.Println("Connected to Redis cache")
		}
	}

	// Initialize repositories
	messageRepo := postgres.NewMessageRepository(sqlDB)
	auditRepo := postgres.NewAuditRepository(db)
	var cacheRepo repository.CacheRepository
	if redisClient != nil {
		cacheRepo = redisRepo.NewCacheRepository(redisClient)
	}

	// Initialize audit service
	auditService := service.NewAuditService(auditRepo)

	// Initialize webhook client
	webhookClient := service.NewWebhookClient(
		cfg.Webhook.URL,
		cfg.Webhook.AuthKey,
		cfg.Webhook.Timeout,
		cfg.Webhook.MaxRetries,
	)

	// Initialize message service
	messageService := service.NewMessageService(
		messageRepo,
		cacheRepo,
		webhookClient,
		cfg.Message.MaxLength,
	)

	// Initialize scheduler with audit service
	scheduler := scheduler.NewScheduler(
		messageService,
		auditService,
		cfg.Scheduler.Interval,
		cfg.Scheduler.BatchSize,
	)

	// Initialize server with audit service
	srv := server.NewServer(cfg, sqlDB, redisClient, messageService, scheduler, auditService)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down gracefully...")
		cancel()
	}()

	// Start the server
	log.Printf("Server started on http://localhost:%s", cfg.Server.Port)
	log.Printf("Health check: http://localhost:%s/api/health", cfg.Server.Port)
	log.Printf("API documentation: http://localhost:%s/api/docs", cfg.Server.Port)

	if err := srv.Start(ctx); err != nil {
		log.Printf("Server shutdown: %v", err)
	}

	log.Println("Server stopped")
}
