package mock

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/pkg/errs"
)

type AssignmentRepo struct {
	mu          sync.Mutex
	assignments map[uuid.UUID]*models.Assignment
}

func NewAssignmentRepo() *AssignmentRepo {
	return &AssignmentRepo{assignments: make(map[uuid.UUID]*models.Assignment)}
}

func (r *AssignmentRepo) Create(_ context.Context, a *models.Assignment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	r.assignments[a.ID] = a
	return nil
}

func (r *AssignmentRepo) BulkCreate(_ context.Context, as []models.Assignment) (int, []repository.BulkError, error) {
	return 0, nil, nil
}

func (r *AssignmentRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Assignment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.assignments[id]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return a, nil
}

func (r *AssignmentRepo) Update(_ context.Context, a *models.Assignment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.assignments[a.ID] = a
	return nil
}

func (r *AssignmentRepo) List(_ context.Context, _ repository.AssignmentFilter, _, _ int) ([]models.Assignment, int64, error) {
	return nil, 0, nil
}

func (r *AssignmentRepo) HasActiveAssignment(_ context.Context, _ uuid.UUID, _ time.Time) (bool, error) {
	return false, nil
}
