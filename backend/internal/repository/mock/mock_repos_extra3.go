package mock

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
	"github.com/kibsoft/amy-mis/pkg/errs"
)

// --- Extended UserRepo methods ---

func (r *UserRepo) CountUsers(_ context.Context) (total int64, active int64, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	total = int64(len(r.users))
	for _, u := range r.users {
		if u.IsActive {
			active++
		}
	}
	return total, active, nil
}

// --- Extended CrewRepo methods ---

func (r *CrewRepo) GetByNationalID(_ context.Context, nationalID string) (*models.CrewMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, m := range r.members {
		if m.NationalID == nationalID {
			return m, nil
		}
	}
	return nil, errs.ErrNotFound
}

func (r *CrewRepo) Count(_ context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.members)), nil
}

func (r *CrewRepo) BulkCreate(ctx context.Context, members []models.CrewMember) ([]repository.BulkError, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var bulkErrors []repository.BulkError
	for i := range members {
		members[i].ID = uuid.New()
		r.seq++
		members[i].CrewID = fmt.Sprintf("CRW-%05d", r.seq)
		r.members[members[i].ID] = &members[i]
	}
	return bulkErrors, nil
}

// --- Extended WalletRepo methods ---

func (r *WalletRepo) List(_ context.Context, page, perPage int) ([]models.Wallet, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []models.Wallet
	for _, w := range r.wallets {
		all = append(all, *w)
	}
	return all, int64(len(all)), nil
}
