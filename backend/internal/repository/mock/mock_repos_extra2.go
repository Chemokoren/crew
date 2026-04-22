package mock

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/internal/repository"
)

// Mock DocumentRepository
type DocumentRepo struct {
	Docs map[uuid.UUID]*models.Document
}

func NewDocumentRepo() *DocumentRepo {
	return &DocumentRepo{Docs: make(map[uuid.UUID]*models.Document)}
}

func (m *DocumentRepo) Create(ctx context.Context, doc *models.Document) error {
	doc.ID = uuid.New()
	m.Docs[doc.ID] = doc
	return nil
}

func (m *DocumentRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Document, error) {
	if d, ok := m.Docs[id]; ok {
		return d, nil
	}
	return nil, errors.New("document not found")
}

func (m *DocumentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if _, ok := m.Docs[id]; !ok {
		return errors.New("document not found")
	}
	delete(m.Docs, id)
	return nil
}

func (m *DocumentRepo) List(ctx context.Context, filter repository.DocumentFilter, page, perPage int) ([]models.Document, int64, error) {
	var docs []models.Document
	for _, d := range m.Docs {
		if filter.DocumentType != "" && string(d.DocumentType) != filter.DocumentType {
			continue
		}
		docs = append(docs, *d)
	}
	return docs, int64(len(docs)), nil
}

// Mock NotificationRepository
type NotificationRepo struct {
	Notifs map[uuid.UUID]*models.Notification
}

func NewNotificationRepo() *NotificationRepo {
	return &NotificationRepo{Notifs: make(map[uuid.UUID]*models.Notification)}
}

func (m *NotificationRepo) Create(ctx context.Context, n *models.Notification) error {
	n.ID = uuid.New()
	m.Notifs[n.ID] = n
	return nil
}

func (m *NotificationRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error) {
	if n, ok := m.Notifs[id]; ok {
		return n, nil
	}
	return nil, errors.New("notification not found")
}

func (m *NotificationRepo) Update(ctx context.Context, n *models.Notification) error {
	m.Notifs[n.ID] = n
	return nil
}

func (m *NotificationRepo) ListByUser(ctx context.Context, userID uuid.UUID, filter repository.NotificationFilter, page, perPage int) ([]models.Notification, int64, error) {
	var list []models.Notification
	for _, n := range m.Notifs {
		if n.UserID == userID {
			list = append(list, *n)
		}
	}
	return list, int64(len(list)), nil
}

func (m *NotificationRepo) MarkRead(ctx context.Context, id uuid.UUID) error {
	n, ok := m.Notifs[id]
	if !ok {
		return errors.New("not found")
	}
	n.Status = models.NotifRead
	return nil
}

func (m *NotificationRepo) GetTemplate(ctx context.Context, eventName string) (*models.NotificationTemplate, error) {
	return &models.NotificationTemplate{
		EventName:     eventName,
		TitleTemplate: "Test Title",
		BodyTemplate:  "Test Body",
		IsActive:      true,
	}, nil
}

func (m *NotificationRepo) CreateTemplate(ctx context.Context, t *models.NotificationTemplate) error {
	return nil
}

func (m *NotificationRepo) UpdateTemplate(ctx context.Context, t *models.NotificationTemplate) error {
	return nil
}

func (m *NotificationRepo) ListTemplates(ctx context.Context) ([]models.NotificationTemplate, error) {
	return []models.NotificationTemplate{}, nil
}

// Mock NotificationPreferenceRepository
type NotificationPreferenceRepo struct {
	mu    sync.RWMutex
	Prefs map[uuid.UUID]*models.NotificationPreference
}

func NewNotificationPreferenceRepo() *NotificationPreferenceRepo {
	return &NotificationPreferenceRepo{Prefs: make(map[uuid.UUID]*models.NotificationPreference)}
}

func (m *NotificationPreferenceRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*models.NotificationPreference, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if p, ok := m.Prefs[userID]; ok {
		return p, nil
	}
	return nil, errors.New("not found")
}

func (m *NotificationPreferenceRepo) Upsert(ctx context.Context, p *models.NotificationPreference) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Prefs[p.UserID] = p
	return nil
}
