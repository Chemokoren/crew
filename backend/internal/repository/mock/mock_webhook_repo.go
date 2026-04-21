package mock

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/pkg/errs"
)

type WebhookEventRepo struct {
	events map[uuid.UUID]*models.WebhookEvent
}

func NewWebhookEventRepo() *WebhookEventRepo {
	return &WebhookEventRepo{events: make(map[uuid.UUID]*models.WebhookEvent)}
}

func (m *WebhookEventRepo) Create(ctx context.Context, event *models.WebhookEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	event.CreatedAt = time.Now()
	m.events[event.ID] = event
	return nil
}

func (m *WebhookEventRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.WebhookEvent, error) {
	if e, ok := m.events[id]; ok {
		return e, nil
	}
	return nil, errs.ErrNotFound
}

func (m *WebhookEventRepo) GetByExternalRef(ctx context.Context, source models.WebhookSource, ref string) (*models.WebhookEvent, error) {
	for _, e := range m.events {
		if e.Source == source && e.ExternalRef == ref {
			return e, nil
		}
	}
	return nil, errs.ErrNotFound
}

func (m *WebhookEventRepo) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	e, ok := m.events[id]
	if !ok {
		return errs.ErrNotFound
	}
	e.IsProcessed = true
	now := time.Now()
	e.ProcessedAt = &now
	return nil
}

func (m *WebhookEventRepo) ListUnprocessed(ctx context.Context, source models.WebhookSource, limit int) ([]models.WebhookEvent, error) {
	var res []models.WebhookEvent
	for _, e := range m.events {
		if !e.IsProcessed && (source == "" || e.Source == source) {
			res = append(res, *e)
			if len(res) == limit {
				break
			}
		}
	}
	return res, nil
}
