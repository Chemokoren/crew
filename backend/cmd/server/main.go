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
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
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
	"github.com/kibsoft/amy-mis/internal/credit"
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
	db, err := database.Connect(cfg.DatabaseURL, cfg.IsDevelopment(), database.PoolConfig{
		MaxOpenConns:   cfg.DBMaxOpenConns,
		MaxIdleConns:   cfg.DBMaxIdleConns,
		ConnMaxLifeMin: cfg.DBConnMaxLifeMin,
		ConnMaxIdleMin: cfg.DBConnMaxIdleMin,
	})
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

	// --- 6. Connect to MinIO (optional — non-fatal if unavailable) ---
	var minioClient *storage.MinIOClient
	if cfg.MinIOEndpoint != "" {
		mc, err := storage.NewMinIOClient(
			cfg.MinIOEndpoint,
			cfg.MinIOAccessKey,
			cfg.MinIOSecretKey,
			cfg.MinIOBucket,
			cfg.MinIOUseSSL,
		)
		if err != nil {
			slog.Warn("MinIO unavailable — document upload/download disabled",
				slog.String("endpoint", cfg.MinIOEndpoint),
				slog.String("error", err.Error()),
			)
		} else {
			minioClient = mc
		}
	} else {
		slog.Warn("MinIO not configured — document upload/download disabled")
	}


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
	notificationPrefRepo := pgRepo.NewNotificationPreferenceRepo(db)
	auditRepo := pgRepo.NewAuditLogRepo(db)
	statutoryRateRepo := pgRepo.NewStatutoryRateRepo(db)
	webhookRepo := pgRepo.NewWebhookEventRepo(db)
	creditScoreRepo := pgRepo.NewCreditScoreRepo(db)
	loanRepo := pgRepo.NewLoanApplicationRepo(db)
	insuranceRepo := pgRepo.NewInsurancePolicyRepo(db)
	snapshotRepo := pgRepo.NewWalletSnapshotRepo(db)
	scoreHistoryRepo := pgRepo.NewCreditScoreHistoryRepo(db)
	negativeEventRepo := pgRepo.NewNegativeEventRepo(db)

	// --- 8. Initialize transaction manager ---
	txMgr := database.NewTxManager(db)

	// --- 9. Initialize JWT manager ---
	jwtManager := jwt.NewManager(cfg.JWTSecret, cfg.JWTExpiryMinutes, cfg.JWTRefreshDays)

	// --- 10. Initialize external integration managers ---
	// All integrations follow the Strategy pattern with config-driven enable/disable.
	// Set *_ENABLED=false to disable a provider, or change *_PRIMARY_PROVIDER to switch.

	// --- 10a. Identity/KYC: IPRS ---
	var iprsProvider *iprs.IPRSProvider
	if cfg.IdentityIPRSEnabled && cfg.IPRSClientID != "" {
		iprsProvider = iprs.NewIPRSProvider(iprs.IPRSConfig{
			BaseURL:             cfg.IPRSBaseURL,
			AccessTokenEndpoint: cfg.IPRSTokenEndpoint,
			ClientID:            cfg.IPRSClientID,
			ClientSecret:        cfg.IPRSClientSecret,
		}, logger)
		slog.Info("IPRS identity provider enabled")
	} else {
		slog.Warn("IPRS identity provider disabled",
			slog.Bool("enabled", cfg.IdentityIPRSEnabled),
			slog.Bool("credentials_present", cfg.IPRSClientID != ""),
		)
	}

	// --- 10b. SMS: Optimize (primary) + Africa's Talking (fallback) ---
	var smsProviders []sms.Provider
	if cfg.SMSOptimizeEnabled && cfg.SMSClientID != "" {
		smsProviders = append(smsProviders, sms.NewOptimizeProvider(sms.OptimizeConfig{
			ClientID:           cfg.SMSClientID,
			ClientSecret:       cfg.SMSClientSecret,
			TokenURL:           cfg.SMSTokenURL,
			SMSURL:             cfg.SMSURL,
			SenderID:           cfg.SMSSenderID,
			CallbackURL:        cfg.SMSCallbackURL,
			TokenExpirySeconds: cfg.SMSTokenExpirySec,
		}, logger))
		slog.Info("Optimize SMS provider enabled")
	}
	if cfg.SMSATEnabled && cfg.ATAPIKey != "" {
		smsProviders = append(smsProviders, sms.NewAfricasTalkingProvider(sms.AfricasTalkingConfig{
			APIKey:    cfg.ATAPIKey,
			Username:  cfg.ATUsername,
			Shortcode: cfg.ATShortCode,
			BaseURL:   cfg.ATBaseURL,
		}, logger))
		slog.Info("Africa's Talking SMS provider enabled")
	}
	var smsMgr *sms.Manager
	if len(smsProviders) > 0 {
		smsMgr = sms.NewManager(logger, smsProviders...)
		// Set the configured primary (no-op if it's already first)
		if err := smsMgr.SetPrimary(cfg.SMSPrimaryProvider); err != nil {
			slog.Warn("SMS primary provider not found, using default order",
				slog.String("requested", cfg.SMSPrimaryProvider),
			)
		}
	} else {
		slog.Warn("no SMS providers configured — SMS functionality disabled")
	}

	// --- 12. Initialize services ---
	auditSvc := service.NewAuditService(auditRepo, logger)
	notifSvc := service.NewNotificationService(notificationRepo, notificationPrefRepo, userRepo, smsMgr, logger)
	authSvc := service.NewAuthService(userRepo, crewRepo, jwtManager, txMgr, logger)

	// CrewService: Identity provider is optional — system continues without it (graceful degradation).
	// If IPRS is available, wrap it in the identity Manager for failover support.
	var crewIdProvider identity.Provider
	if iprsProvider != nil {
		identityMgr := identity.NewManager(logger, iprsProvider)
		if err := identityMgr.SetPrimary(cfg.IdentityPrimaryProvider); err != nil {
			slog.Warn("identity primary provider not found, using default order",
				slog.String("requested", cfg.IdentityPrimaryProvider),
			)
		}
		crewIdProvider = identityMgr
	} else {
		slog.Warn("no identity providers configured — identity verification disabled")
	}
	crewSvc := service.NewCrewService(crewRepo, crewIdProvider, logger)

	walletSvc := service.NewWalletService(walletRepo, crewRepo, auditSvc, logger)
	assignmentSvc := service.NewAssignmentService(assignmentRepo, earningRepo, walletSvc, notifSvc, txMgr, logger)
	saccoSvc := service.NewSACCOService(saccoRepo, membershipRepo, floatRepo, auditSvc, logger)
	vehicleSvc := service.NewVehicleService(vehicleRepo, logger)
	routeSvc := service.NewRouteService(routeRepo, logger)
	docSvc := service.NewDocumentService(documentRepo, logger)
	// --- Credit Scoring Engine (V3 architecture) ---
	featureComputer := credit.NewFeatureComputer(
		earningRepo, assignmentRepo, walletRepo, loanRepo,
		insuranceRepo, crewRepo, userRepo, snapshotRepo, negativeEventRepo, logger,
	)
	creditScorer := credit.NewRulesScorer() // Swap to MLScorer/HybridScorer for V3
	creditEngine := credit.NewEngine(featureComputer, creditScorer, creditScoreRepo, scoreHistoryRepo, logger)
	creditSvc := service.NewCreditService(creditEngine, creditScoreRepo, scoreHistoryRepo)
	loanSvc := service.NewLoanService(loanRepo, creditScoreRepo, walletRepo, txMgr)
	insuranceSvc := service.NewInsuranceService(insuranceRepo, logger)

	// --- 13. Initialize handlers ---
	healthHandler := handler.NewHealthHandler(db, redisClient)
	authHandler := handler.NewAuthHandler(authSvc)
	crewHandler := handler.NewCrewHandler(crewSvc)
	walletHandler := handler.NewWalletHandler(walletSvc, cfg.CSVExportMaxRows)
	assignmentHandler := handler.NewAssignmentHandler(assignmentSvc)
	saccoHandler := handler.NewSACCOHandler(saccoSvc)
	vehicleHandler := handler.NewVehicleHandler(vehicleSvc)
	routeHandler := handler.NewRouteHandler(routeSvc)
	notifHandler := handler.NewNotificationHandler(notifSvc)
	docHandler := handler.NewDocumentHandler(docSvc, minioClient)
	earningHandler := handler.NewEarningHandler(earningRepo)
	creditHandler := handler.NewCreditHandler(creditSvc)
	loanHandler := handler.NewLoanHandler(loanSvc)
	insuranceHandler := handler.NewInsuranceHandler(insuranceSvc)
	adminHandler := handler.NewAdminHandler(authSvc, notifSvc, auditRepo, statutoryRateRepo)


	// --- 13a. Payment: JamboPay (config-driven) ---
	var paymentProviders []payment.Provider
	if cfg.PaymentJamboPayEnabled && cfg.JamboPayClientID != "" {
		jp := jambopay.NewJamboPayProvider(jambopay.JamboPayConfig{
			BaseURL:      cfg.JamboPayBaseURL,
			ClientID:     cfg.JamboPayClientID,
			ClientSecret: cfg.JamboPayClientSecret,
		}, logger)
		paymentProviders = append(paymentProviders, jp)
		slog.Info("JamboPay payment provider enabled")
	}
	// Future: M-Pesa direct provider
	// if cfg.PaymentMpesaEnabled && cfg.MpesaConsumerKey != "" {
	//     mp := mpesa.NewMpesaProvider(mpesa.MpesaConfig{...}, logger)
	//     paymentProviders = append(paymentProviders, mp)
	//     slog.Info("M-Pesa payment provider enabled")
	// }

	var paymentMgr *payment.Manager
	if len(paymentProviders) > 0 {
		paymentMgr = payment.NewManager(logger, paymentProviders...)
		if err := paymentMgr.SetPrimary(cfg.PaymentPrimaryProvider); err != nil {
			slog.Warn("payment primary provider not found, using default order",
				slog.String("requested", cfg.PaymentPrimaryProvider),
			)
		}
	} else {
		slog.Warn("no payment providers configured — payout functionality disabled")
	}
	
	// Initialize PayoutService after paymentMgr is available
	payoutSvc := service.NewPayoutService(walletSvc, paymentMgr, auditSvc, logger)
	payoutHandler := handler.NewPayoutHandler(payoutSvc)

	// --- 13b. Payroll: PerPay (config-driven) ---
	var payrollProviders []payroll.Provider
	if cfg.PayrollPerpayEnabled && cfg.PerpayClientID != "" {
		pp := perpay.NewPerPayProvider(perpay.PerPayConfig{
			BaseURL:      cfg.PerpayBaseURL,
			ClientID:     cfg.PerpayClientID,
			ClientSecret: cfg.PerpayClientSecret,
		}, logger)
		payrollProviders = append(payrollProviders, pp)
		slog.Info("PerPay payroll provider enabled")
	}

	var payrollMgr *payroll.Manager
	if len(payrollProviders) > 0 {
		payrollMgr = payroll.NewManager(logger, payrollProviders...)
		if err := payrollMgr.SetPrimary(cfg.PayrollPrimaryProvider); err != nil {
			slog.Warn("payroll primary provider not found, using default order",
				slog.String("requested", cfg.PayrollPrimaryProvider),
			)
		}
	} else {
		slog.Warn("no payroll providers configured — payroll submission disabled")
	}

	payrollSvc := service.NewPayrollService(payrollRepo, earningRepo, statutoryRateRepo, crewRepo, payrollMgr, logger)
	payrollHandler := handler.NewPayrollHandler(payrollSvc)

	webhookSvc := service.NewWebhookService(webhookRepo, payoutSvc, payrollSvc, walletRepo, payrollRepo, logger)
	webhookHandler := handler.NewWebhookHandler(webhookSvc, cfg.WebhookJamboPaySecret, cfg.WebhookPerpaySecret)

	scheduler := worker.NewScheduler(logger, redisClient)
	dailySummaryJob := worker.NewDailySummaryJob(earningRepo, assignmentRepo, logger)
	scheduler.Register(dailySummaryJob.AsJob())

	// New background jobs
	insuranceLapseJob := worker.NewInsuranceLapseJob(insuranceRepo, logger)
	scheduler.Register(insuranceLapseJob.AsJob())

	payrollAutoSubmitJob := worker.NewPayrollAutoSubmitJob(payrollSvc, payrollRepo, logger)
	scheduler.Register(payrollAutoSubmitJob.AsJob())

	walletReconJob := worker.NewWalletReconciliationJob(walletRepo, logger)
	scheduler.Register(walletReconJob.AsJob())

	balanceSnapshotJob := worker.NewBalanceSnapshotJob(walletRepo, snapshotRepo, logger)
	scheduler.Register(balanceSnapshotJob.AsJob())

	loanDefaultJob := worker.NewLoanDefaultDetectorJob(loanSvc, logger)
	scheduler.Register(loanDefaultJob.AsJob())

	scheduler.Start()

	// --- 14. Setup Gin router ---
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	router.Use(middleware.SecureHeaders())
	router.Use(middleware.RequestID())
	router.Use(otelgin.Middleware("amy-mis-api")) // OTEL distributed traces
	router.Use(middleware.MaxBodySize(int64(cfg.MaxRequestBodyMB) << 20)) // Configurable body size limit
	router.Use(middleware.RateLimit(redisClient, cfg.RateLimitRPM, time.Minute))
	router.Use(middleware.Timeout(time.Duration(cfg.RequestTimeoutSec) * time.Second))
	router.Use(middleware.MetricsMiddleware())
	router.Use(middleware.Logger(logger))
	router.Use(middleware.Recovery(logger))

	// --- 14. Register routes ---

	// Root redirect → Swagger docs
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

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
		auth.POST("/change-password", adminHandler.ChangePassword) // Password change for all users
		auth.GET("/lookup", authHandler.Lookup)                    // USSD user identification
		auth.POST("/pin", authHandler.SetPIN)                      // USSD PIN setup
		auth.POST("/pin/verify", authHandler.VerifyPIN)            // USSD PIN verification
	}

	webhooks := v1.Group("/webhooks")
	{
		webhooks.POST("/jambopay", webhookHandler.HandleJamboPay)
		webhooks.POST("/perpay", webhookHandler.HandlePerpay)
	}

	// API v1 — authenticated endpoints
	serviceAPIKey := os.Getenv("SERVICE_API_KEY")
	secured := v1.Group("")
	secured.Use(middleware.JWTAuth(jwtManager, serviceAPIKey))
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
			crew.POST("/bulk-import", crewHandler.BulkImport)
			crew.GET("/search", crewHandler.SearchByNationalID)
		}

		// Assignments
		assignments := secured.Group("/assignments")
		assignments.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			assignments.POST("", assignmentHandler.Create)
			assignments.GET("", assignmentHandler.List)
			assignments.GET("/:id", assignmentHandler.GetByID)
			assignments.POST("/:id/complete", assignmentHandler.Complete)
			assignments.POST("/:id/cancel", assignmentHandler.Cancel)
			assignments.POST("/:id/reassign", assignmentHandler.Reassign)
		}

		// Wallets (system admin only for direct credit/debit; crew can view own)
		wallets := secured.Group("/wallets")
		{
			wallets.GET("/:crew_member_id", walletHandler.GetBalance)
			wallets.GET("/:crew_member_id/transactions", walletHandler.ListTransactions)
			wallets.GET("/:crew_member_id/export", walletHandler.ExportCSV)

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
			saccos.GET("/:id/float/transactions", saccoHandler.ListFloatTransactions)
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
			payrollRoutes.POST("/:id/submit", payrollHandler.Submit)
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
			notifications.GET("/preferences", notifHandler.GetPreferences)
			notifications.PUT("/preferences", notifHandler.UpdatePreferences)
		}

		// Financials: Credit
		credit := secured.Group("/credit")
		{
			credit.GET("/:crew_member_id", creditHandler.GetScore)
			credit.GET("/:crew_member_id/detailed", creditHandler.GetDetailedScore)
			credit.GET("/:crew_member_id/history", creditHandler.GetScoreHistory)
			credit.POST("/:crew_member_id/calculate", creditHandler.CalculateScore)
		}

		// Financials: Loans
		loans := secured.Group("/loans")
		{
			loans.POST("", loanHandler.Apply)
			loans.GET("", loanHandler.List)
			loans.GET("/tier/:crew_member_id", loanHandler.GetTier)
			loans.POST("/:id/repay", loanHandler.Repay)
			
			loanAdmin := loans.Group("")
			loanAdmin.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleLender))
			{
				loanAdmin.POST("/:id/approve", loanHandler.Approve)
				loanAdmin.POST("/:id/reject", loanHandler.Reject)
				loanAdmin.POST("/:id/disburse", loanHandler.Disburse)
			}
		}

		// Financials: Insurance
		insurance := secured.Group("/insurance")
		{
			insurance.GET("", insuranceHandler.List)
			
			insuranceAdmin := insurance.Group("")
			insuranceAdmin.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleInsurer))
			{
				insuranceAdmin.POST("", insuranceHandler.Create)
				insuranceAdmin.POST("/:id/lapse", insuranceHandler.Lapse)
			}
		}

		// Admin dashboard
		admin := secured.Group("/admin")
		admin.Use(middleware.RequireRole(types.RoleSystemAdmin))
		{
			admin.GET("/stats", adminHandler.SystemStats)
			admin.POST("/users/:id/disable", adminHandler.DisableAccount)
			admin.POST("/users/:id/enable", adminHandler.EnableAccount)
			admin.POST("/users/:id/reset-password", adminHandler.ResetPassword)
			admin.GET("/audit-logs", adminHandler.ListAuditLogs)
			admin.GET("/statutory-rates", adminHandler.ListStatutoryRates)
			admin.GET("/notifications/templates", adminHandler.ListTemplates)
			admin.POST("/notifications/templates", adminHandler.CreateTemplate)
			admin.PUT("/notifications/templates", adminHandler.UpdateTemplate)
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
