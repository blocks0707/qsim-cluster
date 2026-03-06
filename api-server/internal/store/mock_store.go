package store

import (
	"sync"
	"time"
)

// MockJobStore is an in-memory implementation for testing
type MockJobStore struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

func NewMockJobStore() *MockJobStore {
	return &MockJobStore{jobs: make(map[string]*Job)}
}

func (m *MockJobStore) Create(job *Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.jobs[job.ID]; exists {
		return ErrConflict
	}
	now := time.Now()
	job.CreatedAt = now
	job.UpdatedAt = now
	m.jobs[job.ID] = job
	return nil
}

func (m *MockJobStore) GetByID(id string) (*Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, exists := m.jobs[id]
	if !exists {
		return nil, ErrNotFound
	}
	return job, nil
}

func (m *MockJobStore) List(params JobListParams) ([]*Job, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*Job
	for _, job := range m.jobs {
		if job.UserID != params.UserID {
			continue
		}
		if params.Status != "" && job.Status != params.Status {
			continue
		}
		result = append(result, job)
	}
	total := len(result)
	start := (params.Page - 1) * params.Limit
	if start >= len(result) {
		return nil, total, nil
	}
	end := start + params.Limit
	if end > len(result) {
		end = len(result)
	}
	return result[start:end], total, nil
}

func (m *MockJobStore) UpdateStatus(id, userID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, exists := m.jobs[id]
	if !exists || job.UserID != userID {
		return ErrNotFound
	}
	job.Status = status
	job.UpdatedAt = time.Now()
	return nil
}

func (m *MockJobStore) UpdateStatusByID(id, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, exists := m.jobs[id]
	if !exists {
		return ErrNotFound
	}
	job.Status = status
	job.UpdatedAt = time.Now()
	return nil
}

func (m *MockJobStore) UpdateComplexity(id string, qubits, depth, gateCount int, complexityClass, method string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, exists := m.jobs[id]
	if !exists {
		return ErrNotFound
	}
	job.Qubits = &qubits
	job.Depth = &depth
	job.GateCount = &gateCount
	job.ComplexityClass = &complexityClass
	job.Method = &method
	return nil
}

func (m *MockJobStore) UpdateAssignment(id, node, pool string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, exists := m.jobs[id]
	if !exists {
		return ErrNotFound
	}
	job.AssignedNode = &node
	job.AssignedPool = &pool
	return nil
}

func (m *MockJobStore) UpdateExecution(id string, startedAt, completedAt *time.Time, executionTimeMs *int64, resultRef, errorMessage string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, exists := m.jobs[id]
	if !exists {
		return ErrNotFound
	}
	job.StartedAt = startedAt
	job.CompletedAt = completedAt
	job.ExecutionTimeMs = executionTimeMs
	job.ResultRef = &resultRef
	job.ErrorMessage = errorMessage
	return nil
}
