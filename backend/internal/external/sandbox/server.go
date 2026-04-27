// Package sandbox provides a standalone HTTP server that mirrors JamboPay's API
// for development and testing. The backend's JamboPayProvider connects to this
// server unchanged — just set JAMBOPAY_BASE_URL=http://localhost:8091.
//
// Additionally exposes /sandbox/admin/* endpoints for test control:
// credit accounts, simulate earnings, manage loans, etc.
package sandbox

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Server is the sandbox HTTP server.
type Server struct {
	store  *Store
	logger *slog.Logger
	router *gin.Engine
}

// NewServer creates a new sandbox server with auto-seeded test data.
func NewServer(logger *slog.Logger) *Server {
	s := &Server{
		store:  NewStore(),
		logger: logger,
	}
	s.setupRouter()
	s.seedTestData()
	return s
}

// Handler returns the http.Handler for the server.
func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) setupRouter() {
	gin.SetMode(gin.DebugMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(s.loggerMiddleware())

	// ============================================================
	// JamboPay-Compatible Endpoints
	// These mirror the real JamboPay v2 API so the existing
	// JamboPayProvider works unchanged.
	// ============================================================

	// Auth: POST /auth/token → returns a static sandbox token
	r.POST("/auth/token", s.handleAuthToken)

	// Payout: POST /payout → initiates a payout (holds in pending)
	r.POST("/payout", s.handlePayout)

	// Verify: POST /payout/authorize → verifies OTP (accepts "1234")
	r.POST("/payout/authorize", s.handlePayoutAuthorize)

	// Balance: GET /wallet/account → returns sandbox account balance
	r.GET("/wallet/account", s.handleWalletBalance)

	// ============================================================
	// Sandbox Admin/Test Control Endpoints
	// For programmatic test setup and scenario control.
	// ============================================================

	admin := r.Group("/sandbox/admin")
	{
		// Accounts
		admin.POST("/accounts", s.handleCreateAccount)
		admin.GET("/accounts/:id", s.handleGetAccount)
		admin.POST("/accounts/:id/credit", s.handleCreditAccount)
		admin.POST("/accounts/:id/debit", s.handleDebitAccount)
		admin.GET("/accounts/:id/transactions", s.handleListTransactions)

		// PIN
		admin.POST("/accounts/:id/pin", s.handleSetPIN)
		admin.POST("/accounts/:id/pin/verify", s.handleVerifyPIN)

		// Earnings
		admin.POST("/earnings", s.handleAddEarning)
		admin.GET("/earnings/:account_id", s.handleListEarnings)

		// Loans
		admin.POST("/loans/apply", s.handleApplyLoan)
		admin.GET("/loans/:account_id", s.handleListLoans)
		admin.POST("/loans/:id/approve", s.handleApproveLoan)
		admin.POST("/loans/:id/disburse", s.handleDisburseLoan)

		// Insurance
		admin.POST("/insurance/create", s.handleCreateInsurance)
		admin.GET("/insurance/:account_id", s.handleListInsurance)

		// System
		admin.POST("/reset", s.handleReset)
		admin.GET("/stats", s.handleStats)
	}

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "sandbox"})
	})

	s.router = r
}

// --- Middleware ---

func (s *Server) loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		s.logger.Info("sandbox request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", time.Since(start)),
		)
	}
}

// --- Auto-seed test data ---

func (s *Server) seedTestData() {
	s.logger.Info("seeding sandbox test data...")

	// Main test account matching the user's crew member
	acct := &Account{
		AccountNo:    "ed76cd33-b15e-49c5-a0b1-c4432286092d",
		Name:         "Test Crew Member",
		Phone:        "+254713058775",
		BalanceCents: 1000000, // KES 10,000
		Currency:     "KES",
	}
	_ = s.store.CreateAccount(acct)

	// Set default PIN: 1234
	pinHash, _ := bcrypt.GenerateFromPassword([]byte("1234"), bcrypt.DefaultCost)
	_ = s.store.SetPIN(acct.AccountNo, string(pinHash))

	// Seed initial credit transaction
	s.store.Credit(acct.AccountNo, 0, "SEED", "Initial sandbox balance") // dummy to create tx record
	// Fix: the credit was already in BalanceCents, so just add the tx record
	s.store.mu.Lock()
	if len(s.store.transactions[acct.AccountNo]) > 0 {
		s.store.transactions[acct.AccountNo][0].AmountCents = 1000000
		s.store.transactions[acct.AccountNo][0].BalanceAfterCents = 1000000
		s.store.transactions[acct.AccountNo][0].Description = "Sandbox seed: KES 10,000"
	}
	// Reset balance to avoid double-counting from the Credit call
	acct.BalanceCents = 1000000
	s.store.mu.Unlock()

	// Seed earnings — today, this week, last month
	now := time.Now()
	s.store.AddEarning(acct.AccountNo, 150000, "FIXED", "Morning shift — Route 33", now)
	s.store.AddEarning(acct.AccountNo, 85000, "COMMISSION", "Afternoon shift — Route 58", now.AddDate(0, 0, -2))
	s.store.AddEarning(acct.AccountNo, 200000, "HYBRID", "Full day shift — Route 12", now.AddDate(0, 0, -20))

	// Seed a pending loan
	loan, _ := s.store.ApplyLoan(acct.AccountNo, 500000, 30)
	if loan != nil {
		s.logger.Info("seeded loan", slog.String("loan_id", loan.ID), slog.String("status", loan.Status))
	}

	// Seed an insurance policy
	s.store.CreateInsurance(acct.AccountNo, "Sandbox Insurance Co.", "PERSONAL_ACCIDENT", 25000, 365)

	s.logger.Info("sandbox seed complete",
		slog.String("account", acct.AccountNo),
		slog.Int64("balance_cents", acct.BalanceCents),
		slog.String("phone", acct.Phone),
		slog.String("default_pin", "1234"),
	)
}

// ============================================================
// JamboPay-Compatible Handlers
// ============================================================

// POST /auth/token — returns a static sandbox token
func (s *Server) handleAuthToken(c *gin.Context) {
	// Accept any credentials — this is a sandbox
	c.JSON(http.StatusOK, gin.H{
		"access_token": "sandbox-token-" + uuid.New().String()[:8],
		"expires_in":   3600,
		"token_type":   "Bearer",
	})
}

// POST /payout — initiates a payout (holds in pending until OTP verify)
func (s *Server) handlePayout(c *gin.Context) {
	var req struct {
		Amount      string                 `json:"amount"`
		AccountFrom string                 `json:"accountFrom"`
		OrderID     string                 `json:"orderId"`
		Provider    string                 `json:"provider"`
		PayTo       map[string]string      `json:"payTo"`
		CallBackURL string                 `json:"callBackUrl"`
		Narration   string                 `json:"narration"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": 400, "message": []string{err.Error()}})
		return
	}

	// Parse amount (JamboPay sends as string like "100.00")
	amountFloat, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil || amountFloat <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": 400, "message": []string{"invalid amount"}})
		return
	}
	amountCents := int64(amountFloat * 100)

	// Resolve account — the backend sends "wallet-{crewMemberID}" as accountFrom,
	// but sandbox accounts are keyed by bare crewMemberID. Try both.
	accountNo := req.AccountFrom
	acct, err := s.store.GetAccount(accountNo)
	if err != nil {
		// Strip "wallet-" prefix and retry
		stripped := strings.TrimPrefix(accountNo, "wallet-")
		if stripped != accountNo {
			acct, err = s.store.GetAccount(stripped)
			accountNo = stripped
		}
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": 400, "message": []string{err.Error()}})
		return
	}
	if acct.BalanceCents < amountCents {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  400,
			"message": []string{fmt.Sprintf("insufficient balance: have %d, need %d", acct.BalanceCents, amountCents)},
		})
		return
	}

	// In sandbox mode, auto-complete payouts (no OTP step).
	// The USSD flow does not support OTP verification.
	ref := "SBX-PAY-" + uuid.New().String()[:8]

	// Debit the sandbox account immediately
	_, err = s.store.Debit(accountNo, amountCents, "WITHDRAWAL", fmt.Sprintf("Payout via %s to %s", req.Provider, req.PayTo["accountNumber"]))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": 400, "message": []string{err.Error()}})
		return
	}

	s.logger.Info("sandbox payout completed",
		slog.String("ref", ref),
		slog.Int64("amount_cents", amountCents),
		slog.String("account", accountNo),
	)

	c.JSON(http.StatusOK, gin.H{
		"ref":     ref,
		"orderId": req.OrderID,
	})
}

// POST /payout/authorize — verifies OTP (sandbox always accepts "1234")
func (s *Server) handlePayoutAuthorize(c *gin.Context) {
	var req struct {
		Ref string `json:"ref"`
		OTP string `json:"otp"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Accept OTP "1234" or any 4+ digit code in sandbox
	if len(req.OTP) < 4 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid OTP"})
		return
	}

	tx, err := s.store.CompletePayout(req.Ref)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.logger.Info("sandbox payout completed",
		slog.String("ref", req.Ref),
		slog.Int64("amount_cents", tx.AmountCents),
	)

	c.JSON(http.StatusOK, gin.H{
		"ref":    req.Ref,
		"status": "completed",
	})
}

// GET /wallet/account?accountNo=xxx — returns sandbox account balance
func (s *Server) handleWalletBalance(c *gin.Context) {
	accountNo := c.Query("accountNo")
	if accountNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accountNo required"})
		return
	}

	acct, err := s.store.GetAccount(accountNo)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"currentBalance": float64(acct.BalanceCents) / 100,
		"currency":       acct.Currency,
		"accountNo":      acct.AccountNo,
		"name":           acct.Name,
	})
}

// ============================================================
// Sandbox Admin Handlers
// ============================================================

// POST /sandbox/admin/accounts
func (s *Server) handleCreateAccount(c *gin.Context) {
	var req struct {
		AccountNo    string `json:"account_no" binding:"required"`
		Name         string `json:"name" binding:"required"`
		Phone        string `json:"phone" binding:"required"`
		BalanceCents int64  `json:"balance_cents"`
		Currency     string `json:"currency"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Currency == "" {
		req.Currency = "KES"
	}

	acct := &Account{
		AccountNo:    req.AccountNo,
		Name:         req.Name,
		Phone:        req.Phone,
		BalanceCents: req.BalanceCents,
		Currency:     req.Currency,
	}
	if err := s.store.CreateAccount(acct); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, acct)
}

// GET /sandbox/admin/accounts/:id
func (s *Server) handleGetAccount(c *gin.Context) {
	acct, err := s.store.GetAccount(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, acct)
}

// POST /sandbox/admin/accounts/:id/credit
func (s *Server) handleCreditAccount(c *gin.Context) {
	var req struct {
		AmountCents int64  `json:"amount_cents" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Description == "" {
		req.Description = "Admin credit"
	}

	tx, err := s.store.Credit(c.Param("id"), req.AmountCents, "ADMIN_CREDIT", req.Description)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tx)
}

// POST /sandbox/admin/accounts/:id/debit
func (s *Server) handleDebitAccount(c *gin.Context) {
	var req struct {
		AmountCents int64  `json:"amount_cents" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Description == "" {
		req.Description = "Admin debit"
	}

	tx, err := s.store.Debit(c.Param("id"), req.AmountCents, "ADMIN_DEBIT", req.Description)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tx)
}

// GET /sandbox/admin/accounts/:id/transactions
func (s *Server) handleListTransactions(c *gin.Context) {
	txs := s.store.GetTransactions(c.Param("id"))
	if txs == nil {
		txs = []Transaction{}
	}
	c.JSON(http.StatusOK, txs)
}

// POST /sandbox/admin/accounts/:id/pin
func (s *Server) handleSetPIN(c *gin.Context) {
	var req struct {
		PIN string `json:"pin" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.PIN), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash PIN"})
		return
	}

	if err := s.store.SetPIN(c.Param("id"), string(hash)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "PIN set successfully"})
}

// POST /sandbox/admin/accounts/:id/pin/verify
func (s *Server) handleVerifyPIN(c *gin.Context) {
	var req struct {
		PIN string `json:"pin" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := s.store.GetPINHash(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if hash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no PIN set"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.PIN)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid PIN"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "PIN verified"})
}

// POST /sandbox/admin/earnings
func (s *Server) handleAddEarning(c *gin.Context) {
	var req struct {
		AccountNo   string `json:"account_no" binding:"required"`
		AmountCents int64  `json:"amount_cents" binding:"required"`
		EarningType string `json:"earning_type"`
		Description string `json:"description"`
		EarnedAt    string `json:"earned_at"` // YYYY-MM-DD (optional, defaults to now)
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.EarningType == "" {
		req.EarningType = "FIXED"
	}
	if req.Description == "" {
		req.Description = "Simulated earning"
	}

	earnedAt := time.Now()
	if req.EarnedAt != "" {
		if t, err := time.Parse("2006-01-02", req.EarnedAt); err == nil {
			earnedAt = t
		}
	}

	earning, err := s.store.AddEarning(req.AccountNo, req.AmountCents, req.EarningType, req.Description, earnedAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, earning)
}

// GET /sandbox/admin/earnings/:account_id
func (s *Server) handleListEarnings(c *gin.Context) {
	earnings := s.store.GetEarnings(c.Param("account_id"))
	if earnings == nil {
		earnings = []Earning{}
	}
	c.JSON(http.StatusOK, earnings)
}

// POST /sandbox/admin/loans/apply
func (s *Server) handleApplyLoan(c *gin.Context) {
	var req struct {
		AccountNo   string `json:"account_no" binding:"required"`
		AmountCents int64  `json:"amount_cents" binding:"required"`
		TenureDays  int    `json:"tenure_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.TenureDays == 0 {
		req.TenureDays = 30
	}

	loan, err := s.store.ApplyLoan(req.AccountNo, req.AmountCents, req.TenureDays)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, loan)
}

// GET /sandbox/admin/loans/:account_id
func (s *Server) handleListLoans(c *gin.Context) {
	loans := s.store.GetLoans(c.Param("account_id"))
	if loans == nil {
		loans = []Loan{}
	}
	c.JSON(http.StatusOK, loans)
}

// POST /sandbox/admin/loans/:id/approve
func (s *Server) handleApproveLoan(c *gin.Context) {
	loan, err := s.store.ApproveLoan(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, loan)
}

// POST /sandbox/admin/loans/:id/disburse
func (s *Server) handleDisburseLoan(c *gin.Context) {
	loan, err := s.store.DisburseLoan(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, loan)
}

// POST /sandbox/admin/insurance/create
func (s *Server) handleCreateInsurance(c *gin.Context) {
	var req struct {
		AccountNo    string `json:"account_no" binding:"required"`
		Provider     string `json:"provider"`
		PolicyType   string `json:"policy_type"`
		PremiumCents int64  `json:"premium_cents" binding:"required"`
		DurationDays int    `json:"duration_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Provider == "" {
		req.Provider = "Sandbox Insurance"
	}
	if req.PolicyType == "" {
		req.PolicyType = "PERSONAL_ACCIDENT"
	}
	if req.DurationDays == 0 {
		req.DurationDays = 365
	}

	ins, err := s.store.CreateInsurance(req.AccountNo, req.Provider, req.PolicyType, req.PremiumCents, req.DurationDays)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, ins)
}

// GET /sandbox/admin/insurance/:account_id
func (s *Server) handleListInsurance(c *gin.Context) {
	policies := s.store.GetInsurance(c.Param("account_id"))
	if policies == nil {
		policies = []Insurance{}
	}
	c.JSON(http.StatusOK, policies)
}

// POST /sandbox/admin/reset
func (s *Server) handleReset(c *gin.Context) {
	s.store.Reset()
	s.seedTestData()
	c.JSON(http.StatusOK, gin.H{"message": "sandbox reset complete"})
}

// GET /sandbox/admin/stats
func (s *Server) handleStats(c *gin.Context) {
	c.JSON(http.StatusOK, s.store.Stats())
}
