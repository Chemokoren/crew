// AMY MIS — Backend Server
// A Workforce Financial Operating System for Informal Economies

// @title AMY MIS API
// @version 1.0
// @description Workforce Financial Operating System for Kenya's informal transport sector.
// @description Manages crew assignments, earnings, wallets, payroll, and SACCO operations.

// @contact.name AMY MIS Engineering
// @contact.email engineering@amy-mis.co.ke

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter "Bearer {token}" (without quotes)

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kibsoft/amy-mis/internal/config"
	"github.com/kibsoft/amy-mis/internal/database"
	"github.com/kibsoft/amy-mis/internal/external/identity"
	"github.com/kibsoft/amy-mis/internal/external/iprs"
	"github.com/kibsoft/amy-mis/internal/external/jambopay"
	"github.com/kibsoft/amy-mis/internal/external/payment"
	"github.com/kibsoft/amy-mis/internal/external/payroll"
	"github.com/kibsoft/amy-mis/internal/external/perpay"
	"github.com/kibsoft/amy-mis/internal/external/sms"
	"github.com/kibsoft/amy-mis/internal/external/storage"
	"github.com/kibsoft/amy-mis/internal/handler"
	"github.com/kibsoft/amy-mis/internal/middleware"
	pgRepo "github.com/kibsoft/amy-mis/internal/repository/postgres"
	"github.com/kibsoft/amy-mis/internal/service"
	"github.com/kibsoft/amy-mis/internal/worker"
	"github.com/kibsoft/amy-mis/pkg/jwt"
	"github.com/kibsoft/amy-mis/pkg/types"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/kibsoft/amy-mis/docs" // swagger docs
)

func main() {
	// --- 1. Setup structured logging ---
	var logHandler slog.Handler
	logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	slog.Info("starting AMY MIS server...")

	// --- 2. Load configuration ---
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if cfg.IsDevelopment() {
		slog.Info("running in development mode")
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
		logger = slog.New(logHandler)
		slog.SetDefault(logger)
	}

	// --- 3. Connect to PostgreSQL ---
	db, err := database.Connect(cfg.DatabaseURL, cfg.IsDevelopment())
	if err != nil {
		slog.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// --- 4. Run database migrations ---
	if err := database.RunMigrations(cfg.DatabaseURL, cfg.MigrationsPath); err != nil {
		slog.Error("failed to run database migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// --- 5. Connect to Redis ---
	redisClient, err := database.ConnectRedis(cfg.RedisURL)
	if err != nil {
		slog.Error("failed to connect to Redis", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// --- 6. Connect to MinIO ---
	minioClient, err := storage.NewMinIOClient(
		cfg.MinIOEndpoint,
		cfg.MinIOAccessKey,
		cfg.MinIOSecretKey,
		cfg.MinIOBucket,
		cfg.MinIOUseSSL,
	)
	if err != nil {
		slog.Error("failed to connect to MinIO", slog.String("error", err.Error()))
		os.Exit(1)
	}
	_ = minioClient // Injected into document handlers in a future phase

	// --- 7. Initialize repositories ---
	userRepo := pgRepo.NewUserRepo(db)
	crewRepo := pgRepo.NewCrewRepo(db)
	walletRepo := pgRepo.NewWalletRepo(db)
	assignmentRepo := pgRepo.NewAssignmentRepo(db)
	earningRepo := pgRepo.NewEarningRepo(db)
	saccoRepo := pgRepo.NewSACCORepo(db)
	vehicleRepo := pgRepo.NewVehicleRepo(db)
	routeRepo := pgRepo.NewRouteRepo(db)
	payrollRepo := pgRepo.NewPayrollRepo(db)
	membershipRepo := pgRepo.NewMembershipRepo(db)
	floatRepo := pgRepo.NewSACCOFloatRepo(db)
	documentRepo := pgRepo.NewDocumentRepo(db)
	notificationRepo := pgRepo.NewNotificationRepo(db)
	auditRepo := pgRepo.NewAuditLogRepo(db)
	statutoryRateRepo := pgRepo.NewStatutoryRateRepo(db)

	// --- 8. Initialize transaction manager ---
	txMgr := database.NewTxManager(db)

	// --- 9. Initialize JWT manager ---
	jwtManager := jwt.NewManager(cfg.JWTSecret, cfg.JWTExpiryMinutes, cfg.JWTRefreshDays)

	// --- 10. Initialize external integration managers (Identity) ---
	var iprsProvider *iprs.IPRSProvider
	if cfg.IPRSClientID != "" {
		iprsProvider = iprs.NewIPRSProvider(iprs.IPRSConfig{
			BaseURL:             cfg.IPRSBaseURL,
			AccessTokenEndpoint: cfg.IPRSTokenEndpoint,
			ClientID:            cfg.IPRSClientID,
			ClientSecret:        cfg.IPRSClientSecret,
		}, logger)
	}

	// --- 11. Initialize external integration managers (Strategy pattern) ---

	// SMS: Optimize (default) + Africa's Talking (fallback)
	var smsProviders []sms.Provider
	if cfg.SMSClientID != "" {
		smsProviders = append(smsProviders, sms.NewOptimizeProvider(sms.OptimizeConfig{
			ClientID:           cfg.SMSClientID,
			ClientSecret:       cfg.SMSClientSecret,
			TokenURL:           cfg.SMSTokenURL,
			SMSURL:             cfg.SMSURL,
			SenderID:           cfg.SMSSenderID,
			CallbackURL:        cfg.SMSCallbackURL,
			TokenExpirySeconds: cfg.SMSTokenExpirySec,
		}, logger))
	}
	if cfg.ATAPIKey != "" {
		smsProviders = append(smsProviders, sms.NewAfricasTalkingProvider(sms.AfricasTalkingConfig{
			APIKey:    cfg.ATAPIKey,
			Username:  cfg.ATUsername,
			Shortcode: cfg.ATShortCode,
			BaseURL:   cfg.ATBaseURL,
		}, logger))
	}
	var smsMgr *sms.Manager
	if len(smsProviders) > 0 {
		smsMgr = sms.NewManager(logger, smsProviders...)
	} else {
		slog.Warn("no SMS providers configured — SMS functionality disabled")
	}

	// --- 12. Initialize services ---
	notifSvc := service.NewNotificationService(notificationRepo, userRepo, smsMgr, logger)
	authSvc := service.NewAuthService(userRepo, crewRepo, jwtManager, txMgr, logger)
	crewSvc := service.NewCrewService(crewRepo, iprsProvider, logger)
	walletSvc := service.NewWalletService(walletRepo, crewRepo, logger)
	assignmentSvc := service.NewAssignmentService(assignmentRepo, earningRepo, walletSvc, notifSvc, txMgr, logger)
	saccoSvc := service.NewSACCOService(saccoRepo, membershipRepo, floatRepo, logger)
	vehicleSvc := service.NewVehicleService(vehicleRepo, logger)
	routeSvc := service.NewRouteService(routeRepo, logger)
	docSvc := service.NewDocumentService(documentRepo, logger)
	_ = service.NewAuditService(auditRepo, logger) // Wired into middleware hooks

	// --- 13. Initialize handlers ---
	healthHandler := handler.NewHealthHandler(db, redisClient)
	authHandler := handler.NewAuthHandler(authSvc)
	crewHandler := handler.NewCrewHandler(crewSvc)
	walletHandler := handler.NewWalletHandler(walletSvc)
	assignmentHandler := handler.NewAssignmentHandler(assignmentSvc)
	saccoHandler := handler.NewSACCOHandler(saccoSvc)
	vehicleHandler := handler.NewVehicleHandler(vehicleSvc)
	routeHandler := handler.NewRouteHandler(routeSvc)
	notifHandler := handler.NewNotificationHandler(notifSvc)
	docHandler := handler.NewDocumentHandler(docSvc, minioClient)
	earningHandler := handler.NewEarningHandler(earningRepo)

	// Payment: JamboPay



	// Payment: JamboPay
	var paymentMgr *payment.Manager
	if cfg.JamboPayClientID != "" {
		jp := jambopay.NewJamboPayProvider(jambopay.JamboPayConfig{
			BaseURL:      cfg.JamboPayBaseURL,
			ClientID:     cfg.JamboPayClientID,
			ClientSecret: cfg.JamboPayClientSecret,
		}, logger)
		paymentMgr = payment.NewManager(logger, jp)
	} else {
		slog.Warn("JamboPay not configured — payout functionality disabled")
	}
	
	// Initialize PayoutService after paymentMgr is available
	payoutSvc := service.NewPayoutService(walletSvc, paymentMgr, logger)
	payoutHandler := handler.NewPayoutHandler(payoutSvc)

	// Payroll: PerPay
	var payrollMgr *payroll.Manager
	if cfg.PerpayClientID != "" {
		pp := perpay.NewPerPayProvider(perpay.PerPayConfig{
			BaseURL:      cfg.PerpayBaseURL,
			ClientID:     cfg.PerpayClientID,
			ClientSecret: cfg.PerpayClientSecret,
		}, logger)
		payrollMgr = payroll.NewManager(logger, pp)
	} else {
		slog.Warn("PerPay not configured — payroll submission disabled")
	}

	payrollSvc := service.NewPayrollService(payrollRepo, earningRepo, statutoryRateRepo, crewRepo, payrollMgr, logger)
	payrollHandler := handler.NewPayrollHandler(payrollSvc)

	// Identity: IPRS
	var identityMgr *identity.Manager
	if iprsProvider != nil {
		identityMgr = identity.NewManager(logger, iprsProvider)
	} else {
		slog.Warn("IPRS not configured — identity verification disabled")
	}
	_ = identityMgr // Injected into KYC service in a future phase

	// --- 13. Initialize background workers ---
	scheduler := worker.NewScheduler(logger)
	dailySummaryJob := worker.NewDailySummaryJob(earningRepo, assignmentRepo, logger)
	scheduler.Register(dailySummaryJob.AsJob())
	scheduler.Start()

	// --- 14. Setup Gin router ---
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(middleware.CORS())
	router.Use(middleware.SecureHeaders())
	router.Use(middleware.RequestID())
	router.Use(middleware.RateLimit(100, time.Minute)) // 100 req/min per IP
	router.Use(middleware.Timeout(30 * time.Second))
	router.Use(middleware.MetricsMiddleware())
	router.Use(middleware.Logger(logger))
	router.Use(middleware.Recovery(logger))

	// --- 14. Register routes ---

	// Health, readiness, and metrics (no auth)
	router.GET("/health", healthHandler.Health)
	router.GET("/ready", healthHandler.Ready)
	router.GET("/metrics", middleware.MetricsHandler())

	// Swagger API documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 — public endpoints
	v1 := router.Group("/api/v1")

	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
	}

	// API v1 — authenticated endpoints
	secured := v1.Group("")
	secured.Use(middleware.JWTAuth(jwtManager))
	{
		// Current user
		secured.GET("/auth/me", authHandler.Me)

		// Crew members (SACCO admins & system admins)
		crew := secured.Group("/crew")
		crew.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			crew.POST("", crewHandler.Create)
			crew.GET("", crewHandler.List)
			crew.GET("/:id", crewHandler.GetByID)
			crew.PUT("/:id/kyc", crewHandler.UpdateKYC)
			crew.POST("/:id/verify", crewHandler.VerifyNationalID)
			crew.DELETE("/:id", crewHandler.Deactivate)
		}

		// Assignments
		assignments := secured.Group("/assignments")
		assignments.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			assignments.POST("", assignmentHandler.Create)
			assignments.GET("", assignmentHandler.List)
			assignments.GET("/:id", assignmentHandler.GetByID)
			assignments.POST("/:id/complete", assignmentHandler.Complete)
		}

		// Wallets (system admin only for direct credit/debit; crew can view own)
		wallets := secured.Group("/wallets")
		{
			wallets.GET("/:crew_member_id", walletHandler.GetBalance)
			wallets.GET("/:crew_member_id/transactions", walletHandler.ListTransactions)

			walletAdmin := wallets.Group("")
			walletAdmin.Use(middleware.RequireRole(types.RoleSystemAdmin))
			{
				walletAdmin.POST("/credit", walletHandler.Credit)
				walletAdmin.POST("/debit", walletHandler.Debit)
			}
			
			wallets.POST("/:crew_member_id/payout", payoutHandler.Payout)
		}

		// SACCOs (system admin + sacco admin)
		saccos := secured.Group("/saccos")
		saccos.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			saccos.POST("", saccoHandler.Create)
			saccos.GET("", saccoHandler.List)
			saccos.GET("/:id", saccoHandler.GetByID)
			saccos.PUT("/:id", saccoHandler.Update)
			saccos.DELETE("/:id", saccoHandler.Delete)
			saccos.GET("/:id/members", saccoHandler.ListMembers)
			saccos.POST("/:id/members", saccoHandler.AddMember)
			saccos.DELETE("/:id/members/:membership_id", saccoHandler.RemoveMember)
			saccos.GET("/:id/float", saccoHandler.GetFloat)
			saccos.POST("/:id/float/credit", saccoHandler.CreditFloat)
			saccos.POST("/:id/float/debit", saccoHandler.DebitFloat)
		}

		// Vehicles
		vehicles := secured.Group("/vehicles")
		vehicles.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			vehicles.POST("", vehicleHandler.Create)
			vehicles.GET("", vehicleHandler.List)
			vehicles.GET("/:id", vehicleHandler.GetByID)
			vehicles.PUT("/:id", vehicleHandler.Update)
			vehicles.DELETE("/:id", vehicleHandler.Delete)
		}

		// Routes
		routes := secured.Group("/routes")
		routes.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			routes.POST("", routeHandler.Create)
			routes.GET("", routeHandler.List)
			routes.GET("/:id", routeHandler.GetByID)
			routes.PUT("/:id", routeHandler.Update)
			routes.DELETE("/:id", routeHandler.Delete)
		}

		// Payroll (system admin + sacco admin)
		payrollRoutes := secured.Group("/payroll")
		payrollRoutes.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			payrollRoutes.POST("", payrollHandler.Create)
			payrollRoutes.GET("", payrollHandler.List)
			payrollRoutes.GET("/:id", payrollHandler.GetByID)
			payrollRoutes.GET("/:id/entries", payrollHandler.GetEntries)
			payrollRoutes.POST("/:id/process", payrollHandler.Process)
			payrollRoutes.POST("/:id/approve", payrollHandler.Approve)
		}

		// Documents
		documents := secured.Group("/documents")
		documents.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			documents.POST("/upload", docHandler.Upload)
			documents.GET("/:id/download", docHandler.Download)
			documents.GET("", docHandler.List)
			documents.DELETE("/:id", docHandler.Delete)
		}

		// Earnings
		earnings := secured.Group("/earnings")
		{
			earnings.GET("", earningHandler.List)
			earnings.GET("/summary/:crew_member_id", earningHandler.SummaryDashboard)
		}

		// Notifications (all authenticated users)
		notifications := secured.Group("/notifications")
		{
			notifications.GET("", notifHandler.List)
			notifications.PUT("/:id/read", notifHandler.MarkRead)
			notifications.PUT("/preferences", notifHandler.UpdatePreferences)
		}
	}

	// --- 15. Start HTTP server ---
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("HTTP server started",
			slog.Int("port", cfg.Port),
			slog.String("env", cfg.Environment),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// --- 16. Graceful shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	slog.Info("shutdown signal received", slog.String("signal", sig.String()))

	// Create shutdown context with 30s timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Stop background workers
	scheduler.Stop()
	slog.Info("background workers stopped")

	// 2. Stop accepting new HTTP requests, drain in-flight
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("HTTP server shutdown error", slog.String("error", err.Error()))
	}
	slog.Info("HTTP server stopped")

	// 3. Close Redis
	if err := redisClient.Close(); err != nil {
		slog.Error("Redis close error", slog.String("error", err.Error()))
	}
	slog.Info("Redis connection closed")

	// 4. Close database
	sqlDB, err := db.DB()
	if err == nil {
		if err := sqlDB.Close(); err != nil {
			slog.Error("database close error", slog.String("error", err.Error()))
		}
	}
	slog.Info("database connection closed")

	slog.Info("AMY MIS server shutdown complete")
}
