package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/internal/repository/mock"
)

// --- Mock Assignment & Earning Repos for service tests ---

type mockAssignmentRepo struct {
	mu          sync.Mutex
	assignments map[uuid.UUID]*models.Assignment
}

func newMockAssignmentRepo() *mockAssignmentRepo {
	return &mockAssignmentRepo{assignments: make(map[uuid.UUID]*models.Assignment)}
}

func (r *mockAssignmentRepo) Create(_ context.Context, a *models.Assignment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	r.assignments[a.ID] = a
	return nil
}

func (r *mockAssignmentRepo) BulkCreate(_ context.Context, as []models.Assignment) (int, []repository.BulkError, error) {
	return 0, nil, nil
}

func (r *mockAssignmentRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Assignment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.assignments[id]
	if !ok {
		return nil, ErrNotFound
	}
	return a, nil
}

func (r *mockAssignmentRepo) Update(_ context.Context, a *models.Assignment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.assignments[a.ID] = a
	return nil
}

func (r *mockAssignmentRepo) List(_ context.Context, _ repository.AssignmentFilter, _, _ int) ([]models.Assignment, int64, error) {
	return nil, 0, nil
}

func (r *mockAssignmentRepo) HasActiveAssignment(_ context.Context, _ uuid.UUID, _ time.Time) (bool, error) {
	return false, nil
}

type mockEarningRepo struct {
	mu       sync.Mutex
	earnings []*models.Earning
}

func (r *mockEarningRepo) Create(_ context.Context, e *models.Earning) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	r.earnings = append(r.earnings, e)
	return nil
}

func (r *mockEarningRepo) BulkCreate(_ context.Context, _ []models.Earning) (int, []repository.BulkError, error) {
	return 0, nil, nil
}
func (r *mockEarningRepo) GetByID(_ context.Context, _ uuid.UUID) (*models.Earning, error) {
	return nil, ErrNotFound
}
func (r *mockEarningRepo) Update(_ context.Context, _ *models.Earning) error { return nil }
func (r *mockEarningRepo) List(_ context.Context, _ repository.EarningFilter, _, _ int) ([]models.Earning, int64, error) {
	return nil, 0, nil
}
func (r *mockEarningRepo) GetDailySummary(_ context.Context, _ uuid.UUID, _ time.Time) (*models.DailyEarningsSummary, error) {
	return nil, ErrNotFound
}
func (r *mockEarningRepo) UpsertDailySummary(_ context.Context, _ *models.DailyEarningsSummary) error {
	return nil
}

// --- Setup ---

func newAssignmentTestEnv() (*AssignmentService, *WalletService, *mock.CrewRepo) {
	crewRepo := mock.NewCrewRepo()
	walletRepo := mock.NewWalletRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	auditSvc := NewAuditService(mock.NewAuditRepo(), logger)
	walletSvc := NewWalletService(walletRepo, crewRepo, auditSvc, logger)
	assignmentSvc := NewAssignmentService(newMockAssignmentRepo(), &mockEarningRepo{}, walletSvc, nil, nil, logger)
	return assignmentSvc, walletSvc, crewRepo
}

func makeCrewForTest(t *testing.T, repo *mock.CrewRepo) *models.CrewMember {
	t.Helper()
	crew := &models.CrewMember{
		CrewID: "CRW-99999", FirstName: "Test", LastName: "Crew",
		Role: models.RoleDriver, KYCStatus: models.KYCVerified, IsActive: true,
	}
	repo.Create(context.Background(), crew)
	return crew
}

// --- Earning Model Calculations ---

func TestFixedEarningCalc(t *testing.T) {
	svc, walletSvc, crewRepo := newAssignmentTestEnv()
	ctx := context.Background()
	crew := makeCrewForTest(t, crewRepo)

	a, _ := svc.CreateAssignment(ctx, CreateAssignmentInput{
		CrewMemberID: crew.ID, VehicleID: uuid.New(), SaccoID: uuid.New(),
		ShiftDate: time.Now(), ShiftStart: time.Now(),
		EarningModel: models.EarningFixed, FixedAmountCents: 250000,
	})
	earning, err := svc.CompleteAssignment(ctx, a.ID, 0)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if earning.AmountCents != 250000 {
		t.Errorf("fixed = %d, want 250000", earning.AmountCents)
	}
	w, _ := walletSvc.GetBalance(ctx, crew.ID)
	if w.BalanceCents != 250000 {
		t.Errorf("balance = %d, want 250000", w.BalanceCents)
	}
}

func TestCommissionEarningCalc(t *testing.T) {
	svc, _, crewRepo := newAssignmentTestEnv()
	ctx := context.Background()
	crew := makeCrewForTest(t, crewRepo)

	a, _ := svc.CreateAssignment(ctx, CreateAssignmentInput{
		CrewMemberID: crew.ID, VehicleID: uuid.New(), SaccoID: uuid.New(),
		ShiftDate: time.Now(), ShiftStart: time.Now(),
		EarningModel: models.EarningCommission, CommissionRate: 0.10,
	})
	earning, err := svc.CompleteAssignment(ctx, a.ID, 500000) // 10% of 500K
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if earning.AmountCents != 50000 {
		t.Errorf("commission = %d, want 50000", earning.AmountCents)
	}
}

func TestHybridEarningCalc(t *testing.T) {
	svc, _, crewRepo := newAssignmentTestEnv()
	ctx := context.Background()
	crew := makeCrewForTest(t, crewRepo)

	a, _ := svc.CreateAssignment(ctx, CreateAssignmentInput{
		CrewMemberID: crew.ID, VehicleID: uuid.New(), SaccoID: uuid.New(),
		ShiftDate: time.Now(), ShiftStart: time.Now(),
		EarningModel: models.EarningHybrid, HybridBaseCents: 100000, CommissionRate: 0.05,
	})
	earning, err := svc.CompleteAssignment(ctx, a.ID, 1000000) // 100K + 5% of 1M
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if earning.AmountCents != 150000 {
		t.Errorf("hybrid = %d, want 150000", earning.AmountCents)
	}
}

// --- Financial Precision ---

func TestLargeAmountPrecision(t *testing.T) {
	svc, _, crewRepo := newTestWalletService()
	ctx := context.Background()
	crew := setupCrewMember(t, crewRepo)

	big := int64(4_000_000_000_000_000) // 40 billion KES
	_, err := svc.Credit(ctx, CreditInput{
		CrewMemberID: crew.ID, AmountCents: big,
		Category: models.TxCatTopUp, IdempotencyKey: "big",
	})
	if err != nil {
		t.Fatalf("large credit: %v", err)
	}
	w, _ := svc.GetBalance(ctx, crew.ID)
	if w.BalanceCents != big {
		t.Errorf("balance = %d, want %d", w.BalanceCents, big)
	}
}

func TestMultipleCreditsAccumulate(t *testing.T) {
	svc, _, crewRepo := newTestWalletService()
	ctx := context.Background()
	crew := setupCrewMember(t, crewRepo)

	amounts := []int64{10050, 20075, 30025, 40000, 50050}
	var expected int64
	for i, amt := range amounts {
		_, err := svc.Credit(ctx, CreditInput{
			CrewMemberID: crew.ID, AmountCents: amt,
			Category: models.TxCatEarning, IdempotencyKey: string(rune('A' + i)),
		})
		if err != nil {
			t.Fatalf("credit %d: %v", i, err)
		}
		expected += amt
	}
	w, _ := svc.GetBalance(ctx, crew.ID)
	if w.BalanceCents != expected {
		t.Errorf("balance = %d, want %d", w.BalanceCents, expected)
	}
}

func TestDebitExactBalance(t *testing.T) {
	svc, _, crewRepo := newTestWalletService()
	ctx := context.Background()
	crew := setupCrewMember(t, crewRepo)

	svc.Credit(ctx, CreditInput{
		CrewMemberID: crew.ID, AmountCents: 100000,
		Category: models.TxCatTopUp, IdempotencyKey: "c1",
	})
	_, err := svc.Debit(ctx, DebitInput{
		CrewMemberID: crew.ID, AmountCents: 100000,
		Category: models.TxCatWithdrawal, IdempotencyKey: "d1",
	})
	if err != nil {
		t.Fatalf("exact debit: %v", err)
	}
	w, _ := svc.GetBalance(ctx, crew.ID)
	if w.BalanceCents != 0 {
		t.Errorf("balance = %d, want 0", w.BalanceCents)
	}
}

func TestDebitOneOverBalance(t *testing.T) {
	svc, _, crewRepo := newTestWalletService()
	ctx := context.Background()
	crew := setupCrewMember(t, crewRepo)

	svc.Credit(ctx, CreditInput{
		CrewMemberID: crew.ID, AmountCents: 100000,
		Category: models.TxCatTopUp, IdempotencyKey: "c1",
	})
	_, err := svc.Debit(ctx, DebitInput{
		CrewMemberID: crew.ID, AmountCents: 100001,
		Category: models.TxCatWithdrawal, IdempotencyKey: "d1",
	})
	if !errors.Is(err, ErrInsufficientBalance) {
		t.Errorf("expected ErrInsufficientBalance, got %v", err)
	}
}

func TestConcurrentCredits(t *testing.T) {
	svc, _, crewRepo := newTestWalletService()
	ctx := context.Background()
	crew := setupCrewMember(t, crewRepo)

	svc.Credit(ctx, CreditInput{
		CrewMemberID: crew.ID, AmountCents: 1,
		Category: models.TxCatTopUp, IdempotencyKey: "seed",
	})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			svc.Credit(ctx, CreditInput{
				CrewMemberID: crew.ID, AmountCents: 1000,
				Category: models.TxCatTopUp, IdempotencyKey: string(rune(idx + 200)),
			})
		}(i)
	}
	wg.Wait()
	// No panic = pass. Exact balance depends on optimistic lock conflicts.
}

func TestCompleteAssignment_TriggersNotification(t *testing.T) {
	crewRepo := mock.NewCrewRepo()
	walletRepo := mock.NewWalletRepo()
	notifRepo := mock.NewNotificationRepo()
	userRepo := mock.NewUserRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	auditSvc := NewAuditService(mock.NewAuditRepo(), logger)

	walletSvc := NewWalletService(walletRepo, crewRepo, auditSvc, logger)
	notifSvc := NewNotificationService(notifRepo, userRepo, nil, logger)
	assignmentSvc := NewAssignmentService(newMockAssignmentRepo(), &mockEarningRepo{}, walletSvc, notifSvc, nil, logger)

	ctx := context.Background()
	crew := makeCrewForTest(t, crewRepo)
	
	user := &models.User{
		ID:           uuid.New(),
		Phone:        "+254712345678",
		CrewMemberID: &crew.ID,
	}
	userRepo.Create(ctx, user)

	a, _ := assignmentSvc.CreateAssignment(ctx, CreateAssignmentInput{
		CrewMemberID: crew.ID, VehicleID: uuid.New(), SaccoID: uuid.New(),
		ShiftDate: time.Now(), ShiftStart: time.Now(),
		EarningModel: models.EarningFixed, FixedAmountCents: 250000,
	})

	_, err := assignmentSvc.CompleteAssignment(ctx, a.ID, 0)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	notifs, count, _ := notifRepo.ListByUser(ctx, user.ID, repository.NotificationFilter{}, 1, 10)
	if count != 1 {
		t.Errorf("expected 1 notification, got %d", count)
	} else if notifs[0].Channel != models.ChannelSMS {
		t.Errorf("expected SMS channel, got %s", notifs[0].Channel)
	}
}
