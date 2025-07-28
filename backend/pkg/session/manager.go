package session

import (
	"fmt"
	"sync"
	"time"

	"rancher-questions-generator/internal/models"

	"github.com/google/uuid"
)

type Manager struct {
	sessions map[string]*models.Session
	mutex    sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*models.Session),
	}
}

func (m *Manager) CreateSession(chartURL string) *models.Session {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	sessionID := uuid.New().String()
	session := &models.Session{
		ID:        sessionID,
		ChartURL:  chartURL,
		Values:    make(map[string]interface{}),
		Questions: models.Questions{Questions: []models.Question{}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	m.sessions[sessionID] = session
	return session
}

func (m *Manager) GetSession(sessionID string) (*models.Session, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	return session, nil
}

func (m *Manager) UpdateSession(sessionID string, questions models.Questions) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.Questions = questions
	session.UpdatedAt = time.Now()
	return nil
}

func (m *Manager) DeleteSession(sessionID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.sessions[sessionID]; !exists {
		return fmt.Errorf("session not found")
	}

	delete(m.sessions, sessionID)
	return nil
}