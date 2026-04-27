// Package repository defines interfaces for all data access operations.
// Services depend on these interfaces, never on concrete implementations.
// This enables unit testing with mocks.
package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
)

// --- Filter types ---

// CrewFilter specifies filtering criteria for crew member queries.
type CrewFilter struct {
	SaccoID   *uuid.UUID
	Role      string
	KYCStatus string
	IsActive  *bool
	Search    string // Matches first_name, last_name, crew_id
}

// AssignmentFilter specifies filtering criteria for assignment queries.
type AssignmentFilter struct {
	SaccoID      *uuid.UUID
	CrewMemberID *uuid.UUID
	VehicleID    *uuid.UUID
	Status       string
	ShiftDate    *time.Time
	DateFrom     *time.Time
	DateTo       *time.Time
}

// EarningFilter specifies filtering criteria for earning queries.
type EarningFilter struct {
	CrewMemberID *uuid.UUID
	AssignmentID *uuid.UUID
	EarningType  string
	DateFrom     *time.Time
	DateTo       *time.Time
	IsVerified   *bool
}

// TxFilter specifies filtering criteria for wallet transaction queries.
type TxFilter struct {
	Category        string
	TransactionType string
	Status          string
	DateFrom        *time.Time
	DateTo          *time.Time
}

// BulkError represents a single error in a bulk operation.
type BulkError struct {
	Index int    `json:"index"`
	Error string `json:"error"`
}

// --- Repository interfaces ---

// UserRepository handles user data access.
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByPhone(ctx context.Context, phone string) (*models.User, error)
	GetByCrewMemberID(ctx context.Context, crewMemberID uuid.UUID) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	List(ctx context.Context, page, perPage int) ([]models.User, int64, error)
	CountUsers(ctx context.Context) (total int64, active int64, err error)
}

// CrewRepository handles crew member data access.
type CrewRepository interface {
	Create(ctx context.Context, crew *models.CrewMember) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.CrewMember, error)
	GetByCrewID(ctx context.Context, crewID string) (*models.CrewMember, error)
	GetByNationalID(ctx context.Context, nationalID string) (*models.CrewMember, error)
	Update(ctx context.Context, crew *models.CrewMember) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter CrewFilter, page, perPage int) ([]models.CrewMember, int64, error)
	NextCrewID(ctx context.Context) (string, error)
	Count(ctx context.Context) (int64, error)
	BulkCreate(ctx context.Context, members []models.CrewMember) ([]BulkError, error)
}

// SACCORepository handles SACCO data access.
type SACCORepository interface {
	Create(ctx context.Context, sacco *models.SACCO) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.SACCO, error)
	Update(ctx context.Context, sacco *models.SACCO) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, page, perPage int, search string) ([]models.SACCO, int64, error)
}

// VehicleRepository handles vehicle data access.
type VehicleRepository interface {
	Create(ctx context.Context, vehicle *models.Vehicle) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Vehicle, error)
	Update(ctx context.Context, vehicle *models.Vehicle) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, saccoID *uuid.UUID, page, perPage int) ([]models.Vehicle, int64, error)
}

// RouteRepository handles route data access.
type RouteRepository interface {
	Create(ctx context.Context, route *models.Route) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Route, error)
	Update(ctx context.Context, route *models.Route) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, page, perPage int, search string) ([]models.Route, int64, error)
}

// AssignmentRepository handles assignment data access.
type AssignmentRepository interface {
	Create(ctx context.Context, assignment *models.Assignment) error
	BulkCreate(ctx context.Context, assignments []models.Assignment) (int, []BulkError, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Assignment, error)
	Update(ctx context.Context, assignment *models.Assignment) error
	List(ctx context.Context, filter AssignmentFilter, page, perPage int) ([]models.Assignment, int64, error)
	HasActiveAssignment(ctx context.Context, crewMemberID uuid.UUID, date time.Time) (bool, error)
}

// EarningRepository handles earning data access.
type EarningRepository interface {
	Create(ctx context.Context, earning *models.Earning) error
	BulkCreate(ctx context.Context, earnings []models.Earning) (int, []BulkError, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Earning, error)
	Update(ctx context.Context, earning *models.Earning) error
	List(ctx context.Context, filter EarningFilter, page, perPage int) ([]models.Earning, int64, error)
	GetDailySummary(ctx context.Context, crewMemberID uuid.UUID, date time.Time) (*models.DailyEarningsSummary, error)
	UpsertDailySummary(ctx context.Context, summary *models.DailyEarningsSummary) error
}

// WalletRepository handles wallet data access with atomic financial operations.
type WalletRepository interface {
	Create(ctx context.Context, wallet *models.Wallet) error
	GetByCrewMemberID(ctx context.Context, crewMemberID uuid.UUID) (*models.Wallet, error)
	GetWalletByID(ctx context.Context, id uuid.UUID) (*models.Wallet, error)

	// CreditWallet atomically updates balance and creates a transaction.
	// Returns ErrOptimisticLock if version mismatch.
	CreditWallet(ctx context.Context, walletID uuid.UUID, version int, amountCents int64,
		category models.TransactionCategory, idempotencyKey, reference, description string) (*models.WalletTransaction, error)

	// DebitWallet atomically checks balance, updates, and creates a transaction.
	// Returns ErrInsufficientBalance or ErrOptimisticLock.
	DebitWallet(ctx context.Context, walletID uuid.UUID, version int, amountCents int64,
		category models.TransactionCategory, idempotencyKey, reference, description string) (*models.WalletTransaction, error)

	GetTransactions(ctx context.Context, walletID uuid.UUID, filter TxFilter, page, perPage int) ([]models.WalletTransaction, int64, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*models.WalletTransaction, error)
	UpdateTransaction(ctx context.Context, tx *models.WalletTransaction) error
	List(ctx context.Context, page, perPage int) ([]models.Wallet, int64, error)
}

// PayrollRepository handles payroll data access.
type PayrollRepository interface {
	Create(ctx context.Context, run *models.PayrollRun) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.PayrollRun, error)
	Update(ctx context.Context, run *models.PayrollRun) error
	List(ctx context.Context, saccoID *uuid.UUID, page, perPage int) ([]models.PayrollRun, int64, error)
	CreateEntries(ctx context.Context, entries []models.PayrollEntry) error
	GetEntries(ctx context.Context, runID uuid.UUID) ([]models.PayrollEntry, error)
}

// MembershipRepository handles crew-SACCO membership data access.
type MembershipRepository interface {
	Create(ctx context.Context, m *models.CrewSACCOMembership) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.CrewSACCOMembership, error)
	Update(ctx context.Context, m *models.CrewSACCOMembership) error
	ListBySACCO(ctx context.Context, saccoID uuid.UUID, page, perPage int) ([]models.CrewSACCOMembership, int64, error)
	ListByCrewMember(ctx context.Context, crewMemberID uuid.UUID) ([]models.CrewSACCOMembership, error)
	GetActive(ctx context.Context, crewMemberID, saccoID uuid.UUID) (*models.CrewSACCOMembership, error)
}

// SACCOFloatFilter specifies filtering for SACCO float transaction queries.
type SACCOFloatFilter struct {
	TransactionType string
	DateFrom        *time.Time
	DateTo          *time.Time
}

// SACCOFloatRepository handles SACCO float data access with atomic operations.
type SACCOFloatRepository interface {
	GetOrCreate(ctx context.Context, saccoID uuid.UUID) (*models.SACCOFloat, error)
	CreditFloat(ctx context.Context, floatID uuid.UUID, version int, amountCents int64,
		idempotencyKey, reference string) (*models.SACCOFloatTransaction, error)
	DebitFloat(ctx context.Context, floatID uuid.UUID, version int, amountCents int64,
		idempotencyKey, reference string) (*models.SACCOFloatTransaction, error)
	GetTransactions(ctx context.Context, floatID uuid.UUID, filter SACCOFloatFilter, page, perPage int) ([]models.SACCOFloatTransaction, int64, error)
}

// DocumentFilter specifies filtering for document queries.
type DocumentFilter struct {
	CrewMemberID *uuid.UUID
	SaccoID      *uuid.UUID
	VehicleID    *uuid.UUID
	DocumentType string
}

// DocumentRepository handles document metadata data access.
type DocumentRepository interface {
	Create(ctx context.Context, doc *models.Document) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Document, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter DocumentFilter, page, perPage int) ([]models.Document, int64, error)
}

// NotificationFilter specifies filtering for notification queries.
type NotificationFilter struct {
	Channel string
	Status  string
}

// NotificationRepository handles notification data access.
type NotificationRepository interface {
	Create(ctx context.Context, n *models.Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error)
	Update(ctx context.Context, n *models.Notification) error
	ListByUser(ctx context.Context, userID uuid.UUID, filter NotificationFilter, page, perPage int) ([]models.Notification, int64, error)
	MarkRead(ctx context.Context, id uuid.UUID) error
	GetTemplate(ctx context.Context, eventName string) (*models.NotificationTemplate, error)
	CreateTemplate(ctx context.Context, t *models.NotificationTemplate) error
	UpdateTemplate(ctx context.Context, t *models.NotificationTemplate) error
	ListTemplates(ctx context.Context) ([]models.NotificationTemplate, error)
}

// NotificationPreferenceRepository handles user notification settings.
type NotificationPreferenceRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*models.NotificationPreference, error)
	Upsert(ctx context.Context, p *models.NotificationPreference) error
}

// AuditLogRepository handles audit log data access (append-only).
type AuditLogRepository interface {
	Create(ctx context.Context, log *models.AuditLog) error
	List(ctx context.Context, resource string, resourceID *uuid.UUID, page, perPage int) ([]models.AuditLog, int64, error)
}

// WebhookEventRepository handles inbound webhook event storage.
type WebhookEventRepository interface {
	Create(ctx context.Context, event *models.WebhookEvent) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.WebhookEvent, error)
	GetByExternalRef(ctx context.Context, source models.WebhookSource, ref string) (*models.WebhookEvent, error)
	MarkProcessed(ctx context.Context, id uuid.UUID) error
	ListUnprocessed(ctx context.Context, source models.WebhookSource, limit int) ([]models.WebhookEvent, error)
}

// StatutoryRateRepository handles statutory deduction rate data access.
type StatutoryRateRepository interface {
	GetActiveRates(ctx context.Context) ([]models.StatutoryRate, error)
	Create(ctx context.Context, rate *models.StatutoryRate) error
	Update(ctx context.Context, rate *models.StatutoryRate) error
}

// CreditScoreRepository handles credit score data access.
type CreditScoreRepository interface {
	GetByCrewMemberID(ctx context.Context, crewMemberID uuid.UUID) (*models.CreditScore, error)
	Upsert(ctx context.Context, score *models.CreditScore) error
}

// LoanApplicationFilter specifies filtering for loan queries.
type LoanApplicationFilter struct {
	CrewMemberID *uuid.UUID
	Status       string
	LenderID     *uuid.UUID
}

// LoanApplicationRepository handles loan application data access.
type LoanApplicationRepository interface {
	Create(ctx context.Context, loan *models.LoanApplication) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.LoanApplication, error)
	Update(ctx context.Context, loan *models.LoanApplication) error
	List(ctx context.Context, filter LoanApplicationFilter, page, perPage int) ([]models.LoanApplication, int64, error)
}

// InsurancePolicyFilter specifies filtering for insurance queries.
type InsurancePolicyFilter struct {
	CrewMemberID *uuid.UUID
	Status       string
}

// InsurancePolicyRepository handles insurance policy data access.
type InsurancePolicyRepository interface {
	Create(ctx context.Context, policy *models.InsurancePolicy) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.InsurancePolicy, error)
	Update(ctx context.Context, policy *models.InsurancePolicy) error
	List(ctx context.Context, filter InsurancePolicyFilter, page, perPage int) ([]models.InsurancePolicy, int64, error)
}

// WalletSnapshotRepository handles daily wallet balance snapshots.
type WalletSnapshotRepository interface {
	// Upsert creates or updates a daily snapshot for a wallet.
	// Uses ON CONFLICT to handle re-runs on the same day safely.
	Upsert(ctx context.Context, snapshot *models.WalletDailySnapshot) error

	// BatchUpsert efficiently upserts multiple snapshots in a single query.
	BatchUpsert(ctx context.Context, snapshots []models.WalletDailySnapshot) error

	// GetAvgBalance computes the average daily balance for a crew member over a date range.
	GetAvgBalance(ctx context.Context, crewMemberID uuid.UUID, from, to time.Time) (int64, error)

	// GetSnapshots retrieves daily snapshots for a crew member within a date range.
	GetSnapshots(ctx context.Context, crewMemberID uuid.UUID, from, to time.Time) ([]models.WalletDailySnapshot, error)

	// GetLatest retrieves the most recent snapshot for a crew member.
	GetLatest(ctx context.Context, crewMemberID uuid.UUID) (*models.WalletDailySnapshot, error)
}


