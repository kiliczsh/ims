package server

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"ims/internal/config"
	"ims/internal/handlers"
	"ims/internal/middleware"
	"ims/internal/scheduler"
	"ims/internal/service"

	"github.com/redis/go-redis/v9"
)

type Server struct {
	httpServer *http.Server
	scheduler  *scheduler.Scheduler
}

func NewServer(
	cfg *config.Config,
	db *sql.DB,
	redis *redis.Client,
	messageService *service.MessageService,
	scheduler *scheduler.Scheduler,
	auditService service.AuditService,
) *Server {
	mux := http.NewServeMux()

	// Create handlers
	healthHandler := handlers.NewHealthHandler(db, redis, scheduler)
	controlHandler := handlers.NewControlHandler(scheduler)
	messageHandler := handlers.NewMessageHandler(messageService)
	auditHandler := handlers.NewAuditHandler(auditService)

	// Apply authentication middleware to protected routes
	authMiddleware := middleware.AuthMiddleware(cfg.Webhook.AuthKey)

	// Routes
	mux.Handle("/api/health", middleware.LoggingMiddleware(http.HandlerFunc(healthHandler.Handle)))
	mux.Handle("/api/control", middleware.LoggingMiddleware(authMiddleware(http.HandlerFunc(controlHandler.Handle))))
	mux.Handle("/api/messages/sent", middleware.LoggingMiddleware(authMiddleware(http.HandlerFunc(messageHandler.GetSentMessages))))

	// Audit routes
	mux.Handle("/api/audit", middleware.LoggingMiddleware(authMiddleware(http.HandlerFunc(auditHandler.GetAuditLogs))))
	mux.Handle("/api/audit/stats", middleware.LoggingMiddleware(authMiddleware(http.HandlerFunc(auditHandler.GetAuditLogStats))))
	mux.Handle("/api/audit/cleanup", middleware.LoggingMiddleware(authMiddleware(http.HandlerFunc(auditHandler.CleanupOldAuditLogs))))

	// Setup path-based routing for audit endpoints that need path parameters
	// For now, using simple path matching since we don't have a full router
	mux.Handle("/api/audit/batch/", middleware.LoggingMiddleware(authMiddleware(http.HandlerFunc(auditHandler.GetBatchAuditLogs))))
	mux.Handle("/api/audit/message/", middleware.LoggingMiddleware(authMiddleware(http.HandlerFunc(auditHandler.GetMessageAuditLogs))))

	// Setup Swagger UI
	SetupSwagger(mux)

	server := &http.Server{
		Addr:           ":" + cfg.Server.Port,
		Handler:        mux,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	return &Server{
		httpServer: server,
		scheduler:  scheduler,
	}
}

func (s *Server) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		s.Shutdown()
	}()

	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown() error {
	// Stop scheduler first
	if s.scheduler != nil {
		s.scheduler.Stop()
	}

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}

// SetupSwagger configures Swagger UI endpoint with dynamic docs
func SetupSwagger(mux *http.ServeMux) {
	// Serve swagger.yaml dynamically
	mux.HandleFunc("/api/swagger.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		http.ServeFile(w, r, "docs/swagger.yaml")
	})

	// Serve swagger.json dynamically
	mux.HandleFunc("/api/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		http.ServeFile(w, r, "docs/swagger.json")
	})

	// Enhanced Swagger UI HTML
	mux.HandleFunc("/api/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		html := `<!DOCTYPE html>
<html>
<head>
    <title>IMS API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui.css" />
    <style>
        .swagger-ui .topbar { display: none; }
        .swagger-ui .info { margin: 20px 0; }
        .swagger-ui .info .title { font-size: 2em; color: #3b4151; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: '/api/swagger.yaml',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                validatorUrl: null,
                docExpansion: 'list',
                filter: true,
                tryItOutEnabled: true
            });
        };
    </script>
</body>
</html>`
		w.Write([]byte(html))
	})

	// Serve docs.go for Go imports (if generated)
	mux.HandleFunc("/api/docs.go", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		http.ServeFile(w, r, "docs/docs.go")
	})
}
