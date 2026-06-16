package service

import (
	"context"
	"sync"
	"time"

	"github.com/craftbyte/learning_languages/services/internal/model"
	"github.com/craftbyte/learning_languages/services/internal/repository"
)

const settingsCacheTTL = 10 * time.Second

type SettingService interface {
	List(ctx context.Context) ([]model.Setting, error)
	Update(ctx context.Context, key string, value bool) (*model.Setting, error)
	IsMaintenance(ctx context.Context) bool
}

type settingService struct {
	repo repository.SettingRepository

	mu        sync.RWMutex
	cache     map[string]bool
	cacheTime time.Time
}

func NewSettingService(repo repository.SettingRepository) SettingService {
	return &settingService{repo: repo}
}

func (s *settingService) List(ctx context.Context) ([]model.Setting, error) {
	return s.repo.List(ctx)
}

func (s *settingService) Update(ctx context.Context, key string, value bool) (*model.Setting, error) {
	setting, err := s.repo.Update(ctx, key, value)
	if err != nil {
		return nil, err
	}
	s.invalidate()
	return setting, nil
}

// IsMaintenance reads the maintenance flag through a short-lived cache so the
// global middleware doesn't hit the DB on every request. Errors → false (fail
// open) to avoid locking everyone out on a transient DB issue.
func (s *settingService) IsMaintenance(ctx context.Context) bool {
	return s.flags(ctx)["maintenance"]
}

func (s *settingService) flags(ctx context.Context) map[string]bool {
	s.mu.RLock()
	if s.cache != nil && time.Since(s.cacheTime) < settingsCacheTTL {
		c := s.cache
		s.mu.RUnlock()
		return c
	}
	s.mu.RUnlock()

	list, err := s.repo.List(ctx)
	if err != nil {
		return map[string]bool{}
	}
	m := make(map[string]bool, len(list))
	for _, st := range list {
		m[st.Key] = st.Value
	}

	s.mu.Lock()
	s.cache = m
	s.cacheTime = time.Now()
	s.mu.Unlock()
	return m
}

func (s *settingService) invalidate() {
	s.mu.Lock()
	s.cache = nil
	s.mu.Unlock()
}
