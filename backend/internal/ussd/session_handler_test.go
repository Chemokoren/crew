package ussd

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/pkg/errs"
	"log/slog"
)

// === Mock Repositories ===

type mockUserRepo struct {
	users map[string]*models.User
}

func (r *mockUserRepo) Create(_ context.Context, u *models.User) error {
	if u.ID == uuid.Nil { u.ID = uuid.New() }
	r.users[u.Phone] = u
	return nil
}
func (r *mockUserRepo) GetByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	for _, u := range r.users { if u.ID == id { return u, nil } }
	return nil, errs.ErrNotFound
}
func (r *mockUserRepo) GetByPhone(_ context.Context, phone string) (*models.User, error) {
	if u, ok := r.users[phone]; ok { return u, nil }
	return nil, errs.ErrNotFound
}
func (r *mockUserRepo) GetByCrewMemberID(_ context.Context, id uuid.UUID) (*models.User, error) {
	for _, u := range r.users { if u.CrewMemberID != nil && *u.CrewMemberID == id { return u, nil } }
	return nil, errs.ErrNotFound
}
func (r *mockUserRepo) Update(_ context.Context, u *models.User) error { r.users[u.Phone] = u; return nil }
func (r *mockUserRepo) List(_ context.Context, _, _ int, _ string) ([]models.User, int64, error) { return nil, 0, nil }
func (r *mockUserRepo) CountUsers(_ context.Context) (int64, int64, error) { return 0, 0, nil }

type mockCrewRepo struct {
	members map[uuid.UUID]*models.CrewMember
	byNatID map[string]*models.CrewMember
	nextID  int
}

func (r *mockCrewRepo) Create(_ context.Context, c *models.CrewMember) error {
	if c.ID == uuid.Nil { c.ID = uuid.New() }
	r.members[c.ID] = c
	r.byNatID[c.NationalID] = c
	return nil
}
func (r *mockCrewRepo) GetByID(_ context.Context, id uuid.UUID) (*models.CrewMember, error) {
	if c, ok := r.members[id]; ok { return c, nil }
	return nil, errs.ErrNotFound
}
func (r *mockCrewRepo) GetByCrewID(_ context.Context, _ string) (*models.CrewMember, error) { return nil, errs.ErrNotFound }
func (r *mockCrewRepo) GetByNationalID(_ context.Context, natID string) (*models.CrewMember, error) {
	if c, ok := r.byNatID[natID]; ok { return c, nil }
	return nil, errs.ErrNotFound
}
func (r *mockCrewRepo) Update(_ context.Context, c *models.CrewMember) error { r.members[c.ID] = c; return nil }
func (r *mockCrewRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (r *mockCrewRepo) List(_ context.Context, _ repository.CrewFilter, _, _ int) ([]models.CrewMember, int64, error) { return nil, 0, nil }
func (r *mockCrewRepo) NextCrewID(_ context.Context) (string, error) { r.nextID++; return "CRW-"+string(rune('0'+r.nextID)), nil }
func (r *mockCrewRepo) Count(_ context.Context) (int64, error) { return int64(len(r.members)), nil }
func (r *mockCrewRepo) BulkCreate(_ context.Context, _ []models.CrewMember) ([]repository.BulkError, error) { return nil, nil }

type mockAssignmentRepo struct {
	assignments []models.Assignment
}

func (r *mockAssignmentRepo) Create(_ context.Context, a *models.Assignment) error { r.assignments = append(r.assignments, *a); return nil }
func (r *mockAssignmentRepo) BulkCreate(_ context.Context, _ []models.Assignment) (int, []repository.BulkError, error) { return 0, nil, nil }
func (r *mockAssignmentRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Assignment, error) {
	for i := range r.assignments { if r.assignments[i].ID == id { return &r.assignments[i], nil } }
	return nil, errs.ErrNotFound
}
func (r *mockAssignmentRepo) Update(_ context.Context, a *models.Assignment) error {
	for i := range r.assignments { if r.assignments[i].ID == a.ID { r.assignments[i] = *a; return nil } }
	return nil
}
func (r *mockAssignmentRepo) List(_ context.Context, f repository.AssignmentFilter, _, perPage int) ([]models.Assignment, int64, error) {
	var result []models.Assignment
	for _, a := range r.assignments {
		if f.CrewMemberID != nil && a.CrewMemberID != *f.CrewMemberID { continue }
		if f.DateFrom != nil && a.ShiftDate.Before(*f.DateFrom) { continue }
		if f.DateTo != nil && !a.ShiftDate.Before(*f.DateTo) { continue }
		result = append(result, a)
		if len(result) >= perPage { break }
	}
	return result, int64(len(result)), nil
}
func (r *mockAssignmentRepo) HasActiveAssignment(_ context.Context, _ uuid.UUID, _ time.Time) (bool, error) { return false, nil }

type mockEarningRepo struct {
	earnings []models.Earning
}

func (r *mockEarningRepo) Create(_ context.Context, _ *models.Earning) error { return nil }
func (r *mockEarningRepo) BulkCreate(_ context.Context, _ []models.Earning) (int, []repository.BulkError, error) { return 0, nil, nil }
func (r *mockEarningRepo) GetByID(_ context.Context, _ uuid.UUID) (*models.Earning, error) { return nil, errs.ErrNotFound }
func (r *mockEarningRepo) Update(_ context.Context, _ *models.Earning) error { return nil }
func (r *mockEarningRepo) List(_ context.Context, f repository.EarningFilter, _, _ int) ([]models.Earning, int64, error) {
	var result []models.Earning
	for _, e := range r.earnings {
		if f.CrewMemberID != nil && e.CrewMemberID != *f.CrewMemberID { continue }
		if f.DateFrom != nil && e.EarnedAt.Before(*f.DateFrom) { continue }
		if f.DateTo != nil && !e.EarnedAt.Before(*f.DateTo) { continue }
		result = append(result, e)
	}
	return result, int64(len(result)), nil
}
func (r *mockEarningRepo) GetDailySummary(_ context.Context, _ uuid.UUID, _ time.Time) (*models.DailyEarningsSummary, error) { return nil, nil }
func (r *mockEarningRepo) UpsertDailySummary(_ context.Context, _ *models.DailyEarningsSummary) error { return nil }

type mockOrgRepo struct {
	orgs map[uuid.UUID]*models.Organization
}

func (r *mockOrgRepo) Create(_ context.Context, o *models.SACCO) error { r.orgs[o.ID] = o; return nil }
func (r *mockOrgRepo) GetByID(_ context.Context, id uuid.UUID) (*models.SACCO, error) {
	if o, ok := r.orgs[id]; ok { return o, nil }
	return nil, errs.ErrNotFound
}
func (r *mockOrgRepo) Update(_ context.Context, _ *models.SACCO) error { return nil }
func (r *mockOrgRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (r *mockOrgRepo) List(_ context.Context, _, _ int, _ string) ([]models.SACCO, int64, error) { return nil, 0, nil }

type mockJobTypeRepo struct{}

func (r *mockJobTypeRepo) Create(_ context.Context, _ *models.TenantJobType) error { return nil }
func (r *mockJobTypeRepo) GetByID(_ context.Context, _ uuid.UUID) (*models.TenantJobType, error) { return nil, nil }
func (r *mockJobTypeRepo) Update(_ context.Context, _ *models.TenantJobType) error { return nil }
func (r *mockJobTypeRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (r *mockJobTypeRepo) ListByOrganization(_ context.Context, _ uuid.UUID) ([]models.TenantJobType, error) { return nil, nil }
func (r *mockJobTypeRepo) GetByCode(_ context.Context, _ uuid.UUID, _ string) (*models.TenantJobType, error) { return nil, nil }

type mockPayScheduleRepo struct {
	schedules map[uuid.UUID][]models.PaySchedule
}

func (r *mockPayScheduleRepo) Create(_ context.Context, _ *models.PaySchedule) error { return nil }
func (r *mockPayScheduleRepo) GetByID(_ context.Context, _ uuid.UUID) (*models.PaySchedule, error) { return nil, nil }
func (r *mockPayScheduleRepo) Update(_ context.Context, _ *models.PaySchedule) error { return nil }
func (r *mockPayScheduleRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (r *mockPayScheduleRepo) ListByOrganization(_ context.Context, orgID uuid.UUID) ([]models.PaySchedule, error) {
	return r.schedules[orgID], nil
}
func (r *mockPayScheduleRepo) GetDefault(_ context.Context, _ uuid.UUID) (*models.PaySchedule, error) { return nil, nil }

type mockMembershipRepo struct{}

func (r *mockMembershipRepo) Create(_ context.Context, _ *models.CrewSACCOMembership) error { return nil }
func (r *mockMembershipRepo) GetByID(_ context.Context, _ uuid.UUID) (*models.CrewSACCOMembership, error) { return nil, nil }
func (r *mockMembershipRepo) Update(_ context.Context, _ *models.CrewSACCOMembership) error { return nil }
func (r *mockMembershipRepo) ListByCrewMember(_ context.Context, _ uuid.UUID) ([]models.CrewSACCOMembership, error) { return nil, nil }
func (r *mockMembershipRepo) ListByOrganization(_ context.Context, _ uuid.UUID, _, _ int) ([]models.CrewSACCOMembership, int64, error) { return nil, 0, nil }
func (r *mockMembershipRepo) GetActive(_ context.Context, _, _ uuid.UUID) (*models.CrewSACCOMembership, error) { return nil, nil }

type mockWalletRepo struct {
	wallets map[uuid.UUID]*models.Wallet
}

func (r *mockWalletRepo) Create(_ context.Context, _ *models.Wallet) error { return nil }
func (r *mockWalletRepo) GetByCrewMemberID(_ context.Context, id uuid.UUID) (*models.Wallet, error) {
	if w, ok := r.wallets[id]; ok { return w, nil }
	return nil, errs.ErrNotFound
}
func (r *mockWalletRepo) GetWalletByID(_ context.Context, _ uuid.UUID) (*models.Wallet, error) { return nil, nil }
func (r *mockWalletRepo) CreditWallet(_ context.Context, _ uuid.UUID, _ int, _ int64, _ models.TransactionCategory, _, _, _ string) (*models.WalletTransaction, error) { return nil, nil }
func (r *mockWalletRepo) DebitWallet(_ context.Context, _ uuid.UUID, _ int, _ int64, _ models.TransactionCategory, _, _, _ string) (*models.WalletTransaction, error) { return nil, nil }
func (r *mockWalletRepo) GetTransactions(_ context.Context, _ uuid.UUID, _ repository.TxFilter, _, _ int) ([]models.WalletTransaction, int64, error) { return nil, 0, nil }
func (r *mockWalletRepo) GetByIdempotencyKey(_ context.Context, _ string) (*models.WalletTransaction, error) { return nil, nil }
func (r *mockWalletRepo) UpdateTransaction(_ context.Context, _ *models.WalletTransaction) error { return nil }
func (r *mockWalletRepo) List(_ context.Context, _, _ int) ([]models.Wallet, int64, error) { return nil, 0, nil }

// === Test Setup ===

func newTestHandler() (*SessionHandler, *testData) {
	crewID := uuid.New()
	orgID := uuid.New()
	
	crew := &models.CrewMember{
		ID:         crewID,
		CrewID:     "CRW-001",
		NationalID: "12345678",
		FirstName:  "Jane",
		LastName:   "Doe",
		Role:       models.RoleDriver,
		IsActive:   true,
	}

	org := &models.Organization{
		ID:              orgID,
		Name:            "CountyLink SACCO",
		IndustryType:    models.IndustryTransport,
		DefaultLanguage: "sw",
	}

	user := &models.User{
		ID:                uuid.New(),
		Phone:             "254712345678",
		CrewMemberID:      &crewID,
		OrganizationID:    &orgID,
		PreferredLanguage: "en",
		IsActive:          true,
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	assignment := models.Assignment{
		ID:             uuid.New(),
		CrewMemberID:   crewID,
		OrganizationID: orgID,
		ShiftDate:      today,
		ShiftStart:     today.Add(8 * time.Hour),
		Status:         models.AssignmentScheduled,
		WorkType:       models.WorkTypeShift,
		WorkSite:       "CBD Route 46",
	}

	earning := models.Earning{
		ID:           uuid.New(),
		CrewMemberID: crewID,
		AmountCents:  150000, // KES 1,500.00
		EarnedAt:     today,
	}

	friday := 5
	paySchedule := models.PaySchedule{
		ID:        uuid.New(),
		Name:      "Weekly Pay",
		Frequency: models.PayWeekly,
		PayDay:    &friday,
		IsDefault: true,
	}

	wallet := &models.Wallet{
		ID:                 uuid.New(),
		CrewMemberID:       crewID,
		BalanceCents:       500000,  // KES 5,000.00
		TotalCreditedCents: 1000000, // KES 10,000.00
		TotalDebitedCents:  500000,  // KES 5,000.00
	}

	userRepo := &mockUserRepo{users: map[string]*models.User{user.Phone: user}}
	crewRepo := &mockCrewRepo{members: map[uuid.UUID]*models.CrewMember{crewID: crew}, byNatID: map[string]*models.CrewMember{crew.NationalID: crew}}
	assignmentRepo := &mockAssignmentRepo{assignments: []models.Assignment{assignment}}
	earningRepo := &mockEarningRepo{earnings: []models.Earning{earning}}
	orgRepo := &mockOrgRepo{orgs: map[uuid.UUID]*models.Organization{orgID: org}}
	jobTypeRepo := &mockJobTypeRepo{}
	scheduleRepo := &mockPayScheduleRepo{schedules: map[uuid.UUID][]models.PaySchedule{orgID: {paySchedule}}}
	membershipRepo := &mockMembershipRepo{}
	walletRepo := &mockWalletRepo{wallets: map[uuid.UUID]*models.Wallet{crewID: wallet}}

	h := NewSessionHandler(
		userRepo, crewRepo, assignmentRepo, earningRepo,
		orgRepo, jobTypeRepo, scheduleRepo, membershipRepo, walletRepo,
		slog.Default(),
	)

	return h, &testData{user: user, crew: crew, org: org, assignment: assignment}
}

type testData struct {
	user       *models.User
	crew       *models.CrewMember
	org        *models.Organization
	assignment models.Assignment
}

// === Tests ===

func TestMainMenu_RegisteredUser(t *testing.T) {
	h, _ := newTestHandler()
	resp := h.HandleSession(context.Background(), USSDRequest{
		SessionID: "sess-1", PhoneNumber: "+254712345678", Text: "",
	})
	// User pref is "en", so English menu
	if resp[:4] != "CON " {
		t.Fatalf("expected CON, got: %s", resp)
	}
	if !contains(resp, "Welcome Jane") {
		t.Errorf("expected personalized greeting, got: %s", resp)
	}
	if !contains(resp, "Shift") {
		t.Errorf("expected industry-specific label 'Shift', got: %s", resp)
	}
}

func TestMainMenu_UnregisteredUser(t *testing.T) {
	h, _ := newTestHandler()
	resp := h.HandleSession(context.Background(), USSDRequest{
		SessionID: "sess-2", PhoneNumber: "+254799999999", Text: "",
	})
	if resp[:4] != "CON " {
		t.Fatalf("expected CON, got: %s", resp)
	}
	// Swahili default for unregistered
	if !contains(resp, "Jisajili") {
		t.Errorf("expected registration option, got: %s", resp)
	}
}

func TestAssignments_Today(t *testing.T) {
	h, _ := newTestHandler()
	resp := h.HandleSession(context.Background(), USSDRequest{
		SessionID: "sess-3", PhoneNumber: "+254712345678", Text: "1*1",
	})
	if resp[:4] != "END " {
		t.Fatalf("expected END, got: %s", resp)
	}
	if !contains(resp, "CBD Route 46") {
		t.Errorf("expected work site in response, got: %s", resp)
	}
}

func TestAssignments_NoAssignments(t *testing.T) {
	h, _ := newTestHandler()
	// This month — assignment only exists today
	resp := h.HandleSession(context.Background(), USSDRequest{
		SessionID: "sess-4", PhoneNumber: "+254712345678", Text: "1*3",
	})
	if resp[:4] != "END " {
		t.Fatalf("expected END, got: %s", resp)
	}
}

func TestCheckIn_Flow(t *testing.T) {
	h, _ := newTestHandler()

	// Step 1: Show today's assignments
	resp := h.HandleSession(context.Background(), USSDRequest{
		SessionID: "sess-5", PhoneNumber: "+254712345678", Text: "4",
	})
	if resp[:4] != "CON " {
		t.Fatalf("expected CON, got: %s", resp)
	}
	if !contains(resp, "Check In") {
		t.Errorf("expected 'Check In' action, got: %s", resp)
	}

	// Step 2: Select assignment 1
	resp = h.HandleSession(context.Background(), USSDRequest{
		SessionID: "sess-5", PhoneNumber: "+254712345678", Text: "4*1",
	})
	if resp[:4] != "CON " {
		t.Fatalf("expected CON confirmation, got: %s", resp)
	}
	if !contains(resp, "Confirm") || !contains(resp, "CBD Route 46") {
		t.Errorf("expected check-in confirmation, got: %s", resp)
	}

	// Step 3: Confirm check-in
	resp = h.HandleSession(context.Background(), USSDRequest{
		SessionID: "sess-5", PhoneNumber: "+254712345678", Text: "4*1*1",
	})
	if resp[:4] != "END " {
		t.Fatalf("expected END, got: %s", resp)
	}
	if !contains(resp, "Checked in") {
		t.Errorf("expected check-in success, got: %s", resp)
	}
}

func TestEarnings_Today(t *testing.T) {
	h, _ := newTestHandler()
	resp := h.HandleSession(context.Background(), USSDRequest{
		SessionID: "sess-6", PhoneNumber: "+254712345678", Text: "2*1",
	})
	if resp[:4] != "END " {
		t.Fatalf("expected END, got: %s", resp)
	}
	if !contains(resp, "KES 1500.00") {
		t.Errorf("expected earnings total, got: %s", resp)
	}
}

func TestWallet_Balance(t *testing.T) {
	h, _ := newTestHandler()
	resp := h.HandleSession(context.Background(), USSDRequest{
		SessionID: "sess-7", PhoneNumber: "+254712345678", Text: "3",
	})
	if resp[:4] != "END " {
		t.Fatalf("expected END, got: %s", resp)
	}
	if !contains(resp, "KES 5000.00") {
		t.Errorf("expected wallet balance, got: %s", resp)
	}
}

func TestNextPayday(t *testing.T) {
	h, _ := newTestHandler()
	resp := h.HandleSession(context.Background(), USSDRequest{
		SessionID: "sess-8", PhoneNumber: "+254712345678", Text: "6",
	})
	if resp[:4] != "END " {
		t.Fatalf("expected END, got: %s", resp)
	}
	if !contains(resp, "Weekly Pay") {
		t.Errorf("expected schedule name, got: %s", resp)
	}
	if !contains(resp, "WEEKLY") {
		t.Errorf("expected frequency, got: %s", resp)
	}
}

func TestProfile(t *testing.T) {
	h, _ := newTestHandler()
	resp := h.HandleSession(context.Background(), USSDRequest{
		SessionID: "sess-9", PhoneNumber: "+254712345678", Text: "5",
	})
	if resp[:4] != "END " {
		t.Fatalf("expected END, got: %s", resp)
	}
	if !contains(resp, "Jane Doe") {
		t.Errorf("expected full name, got: %s", resp)
	}
	if !contains(resp, "CountyLink SACCO") {
		t.Errorf("expected org name, got: %s", resp)
	}
}

func TestRegistration_NewUser(t *testing.T) {
	h, _ := newTestHandler()

	// Step 1: Unregistered user dials
	resp := h.HandleSession(context.Background(), USSDRequest{
		SessionID: "sess-10", PhoneNumber: "+254799000000", Text: "",
	})
	if !contains(resp, "Jisajili") {
		t.Errorf("expected registration menu with Jisajili, got: %s", resp)
	}

	// Step 2: Choose register
	resp = h.HandleSession(context.Background(), USSDRequest{
		SessionID: "sess-10", PhoneNumber: "+254799000000", Text: "1",
	})
	if !contains(resp, "kitambulisho") {
		t.Errorf("expected national ID prompt, got: %s", resp)
	}

	// Step 3: Enter national ID
	resp = h.HandleSession(context.Background(), USSDRequest{
		SessionID: "sess-10", PhoneNumber: "+254799000000", Text: "1*99887766",
	})
	if !contains(resp, "jina") {
		t.Errorf("expected name prompt, got: %s", resp)
	}

	// Step 4: Enter name — registration completes
	resp = h.HandleSession(context.Background(), USSDRequest{
		SessionID: "sess-10", PhoneNumber: "+254799000000", Text: "1*99887766*John Kamau",
	})
	if resp[:4] != "END " {
		t.Fatalf("expected END, got: %s", resp)
	}
	if !contains(resp, "Umesajiliwa") || !contains(resp, "John") {
		t.Errorf("expected registration success, got: %s", resp)
	}
}

func TestRegistration_DuplicateNationalID(t *testing.T) {
	h, _ := newTestHandler()
	// Try registering with existing national ID
	resp := h.HandleSession(context.Background(), USSDRequest{
		SessionID: "sess-11", PhoneNumber: "+254799111111", Text: "1*12345678*Duplicate User",
	})
	if !contains(resp, "kimesajiliwa") || !contains(resp, "Jane") {
		t.Errorf("expected duplicate ID rejection, got: %s", resp)
	}
}

func TestLanguageResolution_SwahiliDefault(t *testing.T) {
	h, td := newTestHandler()
	// Set user pref to empty → should fall back to org default (sw)
	td.user.PreferredLanguage = ""
	resp := h.HandleSession(context.Background(), USSDRequest{
		SessionID: "sess-12", PhoneNumber: "+254712345678", Text: "",
	})
	if !contains(resp, "Karibu") {
		t.Errorf("expected Swahili menu, got: %s", resp)
	}
}

func TestPhoneNormalization(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"+254712345678", "254712345678"},
		{"0712345678", "254712345678"},
		{"254712345678", "254712345678"},
		{" +254712345678 ", "254712345678"},
	}

	for _, tc := range tests {
		result := normalizePhone(tc.input)
		if result != tc.expected {
			t.Errorf("normalizePhone(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestParsePath(t *testing.T) {
	tests := []struct {
		input    string
		expected int // expected number of parts
	}{
		{"", 0},
		{"1", 1},
		{"1*2", 2},
		{"1*2*3", 3},
	}

	for _, tc := range tests {
		parts := parsePath(tc.input)
		if len(parts) != tc.expected {
			t.Errorf("parsePath(%q) = %d parts, want %d", tc.input, len(parts), tc.expected)
		}
	}
}

func TestCalculateNextPayday(t *testing.T) {
	// Test monthly payday on the 28th
	payDay28 := 28
	sched := models.PaySchedule{Frequency: models.PayMonthly, PayDay: &payDay28}

	// If today is the 15th, next payday should be the 28th of this month
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	next := calculateNextPayday(sched, now)
	if next.Day() != 28 || next.Month() != 5 {
		t.Errorf("expected May 28, got: %s", next.Format("Jan 02"))
	}

	// If today is the 29th, next payday should be the 28th of next month
	now = time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	next = calculateNextPayday(sched, now)
	if next.Day() != 28 || next.Month() != 6 {
		t.Errorf("expected Jun 28, got: %s", next.Format("Jan 02"))
	}
}

func TestFormatKES(t *testing.T) {
	tests := []struct {
		cents    int64
		expected string
	}{
		{150000, "KES 1500.00"},
		{100, "KES 1.00"},
		{50, "KES 0.50"},
		{0, "KES 0.00"},
	}

	for _, tc := range tests {
		result := formatKES(tc.cents)
		if result != tc.expected {
			t.Errorf("formatKES(%d) = %q, want %q", tc.cents, result, tc.expected)
		}
	}
}

// helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub { return true }
	}
	return false
}
