// Package mock provides in-memory implementations of repository interfaces for testing.
package mock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/pkg/errs"
)

// --- UserRepo Mock ---

type UserRepo struct {
	mu    sync.RWMutex
	users map[uuid.UUID]*models.User
}

func NewUserRepo() *UserRepo {
	return &UserRepo{users: make(map[uuid.UUID]*models.User)}
}

func (r *UserRepo) Create(_ context.Context, user *models.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	for _, u := range r.users {
		if u.Phone == user.Phone {
			return fmt.Errorf("duplicate phone")
		}
	}
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	r.users[user.ID] = user
	return nil
}

func (r *UserRepo) GetByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.users[id]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return u, nil
}

func (r *UserRepo) GetByPhone(_ context.Context, phone string) (*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.users {
		if u.Phone == phone {
			return u, nil
		}
	}
	return nil, errs.ErrNotFound
}

func (r *UserRepo) GetByCrewMemberID(_ context.Context, crewMemberID uuid.UUID) (*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.users {
		if u.CrewMemberID != nil && *u.CrewMemberID == crewMemberID {
			return u, nil
		}
	}
	return nil, errs.ErrNotFound
}

func (r *UserRepo) Update(_ context.Context, user *models.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	user.UpdatedAt = time.Now()
	r.users[user.ID] = user
	return nil
}

func (r *UserRepo) List(_ context.Context, page, perPage int) ([]models.User, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []models.User
	for _, u := range r.users {
		all = append(all, *u)
	}
	total := int64(len(all))
	start := (page - 1) * perPage
	if start >= len(all) {
		return nil, total, nil
	}
	end := start + perPage
	if end > len(all) {
		end = len(all)
	}
	return all[start:end], total, nil
}

// --- CrewRepo Mock ---

type CrewRepo struct {
	mu      sync.RWMutex
	members map[uuid.UUID]*models.CrewMember
	seq     int64
}

func NewCrewRepo() *CrewRepo {
	return &CrewRepo{members: make(map[uuid.UUID]*models.CrewMember)}
}

func (r *CrewRepo) Create(_ context.Context, crew *models.CrewMember) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if crew.ID == uuid.Nil {
		crew.ID = uuid.New()
	}
	now := time.Now()
	crew.CreatedAt = now
	crew.UpdatedAt = now
	r.members[crew.ID] = crew
	return nil
}

func (r *CrewRepo) GetByID(_ context.Context, id uuid.UUID) (*models.CrewMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.members[id]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return m, nil
}

func (r *CrewRepo) GetByCrewID(_ context.Context, crewID string) (*models.CrewMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, m := range r.members {
		if m.CrewID == crewID {
			return m, nil
		}
	}
	return nil, errs.ErrNotFound
}

func (r *CrewRepo) Update(_ context.Context, crew *models.CrewMember) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	crew.UpdatedAt = time.Now()
	r.members[crew.ID] = crew
	return nil
}

func (r *CrewRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.members, id)
	return nil
}

func (r *CrewRepo) List(_ context.Context, _ repository.CrewFilter, page, perPage int) ([]models.CrewMember, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []models.CrewMember
	for _, m := range r.members {
		all = append(all, *m)
	}
	total := int64(len(all))
	start := (page - 1) * perPage
	if start >= len(all) {
		return nil, total, nil
	}
	end := start + perPage
	if end > len(all) {
		end = len(all)
	}
	return all[start:end], total, nil
}

func (r *CrewRepo) NextCrewID(_ context.Context) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	return fmt.Sprintf("CRW-%05d", r.seq), nil
}

// --- WalletRepo Mock ---

type WalletRepo struct {
	mu      sync.RWMutex
	wallets map[uuid.UUID]*models.Wallet
	txs     map[uuid.UUID]*models.WalletTransaction
}

func NewWalletRepo() *WalletRepo {
	return &WalletRepo{
		wallets: make(map[uuid.UUID]*models.Wallet),
		txs:     make(map[uuid.UUID]*models.WalletTransaction),
	}
}

func (r *WalletRepo) Create(_ context.Context, wallet *models.Wallet) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if wallet.ID == uuid.Nil {
		wallet.ID = uuid.New()
	}
	r.wallets[wallet.ID] = wallet
	return nil
}

func (r *WalletRepo) GetByCrewMemberID(_ context.Context, crewMemberID uuid.UUID) (*models.Wallet, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, w := range r.wallets {
		if w.CrewMemberID == crewMemberID {
			copy := *w // Return a copy to prevent data races
			return &copy, nil
		}
	}
	return nil, errs.ErrNotFound
}

func (r *WalletRepo) GetWalletByID(_ context.Context, id uuid.UUID) (*models.Wallet, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	w, ok := r.wallets[id]
	if !ok {
		return nil, errs.ErrNotFound
	}
	copy := *w
	return &copy, nil
}

func (r *WalletRepo) CreditWallet(_ context.Context, walletID uuid.UUID, version int, amountCents int64,
	category models.TransactionCategory, idempotencyKey, reference, description string) (*models.WalletTransaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check idempotency
	if idempotencyKey != "" {
		for _, tx := range r.txs {
			if tx.IdempotencyKey == idempotencyKey {
				return tx, nil
			}
		}
	}

	w, ok := r.wallets[walletID]
	if !ok {
		return nil, errs.ErrNotFound
	}
	if w.Version != version {
		return nil, errs.ErrOptimisticLock
	}

	w.BalanceCents += amountCents
	w.TotalCreditedCents += amountCents
	w.Version++

	tx := &models.WalletTransaction{
		ID:                uuid.New(),
		WalletID:          walletID,
		IdempotencyKey:    idempotencyKey,
		TransactionType:   models.TxCredit,
		Category:          category,
		AmountCents:       amountCents,
		BalanceAfterCents: w.BalanceCents,
		Currency:          w.Currency,
		Reference:         reference,
		Description:       description,
		Status:            models.TxCompleted,
	}
	r.txs[tx.ID] = tx
	return tx, nil
}

func (r *WalletRepo) DebitWallet(_ context.Context, walletID uuid.UUID, version int, amountCents int64,
	category models.TransactionCategory, idempotencyKey, reference, description string) (*models.WalletTransaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if idempotencyKey != "" {
		for _, tx := range r.txs {
			if tx.IdempotencyKey == idempotencyKey {
				return tx, nil
			}
		}
	}

	w, ok := r.wallets[walletID]
	if !ok {
		return nil, errs.ErrNotFound
	}
	if w.Version != version {
		return nil, errs.ErrOptimisticLock
	}
	if w.BalanceCents < amountCents {
		return nil, errs.ErrInsufficientBalance
	}

	w.BalanceCents -= amountCents
	w.TotalDebitedCents += amountCents
	w.Version++

	tx := &models.WalletTransaction{
		ID:                uuid.New(),
		WalletID:          walletID,
		IdempotencyKey:    idempotencyKey,
		TransactionType:   models.TxDebit,
		Category:          category,
		AmountCents:       amountCents,
		BalanceAfterCents: w.BalanceCents,
		Currency:          w.Currency,
		Reference:         reference,
		Description:       description,
		Status:            models.TxCompleted,
	}
	r.txs[tx.ID] = tx
	return tx, nil
}

func (r *WalletRepo) GetTransactions(_ context.Context, walletID uuid.UUID, _ repository.TxFilter, _, _ int) ([]models.WalletTransaction, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []models.WalletTransaction
	for _, tx := range r.txs {
		if tx.WalletID == walletID {
			result = append(result, *tx)
		}
	}
	return result, int64(len(result)), nil
}

func (r *WalletRepo) GetByIdempotencyKey(_ context.Context, key string) (*models.WalletTransaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, tx := range r.txs {
		if tx.IdempotencyKey == key {
			return tx, nil
		}
	}
	return nil, errs.ErrNotFound
}

func (r *WalletRepo) UpdateTransaction(_ context.Context, tx *models.WalletTransaction) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, existing := range r.txs {
		if existing.ID == tx.ID {
			r.txs[i] = tx
			return nil
		}
	}
	return errs.ErrNotFound
}

// --- SACCORepo Mock ---

type SACCORepo struct {
	mu     sync.RWMutex
	saccos map[uuid.UUID]*models.SACCO
}

func NewSACCORepo() *SACCORepo {
	return &SACCORepo{saccos: make(map[uuid.UUID]*models.SACCO)}
}

func (r *SACCORepo) Create(_ context.Context, sacco *models.SACCO) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if sacco.ID == uuid.Nil {
		sacco.ID = uuid.New()
	}
	sacco.CreatedAt = time.Now()
	sacco.UpdatedAt = time.Now()
	r.saccos[sacco.ID] = sacco
	return nil
}

func (r *SACCORepo) GetByID(_ context.Context, id uuid.UUID) (*models.SACCO, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.saccos[id]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return s, nil
}

func (r *SACCORepo) Update(_ context.Context, sacco *models.SACCO) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	sacco.UpdatedAt = time.Now()
	r.saccos[sacco.ID] = sacco
	return nil
}

func (r *SACCORepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.saccos, id)
	return nil
}

func (r *SACCORepo) List(_ context.Context, page, perPage int, search string) ([]models.SACCO, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []models.SACCO
	for _, s := range r.saccos {
		all = append(all, *s)
	}
	return all, int64(len(all)), nil
}

// --- MembershipRepo Mock ---

type MembershipRepo struct {
	mu      sync.RWMutex
	members map[uuid.UUID]*models.CrewSACCOMembership
}

func NewMembershipRepo() *MembershipRepo {
	return &MembershipRepo{members: make(map[uuid.UUID]*models.CrewSACCOMembership)}
}

func (r *MembershipRepo) Create(_ context.Context, m *models.CrewSACCOMembership) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	m.JoinedAt = time.Now()
	r.members[m.ID] = m
	return nil
}

func (r *MembershipRepo) GetByID(_ context.Context, id uuid.UUID) (*models.CrewSACCOMembership, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.members[id]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return m, nil
}

func (r *MembershipRepo) Update(_ context.Context, m *models.CrewSACCOMembership) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.members[m.ID] = m
	return nil
}

func (r *MembershipRepo) ListBySACCO(_ context.Context, saccoID uuid.UUID, page, perPage int) ([]models.CrewSACCOMembership, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []models.CrewSACCOMembership
	for _, m := range r.members {
		if m.SaccoID == saccoID && m.IsActive {
			all = append(all, *m)
		}
	}
	return all, int64(len(all)), nil
}

func (r *MembershipRepo) ListByCrewMember(_ context.Context, crewMemberID uuid.UUID) ([]models.CrewSACCOMembership, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []models.CrewSACCOMembership
	for _, m := range r.members {
		if m.CrewMemberID == crewMemberID && m.IsActive {
			all = append(all, *m)
		}
	}
	return all, nil
}

func (r *MembershipRepo) GetActive(_ context.Context, crewMemberID, saccoID uuid.UUID) (*models.CrewSACCOMembership, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, m := range r.members {
		if m.CrewMemberID == crewMemberID && m.SaccoID == saccoID && m.IsActive {
			return m, nil
		}
	}
	return nil, errs.ErrNotFound
}

// --- SACCOFloatRepo Mock ---

type SACCOFloatRepo struct {
	mu     sync.RWMutex
	floats map[uuid.UUID]*models.SACCOFloat
	txs    []models.SACCOFloatTransaction
}

func NewSACCOFloatRepo() *SACCOFloatRepo {
	return &SACCOFloatRepo{
		floats: make(map[uuid.UUID]*models.SACCOFloat),
	}
}

func (r *SACCOFloatRepo) GetOrCreate(_ context.Context, saccoID uuid.UUID) (*models.SACCOFloat, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, f := range r.floats {
		if f.SaccoID == saccoID {
			return f, nil
		}
	}
	f := &models.SACCOFloat{
		ID:       uuid.New(),
		SaccoID:  saccoID,
		Currency: "KES",
	}
	r.floats[f.ID] = f
	return f, nil
}

func (r *SACCOFloatRepo) CreditFloat(_ context.Context, floatID uuid.UUID, version int, amountCents int64, idempotencyKey, reference string) (*models.SACCOFloatTransaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, ok := r.floats[floatID]
	if !ok {
		return nil, errs.ErrNotFound
	}
	if f.Version != version {
		return nil, errs.ErrOptimisticLock
	}
	f.BalanceCents += amountCents
	f.Version++
	tx := models.SACCOFloatTransaction{
		ID:                uuid.New(),
		SACCOFloatID:      floatID,
		IdempotencyKey:    idempotencyKey,
		TransactionType:   "CREDIT",
		AmountCents:       amountCents,
		BalanceAfterCents: f.BalanceCents,
		Status:            models.TxCompleted,
	}
	r.txs = append(r.txs, tx)
	return &tx, nil
}

func (r *SACCOFloatRepo) DebitFloat(_ context.Context, floatID uuid.UUID, version int, amountCents int64, idempotencyKey, reference string) (*models.SACCOFloatTransaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, ok := r.floats[floatID]
	if !ok {
		return nil, errs.ErrNotFound
	}
	if f.Version != version {
		return nil, errs.ErrOptimisticLock
	}
	if f.BalanceCents < amountCents {
		return nil, errs.ErrInsufficientBalance
	}
	f.BalanceCents -= amountCents
	f.Version++
	tx := models.SACCOFloatTransaction{
		ID:                uuid.New(),
		SACCOFloatID:      floatID,
		IdempotencyKey:    idempotencyKey,
		TransactionType:   "DEBIT",
		AmountCents:       amountCents,
		BalanceAfterCents: f.BalanceCents,
		Status:            models.TxCompleted,
	}
	r.txs = append(r.txs, tx)
	return &tx, nil
}

func (r *SACCOFloatRepo) GetTransactions(_ context.Context, floatID uuid.UUID, filter repository.SACCOFloatFilter, page, perPage int) ([]models.SACCOFloatTransaction, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []models.SACCOFloatTransaction
	for _, tx := range r.txs {
		if tx.SACCOFloatID == floatID {
			all = append(all, tx)
		}
	}
	return all, int64(len(all)), nil
}

// --- AuditLogRepo Mock ---

type AuditRepo struct {
	mu   sync.RWMutex
	Logs []models.AuditLog
}

func NewAuditRepo() *AuditRepo {
	return &AuditRepo{
		Logs: make([]models.AuditLog, 0),
	}
}

func (r *AuditRepo) Create(_ context.Context, log *models.AuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Logs = append(r.Logs, *log)
	return nil
}

func (r *AuditRepo) List(_ context.Context, resource string, resourceID *uuid.UUID, page, perPage int) ([]models.AuditLog, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var res []models.AuditLog
	for _, l := range r.Logs {
		if resource != "" && l.Resource != resource {
			continue
		}
		if resourceID != nil && l.ResourceID != nil && *l.ResourceID != *resourceID {
			continue
		}
		res = append(res, l)
	}
	return res, int64(len(res)), nil
}

