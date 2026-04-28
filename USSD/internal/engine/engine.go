// Package engine implements the Finite State Machine (FSM) that drives all USSD
// menu navigation. Each USSD flow is modeled as a deterministic state machine
// with explicit transitions, eliminating brittle if/else chains.
//
// Design principles:
//   - Deterministic: identical (state, input) → identical next state
//   - Shallow: max 3–4 levels deep to minimize session steps
//   - Configurable: transitions and menus are data, not code
//   - Testable: pure functions for state transitions
package engine

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/kibsoft/amy-mis-ussd/internal/backend"
	"github.com/kibsoft/amy-mis-ussd/internal/i18n"
	"github.com/kibsoft/amy-mis-ussd/internal/session"
)

// Response represents the USSD response to send back to the telco gateway.
type Response struct {
	Message    string // The text to display to the user
	EndSession bool   // true = CON (continue), false = END (terminate)
}

// Engine is the FSM-based USSD menu processor.
type Engine struct {
	backendClient *backend.Client
	sessionStore  *session.Store
	translator    *i18n.Translator
	logger        *slog.Logger
}

// NewEngine creates a new USSD processing engine.
func NewEngine(client *backend.Client, store *session.Store, translator *i18n.Translator, logger *slog.Logger) *Engine {
	return &Engine{
		backendClient: client,
		sessionStore:  store,
		translator:    translator,
		logger:        logger,
	}
}

// Process handles a single USSD request by:
// 1. Determining the current FSM state
// 2. Processing user input for that state
// 3. Transitioning to the next state
// 4. Rendering the appropriate menu/response
func (e *Engine) Process(ctx context.Context, sess *session.Data, userInput string) (*Response, error) {
	sess.StepCount++
	sess.LastInputAt = time.Now()

	e.logger.Debug("FSM processing",
		slog.String("session_id", sess.SessionID),
		slog.String("state", string(sess.CurrentState)),
		slog.String("input", userInput),
		slog.Int("step", sess.StepCount),
	)

	switch sess.CurrentState {
	case session.StateInit:
		return e.handleInit(ctx, sess)
	case session.StateMainMenu:
		return e.handleMainMenu(ctx, sess, userInput)
	case session.StateCheckBalance:
		return e.handleCheckBalance(ctx, sess, userInput)
	case session.StateWithdraw:
		return e.handleWithdraw(ctx, sess, userInput)
	case session.StateWithdrawAmount:
		return e.handleWithdrawAmount(ctx, sess, userInput)
	case session.StateWithdrawConfirm:
		return e.handleWithdrawConfirm(ctx, sess, userInput)
	case session.StateWithdrawPIN:
		return e.handleWithdrawPIN(ctx, sess, userInput)
	case session.StateEarnings:
		return e.handleEarnings(ctx, sess, userInput)
	case session.StateEarningsDaily:
		return e.handleEarningsDaily(ctx, sess, userInput)
	case session.StateEarningsWeekly:
		return e.handleEarningsWeekly(ctx, sess, userInput)
	case session.StateEarningsMonthly:
		return e.handleEarningsMonthly(ctx, sess, userInput)
	case session.StateLastPayment:
		return e.handleLastPayment(ctx, sess, userInput)
	case session.StateLoanStatus:
		return e.handleLoanStatus(ctx, sess, userInput)
	case session.StateLoanApply:
		return e.handleLoanApply(ctx, sess, userInput)
	case session.StateLoanCategory:
		return e.handleLoanCategory(ctx, sess, userInput)
	case session.StateLoanAmount:
		return e.handleLoanAmount(ctx, sess, userInput)
	case session.StateLoanTenure:
		return e.handleLoanTenure(ctx, sess, userInput)
	case session.StateLoanConfirm:
		return e.handleLoanConfirm(ctx, sess, userInput)
	case session.StateRegister:
		return e.handleRegister(ctx, sess, userInput)
	case session.StateRegisterName:
		return e.handleRegisterName(ctx, sess, userInput)
	case session.StateRegisterNationalID:
		return e.handleRegisterNationalID(ctx, sess, userInput)
	case session.StateRegisterRole:
		return e.handleRegisterRole(ctx, sess, userInput)
	case session.StateRegisterPIN:
		return e.handleRegisterPIN(ctx, sess, userInput)
	case session.StateRegisterPINConfirm:
		return e.handleRegisterPINConfirm(ctx, sess, userInput)
	case session.StateRegisterConfirm:
		return e.handleRegisterConfirm(ctx, sess, userInput)
	case session.StateMyProfile:
		return e.handleMyProfile(ctx, sess, userInput)
	case session.StateSetPIN:
		return e.handleSetPIN(ctx, sess, userInput)
	case session.StateSetPINConfirm:
		return e.handleSetPINConfirm(ctx, sess, userInput)
	case session.StateChangePIN:
		return e.handleChangePIN(ctx, sess, userInput)
	case session.StateChangePINNew:
		return e.handleChangePINNew(ctx, sess, userInput)
	case session.StateChangePINConfirm:
		return e.handleChangePINConfirm(ctx, sess, userInput)
	case session.StateLanguageSelect:
		return e.handleLanguageSelect(ctx, sess, userInput)
	default:
		return e.endWithMessage(sess, e.t(sess, "error.generic")), nil
	}
}

// --- Init (first dial only) ---

func (e *Engine) handleInit(ctx context.Context, sess *session.Data) (*Response, error) {
	// Load persisted language preference (survives session endings)
	if savedLang, err := e.sessionStore.GetLanguage(ctx, sess.MSISDN); err == nil && savedLang != "" {
		sess.Language = savedLang
	}

	// Look up user by MSISDN (best-effort, non-blocking on failure)
	user, err := e.backendClient.GetUserByPhone(ctx, sess.MSISDN)
	if err != nil {
		e.logger.Debug("user lookup failed (expected for new users)",
			slog.String("msisdn", sess.MSISDN),
			slog.String("error", err.Error()),
		)
	}
	if user != nil {
		sess.UserID = user.ID
		sess.CrewMemberID = user.CrewMemberID
	}

	sess.CurrentState = session.StateMainMenu
	return e.renderMainMenu(sess), nil
}

// --- Main Menu ---

func (e *Engine) handleMainMenu(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	input = strings.TrimSpace(input)

	// If no input provided, re-render the menu
	if input == "" {
		return e.renderMainMenu(sess), nil
	}

	// Process menu selection
	switch input {
	case "1": // Check Balance
		if sess.CrewMemberID == "" {
			return e.endWithMessage(sess, e.t(sess, "error.not_registered")), nil
		}
		sess.PreviousState = sess.CurrentState
		sess.CurrentState = session.StateCheckBalance
		return e.handleCheckBalance(ctx, sess, "")

	case "2": // Withdraw
		if sess.CrewMemberID == "" {
			return e.endWithMessage(sess, e.t(sess, "error.not_registered")), nil
		}
		sess.PreviousState = sess.CurrentState
		sess.CurrentState = session.StateWithdraw
		return e.handleWithdraw(ctx, sess, "")

	case "3": // Earnings
		if sess.CrewMemberID == "" {
			return e.endWithMessage(sess, e.t(sess, "error.not_registered")), nil
		}
		sess.PreviousState = sess.CurrentState
		sess.CurrentState = session.StateEarnings
		return e.handleEarnings(ctx, sess, "")

	case "4": // Last Payment
		if sess.CrewMemberID == "" {
			return e.endWithMessage(sess, e.t(sess, "error.not_registered")), nil
		}
		sess.PreviousState = sess.CurrentState
		sess.CurrentState = session.StateLastPayment
		return e.handleLastPayment(ctx, sess, "")

	case "5": // Loan Status
		if sess.CrewMemberID == "" {
			return e.endWithMessage(sess, e.t(sess, "error.not_registered")), nil
		}
		sess.PreviousState = sess.CurrentState
		sess.CurrentState = session.StateLoanStatus
		return e.handleLoanStatus(ctx, sess, "")

	case "6": // Register or My Profile
		if sess.CrewMemberID != "" {
			// Registered user — go to profile
			sess.PreviousState = sess.CurrentState
			sess.CurrentState = session.StateMyProfile
			return e.handleMyProfile(ctx, sess, "")
		}
		// Unregistered user — register
		sess.PreviousState = sess.CurrentState
		sess.CurrentState = session.StateRegister
		return e.handleRegister(ctx, sess, "")

	case "7": // Language
		sess.PreviousState = sess.CurrentState
		sess.CurrentState = session.StateLanguageSelect
		return e.handleLanguageSelect(ctx, sess, "")

	case "0": // Exit
		return e.endWithMessage(sess, e.t(sess, "goodbye")), nil

	default:
		return e.continueWithMessage(sess, e.t(sess, "error.invalid_input")+"\n\n"+e.renderMainMenuText(sess)), nil
	}
}

func (e *Engine) renderMainMenu(sess *session.Data) *Response {
	return e.continueWithMessage(sess, e.renderMainMenuText(sess))
}

func (e *Engine) renderMainMenuText(sess *session.Data) string {
	welcome := e.t(sess, "menu.welcome")
	if sess.CrewMemberID != "" {
		return fmt.Sprintf("%s\n1. %s\n2. %s\n3. %s\n4. %s\n5. %s\n6. %s\n7. %s\n0. %s",
			welcome,
			e.t(sess, "menu.check_balance"),
			e.t(sess, "menu.withdraw"),
			e.t(sess, "menu.earnings"),
			e.t(sess, "menu.last_payment"),
			e.t(sess, "menu.loan_status"),
			e.t(sess, "menu.my_profile"),
			e.t(sess, "menu.language"),
			e.t(sess, "menu.exit"),
		)
	}
	// Unregistered user — show limited menu
	return fmt.Sprintf("%s\n6. %s\n7. %s\n0. %s",
		welcome,
		e.t(sess, "menu.register"),
		e.t(sess, "menu.language"),
		e.t(sess, "menu.exit"),
	)
}

// --- Check Balance ---

func (e *Engine) handleCheckBalance(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	// Handle back navigation
	if input == "0" {
		return e.backToMainMenu(sess), nil
	}

	wallet, err := e.backendClient.GetWalletBalance(ctx, sess.CrewMemberID)
	if err != nil {
		e.logger.Error("balance check failed",
			slog.String("crew_member_id", sess.CrewMemberID),
			slog.String("error", err.Error()),
		)
		return e.continueWithMessage(sess, e.t(sess, "error.service_unavailable")+"\n0. "+e.t(sess, "menu.back")), nil
	}

	msg := fmt.Sprintf(e.t(sess, "balance.result"),
		formatMoney(wallet.BalanceCents, wallet.Currency),
	)
	return e.continueWithMessage(sess, msg+"\n0. "+e.t(sess, "menu.back")), nil
}

// --- Withdraw Flow ---

func (e *Engine) handleWithdraw(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	// Show withdrawal menu
	wallet, err := e.backendClient.GetWalletBalance(ctx, sess.CrewMemberID)
	if err != nil {
		return e.endWithMessage(sess, e.t(sess, "error.service_unavailable")), nil
	}

	sess.CurrentState = session.StateWithdrawAmount
	msg := fmt.Sprintf(e.t(sess, "withdraw.enter_amount"),
		formatMoney(wallet.BalanceCents, wallet.Currency),
	)
	return e.continueWithMessage(sess, msg), nil
}

func (e *Engine) handleWithdrawAmount(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	if input == "0" {
		return e.backToMainMenu(sess), nil
	}

	amount, err := parseAmount(input)
	if err != nil || amount <= 0 {
		return e.continueWithMessage(sess, e.t(sess, "withdraw.invalid_amount")), nil
	}

	sess.SetInput("withdraw_amount", input)
	sess.CurrentState = session.StateWithdrawConfirm

	msg := fmt.Sprintf(e.t(sess, "withdraw.confirm"),
		formatMoney(amount, "KES"),
	)
	return e.continueWithMessage(sess, msg), nil
}

func (e *Engine) handleWithdrawConfirm(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	switch strings.TrimSpace(input) {
	case "1": // Confirm — ask for PIN
		sess.CurrentState = session.StateWithdrawPIN
		return e.continueWithMessage(sess, e.t(sess, "withdraw.enter_pin")), nil
	case "2": // Cancel
		return e.backToMainMenu(sess), nil
	default:
		return e.continueWithMessage(sess, e.t(sess, "error.invalid_input")+"\n"+e.t(sess, "withdraw.confirm_options")), nil
	}
}

func (e *Engine) handleWithdrawPIN(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	cleaned := strings.TrimSpace(input)
	if len(cleaned) < 4 || len(cleaned) > 6 {
		return e.continueWithMessage(sess, e.t(sess, "withdraw.invalid_pin")), nil
	}

	// Verify PIN against the backend
	if err := e.backendClient.VerifyPIN(ctx, sess.MSISDN, cleaned); err != nil {
		e.logger.Warn("PIN verification failed",
			slog.String("msisdn", sess.MSISDN),
			slog.String("error", err.Error()),
		)
		errMsg := err.Error()
		// No PIN set — user registered before PIN feature was added
		if strings.Contains(errMsg, "no PIN set") || strings.Contains(errMsg, "VALIDATION_ERROR") {
			return e.continueWithMessage(sess, e.t(sess, "withdraw.no_pin_set")), nil
		}
		// Wrong PIN
		if strings.Contains(errMsg, "Invalid") || strings.Contains(errMsg, "UNAUTHORIZED") || strings.Contains(errMsg, "401") {
			return e.continueWithMessage(sess, e.t(sess, "withdraw.wrong_pin")), nil
		}
		return e.endWithMessage(sess, e.t(sess, "error.service_unavailable")), nil
	}

	// PIN verified — proceed with withdrawal
	amount, _ := parseAmount(sess.GetInput("withdraw_amount"))

	result, err := e.backendClient.InitiateWithdrawal(ctx, sess.CrewMemberID, amount, sess.MSISDN)
	if err != nil {
		e.logger.Error("withdrawal failed",
			slog.String("crew_member_id", sess.CrewMemberID),
			slog.Int64("amount_cents", amount),
			slog.String("error", err.Error()),
		)
		if strings.Contains(err.Error(), "insufficient") {
			return e.endWithMessage(sess, e.t(sess, "withdraw.insufficient_funds")), nil
		}
		return e.endWithMessage(sess, e.t(sess, "error.service_unavailable")), nil
	}

	msg := fmt.Sprintf(e.t(sess, "withdraw.success"),
		formatMoney(amount, "KES"),
		result.Reference,
	)
	return e.endWithMessage(sess, msg), nil
}

// --- Earnings Flow ---

func (e *Engine) handleEarnings(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	if input == "" {
		return e.continueWithMessage(sess, e.t(sess, "earnings.menu")), nil
	}

	switch strings.TrimSpace(input) {
	case "1": // Today
		sess.CurrentState = session.StateEarningsDaily
		return e.handleEarningsDaily(ctx, sess, "")
	case "2": // This week
		sess.CurrentState = session.StateEarningsWeekly
		return e.handleEarningsWeekly(ctx, sess, "")
	case "3": // This month
		sess.CurrentState = session.StateEarningsMonthly
		return e.handleEarningsMonthly(ctx, sess, "")
	case "0": // Back
		return e.backToMainMenu(sess), nil
	default:
		return e.continueWithMessage(sess, e.t(sess, "error.invalid_input")+"\n"+e.t(sess, "earnings.menu")), nil
	}
}

func (e *Engine) handleEarningsDaily(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	if input == "0" {
		return e.backToMainMenu(sess), nil
	}

	summary, err := e.backendClient.GetEarningsSummary(ctx, sess.CrewMemberID, "daily")
	if err != nil {
		return e.continueWithMessage(sess, e.t(sess, "error.service_unavailable")+"\n0. "+e.t(sess, "menu.back")), nil
	}

	msg := fmt.Sprintf(e.t(sess, "earnings.daily_result"),
		formatMoney(summary.TotalEarnedCents, summary.Currency),
		summary.AssignmentCount,
	)
	return e.continueWithMessage(sess, msg+"\n0. "+e.t(sess, "menu.back")), nil
}

func (e *Engine) handleEarningsWeekly(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	if input == "0" {
		return e.backToMainMenu(sess), nil
	}

	summary, err := e.backendClient.GetEarningsSummary(ctx, sess.CrewMemberID, "weekly")
	if err != nil {
		return e.continueWithMessage(sess, e.t(sess, "error.service_unavailable")+"\n0. "+e.t(sess, "menu.back")), nil
	}

	msg := fmt.Sprintf(e.t(sess, "earnings.weekly_result"),
		formatMoney(summary.TotalEarnedCents, summary.Currency),
	)
	return e.continueWithMessage(sess, msg+"\n0. "+e.t(sess, "menu.back")), nil
}

func (e *Engine) handleEarningsMonthly(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	if input == "0" {
		return e.backToMainMenu(sess), nil
	}

	summary, err := e.backendClient.GetEarningsSummary(ctx, sess.CrewMemberID, "monthly")
	if err != nil {
		return e.continueWithMessage(sess, e.t(sess, "error.service_unavailable")+"\n0. "+e.t(sess, "menu.back")), nil
	}

	msg := fmt.Sprintf(e.t(sess, "earnings.monthly_result"),
		formatMoney(summary.TotalEarnedCents, summary.Currency),
	)
	return e.continueWithMessage(sess, msg+"\n0. "+e.t(sess, "menu.back")), nil
}

// --- Last Payment ---

func (e *Engine) handleLastPayment(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	if input == "0" {
		return e.backToMainMenu(sess), nil
	}

	tx, err := e.backendClient.GetLastTransaction(ctx, sess.CrewMemberID)
	if err != nil {
		return e.continueWithMessage(sess, e.t(sess, "error.service_unavailable")+"\n0. "+e.t(sess, "menu.back")), nil
	}
	if tx == nil {
		return e.continueWithMessage(sess, e.t(sess, "payment.no_transactions")+"\n0. "+e.t(sess, "menu.back")), nil
	}

	msg := fmt.Sprintf(e.t(sess, "payment.last_result"),
		tx.TransactionType,
		formatMoney(tx.AmountCents, tx.Currency),
		tx.CreatedAt.Format("02/01/2006 15:04"),
		tx.Reference,
	)
	return e.continueWithMessage(sess, msg+"\n0. "+e.t(sess, "menu.back")), nil
}

// --- Loan Status Flow ---

func (e *Engine) handleLoanStatus(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	if input == "" {
		return e.continueWithMessage(sess, e.t(sess, "loan.menu")), nil
	}

	switch strings.TrimSpace(input) {
	case "1": // View status
		loans, err := e.backendClient.GetLoans(ctx, sess.CrewMemberID)
		if err != nil {
			return e.endWithMessage(sess, e.t(sess, "error.service_unavailable")), nil
		}
		if len(loans) == 0 {
			return e.endWithMessage(sess, e.t(sess, "loan.no_loans")), nil
		}
		loan := loans[0]
		msg := fmt.Sprintf(e.t(sess, "loan.status_result"),
			formatMoney(loan.AmountApprovedCents, loan.Currency),
			string(loan.Status),
		)
		return e.endWithMessage(sess, msg), nil

	case "2": // Apply for loan
		sess.CurrentState = session.StateLoanApply
		return e.handleLoanApply(ctx, sess, "")

	case "3": // View Score Details
		detailed, err := e.backendClient.GetDetailedScore(ctx, sess.CrewMemberID)
		if err != nil || detailed == nil {
			return e.endWithMessage(sess, e.t(sess, "error.service_unavailable")), nil
		}

		// Build top 3 factors summary
		var factorLines string
		count := 0
		for _, f := range detailed.Factors {
			if count >= 3 {
				break
			}
			symbol := "●"
			if f.Impact == "POSITIVE" {
				symbol = "✓"
			} else if f.Impact == "NEGATIVE" {
				symbol = "✗"
			}
			factorLines += fmt.Sprintf("\n%s %s: %d/%d", symbol, f.Name, f.Points, f.MaxPoints)
			count++
		}

		// Build top 2 suggestions
		var sugLines string
		for i, s := range detailed.Suggestions {
			if i >= 2 {
				break
			}
			sugLines += fmt.Sprintf("\n→ %s", s)
		}

		msg := fmt.Sprintf(e.t(sess, "loan.score_details"),
			detailed.Score, detailed.Grade,
			factorLines, sugLines,
		)
		return e.endWithMessage(sess, msg), nil

	case "0": // Back
		return e.backToMainMenu(sess), nil

	default:
		return e.continueWithMessage(sess, e.t(sess, "error.invalid_input")+"\n"+e.t(sess, "loan.menu")), nil
	}
}

func (e *Engine) handleLoanApply(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	// First, always get the credit score — we need it either way
	score, scoreErr := e.backendClient.GetCreditScore(ctx, sess.CrewMemberID)
	if scoreErr != nil || score == nil {
		msg := fmt.Sprintf(e.t(sess, "loan.low_score"), 0, 400)
		return e.continueWithMessage(sess, msg), nil
	}

	// Check if score qualifies for any tier at all
	tier := computeLocalTier(score.Score)
	if tier == nil {
		msg := fmt.Sprintf(e.t(sess, "loan.low_score"), score.Score, 400)
		return e.continueWithMessage(sess, msg), nil
	}

	// Store score/tier for use after category selection
	sess.SetInput("loan_score", fmt.Sprintf("%d", score.Score))

	// Show category selection menu
	sess.CurrentState = session.StateLoanCategory
	return e.continueWithMessage(sess, e.t(sess, "loan.select_category")), nil
}

// handleLoanCategory processes the category selection and checks active loan status.
func (e *Engine) handleLoanCategory(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	if input == "0" {
		return e.backToMainMenu(sess), nil
	}

	// Map input to category
	var category string
	switch strings.TrimSpace(input) {
	case "1":
		category = "PERSONAL"
	case "2":
		category = "EMERGENCY"
	case "3":
		category = "EDUCATION"
	case "4":
		category = "BUSINESS"
	case "5":
		category = "ASSET"
	default:
		return e.continueWithMessage(sess, e.t(sess, "error.invalid_input")), nil
	}
	sess.SetInput("loan_category", category)

	// Check for active loans in this category — warn user EARLY
	loans, loansErr := e.backendClient.GetLoans(ctx, sess.CrewMemberID)
	if loansErr == nil && len(loans) > 0 {
		for _, l := range loans {
			if l.Status == "APPLIED" || l.Status == "APPROVED" ||
				l.Status == "DISBURSED" || l.Status == "REPAYING" {
				// Check if this blocks the user depending on policy
				// (Backend enforces policy properly, but we show a warning for UX)
				if l.Category == category {
					return e.continueWithMessage(sess,
						fmt.Sprintf(e.t(sess, "loan.active_in_category"), category)), nil
				}
			}
		}
	}

	// Fetch tier info
	scoreVal := 0
	fmt.Sscanf(sess.GetInput("loan_score"), "%d", &scoreVal)

	tier, err := e.backendClient.GetLoanTier(ctx, sess.CrewMemberID)
	if err != nil || tier == nil {
		tier = computeLocalTier(scoreVal)
	}
	if tier == nil {
		msg := fmt.Sprintf(e.t(sess, "loan.low_score"), scoreVal, 400)
		return e.continueWithMessage(sess, msg), nil
	}

	// Store tier info in session for downstream validation
	sess.SetInput("loan_tier_grade", tier.Grade)
	sess.SetInput("loan_tier_max", fmt.Sprintf("%.0f", tier.MaxLoanKES))
	sess.SetInput("loan_tier_rate", fmt.Sprintf("%.0f", tier.InterestRate))
	sess.SetInput("loan_tier_tenure", fmt.Sprintf("%d", tier.MaxTenureDays))

	// Show tier info and prompt for amount
	msg := fmt.Sprintf(e.t(sess, "loan.tier_info"),
		tier.Grade,
		scoreVal,
		tier.MaxLoanKES,
		tier.InterestRate,
		tier.MaxTenureDays,
	)
	sess.CurrentState = session.StateLoanAmount
	return e.continueWithMessage(sess, msg), nil
}

// computeLocalTier applies the same tier logic as the backend when the
// tier API is unavailable (e.g., auth middleware blocks USSD calls).
func computeLocalTier(score int) *backend.LoanTierResponse {
	type tierDef struct {
		grade       string
		minScore    int
		maxLoanKES  float64
		rate        float64
		tenureDays  int
		cooldown    int
		description string
	}

	tiers := []tierDef{
		{"EXCELLENT", 750, 50000, 5, 30, 0, "Premium — KES 50,000 max at 5%"},
		{"GOOD", 650, 20000, 8, 30, 3, "Standard — KES 20,000 max at 8%"},
		{"FAIR", 500, 5000, 12, 14, 7, "Growth — KES 5,000 max at 12%"},
		{"POOR", 400, 1000, 15, 7, 14, "Starter — KES 1,000 max at 15%"},
	}

	for _, t := range tiers {
		if score >= t.minScore {
			return &backend.LoanTierResponse{
				Score:         score,
				Grade:         t.grade,
				MaxLoanKES:    t.maxLoanKES,
				InterestRate:  t.rate,
				MaxTenureDays: t.tenureDays,
				CooldownDays:  t.cooldown,
				Description:   t.description,
			}
		}
	}
	return nil // Below 400 — not eligible
}

func (e *Engine) handleLoanAmount(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	if input == "0" {
		return e.backToMainMenu(sess), nil
	}

	amount, err := parseAmount(input)
	if err != nil || amount <= 0 {
		return e.continueWithMessage(sess, e.t(sess, "loan.invalid_amount")), nil
	}

	// Validate against tier max
	tierMax := sess.GetInput("loan_tier_max")
	if tierMax != "" {
		var maxKES float64
		fmt.Sscanf(tierMax, "%f", &maxKES)
		amountKES := float64(amount) / 100
		if amountKES > maxKES {
			msg := fmt.Sprintf(e.t(sess, "loan.amount_exceeds_tier"), maxKES)
			return e.continueWithMessage(sess, msg), nil
		}
	}

	sess.SetInput("loan_amount", input)

	// Build dynamic tenure menu based on tier max
	maxTenure := 30
	if t := sess.GetInput("loan_tier_tenure"); t != "" {
		fmt.Sscanf(t, "%d", &maxTenure)
	}

	var tenureMenu string
	if maxTenure >= 30 {
		tenureMenu = e.t(sess, "loan.select_tenure") // All options
	} else if maxTenure >= 14 {
		tenureMenu = e.t(sess, "loan.select_tenure_14") // 7 and 14 only
	} else {
		tenureMenu = e.t(sess, "loan.select_tenure_7") // 7 days only
	}

	sess.CurrentState = session.StateLoanTenure
	return e.continueWithMessage(sess, tenureMenu), nil
}

func (e *Engine) handleLoanTenure(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	var tenureDays int

	// Get tier max tenure
	maxTenure := 30
	if t := sess.GetInput("loan_tier_tenure"); t != "" {
		fmt.Sscanf(t, "%d", &maxTenure)
	}

	switch strings.TrimSpace(input) {
	case "1":
		tenureDays = 7
	case "2":
		if maxTenure >= 14 {
			tenureDays = 14
		} else {
			return e.continueWithMessage(sess, e.t(sess, "error.invalid_input")), nil
		}
	case "3":
		if maxTenure >= 30 {
			tenureDays = 30
		} else {
			return e.continueWithMessage(sess, e.t(sess, "error.invalid_input")), nil
		}
	case "0":
		return e.backToMainMenu(sess), nil
	default:
		return e.continueWithMessage(sess, e.t(sess, "error.invalid_input")), nil
	}

	sess.SetInput("loan_tenure", fmt.Sprintf("%d", tenureDays))
	sess.CurrentState = session.StateLoanConfirm

	amount, _ := parseAmount(sess.GetInput("loan_amount"))
	interestRate := sess.GetInput("loan_tier_rate")
	grade := sess.GetInput("loan_tier_grade")

	msg := fmt.Sprintf(e.t(sess, "loan.confirm_with_rate"),
		formatMoney(amount, "KES"),
		tenureDays,
		interestRate,
		grade,
	)
	return e.continueWithMessage(sess, msg), nil
}

func (e *Engine) handleLoanConfirm(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	switch strings.TrimSpace(input) {
	case "1": // Confirm
		amount, _ := parseAmount(sess.GetInput("loan_amount"))
		tenure := 30
		if t := sess.GetInput("loan_tenure"); t != "" {
			fmt.Sscanf(t, "%d", &tenure)
		}

		category := sess.GetInput("loan_category")
		if category == "" {
			category = "PERSONAL"
		}

		loan, err := e.backendClient.ApplyForLoan(ctx, sess.CrewMemberID, amount, tenure, category, "")
		if err != nil {
			e.logger.Error("loan application failed",
				slog.String("crew_member_id", sess.CrewMemberID),
				slog.String("category", category),
				slog.String("error", err.Error()),
			)
			// Show the actual error to the user if it's a business rule
			errMsg := err.Error()
			if strings.Contains(errMsg, "active loan") {
				return e.continueWithMessage(sess, e.t(sess, "loan.active_loan")), nil
			}
			// For other known errors, show the backend message
			msg := fmt.Sprintf(e.t(sess, "loan.apply_failed"), errMsg)
			return e.continueWithMessage(sess, msg), nil
		}

		msg := fmt.Sprintf(e.t(sess, "loan.applied_success"),
			formatMoney(loan.AmountRequestedCents, loan.Currency),
		)
		return e.endWithMessage(sess, msg), nil

	case "2": // Cancel
		return e.backToMainMenu(sess), nil

	default:
		return e.continueWithMessage(sess, e.t(sess, "error.invalid_input")+"\n"+e.t(sess, "loan.confirm_options")), nil
	}
}

// --- Registration Flow ---

func (e *Engine) handleRegister(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	sess.CurrentState = session.StateRegisterName
	return e.continueWithMessage(sess, e.t(sess, "register.enter_name")), nil
}

func (e *Engine) handleRegisterName(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	if input == "0" {
		return e.backToMainMenu(sess), nil
	}

	parts := strings.Fields(strings.TrimSpace(input))
	if len(parts) < 2 {
		return e.continueWithMessage(sess, e.t(sess, "register.invalid_name")), nil
	}

	sess.SetInput("first_name", parts[0])
	sess.SetInput("last_name", strings.Join(parts[1:], " "))
	sess.CurrentState = session.StateRegisterNationalID
	return e.continueWithMessage(sess, e.t(sess, "register.enter_national_id")), nil
}

func (e *Engine) handleRegisterNationalID(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	if input == "0" {
		return e.backToMainMenu(sess), nil
	}

	cleaned := strings.TrimSpace(input)
	if len(cleaned) < 5 || len(cleaned) > 12 {
		return e.continueWithMessage(sess, e.t(sess, "register.invalid_national_id")), nil
	}

	sess.SetInput("national_id", cleaned)
	sess.CurrentState = session.StateRegisterRole
	return e.continueWithMessage(sess, e.t(sess, "register.select_role")), nil
}

func (e *Engine) handleRegisterRole(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	var role string
	switch strings.TrimSpace(input) {
	case "1":
		role = "DRIVER"
	case "2":
		role = "CONDUCTOR"
	case "3":
		role = "RIDER"
	case "0":
		return e.backToMainMenu(sess), nil
	default:
		return e.continueWithMessage(sess, e.t(sess, "error.invalid_input")+"\n"+e.t(sess, "register.select_role")), nil
	}

	sess.SetInput("role", role)
	sess.CurrentState = session.StateRegisterPIN
	return e.continueWithMessage(sess, e.t(sess, "register.enter_pin")), nil
}

func (e *Engine) handleRegisterPIN(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	cleaned := strings.TrimSpace(input)
	if len(cleaned) < 4 || len(cleaned) > 6 {
		return e.continueWithMessage(sess, e.t(sess, "register.invalid_pin")), nil
	}
	// Validate digits only
	for _, c := range cleaned {
		if c < '0' || c > '9' {
			return e.continueWithMessage(sess, e.t(sess, "register.invalid_pin")), nil
		}
	}

	sess.SetInput("pin", cleaned)
	sess.CurrentState = session.StateRegisterPINConfirm
	return e.continueWithMessage(sess, e.t(sess, "register.confirm_pin")), nil
}

func (e *Engine) handleRegisterPINConfirm(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	cleaned := strings.TrimSpace(input)
	if cleaned != sess.GetInput("pin") {
		// PINs don't match — go back to PIN entry
		sess.CurrentState = session.StateRegisterPIN
		return e.continueWithMessage(sess, e.t(sess, "register.pin_mismatch")), nil
	}

	// PINs match — proceed to details confirmation
	sess.CurrentState = session.StateRegisterConfirm

	msg := fmt.Sprintf(e.t(sess, "register.confirm"),
		sess.GetInput("first_name")+" "+sess.GetInput("last_name"),
		sess.GetInput("role"),
	)
	return e.continueWithMessage(sess, msg), nil
}

func (e *Engine) handleRegisterConfirm(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	switch strings.TrimSpace(input) {
	case "1": // Confirm
		// Generate a temporary password from phone + national ID
		phone := sess.MSISDN
		nid := sess.GetInput("national_id")
		tempPassword := generateTempPassword(phone, nid)

		result, err := e.backendClient.RegisterCrew(ctx, backend.RegisterRequest{
			Phone:      sess.MSISDN,
			Password:   tempPassword,
			FirstName:  sess.GetInput("first_name"),
			LastName:   sess.GetInput("last_name"),
			NationalID: sess.GetInput("national_id"),
			Role:       "CREW",                    // SystemRole
			CrewRole:   sess.GetInput("role"),      // DRIVER, CONDUCTOR, RIDER
		})
		if err != nil {
			e.logger.Error("registration failed",
				slog.String("msisdn", sess.MSISDN),
				slog.String("error", err.Error()),
			)
			errMsg := err.Error()
			if strings.Contains(errMsg, "already exists") ||
				strings.Contains(errMsg, "already registered") ||
				strings.Contains(errMsg, "duplicate") ||
				strings.Contains(errMsg, "409") {
				return e.endWithMessage(sess, e.t(sess, "register.already_registered")), nil
			}
			return e.endWithMessage(sess, e.t(sess, "error.service_unavailable")), nil
		}

		sess.CrewMemberID = result.CrewMemberID
		sess.UserID = result.UserID

		// Save the transaction PIN the user set during registration
		pin := sess.GetInput("pin")
		if pin != "" {
			if err := e.backendClient.SetPIN(ctx, sess.MSISDN, pin); err != nil {
				e.logger.Error("failed to set PIN after registration",
					slog.String("msisdn", sess.MSISDN),
					slog.String("error", err.Error()),
				)
				// Registration succeeded but PIN failed — user can reset later
			}
		}

		return e.endWithMessage(sess, e.t(sess, "register.success")), nil

	case "2": // Cancel
		sess.ClearInputs()
		return e.backToMainMenu(sess), nil

	default:
		return e.continueWithMessage(sess, e.t(sess, "error.invalid_input")+"\n"+e.t(sess, "register.confirm_options")), nil
	}
}

// generateTempPassword creates a temporary password for USSD registration.
// Format: last 4 of phone + last 4 of national ID + "Aa1!" (meets min=8, has upper+lower+digit+special)
func generateTempPassword(phone, nationalID string) string {
	phonePart := phone
	if len(phonePart) > 4 {
		phonePart = phonePart[len(phonePart)-4:]
	}
	nidPart := nationalID
	if len(nidPart) > 4 {
		nidPart = nidPart[len(nidPart)-4:]
	}
	return phonePart + nidPart + "Aa1!"
}

// --- Language Selection ---

func (e *Engine) handleLanguageSelect(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	if input == "" {
		return e.continueWithMessage(sess, e.t(sess, "language.menu")), nil
	}

	switch strings.TrimSpace(input) {
	case "1":
		sess.Language = "en"
	case "2":
		sess.Language = "sw"
	case "0":
		return e.backToMainMenu(sess), nil
	default:
		return e.continueWithMessage(sess, e.t(sess, "error.invalid_input")+"\n"+e.t(sess, "language.menu")), nil
	}

	// Persist language preference across sessions
	if err := e.sessionStore.SaveLanguage(ctx, sess.MSISDN, sess.Language); err != nil {
		e.logger.Error("failed to persist language preference",
			slog.String("msisdn", sess.MSISDN),
			slog.String("error", err.Error()),
		)
	}

	return e.endWithMessage(sess, e.t(sess, "language.changed")), nil
}

// --- My Profile ---

func (e *Engine) handleMyProfile(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	if input == "" {
		return e.continueWithMessage(sess, e.t(sess, "profile.menu")), nil
	}

	switch strings.TrimSpace(input) {
	case "1": // Set PIN (for users who don't have one)
		sess.CurrentState = session.StateSetPIN
		return e.continueWithMessage(sess, e.t(sess, "profile.set_pin")), nil
	case "2": // Change PIN
		sess.CurrentState = session.StateChangePIN
		return e.continueWithMessage(sess, e.t(sess, "profile.enter_current_pin")), nil
	case "3": // View Profile
		crew, err := e.backendClient.GetCrewMember(ctx, sess.CrewMemberID)
		name := "N/A"
		if err == nil && crew != nil {
			name = crew.FullName
		}
		msg := fmt.Sprintf(e.t(sess, "profile.view"),
			sess.MSISDN,
			name,
		)
		return e.continueWithMessage(sess, msg+"\n0. "+e.t(sess, "menu.back")), nil
	case "0": // Back
		return e.backToMainMenu(sess), nil
	default:
		return e.continueWithMessage(sess, e.t(sess, "error.invalid_input")+"\n"+e.t(sess, "profile.menu")), nil
	}
}

// --- Set PIN (first time) ---

func (e *Engine) handleSetPIN(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	cleaned := strings.TrimSpace(input)
	if cleaned == "0" {
		sess.CurrentState = session.StateMyProfile
		return e.handleMyProfile(ctx, sess, "")
	}
	if len(cleaned) < 4 || len(cleaned) > 6 || !isDigits(cleaned) {
		return e.continueWithMessage(sess, e.t(sess, "register.invalid_pin")), nil
	}

	sess.SetInput("new_pin", cleaned)
	sess.CurrentState = session.StateSetPINConfirm
	return e.continueWithMessage(sess, e.t(sess, "register.confirm_pin")), nil
}

func (e *Engine) handleSetPINConfirm(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	cleaned := strings.TrimSpace(input)
	if cleaned != sess.GetInput("new_pin") {
		sess.CurrentState = session.StateSetPIN
		return e.continueWithMessage(sess, e.t(sess, "register.pin_mismatch")), nil
	}

	// Save PIN to backend
	if err := e.backendClient.SetPIN(ctx, sess.MSISDN, cleaned); err != nil {
		e.logger.Error("failed to set PIN",
			slog.String("msisdn", sess.MSISDN),
			slog.String("error", err.Error()),
		)
		return e.endWithMessage(sess, e.t(sess, "error.service_unavailable")), nil
	}

	sess.CurrentState = session.StateMyProfile
	return e.continueWithMessage(sess, e.t(sess, "profile.pin_set_success")+"\n0. "+e.t(sess, "menu.back")), nil
}

// --- Change PIN (requires current PIN) ---

func (e *Engine) handleChangePIN(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	cleaned := strings.TrimSpace(input)
	if cleaned == "0" {
		sess.CurrentState = session.StateMyProfile
		return e.handleMyProfile(ctx, sess, "")
	}
	if len(cleaned) < 4 || len(cleaned) > 6 {
		return e.continueWithMessage(sess, e.t(sess, "withdraw.invalid_pin")), nil
	}

	// Verify current PIN
	if err := e.backendClient.VerifyPIN(ctx, sess.MSISDN, cleaned); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "no PIN set") || strings.Contains(errMsg, "VALIDATION_ERROR") {
			// Redirect to Set PIN flow
			sess.CurrentState = session.StateSetPIN
			return e.continueWithMessage(sess, e.t(sess, "profile.no_pin_redirect")), nil
		}
		if strings.Contains(errMsg, "Invalid") || strings.Contains(errMsg, "UNAUTHORIZED") || strings.Contains(errMsg, "401") {
			return e.continueWithMessage(sess, e.t(sess, "withdraw.wrong_pin")+"\n0. "+e.t(sess, "menu.back")), nil
		}
		return e.endWithMessage(sess, e.t(sess, "error.service_unavailable")), nil
	}

	sess.CurrentState = session.StateChangePINNew
	return e.continueWithMessage(sess, e.t(sess, "profile.enter_new_pin")), nil
}

func (e *Engine) handleChangePINNew(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	cleaned := strings.TrimSpace(input)
	if cleaned == "0" {
		sess.CurrentState = session.StateMyProfile
		return e.handleMyProfile(ctx, sess, "")
	}
	if len(cleaned) < 4 || len(cleaned) > 6 || !isDigits(cleaned) {
		return e.continueWithMessage(sess, e.t(sess, "register.invalid_pin")), nil
	}

	sess.SetInput("new_pin", cleaned)
	sess.CurrentState = session.StateChangePINConfirm
	return e.continueWithMessage(sess, e.t(sess, "register.confirm_pin")), nil
}

func (e *Engine) handleChangePINConfirm(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	cleaned := strings.TrimSpace(input)
	if cleaned != sess.GetInput("new_pin") {
		sess.CurrentState = session.StateChangePINNew
		return e.continueWithMessage(sess, e.t(sess, "register.pin_mismatch")), nil
	}

	if err := e.backendClient.SetPIN(ctx, sess.MSISDN, cleaned); err != nil {
		e.logger.Error("failed to change PIN",
			slog.String("msisdn", sess.MSISDN),
			slog.String("error", err.Error()),
		)
		return e.endWithMessage(sess, e.t(sess, "error.service_unavailable")), nil
	}

	sess.CurrentState = session.StateMyProfile
	return e.continueWithMessage(sess, e.t(sess, "profile.pin_changed_success")+"\n0. "+e.t(sess, "menu.back")), nil
}

// isDigits checks if a string contains only digit characters.
func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// --- Helper methods ---

func (e *Engine) t(sess *session.Data, key string) string {
	return e.translator.T(sess.Language, key)
}

func (e *Engine) continueWithMessage(sess *session.Data, msg string) *Response {
	return &Response{Message: msg, EndSession: false}
}

func (e *Engine) endWithMessage(sess *session.Data, msg string) *Response {
	sess.CurrentState = session.StateEnd
	return &Response{Message: msg, EndSession: true}
}

func (e *Engine) backToMainMenu(sess *session.Data) *Response {
	sess.CurrentState = session.StateMainMenu
	sess.ClearInputs()
	return e.renderMainMenu(sess)
}

// formatMoney formats cents to human-readable currency string.
func formatMoney(cents int64, currency string) string {
	whole := cents / 100
	frac := cents % 100
	if frac < 0 {
		frac = -frac
	}
	return fmt.Sprintf("%s %d.%02d", currency, whole, frac)
}

// parseAmount converts a string amount (in whole units) to cents.
func parseAmount(input string) (int64, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return 0, fmt.Errorf("empty amount")
	}

	// Handle decimal amounts
	parts := strings.Split(input, ".")
	var whole, frac int64
	if _, err := fmt.Sscanf(parts[0], "%d", &whole); err != nil {
		return 0, fmt.Errorf("invalid amount: %w", err)
	}
	if len(parts) > 1 && len(parts[1]) > 0 {
		fracStr := parts[1]
		if len(fracStr) > 2 {
			fracStr = fracStr[:2]
		}
		if len(fracStr) == 1 {
			fracStr += "0"
		}
		fmt.Sscanf(fracStr, "%d", &frac)
	}

	return whole*100 + frac, nil
}
