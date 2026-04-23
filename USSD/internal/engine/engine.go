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
	translator    *i18n.Translator
	logger        *slog.Logger
}

// NewEngine creates a new USSD processing engine.
func NewEngine(client *backend.Client, translator *i18n.Translator, logger *slog.Logger) *Engine {
	return &Engine{
		backendClient: client,
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
		return e.handleCheckBalance(ctx, sess)
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
		return e.handleEarningsDaily(ctx, sess)
	case session.StateEarningsWeekly:
		return e.handleEarningsWeekly(ctx, sess)
	case session.StateEarningsMonthly:
		return e.handleEarningsMonthly(ctx, sess)
	case session.StateLastPayment:
		return e.handleLastPayment(ctx, sess)
	case session.StateLoanStatus:
		return e.handleLoanStatus(ctx, sess, userInput)
	case session.StateLoanApply:
		return e.handleLoanApply(ctx, sess, userInput)
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
	case session.StateRegisterConfirm:
		return e.handleRegisterConfirm(ctx, sess, userInput)
	case session.StateLanguageSelect:
		return e.handleLanguageSelect(ctx, sess, userInput)
	default:
		return e.endWithMessage(sess, e.t(sess, "error.generic")), nil
	}
}

// --- Init (first dial only) ---

func (e *Engine) handleInit(ctx context.Context, sess *session.Data) (*Response, error) {
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
		return e.handleCheckBalance(ctx, sess)

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
		return e.handleLastPayment(ctx, sess)

	case "5": // Loan Status
		if sess.CrewMemberID == "" {
			return e.endWithMessage(sess, e.t(sess, "error.not_registered")), nil
		}
		sess.PreviousState = sess.CurrentState
		sess.CurrentState = session.StateLoanStatus
		return e.handleLoanStatus(ctx, sess, "")

	case "6": // Register
		if sess.CrewMemberID != "" {
			return e.endWithMessage(sess, e.t(sess, "register.already_registered")), nil
		}
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
		return fmt.Sprintf("%s\n1. %s\n2. %s\n3. %s\n4. %s\n5. %s\n7. %s\n0. %s",
			welcome,
			e.t(sess, "menu.check_balance"),
			e.t(sess, "menu.withdraw"),
			e.t(sess, "menu.earnings"),
			e.t(sess, "menu.last_payment"),
			e.t(sess, "menu.loan_status"),
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

func (e *Engine) handleCheckBalance(ctx context.Context, sess *session.Data) (*Response, error) {
	wallet, err := e.backendClient.GetWalletBalance(ctx, sess.CrewMemberID)
	if err != nil {
		e.logger.Error("balance check failed",
			slog.String("crew_member_id", sess.CrewMemberID),
			slog.String("error", err.Error()),
		)
		return e.endWithMessage(sess, e.t(sess, "error.service_unavailable")), nil
	}

	msg := fmt.Sprintf(e.t(sess, "balance.result"),
		formatMoney(wallet.BalanceCents, wallet.Currency),
	)
	return e.endWithMessage(sess, msg), nil
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
	case "1": // Confirm
		sess.CurrentState = session.StateWithdrawPIN
		return e.continueWithMessage(sess, e.t(sess, "withdraw.enter_pin")), nil
	case "2": // Cancel
		return e.backToMainMenu(sess), nil
	default:
		return e.continueWithMessage(sess, e.t(sess, "error.invalid_input")+"\n"+e.t(sess, "withdraw.confirm_options")), nil
	}
}

func (e *Engine) handleWithdrawPIN(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	if len(input) < 4 || len(input) > 6 {
		return e.continueWithMessage(sess, e.t(sess, "withdraw.invalid_pin")), nil
	}

	amount, _ := parseAmount(sess.GetInput("withdraw_amount"))

	result, err := e.backendClient.InitiateWithdrawal(ctx, sess.CrewMemberID, amount)
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
		return e.handleEarningsDaily(ctx, sess)
	case "2": // This week
		sess.CurrentState = session.StateEarningsWeekly
		return e.handleEarningsWeekly(ctx, sess)
	case "3": // This month
		sess.CurrentState = session.StateEarningsMonthly
		return e.handleEarningsMonthly(ctx, sess)
	case "0": // Back
		return e.backToMainMenu(sess), nil
	default:
		return e.continueWithMessage(sess, e.t(sess, "error.invalid_input")+"\n"+e.t(sess, "earnings.menu")), nil
	}
}

func (e *Engine) handleEarningsDaily(ctx context.Context, sess *session.Data) (*Response, error) {
	summary, err := e.backendClient.GetEarningsSummary(ctx, sess.CrewMemberID, "daily")
	if err != nil {
		return e.endWithMessage(sess, e.t(sess, "error.service_unavailable")), nil
	}

	msg := fmt.Sprintf(e.t(sess, "earnings.daily_result"),
		formatMoney(summary.TotalEarnedCents, summary.Currency),
		summary.AssignmentCount,
	)
	return e.endWithMessage(sess, msg), nil
}

func (e *Engine) handleEarningsWeekly(ctx context.Context, sess *session.Data) (*Response, error) {
	summary, err := e.backendClient.GetEarningsSummary(ctx, sess.CrewMemberID, "weekly")
	if err != nil {
		return e.endWithMessage(sess, e.t(sess, "error.service_unavailable")), nil
	}

	msg := fmt.Sprintf(e.t(sess, "earnings.weekly_result"),
		formatMoney(summary.TotalEarnedCents, summary.Currency),
	)
	return e.endWithMessage(sess, msg), nil
}

func (e *Engine) handleEarningsMonthly(ctx context.Context, sess *session.Data) (*Response, error) {
	summary, err := e.backendClient.GetEarningsSummary(ctx, sess.CrewMemberID, "monthly")
	if err != nil {
		return e.endWithMessage(sess, e.t(sess, "error.service_unavailable")), nil
	}

	msg := fmt.Sprintf(e.t(sess, "earnings.monthly_result"),
		formatMoney(summary.TotalEarnedCents, summary.Currency),
	)
	return e.endWithMessage(sess, msg), nil
}

// --- Last Payment ---

func (e *Engine) handleLastPayment(ctx context.Context, sess *session.Data) (*Response, error) {
	tx, err := e.backendClient.GetLastTransaction(ctx, sess.CrewMemberID)
	if err != nil {
		return e.endWithMessage(sess, e.t(sess, "error.service_unavailable")), nil
	}
	if tx == nil {
		return e.endWithMessage(sess, e.t(sess, "payment.no_transactions")), nil
	}

	msg := fmt.Sprintf(e.t(sess, "payment.last_result"),
		tx.TransactionType,
		formatMoney(tx.AmountCents, tx.Currency),
		tx.CreatedAt.Format("02/01/2006 15:04"),
		tx.Reference,
	)
	return e.endWithMessage(sess, msg), nil
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

	case "0": // Back
		return e.backToMainMenu(sess), nil

	default:
		return e.continueWithMessage(sess, e.t(sess, "error.invalid_input")+"\n"+e.t(sess, "loan.menu")), nil
	}
}

func (e *Engine) handleLoanApply(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	score, err := e.backendClient.GetCreditScore(ctx, sess.CrewMemberID)
	if err != nil || score == nil || score.Score < 400 {
		return e.endWithMessage(sess, e.t(sess, "loan.low_score")), nil
	}

	sess.CurrentState = session.StateLoanAmount
	return e.continueWithMessage(sess, e.t(sess, "loan.enter_amount")), nil
}

func (e *Engine) handleLoanAmount(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	if input == "0" {
		return e.backToMainMenu(sess), nil
	}

	amount, err := parseAmount(input)
	if err != nil || amount <= 0 {
		return e.continueWithMessage(sess, e.t(sess, "loan.invalid_amount")), nil
	}

	sess.SetInput("loan_amount", input)
	sess.CurrentState = session.StateLoanTenure
	return e.continueWithMessage(sess, e.t(sess, "loan.select_tenure")), nil
}

func (e *Engine) handleLoanTenure(ctx context.Context, sess *session.Data, input string) (*Response, error) {
	var tenureDays int
	switch strings.TrimSpace(input) {
	case "1":
		tenureDays = 7
	case "2":
		tenureDays = 14
	case "3":
		tenureDays = 30
	case "0":
		return e.backToMainMenu(sess), nil
	default:
		return e.continueWithMessage(sess, e.t(sess, "error.invalid_input")+"\n"+e.t(sess, "loan.select_tenure")), nil
	}

	sess.SetInput("loan_tenure", fmt.Sprintf("%d", tenureDays))
	sess.CurrentState = session.StateLoanConfirm

	amount, _ := parseAmount(sess.GetInput("loan_amount"))
	msg := fmt.Sprintf(e.t(sess, "loan.confirm"),
		formatMoney(amount, "KES"),
		tenureDays,
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

		loan, err := e.backendClient.ApplyForLoan(ctx, sess.CrewMemberID, amount, tenure)
		if err != nil {
			e.logger.Error("loan application failed",
				slog.String("crew_member_id", sess.CrewMemberID),
				slog.String("error", err.Error()),
			)
			return e.endWithMessage(sess, e.t(sess, "error.service_unavailable")), nil
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
	sess.CurrentState = session.StateRegisterConfirm

	msg := fmt.Sprintf(e.t(sess, "register.confirm"),
		sess.GetInput("first_name")+" "+sess.GetInput("last_name"),
		role,
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

	return e.endWithMessage(sess, e.t(sess, "language.changed")), nil
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
