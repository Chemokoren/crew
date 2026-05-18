package mock

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kibsoft/amy-mis/internal/models"
	"github.com/kibsoft/amy-mis/pkg/errs"
)

// SystemSettingRepo is an in-memory mock of repository.SystemSettingRepository.
type SystemSettingRepo struct {
	mu       sync.RWMutex
	settings map[string]*models.SystemSetting
}

func NewSystemSettingRepo() *SystemSettingRepo {
	return &SystemSettingRepo{settings: make(map[string]*models.SystemSetting)}
}

func (m *SystemSettingRepo) Get(ctx context.Context, key string) (*models.SystemSetting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.settings[key]; ok {
		return s, nil
	}
	return nil, errs.ErrNotFound
}

func (m *SystemSettingRepo) Set(ctx context.Context, setting *models.SystemSetting) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.settings[setting.Key]; ok {
		existing.Value = setting.Value
		existing.ValueType = setting.ValueType
		existing.Category = setting.Category
		existing.Label = setting.Label
		existing.UpdatedAt = time.Now()
	} else {
		if setting.ID == uuid.Nil {
			setting.ID = uuid.New()
		}
		setting.CreatedAt = time.Now()
		setting.UpdatedAt = time.Now()
		m.settings[setting.Key] = setting
	}
	return nil
}

func (m *SystemSettingRepo) GetAll(ctx context.Context) ([]models.SystemSetting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]models.SystemSetting, 0, len(m.settings))
	for _, s := range m.settings {
		result = append(result, *s)
	}
	return result, nil
}

func (m *SystemSettingRepo) GetByPrefix(ctx context.Context, prefix string) ([]models.SystemSetting, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []models.SystemSetting
	for key, s := range m.settings {
		if strings.HasPrefix(key, prefix) {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *SystemSettingRepo) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.settings[key]; !ok {
		return errs.ErrNotFound
	}
	delete(m.settings, key)
	return nil
}

func (m *SystemSettingRepo) BulkSet(ctx context.Context, settings []models.SystemSetting) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range settings {
		s := settings[i]
		if s.ID == uuid.Nil {
			s.ID = uuid.New()
		}
		s.CreatedAt = time.Now()
		s.UpdatedAt = time.Now()
		m.settings[s.Key] = &s
	}
	return nil
}
