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

// --- VehicleRepo Mock ---

type VehicleRepo struct {
	mu       sync.RWMutex
	vehicles map[uuid.UUID]*models.Vehicle
}

func NewVehicleRepo() *VehicleRepo {
	return &VehicleRepo{vehicles: make(map[uuid.UUID]*models.Vehicle)}
}

func (r *VehicleRepo) Create(_ context.Context, v *models.Vehicle) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	v.CreatedAt = time.Now()
	v.UpdatedAt = time.Now()
	r.vehicles[v.ID] = v
	return nil
}

func (r *VehicleRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Vehicle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.vehicles[id]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return v, nil
}

func (r *VehicleRepo) Update(_ context.Context, v *models.Vehicle) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	v.UpdatedAt = time.Now()
	r.vehicles[v.ID] = v
	return nil
}

func (r *VehicleRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.vehicles, id)
	return nil
}

func (r *VehicleRepo) List(_ context.Context, saccoID *uuid.UUID, page, perPage int) ([]models.Vehicle, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []models.Vehicle
	for _, v := range r.vehicles {
		if saccoID == nil || v.SaccoID == *saccoID {
			all = append(all, *v)
		}
	}
	return all, int64(len(all)), nil
}

// --- RouteRepo Mock ---

type RouteRepo struct {
	mu     sync.RWMutex
	routes map[uuid.UUID]*models.Route
}

func NewRouteRepo() *RouteRepo {
	return &RouteRepo{routes: make(map[uuid.UUID]*models.Route)}
}

func (r *RouteRepo) Create(_ context.Context, route *models.Route) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if route.ID == uuid.Nil {
		route.ID = uuid.New()
	}
	route.CreatedAt = time.Now()
	route.UpdatedAt = time.Now()
	r.routes[route.ID] = route
	return nil
}

func (r *RouteRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Route, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	route, ok := r.routes[id]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return route, nil
}

func (r *RouteRepo) Update(_ context.Context, route *models.Route) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	route.UpdatedAt = time.Now()
	r.routes[route.ID] = route
	return nil
}

func (r *RouteRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.routes, id)
	return nil
}

func (r *RouteRepo) List(_ context.Context, page, perPage int, search string) ([]models.Route, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []models.Route
	for _, route := range r.routes {
		all = append(all, *route)
	}
	return all, int64(len(all)), nil
}

// --- PayrollRepo Mock ---

type PayrollRepo struct {
	mu      sync.RWMutex
	runs    map[uuid.UUID]*models.PayrollRun
	entries []models.PayrollEntry
}

func NewPayrollRepo() *PayrollRepo {
	return &PayrollRepo{runs: make(map[uuid.UUID]*models.PayrollRun)}
}

func (r *PayrollRepo) Create(_ context.Context, run *models.PayrollRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if run.ID == uuid.Nil {
		run.ID = uuid.New()
	}
	run.CreatedAt = time.Now()
	run.UpdatedAt = time.Now()
	r.runs[run.ID] = run
	return nil
}

func (r *PayrollRepo) GetByID(_ context.Context, id uuid.UUID) (*models.PayrollRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	run, ok := r.runs[id]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return run, nil
}

func (r *PayrollRepo) Update(_ context.Context, run *models.PayrollRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	run.UpdatedAt = time.Now()
	r.runs[run.ID] = run
	return nil
}

func (r *PayrollRepo) List(_ context.Context, saccoID *uuid.UUID, page, perPage int) ([]models.PayrollRun, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []models.PayrollRun
	for _, run := range r.runs {
		if saccoID == nil || run.SaccoID == *saccoID {
			all = append(all, *run)
		}
	}
	return all, int64(len(all)), nil
}

func (r *PayrollRepo) CreateEntries(_ context.Context, entries []models.PayrollEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range entries {
		if entries[i].ID == uuid.Nil {
			entries[i].ID = uuid.New()
		}
		r.entries = append(r.entries, entries[i])
	}
	return nil
}

func (r *PayrollRepo) GetEntries(_ context.Context, runID uuid.UUID) ([]models.PayrollEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var runEntries []models.PayrollEntry
	for _, e := range r.entries {
		if e.PayrollRunID == runID {
			runEntries = append(runEntries, e)
		}
	}
	return runEntries, nil
}

// --- StatutoryRateRepo Mock ---

type StatutoryRateRepo struct {
	mu    sync.RWMutex
	rates []models.StatutoryRate
}

func NewStatutoryRateRepo(initial []models.StatutoryRate) *StatutoryRateRepo {
	return &StatutoryRateRepo{rates: initial}
}

func (r *StatutoryRateRepo) GetActiveRates(_ context.Context) ([]models.StatutoryRate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var active []models.StatutoryRate
	for _, rate := range r.rates {
		if rate.IsActive {
			active = append(active, rate)
		}
	}
	return active, nil
}

func (r *StatutoryRateRepo) Create(_ context.Context, rate *models.StatutoryRate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rate.ID == uuid.Nil {
		rate.ID = uuid.New()
	}
	r.rates = append(r.rates, *rate)
	return nil
}

func (r *StatutoryRateRepo) Update(_ context.Context, rate *models.StatutoryRate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, existing := range r.rates {
		if existing.ID == rate.ID {
			r.rates[i] = *rate
			return nil
		}
	}
	return errs.ErrNotFound
}

// --- EarningRepo Mock ---

type EarningRepo struct {
	mu       sync.RWMutex
	earnings map[uuid.UUID]*models.Earning
}

func NewEarningRepo() *EarningRepo {
	return &EarningRepo{earnings: make(map[uuid.UUID]*models.Earning)}
}

func (r *EarningRepo) Create(_ context.Context, earning *models.Earning) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if earning.ID == uuid.Nil {
		earning.ID = uuid.New()
	}
	r.earnings[earning.ID] = earning
	return nil
}

func (r *EarningRepo) BulkCreate(_ context.Context, earnings []models.Earning) (int, []repository.BulkError, error) {
	return 0, nil, nil
}

func (r *EarningRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Earning, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.earnings[id]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return e, nil
}

func (r *EarningRepo) Update(_ context.Context, earning *models.Earning) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.earnings[earning.ID] = earning
	return nil
}

func (r *EarningRepo) List(_ context.Context, filter repository.EarningFilter, page, perPage int) ([]models.Earning, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []models.Earning
	for _, e := range r.earnings {
		all = append(all, *e)
	}
	return all, int64(len(all)), nil
}

func (r *EarningRepo) GetDailySummary(_ context.Context, crewMemberID uuid.UUID, date time.Time) (*models.DailyEarningsSummary, error) {
	return nil, errs.ErrNotFound
}

func (r *EarningRepo) UpsertDailySummary(_ context.Context, summary *models.DailyEarningsSummary) error {
	return nil
}
