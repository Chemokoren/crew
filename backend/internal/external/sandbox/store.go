// Package sandbox provides an in-memory financial data store for the sandbox server.
// All state is held in memory — restart for a clean slate.
package sandbox

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// --- Data models ---

// Account represents a sandbox financial account.
type Account struct {
	AccountNo    string    `json:"account_no"`
	Name         string    `json:"name"`
	Phone        string    `json:"phone"`
	BalanceCents int64     `json:"balance_cents"`
	Currency     string    `json:"currency"`
	PINHash      string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// Transaction represents a financial transaction in the sandbox.
type Transaction struct {
	ID                string    `json:"id"`
	AccountNo         string    `json:"account_no"`
	TransactionType   string    `json:"transaction_type"` // CREDIT, DEBIT, PAYOUT
	Category          string    `json:"category"`         // EARNING, WITHDRAWAL, SEED, LOAN_DISBURSEMENT
	AmountCents       int64     `json:"amount_cents"`
	BalanceAfterCents int64     `json:"balance_after_cents"`
	Currency          string    `json:"currency"`
	Reference         string    `json:"reference"`
	Description       string    `json:"description"`
	Status            string    `json:"status"` // COMPLETED, PENDING, FAILED
	CreatedAt         time.Time `json:"created_at"`
}

// Earning represents a simulated earning event.
type Earning struct {
	ID           string    `json:"id"`
	AccountNo    string    `json:"account_no"`
	AmountCents  int64     `json:"amount_cents"`
	EarningType  string    `json:"earning_type"` // FIXED, COMMISSION, HYBRID
	Currency     string    `json:"currency"`
	Description  string    `json:"description"`
	EarnedAt     time.Time `json:"earned_at"`
}

// Loan represents a sandbox loan.
type Loan struct {
	ID                   string    `json:"id"`
	AccountNo            string    `json:"account_no"`
	AmountRequestedCents int64     `json:"amount_requested_cents"`
	AmountApprovedCents  int64     `json:"amount_approved_cents"`
	Currency             string    `json:"currency"`
	Status               string    `json:"status"` // PENDING, APPROVED, DISBURSED, REJECTED
	TenureDays           int       `json:"tenure_days"`
	CreditScore          int       `json:"credit_score"`
	CreatedAt            time.Time `json:"created_at"`
}

// Insurance represents a sandbox insurance policy.
type Insurance struct {
	ID           string    `json:"id"`
	AccountNo    string    `json:"account_no"`
	Provider     string    `json:"provider"`
	PolicyType   string    `json:"policy_type"` // PERSONAL_ACCIDENT, VEHICLE, HEALTH
	PremiumCents int64     `json:"premium_cents"`
	Currency     string    `json:"currency"`
	Status       string    `json:"status"` // ACTIVE, LAPSED, CANCELLED
	StartsAt     time.Time `json:"starts_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// PendingPayout tracks payouts awaiting OTP verification.
type PendingPayout struct {
	Reference   string    `json:"reference"`
	AccountNo   string    `json:"account_no"`
	AmountCents int64     `json:"amount_cents"`
	OrderID     string    `json:"order_id"`
	Channel     string    `json:"channel"`
	CreatedAt   time.Time `json:"created_at"`
}

// --- Store ---

// Store holds all sandbox financial state in memory.
type Store struct {
	mu             sync.RWMutex
	accounts       map[string]*Account       // keyed by account_no
	transactions   map[string][]Transaction  // keyed by account_no
	earnings       map[string][]Earning      // keyed by account_no
	loans          map[string][]Loan         // keyed by account_no
	insurance      map[string][]Insurance    // keyed by account_no
	pendingPayouts map[string]*PendingPayout // keyed by reference
	allLoans       map[string]*Loan          // keyed by loan ID (for approve/disburse)
}

// NewStore creates an empty sandbox store.
func NewStore() *Store {
	return &Store{
		accounts:       make(map[string]*Account),
		transactions:   make(map[string][]Transaction),
		earnings:       make(map[string][]Earning),
		loans:          make(map[string][]Loan),
		insurance:      make(map[string][]Insurance),
		pendingPayouts: make(map[string]*PendingPayout),
		allLoans:       make(map[string]*Loan),
	}
}

// --- Account operations ---

// CreateAccount adds a new account to the store.
func (s *Store) CreateAccount(acct *Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.accounts[acct.AccountNo]; exists {
		return fmt.Errorf("account %s already exists", acct.AccountNo)
	}
	acct.CreatedAt = time.Now()
	s.accounts[acct.AccountNo] = acct
	return nil
}

// GetAccount retrieves an account by account number.
func (s *Store) GetAccount(accountNo string) (*Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	acct, ok := s.accounts[accountNo]
	if !ok {
		return nil, fmt.Errorf("account %s not found", accountNo)
	}
	return acct, nil
}

// FindAccountByPhone finds an account by phone number.
func (s *Store) FindAccountByPhone(phone string) (*Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, acct := range s.accounts {
		if acct.Phone == phone {
			return acct, nil
		}
	}
	return nil, fmt.Errorf("no account found for phone %s", phone)
}

// Credit adds funds to an account.
func (s *Store) Credit(accountNo string, amountCents int64, category, description string) (*Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	acct, ok := s.accounts[accountNo]
	if !ok {
		return nil, fmt.Errorf("account %s not found", accountNo)
	}

	acct.BalanceCents += amountCents

	tx := Transaction{
		ID:                uuid.New().String(),
		AccountNo:         accountNo,
		TransactionType:   "CREDIT",
		Category:          category,
		AmountCents:       amountCents,
		BalanceAfterCents: acct.BalanceCents,
		Currency:          acct.Currency,
		Reference:         "SBX-" + uuid.New().String()[:8],
		Description:       description,
		Status:            "COMPLETED",
		CreatedAt:         time.Now(),
	}
	s.transactions[accountNo] = append(s.transactions[accountNo], tx)
	return &tx, nil
}

// Debit removes funds from an account.
func (s *Store) Debit(accountNo string, amountCents int64, category, description string) (*Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	acct, ok := s.accounts[accountNo]
	if !ok {
		return nil, fmt.Errorf("account %s not found", accountNo)
	}
	if acct.BalanceCents < amountCents {
		return nil, fmt.Errorf("insufficient balance: have %d, need %d", acct.BalanceCents, amountCents)
	}

	acct.BalanceCents -= amountCents

	tx := Transaction{
		ID:                uuid.New().String(),
		AccountNo:         accountNo,
		TransactionType:   "DEBIT",
		Category:          category,
		AmountCents:       amountCents,
		BalanceAfterCents: acct.BalanceCents,
		Currency:          acct.Currency,
		Reference:         "SBX-" + uuid.New().String()[:8],
		Description:       description,
		Status:            "COMPLETED",
		CreatedAt:         time.Now(),
	}
	s.transactions[accountNo] = append(s.transactions[accountNo], tx)
	return &tx, nil
}

// GetTransactions returns transactions for an account.
func (s *Store) GetTransactions(accountNo string) []Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.transactions[accountNo]
}

// --- Payout operations ---

// CreatePendingPayout stores a payout awaiting OTP verification.
func (s *Store) CreatePendingPayout(p *PendingPayout) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingPayouts[p.Reference] = p
}

// CompletePayout finalizes a pending payout by debiting the account.
func (s *Store) CompletePayout(reference string) (*Transaction, error) {
	s.mu.Lock()
	pending, ok := s.pendingPayouts[reference]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("payout %s not found", reference)
	}
	delete(s.pendingPayouts, reference)
	s.mu.Unlock()

	return s.Debit(pending.AccountNo, pending.AmountCents, "WITHDRAWAL", "Payout via "+pending.Channel)
}

// --- PIN operations ---

// SetPIN sets the PIN hash for an account.
func (s *Store) SetPIN(accountNo, pinHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	acct, ok := s.accounts[accountNo]
	if !ok {
		return fmt.Errorf("account %s not found", accountNo)
	}
	acct.PINHash = pinHash
	return nil
}

// GetPINHash returns the PIN hash for an account.
func (s *Store) GetPINHash(accountNo string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	acct, ok := s.accounts[accountNo]
	if !ok {
		return "", fmt.Errorf("account %s not found", accountNo)
	}
	return acct.PINHash, nil
}

// --- Earning operations ---

// AddEarning adds an earning record to an account and credits the balance.
func (s *Store) AddEarning(accountNo string, amountCents int64, earningType, description string, earnedAt time.Time) (*Earning, error) {
	s.mu.Lock()
	acct, ok := s.accounts[accountNo]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("account %s not found", accountNo)
	}

	earning := Earning{
		ID:          uuid.New().String(),
		AccountNo:   accountNo,
		AmountCents: amountCents,
		EarningType: earningType,
		Currency:    acct.Currency,
		Description: description,
		EarnedAt:    earnedAt,
	}
	s.earnings[accountNo] = append(s.earnings[accountNo], earning)

	// Also credit the balance
	acct.BalanceCents += amountCents

	tx := Transaction{
		ID:                uuid.New().String(),
		AccountNo:         accountNo,
		TransactionType:   "CREDIT",
		Category:          "EARNING",
		AmountCents:       amountCents,
		BalanceAfterCents: acct.BalanceCents,
		Currency:          acct.Currency,
		Reference:         "EARN-" + earning.ID[:8],
		Description:       description,
		Status:            "COMPLETED",
		CreatedAt:         time.Now(),
	}
	s.transactions[accountNo] = append(s.transactions[accountNo], tx)
	s.mu.Unlock()

	return &earning, nil
}

// GetEarnings returns earnings for an account.
func (s *Store) GetEarnings(accountNo string) []Earning {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.earnings[accountNo]
}

// --- Loan operations ---

// ApplyLoan creates a new loan application.
func (s *Store) ApplyLoan(accountNo string, amountCents int64, tenureDays int) (*Loan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.accounts[accountNo]; !ok {
		return nil, fmt.Errorf("account %s not found", accountNo)
	}

	loan := Loan{
		ID:                   uuid.New().String(),
		AccountNo:            accountNo,
		AmountRequestedCents: amountCents,
		Currency:             "KES",
		Status:               "PENDING",
		TenureDays:           tenureDays,
		CreditScore:          650, // Default sandbox score
		CreatedAt:            time.Now(),
	}
	s.loans[accountNo] = append(s.loans[accountNo], loan)
	s.allLoans[loan.ID] = &loan
	return &loan, nil
}

// GetLoans returns loans for an account.
func (s *Store) GetLoans(accountNo string) []Loan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loans[accountNo]
}

// ApproveLoan approves a pending loan.
func (s *Store) ApproveLoan(loanID string) (*Loan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	loan, ok := s.allLoans[loanID]
	if !ok {
		return nil, fmt.Errorf("loan %s not found", loanID)
	}
	if loan.Status != "PENDING" {
		return nil, fmt.Errorf("loan %s is not pending (status: %s)", loanID, loan.Status)
	}
	loan.Status = "APPROVED"
	loan.AmountApprovedCents = loan.AmountRequestedCents
	return loan, nil
}

// DisburseLoan disburses an approved loan and credits the account.
func (s *Store) DisburseLoan(loanID string) (*Loan, error) {
	s.mu.Lock()
	loan, ok := s.allLoans[loanID]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("loan %s not found", loanID)
	}
	if loan.Status != "APPROVED" {
		s.mu.Unlock()
		return nil, fmt.Errorf("loan %s is not approved (status: %s)", loanID, loan.Status)
	}
	loan.Status = "DISBURSED"

	// Credit the account
	acct := s.accounts[loan.AccountNo]
	acct.BalanceCents += loan.AmountApprovedCents

	tx := Transaction{
		ID:                uuid.New().String(),
		AccountNo:         loan.AccountNo,
		TransactionType:   "CREDIT",
		Category:          "LOAN_DISBURSEMENT",
		AmountCents:       loan.AmountApprovedCents,
		BalanceAfterCents: acct.BalanceCents,
		Currency:          acct.Currency,
		Reference:         "LOAN-" + loanID[:8],
		Description:       fmt.Sprintf("Loan disbursement %s", loanID[:8]),
		Status:            "COMPLETED",
		CreatedAt:         time.Now(),
	}
	s.transactions[loan.AccountNo] = append(s.transactions[loan.AccountNo], tx)
	s.mu.Unlock()

	return loan, nil
}

// --- Insurance operations ---

// CreateInsurance creates a new insurance policy.
func (s *Store) CreateInsurance(accountNo, provider, policyType string, premiumCents int64, durationDays int) (*Insurance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.accounts[accountNo]; !ok {
		return nil, fmt.Errorf("account %s not found", accountNo)
	}

	now := time.Now()
	ins := Insurance{
		ID:           uuid.New().String(),
		AccountNo:    accountNo,
		Provider:     provider,
		PolicyType:   policyType,
		PremiumCents: premiumCents,
		Currency:     "KES",
		Status:       "ACTIVE",
		StartsAt:     now,
		ExpiresAt:    now.AddDate(0, 0, durationDays),
	}
	s.insurance[accountNo] = append(s.insurance[accountNo], ins)
	return &ins, nil
}

// GetInsurance returns insurance policies for an account.
func (s *Store) GetInsurance(accountNo string) []Insurance {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.insurance[accountNo]
}

// --- Admin operations ---

// Reset clears all sandbox data.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accounts = make(map[string]*Account)
	s.transactions = make(map[string][]Transaction)
	s.earnings = make(map[string][]Earning)
	s.loans = make(map[string][]Loan)
	s.insurance = make(map[string][]Insurance)
	s.pendingPayouts = make(map[string]*PendingPayout)
	s.allLoans = make(map[string]*Loan)
}

// Stats returns summary statistics.
func (s *Store) Stats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalBalance := int64(0)
	for _, acct := range s.accounts {
		totalBalance += acct.BalanceCents
	}

	totalTxns := 0
	for _, txs := range s.transactions {
		totalTxns += len(txs)
	}

	totalEarnings := 0
	for _, es := range s.earnings {
		totalEarnings += len(es)
	}

	totalLoans := 0
	for _, ls := range s.loans {
		totalLoans += len(ls)
	}

	totalInsurance := 0
	for _, is := range s.insurance {
		totalInsurance += len(is)
	}

	return map[string]interface{}{
		"accounts":         len(s.accounts),
		"total_balance":    totalBalance,
		"transactions":     totalTxns,
		"earnings":         totalEarnings,
		"loans":            totalLoans,
		"insurance":        totalInsurance,
		"pending_payouts":  len(s.pendingPayouts),
	}
}
