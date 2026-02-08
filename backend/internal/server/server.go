package server

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/tessera/tessera/internal/config"
	"github.com/tessera/tessera/internal/handlers"
	"github.com/tessera/tessera/internal/jobs"
	"github.com/tessera/tessera/internal/middleware"
	"github.com/tessera/tessera/internal/repository"
	"github.com/tessera/tessera/internal/security"
	"github.com/tessera/tessera/internal/services"
	"github.com/tessera/tessera/internal/storage"
	"github.com/tessera/tessera/internal/webdav"
	ws "github.com/tessera/tessera/internal/websocket"
)

// Server represents the HTTP server
type Server struct {
	app       *fiber.App
	cfg       *config.Config
	db        *pgxpool.Pool
	rdb       *redis.Client
	store     *storage.MinIOStorage
	log       zerolog.Logger
	hub       *ws.Hub
	jobWorker *jobs.Worker
	scheduler *jobs.Scheduler
}

// New creates a new server instance
func New(cfg *config.Config, db *pgxpool.Pool, rdb *redis.Client, store *storage.MinIOStorage, log zerolog.Logger) *Server {
	app := fiber.New(fiber.Config{
		AppName:               "Tessera API",
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		IdleTimeout:           120 * time.Second,
		BodyLimit:             int(cfg.Upload.MaxSize),
		DisableStartupMessage: false,
		ErrorHandler:          customErrorHandler,
	})

	// Create WebSocket hub
	hub := ws.NewHub(log)
	go hub.Run()

	// Create job queue and worker
	jobQueue := jobs.NewMemoryQueue(1000)
	jobWorker := jobs.NewWorker(jobQueue, 4)

	// Register job handlers
	jobWorker.RegisterHandler(jobs.JobTypeThumbnail, jobs.NewThumbnailHandler())
	jobWorker.RegisterHandler(jobs.JobTypeCleanup, jobs.NewCleanupHandler())
	jobWorker.RegisterHandler(jobs.JobTypeNotification, jobs.NewNotificationHandler())
	jobWorker.RegisterHandler(jobs.JobTypeQuotaCheck, jobs.NewQuotaCheckHandler())
	jobWorker.RegisterHandler(jobs.JobTypeVersionCleanup, jobs.NewVersionCleanupHandler())

	// Create scheduler for recurring jobs
	scheduler := jobs.NewScheduler(jobWorker)

	srv := &Server{
		app:       app,
		cfg:       cfg,
		db:        db,
		rdb:       rdb,
		store:     store,
		log:       log,
		hub:       hub,
		jobWorker: jobWorker,
		scheduler: scheduler,
	}

	srv.setupMiddleware()
	srv.setupRoutes()

	// Start job worker and scheduler
	ctx := context.Background()
	jobWorker.Start(ctx)
	scheduler.Start(ctx)

	return srv
}

func (s *Server) setupMiddleware() {
	// Recover from panics
	s.app.Use(recover.New())

	// Request ID
	s.app.Use(requestid.New())

	// Prometheus metrics middleware
	s.app.Use(middleware.Metrics())

	// Logger
	s.app.Use(logger.New(logger.Config{
		Format:     "${time} | ${status} | ${latency} | ${ip} | ${method} | ${path}\n",
		TimeFormat: "15:04:05",
	}))

	// CORS
	s.app.Use(cors.New(cors.Config{
		AllowOrigins:     s.cfg.Server.FrontendURL,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Request-ID",
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Rate limiting
	s.app.Use(limiter.New(limiter.Config{
		Max:               100,
		Expiration:        1 * time.Minute,
		LimiterMiddleware: limiter.SlidingWindow{},
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": "Too many requests",
			})
		},
	}))
}

func (s *Server) setupRoutes() {
	// Initialize repositories
	userRepo := repository.NewUserRepository(s.db)
	fileRepo := repository.NewFileRepository(s.db)
	sessionRepo := repository.NewSessionRepository(s.rdb)
	activityRepo := repository.NewActivityRepository(s.db)
	settingsRepo := repository.NewSettingsRepository(s.db)
	emailRepo := repository.NewEmailRepository(s.db)

	// Initialize encryptor for sensitive data (email passwords, etc.)
	var encryptor *security.Encryptor
	if s.cfg.Encryption.MasterKey != "" {
		var err error
		encryptor, err = security.NewEncryptor(s.cfg.Encryption.MasterKey)
		if err != nil {
			s.log.Warn().Err(err).Msg("Failed to initialize encryptor - email passwords will not be encrypted")
		} else {
			s.log.Info().Msg("Encryption enabled for sensitive data")
		}
	} else {
		s.log.Warn().Msg("ENCRYPTION_KEY not set - email passwords will be stored in plain text")
	}

	// Initialize services
	authService := services.NewAuthService(userRepo, sessionRepo, s.cfg.JWT)
	fileService := services.NewFileService(fileRepo, userRepo, s.store, s.log)
	emailService := services.NewEmailService(emailRepo, s.store, encryptor)

	// Register email sync handler now that we have the email service
	s.jobWorker.RegisterHandler(jobs.JobTypeEmailSync, jobs.NewEmailSyncHandler(emailService))
	s.scheduler.SetEmailService(emailService)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, s.log)
	fileHandler := handlers.NewFileHandler(fileService, s.log, s.hub)
	healthHandler := handlers.NewHealthHandler(s.log, s.db, s.rdb, s.store.Client())
	wsHandler := ws.NewHandler(s.hub, s.log)
	webdavServer := webdav.NewServer(fileRepo, s.store, authService, fileService, s.log)
	adminHandler := handlers.NewAdminHandler(s.db, userRepo, fileRepo, activityRepo, s.log)
	moduleHandler := handlers.NewModuleHandler(s.log, settingsRepo)
	taskHandler := handlers.NewTaskHandler(s.log)
	documentHandler := handlers.NewDocumentHandler(s.log)
	emailHandler := handlers.NewEmailHandler(emailService)
	calendarHandler := handlers.NewCalendarHandler(s.log)
	contactsHandler := handlers.NewContactsHandler(s.log)

	// Auth middleware
	authMiddleware := middleware.NewAuthMiddleware(authService, s.cfg.JWT)

	// API routes
	api := s.app.Group("/api")

	// Health check (public)
	api.Get("/health", healthHandler.Liveness)
	api.Get("/ready", healthHandler.Readiness)

	// Prometheus metrics endpoint (public for scraping)
	s.app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	// Jobs stats (for admin monitoring)
	api.Get("/jobs/stats", s.jobsStatsHandler)

	// Auth routes (public)
	auth := api.Group("/auth")
	auth.Get("/setup-status", authHandler.SetupStatus)
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/login/totp", authHandler.CompleteTOTPLogin)
	auth.Post("/refresh", authHandler.RefreshToken)
	auth.Post("/forgot-password", authHandler.ForgotPassword)
	auth.Post("/reset-password", authHandler.ResetPassword)

	// Protected routes
	protected := api.Group("", authMiddleware.Authenticate)

	// Auth (protected)
	protected.Post("/auth/logout", authHandler.Logout)
	protected.Get("/auth/me", authHandler.Me)
	protected.Put("/auth/password", authHandler.ChangePassword)
	protected.Put("/auth/settings", authHandler.UpdateSettings)
	protected.Get("/auth/ws-ticket", authHandler.GetWebSocketTicket)

	// Two-Factor Authentication (protected)
	protected.Get("/auth/totp/status", authHandler.GetTOTPStatus)
	protected.Post("/auth/totp/setup", authHandler.InitiateTOTPSetup)
	protected.Post("/auth/totp/confirm", authHandler.ConfirmTOTPSetup)
	protected.Delete("/auth/totp", authHandler.DisableTOTP)
	protected.Post("/auth/totp/backup-codes", authHandler.RegenerateBackupCodes)

	// Files
	files := protected.Group("/files")
	files.Get("/", fileHandler.List)
	files.Get("/:id", fileHandler.Get)
	files.Post("/folder", fileHandler.CreateFolder)
	files.Put("/:id", fileHandler.Update)
	files.Delete("/:id", fileHandler.Delete)
	files.Post("/:id/restore", fileHandler.Restore)
	files.Post("/:id/copy", fileHandler.Copy)
	files.Get("/:id/download", fileHandler.Download)
	files.Get("/:id/versions", fileHandler.GetVersions)
	files.Post("/:id/versions/:version/restore", fileHandler.RestoreVersion)
	files.Post("/:id/share", fileHandler.CreateShare)
	files.Post("/:id/share/user", fileHandler.ShareWithUser)
	files.Get("/:id/share/analytics", fileHandler.GetShareAnalytics)
	files.Get("/:id/shares", fileHandler.GetFileShares)
	files.Delete("/shares/:shareId", fileHandler.RevokeShare)
	// Document file content endpoints
	files.Get("/:id/content", fileHandler.GetDocumentContent)
	files.Put("/:id/content", fileHandler.UpdateDocumentContent)

	// Shared with me
	protected.Get("/shared", fileHandler.GetSharedWithMe)

	// Upload (using Tus protocol)
	upload := protected.Group("/upload")
	upload.Post("/", fileHandler.InitiateUpload)
	upload.Patch("/:uploadId", fileHandler.ChunkUpload)
	upload.Head("/:uploadId", fileHandler.UploadStatus)

	// Simple upload (multipart form)
	files.Post("/upload", fileHandler.SimpleUpload)

	// Search
	protected.Get("/search", fileHandler.Search)

	// Trash
	trash := protected.Group("/trash")
	trash.Get("/", fileHandler.ListTrash)
	trash.Delete("/", fileHandler.EmptyTrash)

	// Starred
	protected.Get("/starred", fileHandler.ListStarred)

	// Storage stats
	protected.Get("/storage", fileHandler.StorageStats)

	// Admin routes (require admin role)
	admin := protected.Group("/admin", authMiddleware.RequireAdmin)
	admin.Get("/stats", adminHandler.GetStats)
	admin.Get("/settings", adminHandler.GetSettings)
	admin.Patch("/settings", adminHandler.UpdateSettings)
	admin.Get("/users", adminHandler.ListUsers)
	admin.Post("/users", adminHandler.CreateUser)
	admin.Get("/users/:id", adminHandler.GetUser)
	admin.Patch("/users/:id", adminHandler.UpdateUser)
	admin.Delete("/users/:id", adminHandler.DeleteUser)
	admin.Get("/logs", adminHandler.GetActivityLogs)
	admin.Post("/cache/clear", adminHandler.ClearCache)
	admin.Post("/cleanup", adminHandler.RunCleanup)
	admin.Get("/modules", moduleHandler.GetAdminModules)
	admin.Put("/modules/:id", moduleHandler.UpdateModule)
	admin.Put("/modules", moduleHandler.UpdateAllModules)

	// Module settings (public for users to know what's enabled)
	protected.Get("/modules", moduleHandler.GetModules)

	// Tasks routes (optional module)
	tasks := protected.Group("/tasks")
	tasks.Get("/", taskHandler.ListTasks)
	tasks.Post("/", taskHandler.CreateTask)
	tasks.Put("/reorder", taskHandler.ReorderTasks)
	tasks.Get("/groups", taskHandler.ListGroups)
	tasks.Post("/groups", taskHandler.CreateGroup)
	tasks.Put("/groups/:id", taskHandler.UpdateGroup)
	tasks.Delete("/groups/:id", taskHandler.DeleteGroup)
	tasks.Get("/:id", taskHandler.GetTask)
	tasks.Put("/:id", taskHandler.UpdateTask)
	tasks.Put("/:id/move", taskHandler.MoveTask)
	tasks.Delete("/:id", taskHandler.DeleteTask)

	// Documents routes (optional module)
	docs := protected.Group("/documents")
	docs.Get("/", documentHandler.ListDocuments)
	docs.Post("/", documentHandler.CreateDocument)
	docs.Post("/create-file", fileHandler.CreateDocumentFile) // Create document as file
	docs.Get("/:id", documentHandler.GetDocument)
	docs.Put("/:id", documentHandler.UpdateDocument)
	docs.Delete("/:id", documentHandler.DeleteDocument)
	docs.Post("/:id/share", documentHandler.ShareDocument)
	docs.Delete("/:id/share/:userId", documentHandler.RemoveCollaborator)

	// Email routes (optional module)
	email := protected.Group("/email")
	email.Get("/accounts", emailHandler.GetAccounts)
	email.Post("/accounts", emailHandler.CreateAccount)
	email.Get("/accounts/:accountId", emailHandler.GetAccount)
	email.Put("/accounts/:accountId", emailHandler.UpdateAccount)
	email.Delete("/accounts/:accountId", emailHandler.DeleteAccount)
	email.Post("/accounts/:accountId/sync", emailHandler.SyncAccount)
	email.Get("/accounts/:accountId/sync/stream", emailHandler.SyncAccountStream)
	email.Get("/accounts/:accountId/folders", emailHandler.GetFolders)
	email.Get("/accounts/:accountId/folders/tree", emailHandler.GetFoldersTree)
	email.Post("/accounts/:accountId/folders", emailHandler.CreateFolder)
	email.Post("/accounts/:accountId/folders/reorder", emailHandler.ReorderFolders)
	email.Get("/accounts/:accountId/search", emailHandler.SearchEmails)
	email.Get("/accounts/:accountId/starred", emailHandler.GetStarredEmails)
	email.Get("/accounts/:accountId/drafts", emailHandler.GetDraftEmails)
	email.Get("/accounts/:accountId/counts", emailHandler.GetCounts)
	email.Get("/folders/:folderId/emails", emailHandler.GetEmails)
	email.Get("/folders/:folderId/threads", emailHandler.GetThreads)
	email.Put("/folders/:folderId", emailHandler.UpdateFolder)
	email.Delete("/folders/:folderId", emailHandler.DeleteFolder)
	email.Patch("/folders/:folderId/move", emailHandler.MoveFolder)
	email.Post("/folders/:folderId/read", emailHandler.MarkFolderAsRead)
	email.Get("/threads/:threadId/emails", emailHandler.GetThreadEmails)
	email.Get("/threads/:threadId/conversation", emailHandler.GetThreadConversation)
	email.Post("/accounts/:accountId/threads/reindex", emailHandler.ReindexThreads)
	email.Get("/emails/:emailId", emailHandler.GetEmail)
	email.Patch("/emails/:emailId/read", emailHandler.MarkAsRead)
	email.Patch("/emails/:emailId/star", emailHandler.MarkAsStarred)
	email.Patch("/emails/:emailId/move", emailHandler.MoveEmail)
	email.Delete("/emails/:emailId", emailHandler.DeleteEmail)
	email.Post("/send", emailHandler.SendEmail)
	email.Post("/send/queue", emailHandler.QueueSend)
	email.Post("/send/:sendId/cancel", emailHandler.CancelSend)

	// Batch operations
	email.Post("/batch/read", emailHandler.BatchMarkAsRead)
	email.Post("/batch/star", emailHandler.BatchMarkAsStarred)
	email.Post("/batch/move", emailHandler.BatchMoveEmails)
	email.Post("/batch/delete", emailHandler.BatchDeleteEmails)
	email.Post("/batch/label", emailHandler.BatchAssignLabel)

	// Compose drafts
	email.Post("/drafts", emailHandler.SaveDraft)
	email.Get("/drafts/:draftId", emailHandler.GetDraft)
	email.Delete("/drafts/:draftId", emailHandler.DeleteDraft)
	email.Get("/accounts/:accountId/compose-drafts", emailHandler.GetDrafts)

	// Account settings
	email.Put("/accounts/:accountId/signature", emailHandler.UpdateSignature)
	email.Put("/accounts/:accountId/send-delay", emailHandler.UpdateSendDelay)

	// Email labels
	email.Get("/accounts/:accountId/labels", emailHandler.GetLabels)
	email.Post("/accounts/:accountId/labels", emailHandler.CreateLabel)
	email.Put("/labels/:labelId", emailHandler.UpdateLabel)
	email.Delete("/labels/:labelId", emailHandler.DeleteLabel)
	email.Get("/labels/:labelId/emails", emailHandler.GetEmailsByLabel)
	email.Get("/emails/:emailId/labels", emailHandler.GetEmailLabels)
	email.Post("/emails/:emailId/labels/:labelId", emailHandler.AssignLabel)
	email.Delete("/emails/:emailId/labels/:labelId", emailHandler.RemoveLabel)

	// Email rules
	email.Get("/accounts/:accountId/rules", emailHandler.GetRules)
	email.Post("/accounts/:accountId/rules", emailHandler.CreateRule)
	email.Put("/rules/:ruleId", emailHandler.UpdateRule)
	email.Post("/rules/:ruleId/run", emailHandler.RunRule)
	email.Delete("/rules/:ruleId", emailHandler.DeleteRule)

	// Email attachments
	email.Get("/attachments/:attachmentId", emailHandler.GetAttachment)
	email.Get("/attachments/:attachmentId/download", emailHandler.DownloadAttachment)

	// Calendar routes (optional module)
	calendar := protected.Group("/calendar")
	calendar.Get("/events", calendarHandler.ListEvents)
	calendar.Post("/events", calendarHandler.CreateEvent)
	calendar.Delete("/events/by-task/:taskId", calendarHandler.DeleteEventByTask)
	calendar.Get("/events/:id", calendarHandler.GetEvent)
	calendar.Put("/events/:id", calendarHandler.UpdateEvent)
	calendar.Delete("/events/:id", calendarHandler.DeleteEvent)

	// Contacts routes (optional module)
	contacts := protected.Group("/contacts")
	contacts.Get("/", contactsHandler.ListContacts)
	contacts.Post("/", contactsHandler.CreateContact)
	contacts.Get("/:id", contactsHandler.GetContact)
	contacts.Put("/:id", contactsHandler.UpdateContact)
	contacts.Patch("/:id/favorite", contactsHandler.ToggleFavorite)
	contacts.Delete("/:id", contactsHandler.DeleteContact)

	// Public share access (no auth required)
	api.Get("/share/:token", fileHandler.GetShare)
	api.Get("/share/:token/download", fileHandler.DownloadShare)

	// WebSocket endpoint (requires authentication via short-lived ticket)
	api.Use("/ws", func(c *fiber.Ctx) error {
		// Check if it's a WebSocket upgrade request
		if websocket.IsWebSocketUpgrade(c) {
			// Validate ticket from query parameter
			// Tickets are short-lived (30s) and single-use, avoiding JWT exposure in URL
			ticket := c.Query("ticket")
			if ticket == "" {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Ticket required",
				})
			}

			// Validate ticket and get user ID
			userID, err := authService.ValidateWebSocketTicket(c.Context(), ticket)
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Invalid or expired ticket",
				})
			}

			c.Locals("userID", userID.String())
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	api.Get("/ws", websocket.New(wsHandler.HandleConnection))

	// WebDAV endpoint (handles all methods)
	s.app.All("/webdav/*", webdavServer.Handler())
	s.app.All("/webdav", webdavServer.Handler())
}

// Start begins listening for requests
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	return s.app.Listen(addr)
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown() error {
	// Stop background jobs first
	s.scheduler.Stop()
	s.jobWorker.Stop()

	// Stop all rate limiters to prevent goroutine leaks
	middleware.StopAllRateLimiters()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = ctx // For future use with graceful shutdown
	return s.app.Shutdown()
}

// customErrorHandler handles errors returned from handlers
func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(fiber.Map{
		"error": err.Error(),
	})
}

// jobsStatsHandler returns job queue statistics
func (s *Server) jobsStatsHandler(c *fiber.Ctx) error {
	queue := s.jobWorker
	if queue == nil {
		return c.JSON(fiber.Map{
			"status": "not available",
		})
	}

	// Get stats from the memory queue
	return c.JSON(fiber.Map{
		"status":  "running",
		"workers": 4,
	})
}
