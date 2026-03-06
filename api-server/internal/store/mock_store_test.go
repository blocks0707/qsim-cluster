package store

import (
	"testing"
)

func TestMockJobStore_CreateAndGet(t *testing.T) {
	s := NewMockJobStore()
	job := &Job{ID: "test-1", UserID: "user-1", Status: "pending", Code: "print(1)", Language: "python", Priority: "normal"}

	if err := s.Create(job); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := s.GetByID("test-1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.UserID != "user-1" {
		t.Errorf("Expected user-1, got %s", got.UserID)
	}
}

func TestMockJobStore_CreateDuplicate(t *testing.T) {
	s := NewMockJobStore()
	job := &Job{ID: "dup-1", UserID: "u", Status: "pending", Code: "x", Language: "python", Priority: "normal"}
	s.Create(job)

	err := s.Create(job)
	if err != ErrConflict {
		t.Errorf("Expected ErrConflict, got %v", err)
	}
}

func TestMockJobStore_GetNotFound(t *testing.T) {
	s := NewMockJobStore()
	_, err := s.GetByID("nonexistent")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestMockJobStore_UpdateStatus(t *testing.T) {
	s := NewMockJobStore()
	s.Create(&Job{ID: "j1", UserID: "u1", Status: "pending", Code: "x", Language: "python", Priority: "normal"})

	if err := s.UpdateStatus("j1", "u1", "running"); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	got, _ := s.GetByID("j1")
	if got.Status != "running" {
		t.Errorf("Expected running, got %s", got.Status)
	}
}

func TestMockJobStore_UpdateStatusByID(t *testing.T) {
	s := NewMockJobStore()
	s.Create(&Job{ID: "j2", UserID: "u1", Status: "submitted", Code: "x", Language: "python", Priority: "normal"})

	if err := s.UpdateStatusByID("j2", "completed"); err != nil {
		t.Fatalf("UpdateStatusByID failed: %v", err)
	}

	got, _ := s.GetByID("j2")
	if got.Status != "completed" {
		t.Errorf("Expected completed, got %s", got.Status)
	}
}

func TestMockJobStore_List(t *testing.T) {
	s := NewMockJobStore()
	s.Create(&Job{ID: "a", UserID: "u1", Status: "pending", Code: "x", Language: "python", Priority: "normal"})
	s.Create(&Job{ID: "b", UserID: "u1", Status: "running", Code: "x", Language: "python", Priority: "normal"})
	s.Create(&Job{ID: "c", UserID: "u2", Status: "pending", Code: "x", Language: "python", Priority: "normal"})

	jobs, total, err := s.List(JobListParams{UserID: "u1", Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 2 {
		t.Errorf("Expected total 2, got %d", total)
	}
	if len(jobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(jobs))
	}

	// Filter by status
	jobs2, total2, _ := s.List(JobListParams{UserID: "u1", Status: "pending", Page: 1, Limit: 10})
	if total2 != 1 {
		t.Errorf("Expected 1 pending job, got %d", total2)
	}
	if len(jobs2) != 1 || jobs2[0].ID != "a" {
		t.Errorf("Expected job 'a', got %v", jobs2)
	}
}

func TestMockJobStore_UpdateComplexity(t *testing.T) {
	s := NewMockJobStore()
	s.Create(&Job{ID: "j3", UserID: "u1", Status: "pending", Code: "x", Language: "python", Priority: "normal"})

	err := s.UpdateComplexity("j3", 4, 10, 20, "B", "statevector")
	if err != nil {
		t.Fatalf("UpdateComplexity failed: %v", err)
	}

	got, _ := s.GetByID("j3")
	if *got.Qubits != 4 || *got.ComplexityClass != "B" {
		t.Errorf("Unexpected complexity: qubits=%d, class=%s", *got.Qubits, *got.ComplexityClass)
	}
}
