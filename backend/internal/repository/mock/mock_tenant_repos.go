package mock

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/pkg/errs"
)

// --- TenantJobTypeRepo Mock ---

type TenantJobTypeRepo struct {
	mu       sync.RWMutex
	jobTypes map[uuid.UUID]*models.TenantJobType
}

func NewTenantJobTypeRepo() *TenantJobTypeRepo {
	return &TenantJobTypeRepo{jobTypes: make(map[uuid.UUID]*models.TenantJobType)}
}

func (r *TenantJobTypeRepo) Create(_ context.Context, jt *models.TenantJobType) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if jt.ID == uuid.Nil {
		jt.ID = uuid.New()
	}
	now := time.Now()
	jt.CreatedAt = now
	jt.UpdatedAt = now
	r.jobTypes[jt.ID] = jt
	return nil
}

func (r *TenantJobTypeRepo) GetByID(_ context.Context, id uuid.UUID) (*models.TenantJobType, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	jt, ok := r.jobTypes[id]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return jt, nil
}

func (r *TenantJobTypeRepo) Update(_ context.Context, jt *models.TenantJobType) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	jt.UpdatedAt = time.Now()
	r.jobTypes[jt.ID] = jt
	return nil
}

func (r *TenantJobTypeRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.jobTypes, id)
	return nil
}

func (r *TenantJobTypeRepo) ListByOrganization(_ context.Context, orgID uuid.UUID) ([]models.TenantJobType, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []models.TenantJobType
	for _, jt := range r.jobTypes {
		if jt.OrganizationID == orgID && jt.IsActive {
			result = append(result, *jt)
		}
	}
	return result, nil
}

func (r *TenantJobTypeRepo) GetByCode(_ context.Context, orgID uuid.UUID, code string) (*models.TenantJobType, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, jt := range r.jobTypes {
		if jt.OrganizationID == orgID && jt.Code == code && jt.IsActive {
			return jt, nil
		}
	}
	return nil, errs.ErrNotFound
}

// --- PayScheduleRepo Mock ---

type PayScheduleRepo struct {
	mu        sync.RWMutex
	schedules map[uuid.UUID]*models.PaySchedule
}

func NewPayScheduleRepo() *PayScheduleRepo {
	return &PayScheduleRepo{schedules: make(map[uuid.UUID]*models.PaySchedule)}
}

func (r *PayScheduleRepo) Create(_ context.Context, ps *models.PaySchedule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ps.ID == uuid.Nil {
		ps.ID = uuid.New()
	}
	now := time.Now()
	ps.CreatedAt = now
	ps.UpdatedAt = now
	r.schedules[ps.ID] = ps
	return nil
}

func (r *PayScheduleRepo) GetByID(_ context.Context, id uuid.UUID) (*models.PaySchedule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ps, ok := r.schedules[id]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return ps, nil
}

func (r *PayScheduleRepo) Update(_ context.Context, ps *models.PaySchedule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ps.UpdatedAt = time.Now()
	r.schedules[ps.ID] = ps
	return nil
}

func (r *PayScheduleRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.schedules, id)
	return nil
}

func (r *PayScheduleRepo) ListByOrganization(_ context.Context, orgID uuid.UUID) ([]models.PaySchedule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []models.PaySchedule
	for _, ps := range r.schedules {
		if ps.OrganizationID == orgID && ps.IsActive {
			result = append(result, *ps)
		}
	}
	return result, nil
}

func (r *PayScheduleRepo) GetDefault(_ context.Context, orgID uuid.UUID) (*models.PaySchedule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, ps := range r.schedules {
		if ps.OrganizationID == orgID && ps.IsDefault && ps.IsActive {
			return ps, nil
		}
	}
	return nil, errs.ErrNotFound
}
