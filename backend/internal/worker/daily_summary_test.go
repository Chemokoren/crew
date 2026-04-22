package worker_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/internal/worker"
	"github.com/kibsoft/amy-mis/pkg/errs"
)

type mockAssignmentRepo struct {
	assignments []models.Assignment
}

func (m *mockAssignmentRepo) Create(ctx context.Context, assignment *models.Assignment) error { return nil }
func (m *mockAssignmentRepo) BulkCreate(ctx context.Context, assignments []models.Assignment) (int, []repository.BulkError, error) { return 0, nil, nil }
func (m *mockAssignmentRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Assignment, error) { return nil, nil }
func (m *mockAssignmentRepo) Update(ctx context.Context, assignment *models.Assignment) error { return nil }
func (m *mockAssignmentRepo) HasActiveAssignment(ctx context.Context, crewMemberID uuid.UUID, date time.Time) (bool, error) { return false, nil }
func (m *mockAssignmentRepo) List(ctx context.Context, filter repository.AssignmentFilter, page, perPage int) ([]models.Assignment, int64, error) {
	return m.assignments, int64(len(m.assignments)), nil
}

type mockEarningRepo struct {
	earnings []models.Earning
	upserted []models.DailyEarningsSummary
}

func (m *mockEarningRepo) Create(ctx context.Context, earning *models.Earning) error { return nil }
func (m *mockEarningRepo) BulkCreate(ctx context.Context, earnings []models.Earning) (int, []repository.BulkError, error) { return 0, nil, nil }
func (m *mockEarningRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Earning, error) { return nil, nil }
func (m *mockEarningRepo) Update(ctx context.Context, earning *models.Earning) error { return nil }
func (m *mockEarningRepo) GetDailySummary(ctx context.Context, crewMemberID uuid.UUID, date time.Time) (*models.DailyEarningsSummary, error) {
	return nil, errs.ErrNotFound // return not found
}
func (m *mockEarningRepo) UpsertDailySummary(ctx context.Context, summary *models.DailyEarningsSummary) error {
	m.upserted = append(m.upserted, *summary)
	return nil
}
func (m *mockEarningRepo) List(ctx context.Context, filter repository.EarningFilter, page, perPage int) ([]models.Earning, int64, error) {
	var filtered []models.Earning
	for _, e := range m.earnings {
		if filter.AssignmentID != nil && e.AssignmentID == *filter.AssignmentID {
			filtered = append(filtered, e)
		}
	}
	return filtered, int64(len(filtered)), nil
}

func TestDailySummaryJob_Run(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	crewID1 := uuid.New()
	crewID2 := uuid.New()
	assignID1 := uuid.New()
	assignID2 := uuid.New()

	assignments := []models.Assignment{
		{ID: assignID1, CrewMemberID: crewID1, Status: models.AssignmentCompleted},
		{ID: assignID2, CrewMemberID: crewID2, Status: models.AssignmentCompleted},
	}
	
	earnings := []models.Earning{
		{ID: uuid.New(), AssignmentID: assignID1, AmountCents: 5000},
		{ID: uuid.New(), AssignmentID: assignID1, AmountCents: 2000},
		{ID: uuid.New(), AssignmentID: assignID2, AmountCents: 10000},
	}

	assignRepo := &mockAssignmentRepo{assignments: assignments}
	earnRepo := &mockEarningRepo{earnings: earnings}

	job := worker.NewDailySummaryJob(earnRepo, assignRepo, logger)
	
	err := job.Run(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(earnRepo.upserted) != 2 {
		t.Errorf("expected 2 summaries, got %d", len(earnRepo.upserted))
	}

	var found1, found2 bool
	for _, summary := range earnRepo.upserted {
		if summary.CrewMemberID == crewID1 {
			found1 = true
			if summary.TotalEarnedCents != 7000 {
				t.Errorf("expected 7000 cents for crew 1, got %d", summary.TotalEarnedCents)
			}
		}
		if summary.CrewMemberID == crewID2 {
			found2 = true
			if summary.TotalEarnedCents != 10000 {
				t.Errorf("expected 10000 cents for crew 2, got %d", summary.TotalEarnedCents)
			}
		}
	}

	if !found1 || !found2 {
		t.Errorf("missing summaries for one or both crews")
	}
}
