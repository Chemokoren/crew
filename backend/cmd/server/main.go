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
	"github.com/kibsoft/amy-mis/internal/config"
	"github.com/kibsoft/amy-mis/internal/credit"
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
	"github.com/kibsoft/amy-mis/internal/service"
	"github.com/kibsoft/amy-mis/internal/ussd"
	"github.com/kibsoft/amy-mis/internal/worker"
	"github.com/kibsoft/amy-mis/pkg/jwt"
	"github.com/kibsoft/amy-mis/pkg/retry"
	"github.com/kibsoft/amy-mis/pkg/types"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

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

	// Auto-migrate new system settings tables
	if err := db.AutoMigrate(&models.SystemSetting{}, &models.SystemAnnouncement{}); err != nil {
		slog.Error("failed to auto-migrate system settings tables", slog.String("error", err.Error()))
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
	rbacRepo := pgRepo.NewRBACRepo(db)
	systemSettingRepo := pgRepo.NewSystemSettingRepo(db)
	systemAnnouncementRepo := pgRepo.NewSystemAnnouncementRepo(db)

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
	systemSettingsHandler := handler.NewSystemSettingsHandler(systemSettingRepo, systemAnnouncementRepo, statutoryRateRepo)

	// --- RBAC (Enterprise Roles & Permissions) ---
	permCache := service.NewPermissionCache(redisClient, 5*time.Minute)
	go permCache.SubscribeInvalidations(context.Background())
	rbacSvc := service.NewRBACService(rbacRepo, auditSvc, permCache)
	rbacHandler := handler.NewRBACHandler(rbacSvc)

	// Sync permission registry and templates to database on startup
	go func() {
		ctx := context.Background()
		if err := rbacSvc.SyncRegistryPermissions(ctx); err != nil {
			slog.Error("failed to sync RBAC permissions", slog.String("error", err.Error()))
		} else {
			slog.Info("RBAC permissions synced to database")
		}
		if err := rbacSvc.SyncSystemRoles(ctx); err != nil {
			slog.Error("failed to sync RBAC system roles", slog.String("error", err.Error()))
		} else {
			slog.Info("RBAC system roles synced")
		}
		if err := rbacSvc.SyncTemplates(ctx); err != nil {
			slog.Error("failed to sync RBAC templates", slog.String("error", err.Error()))
		} else {
			slog.Info("RBAC industry templates synced")
		}
	}()

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
	router.Use(otelgin.Middleware("amy-mis-api"))                         // OTEL distributed traces
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
	secured.Use(middleware.InjectPermissionChecker(rbacSvc)) // RBAC permission checker (additive)
	{
		// Current user
		secured.GET("/auth/me", authHandler.Me)
		secured.PUT("/auth/profile", authHandler.UpdateProfile)
		secured.POST("/auth/kyc/initiate", authHandler.InitiateKYC)
		secured.POST("/auth/kyc/upload", authHandler.UploadKYC)
		secured.POST("/auth/change-password", adminHandler.ChangePassword) // Password change (requires auth)
		secured.GET("/announcements/active", systemSettingsHandler.ListActiveAnnouncements) // System banners for all users

		// Crew members (SACCO admins & system admins)
		crew := secured.Group("/crew")
		crew.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			crew.POST("", middleware.RequirePermission(models.PermWorkersCreate), crewHandler.Create)
			crew.GET("", middleware.RequirePermission(models.PermWorkersView), crewHandler.List)
			crew.GET("/:id", middleware.RequirePermission(models.PermWorkersView), crewHandler.GetByID)
			crew.PUT("/:id/kyc", middleware.RequireAnyPermission(models.PermWorkersVerifyKYC, models.PermWorkersUpdate), crewHandler.UpdateKYC)
			crew.POST("/:id/verify", middleware.RequirePermission(models.PermWorkersVerifyKYC), crewHandler.VerifyNationalID)
			crew.DELETE("/:id", middleware.RequireAnyPermission(models.PermWorkersDelete, models.PermWorkersArchive), crewHandler.Deactivate)
			crew.POST("/bulk-import", middleware.RequirePermission(models.PermWorkersBulkImport), crewHandler.BulkImport)
			crew.GET("/search", middleware.RequirePermission(models.PermWorkersView), crewHandler.SearchByNationalID)
			crew.GET("/lookup", middleware.RequirePermission(models.PermWorkersView), crewHandler.LookupByNationalID)
			crew.POST("/:id/resend-credentials", middleware.RequireAnyPermission(models.PermWorkersUpdate, models.PermUsersUpdate), crewHandler.ResendCredentials)
		}

		// Assignments
		assignments := secured.Group("/assignments")
		{
			// Read-only access for CREW, full access for SACCO_ADMIN and SYSTEM_ADMIN
			assignments.GET("", middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin, types.RoleCrewUser), middleware.RequirePermission(models.PermAssignmentsView), assignmentHandler.List)
			assignments.GET("/:id", middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin, types.RoleCrewUser), middleware.RequirePermission(models.PermAssignmentsView), assignmentHandler.GetByID)

			adminAssignments := assignments.Group("")
			adminAssignments.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
			{
				adminAssignments.POST("", middleware.RequirePermission(models.PermAssignmentsCreate), assignmentHandler.Create)
				adminAssignments.POST("/bulk", middleware.RequireAnyPermission(models.PermAssignmentsBulkAssign, models.PermAssignmentsCreate), assignmentHandler.BulkCreate)
				adminAssignments.PUT("/:id", middleware.RequirePermission(models.PermAssignmentsUpdate), assignmentHandler.Update)
				adminAssignments.POST("/:id/complete", middleware.RequireAnyPermission(models.PermAssignmentsApprove, models.PermAssignmentsUpdate), assignmentHandler.Complete)
				adminAssignments.POST("/:id/cancel", middleware.RequireAnyPermission(models.PermAssignmentsReject, models.PermAssignmentsUpdate), assignmentHandler.Cancel)
				adminAssignments.POST("/:id/reassign", middleware.RequirePermission(models.PermAssignmentsUpdate), assignmentHandler.Reassign)
			}

			// Check-in/check-out accessible to crew, sacco admin, and system admin
			assignments.POST("/:id/check-in", middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin, types.RoleCrewUser), middleware.RequirePermission(models.PermAssignmentsClockIn), assignmentHandler.CheckIn)
			assignments.POST("/:id/check-out", middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin, types.RoleCrewUser), middleware.RequirePermission(models.PermAssignmentsClockOut), assignmentHandler.CheckOut)
		}

		// Wallets (all authenticated users; handler enforces ownership for CREW users)
		wallets := secured.Group("/wallets")
		{
			wallets.GET("/:crew_member_id", middleware.RequirePermission(models.PermWalletView), walletHandler.GetBalance)
			wallets.GET("/:crew_member_id/transactions", middleware.RequireAnyPermission(models.PermWalletViewTransactions, models.PermWalletView), walletHandler.ListTransactions)
			wallets.GET("/:crew_member_id/export", middleware.RequirePermission(models.PermWalletExport), walletHandler.ExportCSV)
			wallets.POST("/credit", middleware.RequirePermission(models.PermWalletFundFloat), walletHandler.Credit)
			wallets.POST("/debit", middleware.RequireAnyPermission(models.PermWalletReverseTransaction, models.PermWalletReconcile), walletHandler.Debit)
			wallets.POST("/:crew_member_id/payout", middleware.RequireAnyPermission(models.PermWalletApprovePayout, models.PermWalletWithdraw), payoutHandler.Payout)
		}

		// Atomic financial transactions (idempotent, all-or-nothing)
		transactions := secured.Group("/transactions")
		{
			// Employee payout: debit org float (gross) + credit wallet (net) in one TX
			transactions.POST("/employee-payout",
				middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin),
				middleware.RequireAnyPermission(models.PermPayrollProcess, models.PermWalletApprovePayout),
				transactionHandler.EmployeePayout)

			// Bulk employee payout: process multiple payouts sequentially, returns per-item result
			transactions.POST("/bulk-employee-payout",
				middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin),
				middleware.RequireAnyPermission(models.PermPayrollProcess, models.PermWalletApprovePayout),
				transactionHandler.BulkEmployeePayout)

			// Wallet-to-wallet transfer: debit sender + credit recipient in one TX
			transactions.POST("/transfer",
				middleware.RequirePermission(models.PermWalletTransfer),
				transactionHandler.WalletTransfer)
		}

		// SACCOs / Organizations (system admin + sacco admin)
		// Original route kept for backward compat
		saccos := secured.Group("/saccos")
		saccos.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			saccos.POST("", middleware.RequirePermission(models.PermOrganizationsCreate), orgHandler.Create)
			saccos.GET("", middleware.RequirePermission(models.PermOrganizationsView), orgHandler.List)
			saccos.GET("/:id", middleware.RequirePermission(models.PermOrganizationsView), orgHandler.GetByID)
			saccos.PUT("/:id", middleware.RequirePermission(models.PermOrganizationsUpdate), orgHandler.Update)
			saccos.DELETE("/:id", middleware.RequirePermission(models.PermOrganizationsDelete), orgHandler.Delete)
			saccos.GET("/:id/members", middleware.RequireAnyPermission(models.PermOrganizationsView, models.PermUsersView), orgHandler.ListMembers)
			saccos.POST("/:id/members", middleware.RequireAnyPermission(models.PermOrganizationsUpdate, models.PermUsersCreate), orgHandler.AddMember)
			saccos.PUT("/:id/members/:membership_id", middleware.RequireAnyPermission(models.PermOrganizationsUpdate, models.PermUsersUpdate), orgHandler.UpdateMember)
			saccos.DELETE("/:id/members/:membership_id", middleware.RequireAnyPermission(models.PermOrganizationsUpdate, models.PermUsersDeactivate), orgHandler.RemoveMember)
			saccos.GET("/:id/float", middleware.RequireAnyPermission(models.PermWalletView, models.PermOrganizationsView), orgHandler.GetFloat)
			saccos.POST("/:id/float/credit", middleware.RequirePermission(models.PermWalletFundFloat), orgHandler.CreditFloat)
			saccos.POST("/:id/float/topup", middleware.RequirePermission(models.PermWalletFundFloat), orgHandler.TopUpFloat)
			saccos.POST("/:id/float/topup/:tx_id/confirm", middleware.RequirePermission(models.PermWalletReconcile), orgHandler.ConfirmTopUp)
			saccos.POST("/:id/float/topup/:tx_id/reject", middleware.RequirePermission(models.PermWalletReconcile), orgHandler.RejectTopUp)
			saccos.POST("/:id/float/poll-stk", middleware.RequirePermission(models.PermWalletReconcile), orgHandler.PollPendingSTK)
			saccos.POST("/:id/float/poll-stk/:tx_id", middleware.RequirePermission(models.PermWalletReconcile), orgHandler.PollSingleSTK)
			saccos.POST("/:id/float/debit", middleware.RequireAnyPermission(models.PermWalletReverseTransaction, models.PermWalletReconcile), orgHandler.DebitFloat)
			saccos.GET("/:id/float/transactions", middleware.RequireAnyPermission(models.PermWalletViewTransactions, models.PermWalletView), orgHandler.ListFloatTransactions)
		}

		// D1: /organizations/* aliases — same handlers, industry-agnostic URL
		orgs := secured.Group("/organizations")
		orgs.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			orgs.POST("", middleware.RequirePermission(models.PermOrganizationsCreate), orgHandler.Create)
			orgs.GET("", middleware.RequirePermission(models.PermOrganizationsView), orgHandler.List)
			orgs.GET("/:id", middleware.RequirePermission(models.PermOrganizationsView), orgHandler.GetByID)
			orgs.PUT("/:id", middleware.RequirePermission(models.PermOrganizationsUpdate), orgHandler.Update)
			orgs.DELETE("/:id", middleware.RequirePermission(models.PermOrganizationsDelete), orgHandler.Delete)
			orgs.GET("/:id/members", middleware.RequireAnyPermission(models.PermOrganizationsView, models.PermUsersView), orgHandler.ListMembers)
			orgs.POST("/:id/members", middleware.RequireAnyPermission(models.PermOrganizationsUpdate, models.PermUsersCreate), orgHandler.AddMember)
			orgs.PUT("/:id/members/:membership_id", middleware.RequireAnyPermission(models.PermOrganizationsUpdate, models.PermUsersUpdate), orgHandler.UpdateMember)
			orgs.DELETE("/:id/members/:membership_id", middleware.RequireAnyPermission(models.PermOrganizationsUpdate, models.PermUsersDeactivate), orgHandler.RemoveMember)
			orgs.GET("/:id/float", middleware.RequireAnyPermission(models.PermWalletView, models.PermOrganizationsView), orgHandler.GetFloat)
			orgs.POST("/:id/float/credit", middleware.RequirePermission(models.PermWalletFundFloat), orgHandler.CreditFloat)
			orgs.POST("/:id/float/topup", middleware.RequirePermission(models.PermWalletFundFloat), orgHandler.TopUpFloat)
			orgs.POST("/:id/float/topup/:tx_id/confirm", middleware.RequirePermission(models.PermWalletReconcile), orgHandler.ConfirmTopUp)
			orgs.POST("/:id/float/topup/:tx_id/reject", middleware.RequirePermission(models.PermWalletReconcile), orgHandler.RejectTopUp)
			orgs.POST("/:id/float/poll-stk", middleware.RequirePermission(models.PermWalletReconcile), orgHandler.PollPendingSTK)
			orgs.POST("/:id/float/poll-stk/:tx_id", middleware.RequirePermission(models.PermWalletReconcile), orgHandler.PollSingleSTK)
			orgs.POST("/:id/float/debit", middleware.RequireAnyPermission(models.PermWalletReverseTransaction, models.PermWalletReconcile), orgHandler.DebitFloat)
			orgs.GET("/:id/float/transactions", middleware.RequireAnyPermission(models.PermWalletViewTransactions, models.PermWalletView), orgHandler.ListFloatTransactions)
			// Tenant config, job types, pay schedules — also under /organizations/
			orgs.GET("/:id/config", middleware.RequireAnyPermission(models.PermSettingsView, models.PermOrganizationsView), tenantHandler.GetConfig)
			orgs.PUT("/:id/config", middleware.RequireAnyPermission(models.PermSettingsUpdate, models.PermOrganizationsManageConfig), tenantHandler.UpdateConfig)
			orgs.GET("/:id/job-types", middleware.RequireAnyPermission(models.PermSettingsView, models.PermOrganizationsView), tenantHandler.ListJobTypes)
			orgs.POST("/:id/job-types", middleware.RequireAnyPermission(models.PermSettingsManageTenant, models.PermOrganizationsManageConfig), tenantHandler.CreateJobType)
			orgs.PUT("/:id/job-types/:job_type_id", middleware.RequireAnyPermission(models.PermSettingsManageTenant, models.PermOrganizationsManageConfig), tenantHandler.UpdateJobType)
			orgs.DELETE("/:id/job-types/:job_type_id", middleware.RequireAnyPermission(models.PermSettingsManageTenant, models.PermOrganizationsManageConfig), tenantHandler.DeleteJobType)
			orgs.GET("/:id/pay-schedules", middleware.RequireAnyPermission(models.PermPayrollView, models.PermPayrollManageSchedules), tenantHandler.ListPaySchedules)
			orgs.POST("/:id/pay-schedules", middleware.RequirePermission(models.PermPayrollManageSchedules), tenantHandler.CreatePaySchedule)
			orgs.PUT("/:id/pay-schedules/:schedule_id", middleware.RequirePermission(models.PermPayrollManageSchedules), tenantHandler.UpdatePaySchedule)
			orgs.DELETE("/:id/pay-schedules/:schedule_id", middleware.RequirePermission(models.PermPayrollManageSchedules), tenantHandler.DeletePaySchedule)
			orgs.POST("/:id/bootstrap", middleware.RequireAnyPermission(models.PermSettingsManageTenant, models.PermOrganizationsManageConfig), tenantHandler.BootstrapIndustry)
		}

		// Tenant configuration — legacy /tenants/* routes (backward compat)
		tenants := secured.Group("/tenants")
		tenants.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			tenants.GET("/:id/config", middleware.RequireAnyPermission(models.PermSettingsView, models.PermOrganizationsView), tenantHandler.GetConfig)
			tenants.PUT("/:id/config", middleware.RequireAnyPermission(models.PermSettingsUpdate, models.PermOrganizationsManageConfig), tenantHandler.UpdateConfig)
			tenants.GET("/:id/job-types", middleware.RequireAnyPermission(models.PermSettingsView, models.PermOrganizationsView), tenantHandler.ListJobTypes)
			tenants.POST("/:id/job-types", middleware.RequireAnyPermission(models.PermSettingsManageTenant, models.PermOrganizationsManageConfig), tenantHandler.CreateJobType)
			tenants.PUT("/:id/job-types/:job_type_id", middleware.RequireAnyPermission(models.PermSettingsManageTenant, models.PermOrganizationsManageConfig), tenantHandler.UpdateJobType)
			tenants.DELETE("/:id/job-types/:job_type_id", middleware.RequireAnyPermission(models.PermSettingsManageTenant, models.PermOrganizationsManageConfig), tenantHandler.DeleteJobType)
			tenants.GET("/:id/pay-schedules", middleware.RequireAnyPermission(models.PermPayrollView, models.PermPayrollManageSchedules), tenantHandler.ListPaySchedules)
			tenants.POST("/:id/pay-schedules", middleware.RequirePermission(models.PermPayrollManageSchedules), tenantHandler.CreatePaySchedule)
			tenants.PUT("/:id/pay-schedules/:schedule_id", middleware.RequirePermission(models.PermPayrollManageSchedules), tenantHandler.UpdatePaySchedule)
			tenants.DELETE("/:id/pay-schedules/:schedule_id", middleware.RequirePermission(models.PermPayrollManageSchedules), tenantHandler.DeletePaySchedule)
			tenants.POST("/:id/bootstrap", middleware.RequireAnyPermission(models.PermSettingsManageTenant, models.PermOrganizationsManageConfig), tenantHandler.BootstrapIndustry)
		}

		// Industry templates (public, read-only)
		secured.GET("/industry-templates", middleware.RequireAnyPermission(models.PermSettingsView, models.PermOrganizationsView), tenantHandler.GetIndustryTemplate)

		// Vehicles
		vehicles := secured.Group("/vehicles")
		vehicles.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			vehicles.POST("", middleware.RequirePermission(models.PermVehiclesCreate), vehicleHandler.Create)
			vehicles.GET("", middleware.RequirePermission(models.PermVehiclesView), vehicleHandler.List)
			vehicles.GET("/:id", middleware.RequirePermission(models.PermVehiclesView), vehicleHandler.GetByID)
			vehicles.PUT("/:id", middleware.RequirePermission(models.PermVehiclesUpdate), vehicleHandler.Update)
			vehicles.DELETE("/:id", middleware.RequirePermission(models.PermVehiclesDelete), vehicleHandler.Delete)
		}

		// Routes
		routes := secured.Group("/routes")
		routes.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			routes.POST("", middleware.RequirePermission(models.PermRoutesCreate), routeHandler.Create)
			routes.GET("", middleware.RequirePermission(models.PermRoutesView), routeHandler.List)
			routes.GET("/:id", middleware.RequirePermission(models.PermRoutesView), routeHandler.GetByID)
			routes.PUT("/:id", middleware.RequirePermission(models.PermRoutesUpdate), routeHandler.Update)
			routes.DELETE("/:id", middleware.RequirePermission(models.PermRoutesDelete), routeHandler.Delete)
		}

		// Work Sites (full CRUD, org-scoped)
		workSites := secured.Group("/work-sites")
		workSites.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			workSites.POST("", middleware.RequirePermission(models.PermWorkSitesCreate), workSiteHandler.Create)
			workSites.GET("", middleware.RequirePermission(models.PermWorkSitesView), workSiteHandler.List)
			workSites.GET("/:id", middleware.RequirePermission(models.PermWorkSitesView), workSiteHandler.GetByID)
			workSites.PUT("/:id", middleware.RequirePermission(models.PermWorkSitesUpdate), workSiteHandler.Update)
			workSites.DELETE("/:id", middleware.RequirePermission(models.PermWorkSitesDelete), workSiteHandler.Delete)
		}

		// Payroll (system admin + sacco admin)
		payrollRoutes := secured.Group("/payroll")
		payrollRoutes.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			payrollRoutes.POST("", middleware.RequireAnyPermission(models.PermPayrollCreate, models.PermPayrollRun), payrollHandler.Create)
			payrollRoutes.GET("", middleware.RequirePermission(models.PermPayrollView), payrollHandler.List)
			// Static paths MUST be registered before /:id to avoid Gin matching "periods" as a UUID
			payrollRoutes.GET("/periods", middleware.RequireAnyPermission(models.PermPayrollView, models.PermPayrollManagePeriods), payrollHandler.ListPeriods)
			payrollRoutes.GET("/:id", middleware.RequirePermission(models.PermPayrollView), payrollHandler.GetByID)
			payrollRoutes.GET("/:id/entries", middleware.RequireAnyPermission(models.PermPayrollViewEntries, models.PermPayrollView), payrollHandler.GetEntries)
			payrollRoutes.POST("/:id/process", middleware.RequirePermission(models.PermPayrollProcess), payrollHandler.Process)
			payrollRoutes.POST("/:id/approve", middleware.RequirePermission(models.PermPayrollApprove), payrollHandler.Approve)
			payrollRoutes.POST("/:id/submit", middleware.RequirePermission(models.PermComplianceSubmitStatutory), payrollHandler.Submit)
		}

		// Documents
		documents := secured.Group("/documents")
		documents.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleSaccoAdmin))
		{
			documents.POST("/upload", middleware.RequirePermission(models.PermDocumentsUpload), docHandler.Upload)
			documents.GET("/:id/download", middleware.RequirePermission(models.PermDocumentsView), docHandler.Download)
			documents.GET("", middleware.RequirePermission(models.PermDocumentsView), docHandler.List)
			documents.DELETE("/:id", middleware.RequirePermission(models.PermDocumentsDelete), docHandler.Delete)
		}

		// Earnings
		earnings := secured.Group("/earnings")
		{
			earnings.GET("", middleware.RequirePermission(models.PermEarningsView), earningHandler.List)
			earnings.GET("/summary/:crew_member_id", middleware.RequirePermission(models.PermEarningsView), earningHandler.SummaryDashboard)
		}

		// Notifications (all authenticated users)
		notifications := secured.Group("/notifications")
		{
			notifications.GET("", middleware.RequirePermission(models.PermNotificationsView), notifHandler.List)
			notifications.PUT("/:id/read", middleware.RequirePermission(models.PermNotificationsView), notifHandler.MarkRead)
			notifications.GET("/preferences", middleware.RequirePermission(models.PermNotificationsView), notifHandler.GetPreferences)
			notifications.PUT("/preferences", middleware.RequirePermission(models.PermNotificationsView), notifHandler.UpdatePreferences)
		}

		// Financials: Credit
		credit := secured.Group("/credit")
		{
			credit.GET("/:crew_member_id", middleware.RequirePermission(models.PermCreditView), creditHandler.GetScore)
			credit.GET("/:crew_member_id/detailed", middleware.RequirePermission(models.PermCreditView), creditHandler.GetDetailedScore)
			credit.GET("/:crew_member_id/history", middleware.RequirePermission(models.PermCreditView), creditHandler.GetScoreHistory)
			credit.POST("/:crew_member_id/calculate", middleware.RequirePermission(models.PermCreditScoreCompute), creditHandler.CalculateScore)
		}

		// Financials: Loans
		loans := secured.Group("/loans")
		{
			loans.POST("", middleware.RequirePermission(models.PermLoansApply), loanHandler.Apply)
			loans.GET("", middleware.RequirePermission(models.PermLoansView), loanHandler.List)
			loans.GET("/tier/:crew_member_id", middleware.RequirePermission(models.PermLoansView), loanHandler.GetTier)
			loans.POST("/:id/repay", middleware.RequireAnyPermission(models.PermLoansApply, models.PermLoansManage), loanHandler.Repay)

			loanAdmin := loans.Group("")
			loanAdmin.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleLender))
			{
				loanAdmin.POST("/:id/approve", middleware.RequirePermission(models.PermLoansApprove), loanHandler.Approve)
				loanAdmin.POST("/:id/reject", middleware.RequirePermission(models.PermLoansReject), loanHandler.Reject)
				loanAdmin.POST("/:id/disburse", middleware.RequirePermission(models.PermLoansDisburse), loanHandler.Disburse)
			}
		}

		// Financials: Insurance
		insurance := secured.Group("/insurance")
		{
			insurance.GET("", middleware.RequirePermission(models.PermInsuranceView), insuranceHandler.List)

			insuranceAdmin := insurance.Group("")
			insuranceAdmin.Use(middleware.RequireRole(types.RoleSystemAdmin, types.RoleInsurer))
			{
				insuranceAdmin.POST("", middleware.RequireAnyPermission(models.PermInsuranceEnroll, models.PermInsuranceManagePolicies), insuranceHandler.Create)
				insuranceAdmin.POST("/:id/lapse", middleware.RequireAnyPermission(models.PermInsuranceCancel, models.PermInsuranceManagePolicies), insuranceHandler.Lapse)
			}
		}

		// Admin dashboard
		admin := secured.Group("/admin")
		admin.Use(middleware.RequireRole(types.RoleSystemAdmin))
		{
			admin.GET("/stats", middleware.RequirePermission(models.PermPlatformViewAnalytics), adminHandler.SystemStats)
			admin.GET("/users", middleware.RequireAnyPermission(models.PermUsersView, models.PermPlatformManageUsers), adminHandler.ListUsers)
			admin.POST("/users/:id/disable", middleware.RequireAnyPermission(models.PermUsersDeactivate, models.PermPlatformManageUsers), adminHandler.DisableAccount)
			admin.POST("/users/:id/enable", middleware.RequireAnyPermission(models.PermUsersUpdate, models.PermPlatformManageUsers), adminHandler.EnableAccount)
			admin.POST("/users/:id/reset-password", middleware.RequireAnyPermission(models.PermUsersUpdate, models.PermPlatformManageUsers), adminHandler.ResetPassword)
			admin.GET("/audit-logs", middleware.RequireAnyPermission(models.PermAuditView, models.PermPlatformViewAudit), adminHandler.ListAuditLogs)
			admin.GET("/statutory-rates", middleware.RequireAnyPermission(models.PermComplianceView, models.PermPlatformManageCompliance), adminHandler.ListStatutoryRates)
			admin.POST("/statutory-rates", middleware.RequireAnyPermission(models.PermComplianceManageRates, models.PermPlatformManageCompliance), systemSettingsHandler.CreateStatutoryRate)
			admin.PUT("/statutory-rates/:id", middleware.RequireAnyPermission(models.PermComplianceManageRates, models.PermPlatformManageCompliance), systemSettingsHandler.UpdateStatutoryRate)
			admin.GET("/notifications/templates", middleware.RequirePermission(models.PermNotificationsManageTemplates), adminHandler.ListTemplates)
			admin.POST("/notifications/templates", middleware.RequirePermission(models.PermNotificationsManageTemplates), adminHandler.CreateTemplate)
			admin.PUT("/notifications/templates", middleware.RequirePermission(models.PermNotificationsManageTemplates), adminHandler.UpdateTemplate)

			// System Settings (key-value store)
			admin.GET("/system-settings", middleware.RequireAnyPermission(models.PermSettingsView, models.PermPlatformManageSettings), systemSettingsHandler.ListSettings)
			admin.PUT("/system-settings", middleware.RequireAnyPermission(models.PermSettingsUpdate, models.PermPlatformManageSettings), systemSettingsHandler.UpsertSetting)
			admin.PUT("/system-settings/bulk", middleware.RequireAnyPermission(models.PermSettingsUpdate, models.PermPlatformManageSettings), systemSettingsHandler.BulkUpsertSettings)
			admin.DELETE("/system-settings/:key", middleware.RequireAnyPermission(models.PermSettingsUpdate, models.PermPlatformManageSettings), systemSettingsHandler.DeleteSetting)

			// System Announcements
			admin.GET("/announcements", middleware.RequireAnyPermission(models.PermNotificationsView, models.PermPlatformManageSettings), systemSettingsHandler.ListAnnouncements)
			admin.POST("/announcements", middleware.RequireAnyPermission(models.PermNotificationsSend, models.PermPlatformManageSettings), systemSettingsHandler.CreateAnnouncement)
			admin.PUT("/announcements/:id", middleware.RequireAnyPermission(models.PermNotificationsSend, models.PermPlatformManageSettings), systemSettingsHandler.UpdateAnnouncement)
			admin.DELETE("/announcements/:id", middleware.RequireAnyPermission(models.PermNotificationsSend, models.PermPlatformManageSettings), systemSettingsHandler.DeleteAnnouncement)
		}

		// RBAC APIs (uses rate limiting for mutations)
		rbacLimiter := middleware.RateLimit(redisClient, 20, time.Minute)
		rbacHandler.RegisterRoutes(secured, rbacLimiter)
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
