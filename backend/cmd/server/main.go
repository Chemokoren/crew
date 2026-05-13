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
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"github.com/kibsoft/amy-mis/internal/config"
	"github.com/kibsoft/amy-mis/internal/database"
	"github.com/kibsoft/amy-mis/internal/external/email"
	"github.com/kibsoft/amy-mis/internal/external/identity"
	"github.com/kibsoft/amy-mis/internal/external/iprs"
	"github.com/kibsoft/amy-mis/internal/external/jambopay"
	"github.com/kibsoft/amy-mis/internal/external/messaging"
	"github.com/kibsoft/amy-mis/internal/external/payment"
	"github.com/kibsoft/amy-mis/internal/external/payroll"
	"github.com/kibsoft/amy-mis/internal/external/perpay"
	"github.com/kibsoft/amy-mis/internal/external/sms"
	"github.com/kibsoft/amy-mis/internal/external/storage"
	"github.com/kibsoft/amy-mis/internal/external/whatsapp"
	"github.com/kibsoft/amy-mis/internal/handler"
	"github.com/kibsoft/amy-mis/internal/middleware"
	"github.com/kibsoft/amy-mis/internal/models"
	pgRepo "github.com/kibsoft/amy-mis/internal/repository/postgres"
	"github.com/kibsoft/amy-mis/internal/credit"
	"github.com/kibsoft/amy-mis/internal/service"
	"github.com/kibsoft/amy-mis/internal/ussd"
	"github.com/kibsoft/amy-mis/internal/worker"
	"github.com/kibsoft/amy-mis/pkg/jwt"
	"github.com/kibsoft/amy-mis/pkg/retry"
	"github.com/kibsoft/amy-mis/pkg/types"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/kibsoft/amy-mis/docs" // swagger docs
)

func main() {
	// --- 1. Setup structured logging (stdout + file) ---
	logDir := os.Getenv("LOG_DIR")
	if logDir == "" {
		logDir = "/var/log/crew"
	}

	// Open log files (create dir if needed)
	var allWriter io.Writer = os.Stdout
	var errorWriter io.Writer
	var logFiles []*os.File

	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "WARN: cannot create log dir %s: %v — logging to stdout only\n", logDir, err)
	} else {
		serverLog, err1 := os.OpenFile(filepath.Join(logDir, "server.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		errLog, err2 := os.OpenFile(filepath.Join(logDir, "error.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err1 != nil || err2 != nil {
			fmt.Fprintf(os.Stderr, "WARN: cannot open log files: %v / %v — logging to stdout only\n", err1, err2)
		} else {
			logFiles = append(logFiles, serverLog, errLog)
			allWriter = io.MultiWriter(os.Stdout, serverLog)
			errorWriter = errLog
		}
	}

	// Ensure log files are flushed on exit
	defer func() {
		for _, f := range logFiles {
			_ = f.Sync()
			_ = f.Close()
		}
	}()

	var logHandler slog.Handler
	logHandler = slog.NewJSONHandler(allWriter, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	// Error-only handler for error.log
	if errorWriter != nil {
		errHandler := slog.NewJSONHandler(errorWriter, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})
		logHandler = &multiHandler{handlers: []slog.Handler{logHandler, errHandler}}
		logger = slog.New(logHandler)
		slog.SetDefault(logger)
	}

	slog.Info("starting AMY MIS server...",
		slog.String("log_dir", logDir),
		slog.String("server_log", filepath.Join(logDir, "server.log")),
		slog.String("error_log", filepath.Join(logDir, "error.log")),
	)

	// --- 2. Load configuration ---
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if cfg.IsDevelopment() {
		slog.Info("running in development mode")
		devHandler := slog.NewTextHandler(allWriter, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
		if errorWriter != nil {
			errHandler := slog.NewJSONHandler(errorWriter, &slog.HandlerOptions{
				Level: slog.LevelWarn,
			})
			logHandler = &multiHandler{handlers: []slog.Handler{devHandler, errHandler}}
		} else {
			logHandler = devHandler
		}
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

	// --- 6. Initialize Storage (MinIO with Local Fallback) ---
	var fileStorage storage.Storage
	if cfg.MinIOEndpoint != "" {
		mc, err := storage.NewMinIOClient(
			cfg.MinIOEndpoint,
			cfg.MinIOAccessKey,
			cfg.MinIOSecretKey,
			cfg.MinIOBucket,
			cfg.MinIOUseSSL,
		)
		if err != nil {
			slog.Warn("MinIO unavailable — falling back to local filesystem storage",
				slog.String("endpoint", cfg.MinIOEndpoint),
				slog.String("error", err.Error()),
			)
			// Fallback to local storage in dev/test
			ls, lerr := storage.NewLocalStorageClient("./storage", "")
			if lerr != nil {
				slog.Error("failed to initialize local storage fallback", slog.String("error", lerr.Error()))
			} else {
				fileStorage = ls
			}
		} else {
			fileStorage = mc
		}
	} else {
		slog.Warn("MinIO not configured — falling back to local filesystem storage")
		ls, _ := storage.NewLocalStorageClient("./storage", "")
		fileStorage = ls
	}


	// --- 7. Initialize repositories ---
	userRepo := pgRepo.NewUserRepo(db)
	crewRepo := pgRepo.NewCrewRepo(db)
	walletRepo := pgRepo.NewWalletRepo(db)
	assignmentRepo := pgRepo.NewAssignmentRepo(db)
	earningRepo := pgRepo.NewEarningRepo(db)
	orgRepo := pgRepo.NewSACCORepo(db)
	vehicleRepo := pgRepo.NewVehicleRepo(db)
	routeRepo := pgRepo.NewRouteRepo(db)
	payrollRepo := pgRepo.NewPayrollRepo(db)
	membershipRepo := pgRepo.NewMembershipRepo(db)
	floatRepo := pgRepo.NewOrganizationFloatRepo(db)
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
	jobTypeRepo := pgRepo.NewTenantJobTypeRepo(db)
	payScheduleRepo := pgRepo.NewPayScheduleRepo(db)
	workSiteRepo := pgRepo.NewWorkSiteRepo(db)

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
		smsMgr.SetRetryPolicy(retry.Policy{
			MaxAttempts:  cfg.RetryMaxAttempts,
			InitialDelay: time.Duration(cfg.RetryInitialDelayMs) * time.Millisecond,
			MaxDelay:     time.Duration(cfg.RetryMaxDelayMs) * time.Millisecond,
		})
	} else {
		slog.Warn("no SMS providers configured — SMS functionality disabled")
	}

	// --- 12. Initialize services ---
	auditSvc := service.NewAuditService(auditRepo, logger)
	notifSvc := service.NewNotificationService(notificationRepo, notificationPrefRepo, userRepo, smsMgr, logger)
	tenantSvc := service.NewTenantService(orgRepo, jobTypeRepo, payScheduleRepo, logger)
	authSvc := service.NewAuthService(userRepo, crewRepo, jwtManager, txMgr, logger,
		service.WithOrgRepo(orgRepo),
		service.WithTenantSvc(tenantSvc),
		service.WithNotifSvc(notifSvc),
	)

	// --- 12a. Email Provider Strategy ---
	var emailProviders []email.Provider
	if cfg.EmailGmailEnabled && cfg.EmailHostUser != "" {
		gmailProvider := email.NewGmailProvider(email.GmailConfig{
			Host:     cfg.EmailHost,
			Port:     cfg.EmailPort,
			Username: cfg.EmailHostUser,
			Password: cfg.EmailHostPassword,
			FromAddr: cfg.EmailFromAddress,
			FromName: cfg.EmailFromName,
			UseTLS:   cfg.EmailUseTLS,
		}, logger)
		emailProviders = append(emailProviders, gmailProvider)
		slog.Info("Gmail SMTP email provider enabled")
	}
	// Future: SendGrid, Twilio SendGrid, etc.
	// if cfg.EmailSendGridEnabled && cfg.SendGridAPIKey != "" {
	//     emailProviders = append(emailProviders, email.NewSendGridProvider(...))
	// }
	var emailMgr *email.Manager
	if len(emailProviders) > 0 {
		emailMgr = email.NewManager(logger, emailProviders...)
		if err := emailMgr.SetPrimary(cfg.EmailPrimaryProvider); err != nil {
			slog.Warn("email primary provider not found, using default order",
				slog.String("requested", cfg.EmailPrimaryProvider),
			)
		}
	} else {
		slog.Warn("no email providers configured — email functionality disabled")
	}

	// --- 12b. WhatsApp Provider Strategy ---
	var whatsappProviders []whatsapp.Provider
	if cfg.WhatsAppMetaEnabled && cfg.WhatsAppPhoneNumberID != "" {
		metaProvider := whatsapp.NewMetaProvider(whatsapp.MetaConfig{
			PhoneNumberID: cfg.WhatsAppPhoneNumberID,
			AccessToken:   cfg.WhatsAppAccessToken,
			APIVersion:    cfg.WhatsAppAPIVersion,
		}, logger)
		whatsappProviders = append(whatsappProviders, metaProvider)
		slog.Info("Meta WhatsApp Cloud API provider enabled")
	}
	// Future: Twilio WhatsApp, etc.
	var whatsappMgr *whatsapp.Manager
	if len(whatsappProviders) > 0 {
		whatsappMgr = whatsapp.NewManager(logger, whatsappProviders...)
		if err := whatsappMgr.SetPrimary(cfg.WhatsAppPrimaryProvider); err != nil {
			slog.Warn("WhatsApp primary provider not found, using default order",
				slog.String("requested", cfg.WhatsAppPrimaryProvider),
			)
		}
	} else {
		slog.Warn("no WhatsApp providers configured — WhatsApp functionality disabled")
	}

	// --- 12c. Unified Messaging Engine ---
	msgEngine := messaging.NewEngine(emailMgr, smsMgr, whatsappMgr, logger)

	// --- 12d. OTP Service (uses messaging engine) ---
	otpSvc := service.NewOTPService(redisClient, msgEngine, service.OTPConfig{
		DefaultChannel: cfg.OTPDefaultChannel,
		Enabled:        cfg.OTPEnabled,
	}, logger)

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
	crewSvc := service.NewCrewService(crewRepo, membershipRepo, crewIdProvider, logger)
	crewSvc.WithUserRepo(userRepo)
	crewSvc.WithNotificationSvc(notifSvc)

	walletSvc := service.NewWalletService(walletRepo, crewRepo, auditSvc, logger)
	assignmentSvc := service.NewAssignmentService(assignmentRepo, earningRepo, walletSvc, notifSvc, txMgr, logger)
	saccoSvc := service.NewOrganizationService(orgRepo, membershipRepo, floatRepo, auditSvc, logger)

	vehicleSvc := service.NewVehicleService(vehicleRepo, logger)
	routeSvc := service.NewRouteService(routeRepo, logger)
	docSvc := service.NewDocumentService(documentRepo, logger)
	// --- Credit Scoring Engine (V3 architecture) ---
	featureComputer := credit.NewFeatureComputer(
		earningRepo, assignmentRepo, walletRepo, loanRepo,
		insuranceRepo, crewRepo, userRepo, snapshotRepo, negativeEventRepo, membershipRepo, logger,
	)
	creditScorer := credit.NewRulesScorer() // Swap to MLScorer/HybridScorer for V3
	creditEngine := credit.NewEngine(featureComputer, creditScorer, creditScoreRepo, scoreHistoryRepo, logger)
	creditSvc := service.NewCreditService(creditEngine, creditScoreRepo, scoreHistoryRepo)
	loanPolicy := buildLoanPolicy(cfg)
	loanSvc := service.NewLoanService(loanRepo, creditScoreRepo, walletRepo, txMgr,
		service.WithLoanPolicy(loanPolicy))
	insuranceSvc := service.NewInsuranceService(insuranceRepo, logger)

	// --- 13. Initialize handlers ---
	healthHandler := handler.NewHealthHandler(db, redisClient)
	authHandler := handler.NewAuthHandler(authSvc, otpSvc)
	authHandler.WithDocUpload(docSvc, fileStorage)
	crewHandler := handler.NewCrewHandler(crewSvc, notifSvc)
	walletHandler := handler.NewWalletHandler(walletSvc, cfg.CSVExportMaxRows)
	assignmentHandler := handler.NewAssignmentHandler(assignmentSvc)
	vehicleHandler := handler.NewVehicleHandler(vehicleSvc)
	routeHandler := handler.NewRouteHandler(routeSvc)
	notifHandler := handler.NewNotificationHandler(notifSvc)
	docHandler := handler.NewDocumentHandler(docSvc, fileStorage)
	earningHandler := handler.NewEarningHandler(earningRepo)
	creditHandler := handler.NewCreditHandler(creditSvc)
	loanHandler := handler.NewLoanHandler(loanSvc)
	insuranceHandler := handler.NewInsuranceHandler(insuranceSvc)
	tenantHandler := handler.NewTenantHandler(tenantSvc)
	adminHandler := handler.NewAdminHandler(authSvc, notifSvc, auditRepo, statutoryRateRepo)
	workSiteHandler := handler.NewWorkSiteHandler(workSiteRepo)

	// --- USSD Session Handler (Phase G) ---
	ussdSession := ussd.NewSessionHandler(
		userRepo, crewRepo, assignmentRepo, earningRepo,
		orgRepo, jobTypeRepo, payScheduleRepo, membershipRepo, walletRepo,
		logger,
	)
	ussdHandler := handler.NewUSSDHandler(ussdSession)


	// --- 13a. Payment: JamboPay (config-driven) ---
	var paymentProviders []payment.Provider
	var jamboPayProvider *jambopay.JamboPayProvider // held for checksum verifier injection
	if cfg.PaymentJamboPayEnabled && cfg.JamboPayClientID != "" {
		jamboPayProvider = jambopay.NewJamboPayProvider(jambopay.JamboPayConfig{
			BaseURL:           cfg.JamboPayBaseURL,
			AuthURL:           cfg.JamboPayAuthURL,
			ClientID:          cfg.JamboPayClientID,
			ClientSecret:      cfg.JamboPayClientSecret,
			CollectionAccount: cfg.JamboPayCollectionAccount,
			PayoutAccount:     cfg.JamboPayPayoutAccount,
			CallbackURL:       cfg.JamboPayCallbackURL,
			PartnerCode:       cfg.JamboPayPartnerCode,
		}, logger)
		paymentProviders = append(paymentProviders, jamboPayProvider)
		slog.Info("JamboPay payment provider enabled",
			slog.String("base_url", cfg.JamboPayBaseURL),
			slog.String("collection_account", cfg.JamboPayCollectionAccount),
			slog.String("payout_account", cfg.JamboPayPayoutAccount),
		)
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
		// Apply admin-configurable retry policy for external integration calls
		paymentMgr.SetRetryPolicy(retry.Policy{
			MaxAttempts:  cfg.RetryMaxAttempts,
			InitialDelay: time.Duration(cfg.RetryInitialDelayMs) * time.Millisecond,
			MaxDelay:     time.Duration(cfg.RetryMaxDelayMs) * time.Millisecond,
		})
	} else {
		slog.Warn("no payment providers configured — payout functionality disabled")
	}

	// Create orgHandler after paymentMgr so it can trigger STK push for float top-ups
	orgHandler := handler.NewOrganizationHandler(saccoSvc, paymentMgr)

	// Inject JamboPay STK poller for callback-less status reconciliation
	if jamboPayProvider != nil {
		orgHandler.WithSTKPoller(jamboPayProvider, floatRepo)
	}

	// Initialize PayoutService after paymentMgr is available
	payoutSvc := service.NewPayoutService(walletSvc, paymentMgr, auditSvc, logger)
	payoutHandler := handler.NewPayoutHandler(payoutSvc)

	// Initialize TransactionService for atomic multi-repo operations (employee payout, wallet transfer)
	transactionSvc := service.NewTransactionService(txMgr, floatRepo, walletSvc, auditSvc, logger)
	transactionHandler := handler.NewTransactionHandler(transactionSvc)

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
		payrollMgr.SetRetryPolicy(retry.Policy{
			MaxAttempts:  cfg.RetryMaxAttempts,
			InitialDelay: time.Duration(cfg.RetryInitialDelayMs) * time.Millisecond,
			MaxDelay:     time.Duration(cfg.RetryMaxDelayMs) * time.Millisecond,
		})
	} else {
		slog.Warn("no payroll providers configured — payroll submission disabled")
	}

	payrollSvc := service.NewPayrollService(payrollRepo, earningRepo, statutoryRateRepo, crewRepo, payrollMgr, logger)
	payrollHandler := handler.NewPayrollHandler(payrollSvc)

	webhookSvc := service.NewWebhookService(webhookRepo, payoutSvc, payrollSvc, saccoSvc, walletRepo, payrollRepo, logger)

	// Build the JamboPay checksum verifier (SHA256-based, per v2 API spec).
	// Injected into WebhookHandler so it can verify callback authenticity without
	// the handler importing the jambopay package directly.
	var jamboChecksumVerifier handler.ChecksumVerifier
	if jamboPayProvider != nil {
		jamboChecksumVerifier = jamboPayProvider.VerifyCallbackChecksum
	}
	webhookHandler := handler.NewWebhookHandler(webhookSvc, jamboChecksumVerifier, cfg.WebhookPerpaySecret)

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

	// Serve local uploads if they exist
	router.Static("/uploads", "./storage")

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
		auth.GET("/lookup", authHandler.Lookup)                    // USSD user identification
		auth.POST("/pin", authHandler.SetPIN)                      // USSD PIN setup
		auth.POST("/pin/verify", authHandler.VerifyPIN)            // USSD PIN verification
		auth.POST("/forgot-password", authHandler.ForgotPassword)  // Self-service OTP request
		auth.POST("/verify-otp", authHandler.VerifyOTP)            // OTP verification
		auth.POST("/reset-password", authHandler.ResetPasswordOTP) // OTP-based password reset
		auth.GET("/otp-channels", authHandler.OTPChannels)         // Available OTP channels
	}

	webhooks := v1.Group("/webhooks")
	{
		webhooks.POST("/jambopay", webhookHandler.HandleJamboPay)
		webhooks.POST("/perpay", webhookHandler.HandlePerpay)
	}

	// USSD callback (Africa's Talking — no JWT, public endpoint)
	v1.POST("/ussd/callback", ussdHandler.Callback)

	// API v1 — authenticated endpoints
	serviceAPIKey := os.Getenv("SERVICE_API_KEY")
	secured := v1.Group("")
	secured.Use(middleware.JWTAuth(jwtManager, serviceAPIKey))
	{
		// Current user
		secured.GET("/auth/me", authHandler.Me)
		secured.PUT("/auth/profile", authHandler.UpdateProfile)
		secured.POST("/auth/kyc/initiate", authHandler.InitiateKYC)
		secured.POST("/auth/kyc/upload", authHandler.UploadKYC)
		secured.POST("/auth/change-password", adminHandler.ChangePassword) // Password change (requires auth)

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
			crew.GET("/lookup", crewHandler.LookupByNationalID)
			crew.POST("/:id/resend-credentials", crewHandler.ResendCredentials)
		}

		// Assignments
		assignments := secured.Group("/assignments")
		{
			// Read-only access for CREW, full access for SACCO_ADMIN and SYSTEM_ADMIN
			assignments.GET("", middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin, types.RoleCrewUser), assignmentHandler.List)
			assignments.GET("/:id", middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin, types.RoleCrewUser), assignmentHandler.GetByID)

			adminAssignments := assignments.Group("")
			adminAssignments.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
			{
				adminAssignments.POST("", assignmentHandler.Create)
			adminAssignments.POST("/bulk", assignmentHandler.BulkCreate)
				adminAssignments.PUT("/:id", assignmentHandler.Update)
				adminAssignments.POST("/:id/complete", assignmentHandler.Complete)
				adminAssignments.POST("/:id/cancel", assignmentHandler.Cancel)
				adminAssignments.POST("/:id/reassign", assignmentHandler.Reassign)
			}

			// Check-in/check-out accessible to crew, sacco admin, and system admin
			assignments.POST("/:id/check-in", middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin, types.RoleCrewUser), assignmentHandler.CheckIn)
			assignments.POST("/:id/check-out", middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin, types.RoleCrewUser), assignmentHandler.CheckOut)
		}

		// Wallets (all authenticated users; handler enforces ownership for CREW users)
		wallets := secured.Group("/wallets")
		{
			wallets.GET("/:crew_member_id", walletHandler.GetBalance)
			wallets.GET("/:crew_member_id/transactions", walletHandler.ListTransactions)
			wallets.GET("/:crew_member_id/export", walletHandler.ExportCSV)
			wallets.POST("/credit", walletHandler.Credit)
			wallets.POST("/debit", walletHandler.Debit)
			wallets.POST("/:crew_member_id/payout", payoutHandler.Payout)
		}

		// Atomic financial transactions (idempotent, all-or-nothing)
		transactions := secured.Group("/transactions")
		{
			// Employee payout: debit org float (gross) + credit wallet (net) in one TX
			transactions.POST("/employee-payout",
				middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin),
				transactionHandler.EmployeePayout)

			// Bulk employee payout: process multiple payouts sequentially, returns per-item result
			transactions.POST("/bulk-employee-payout",
				middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin),
				transactionHandler.BulkEmployeePayout)

			// Wallet-to-wallet transfer: debit sender + credit recipient in one TX
			transactions.POST("/transfer", transactionHandler.WalletTransfer)
		}

		// SACCOs / Organizations (system admin + sacco admin)
		// Original route kept for backward compat
		saccos := secured.Group("/saccos")
		saccos.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			saccos.POST("", orgHandler.Create)
			saccos.GET("", orgHandler.List)
			saccos.GET("/:id", orgHandler.GetByID)
			saccos.PUT("/:id", orgHandler.Update)
			saccos.DELETE("/:id", orgHandler.Delete)
			saccos.GET("/:id/members", orgHandler.ListMembers)
			saccos.POST("/:id/members", orgHandler.AddMember)
			saccos.PUT("/:id/members/:membership_id", orgHandler.UpdateMember)
			saccos.DELETE("/:id/members/:membership_id", orgHandler.RemoveMember)
			saccos.GET("/:id/float", orgHandler.GetFloat)
			saccos.POST("/:id/float/credit", orgHandler.CreditFloat)
			saccos.POST("/:id/float/topup", orgHandler.TopUpFloat)
			saccos.POST("/:id/float/topup/:tx_id/confirm", orgHandler.ConfirmTopUp)
			saccos.POST("/:id/float/topup/:tx_id/reject", orgHandler.RejectTopUp)
		saccos.POST("/:id/float/poll-stk", orgHandler.PollPendingSTK)
		saccos.POST("/:id/float/poll-stk/:tx_id", orgHandler.PollSingleSTK)
			saccos.POST("/:id/float/debit", orgHandler.DebitFloat)
			saccos.GET("/:id/float/transactions", orgHandler.ListFloatTransactions)
		}

		// D1: /organizations/* aliases — same handlers, industry-agnostic URL
		orgs := secured.Group("/organizations")
		orgs.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			orgs.POST("", orgHandler.Create)
			orgs.GET("", orgHandler.List)
			orgs.GET("/:id", orgHandler.GetByID)
			orgs.PUT("/:id", orgHandler.Update)
			orgs.DELETE("/:id", orgHandler.Delete)
			orgs.GET("/:id/members", orgHandler.ListMembers)
			orgs.POST("/:id/members", orgHandler.AddMember)
			orgs.PUT("/:id/members/:membership_id", orgHandler.UpdateMember)
			orgs.DELETE("/:id/members/:membership_id", orgHandler.RemoveMember)
			orgs.GET("/:id/float", orgHandler.GetFloat)
			orgs.POST("/:id/float/credit", orgHandler.CreditFloat)
			orgs.POST("/:id/float/topup", orgHandler.TopUpFloat)
			orgs.POST("/:id/float/topup/:tx_id/confirm", orgHandler.ConfirmTopUp)
			orgs.POST("/:id/float/topup/:tx_id/reject", orgHandler.RejectTopUp)
		orgs.POST("/:id/float/poll-stk", orgHandler.PollPendingSTK)
		orgs.POST("/:id/float/poll-stk/:tx_id", orgHandler.PollSingleSTK)
			orgs.POST("/:id/float/debit", orgHandler.DebitFloat)
			orgs.GET("/:id/float/transactions", orgHandler.ListFloatTransactions)
			// Tenant config, job types, pay schedules — also under /organizations/
			orgs.GET("/:id/config", tenantHandler.GetConfig)
			orgs.PUT("/:id/config", tenantHandler.UpdateConfig)
			orgs.GET("/:id/job-types", tenantHandler.ListJobTypes)
			orgs.POST("/:id/job-types", tenantHandler.CreateJobType)
			orgs.PUT("/:id/job-types/:job_type_id", tenantHandler.UpdateJobType)
			orgs.DELETE("/:id/job-types/:job_type_id", tenantHandler.DeleteJobType)
			orgs.GET("/:id/pay-schedules", tenantHandler.ListPaySchedules)
			orgs.POST("/:id/pay-schedules", tenantHandler.CreatePaySchedule)
			orgs.PUT("/:id/pay-schedules/:schedule_id", tenantHandler.UpdatePaySchedule)
			orgs.DELETE("/:id/pay-schedules/:schedule_id", tenantHandler.DeletePaySchedule)
			orgs.POST("/:id/bootstrap", tenantHandler.BootstrapIndustry)
		}

		// Tenant configuration — legacy /tenants/* routes (backward compat)
		tenants := secured.Group("/tenants")
		tenants.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			tenants.GET("/:id/config", tenantHandler.GetConfig)
			tenants.PUT("/:id/config", tenantHandler.UpdateConfig)
			tenants.GET("/:id/job-types", tenantHandler.ListJobTypes)
			tenants.POST("/:id/job-types", tenantHandler.CreateJobType)
			tenants.PUT("/:id/job-types/:job_type_id", tenantHandler.UpdateJobType)
			tenants.DELETE("/:id/job-types/:job_type_id", tenantHandler.DeleteJobType)
			tenants.GET("/:id/pay-schedules", tenantHandler.ListPaySchedules)
			tenants.POST("/:id/pay-schedules", tenantHandler.CreatePaySchedule)
			tenants.PUT("/:id/pay-schedules/:schedule_id", tenantHandler.UpdatePaySchedule)
			tenants.DELETE("/:id/pay-schedules/:schedule_id", tenantHandler.DeletePaySchedule)
			tenants.POST("/:id/bootstrap", tenantHandler.BootstrapIndustry)
		}

		// Industry templates (public, read-only)
		secured.GET("/industry-templates", tenantHandler.GetIndustryTemplate)

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

		// Work Sites (full CRUD, org-scoped)
		workSites := secured.Group("/work-sites")
		workSites.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			workSites.POST("", workSiteHandler.Create)
			workSites.GET("", workSiteHandler.List)
			workSites.GET("/:id", workSiteHandler.GetByID)
			workSites.PUT("/:id", workSiteHandler.Update)
			workSites.DELETE("/:id", workSiteHandler.Delete)
		}

		// Payroll (system admin + sacco admin)
		payrollRoutes := secured.Group("/payroll")
		payrollRoutes.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			payrollRoutes.POST("", payrollHandler.Create)
			payrollRoutes.GET("", payrollHandler.List)
			// Static paths MUST be registered before /:id to avoid Gin matching "periods" as a UUID
			payrollRoutes.GET("/periods", payrollHandler.ListPeriods)
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
			admin.GET("/users", adminHandler.ListUsers)
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

// buildLoanPolicy constructs a LoanPolicyConfig from env-loaded Config.
func buildLoanPolicy(cfg *config.Config) *models.LoanPolicyConfig {
	policy := models.DefaultLoanPolicy()

	// Set concurrency policy
	switch cfg.LoanConcurrencyPolicy {
	case "PER_CATEGORY":
		policy.ConcurrencyPolicy = models.PolicyPerCategory
	case "AGGREGATE":
		policy.ConcurrencyPolicy = models.PolicyAggregate
	default:
		policy.ConcurrencyPolicy = models.PolicySingle
	}

	policy.MaxConcurrentLoans = cfg.LoanMaxConcurrent
	policy.AggregateExposureMultiplier = cfg.LoanAggregateExposureMultiplier

	// Parse enabled categories
	if cfg.LoanCategoriesEnabled != "" {
		policy.CategoryEnabled = make(map[models.LoanCategory]bool)
		for _, cat := range strings.Split(cfg.LoanCategoriesEnabled, ",") {
			cat = strings.TrimSpace(cat)
			lc := models.LoanCategory(cat)
			if lc.IsValid() {
				policy.CategoryEnabled[lc] = true
			}
		}
	}

	slog.Info("loan policy configured",
		slog.String("concurrency", string(policy.ConcurrencyPolicy)),
		slog.Int("max_concurrent", policy.MaxConcurrentLoans),
		slog.Float64("exposure_multiplier", policy.AggregateExposureMultiplier),
		slog.Int("categories_enabled", len(policy.EnabledCategories())),
	)

	return policy
}

// multiHandler fans a single log record out to multiple slog.Handler
// instances. This lets us write all logs to stdout+server.log AND
// additionally write WARN+ to error.log.
type multiHandler struct {
	handlers []slog.Handler
}

func (h *multiHandler) Enabled(_ context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(context.Background(), level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			if err := handler.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}
