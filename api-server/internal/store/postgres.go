package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// PostgresJobStore implements JobStore using PostgreSQL
type PostgresJobStore struct {
	db *sql.DB
}

// NewPostgresJobStore creates a new PostgreSQL job store
func NewPostgresJobStore(db *sql.DB) JobStore {
	return &PostgresJobStore{db: db}
}

// Create creates a new job in the database
func (s *PostgresJobStore) Create(job *Job) error {
	query := `
		INSERT INTO quantum_jobs (
			id, user_id, status, code, language, priority, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	now := time.Now()
	_, err := s.db.Exec(query,
		job.ID, job.UserID, job.Status, job.Code, job.Language,
		job.Priority, now, now,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" { // unique_violation
			return ErrConflict
		}
		return fmt.Errorf("failed to create job: %w", err)
	}

	job.CreatedAt = now
	job.UpdatedAt = now
	return nil
}

// GetByID retrieves a job by its ID
func (s *PostgresJobStore) GetByID(id string) (*Job, error) {
	query := `
		SELECT id, user_id, status, code, language, qubits, depth, gate_count,
			   complexity_class, method, priority, assigned_node, assigned_pool,
			   started_at, completed_at, execution_time_ms, result_ref,
			   error_message, retry_count, created_at, updated_at
		FROM quantum_jobs 
		WHERE id = $1
	`

	job := &Job{}
	err := s.db.QueryRow(query, id).Scan(
		&job.ID, &job.UserID, &job.Status, &job.Code, &job.Language,
		&job.Qubits, &job.Depth, &job.GateCount, &job.ComplexityClass, &job.Method,
		&job.Priority, &job.AssignedNode, &job.AssignedPool,
		&job.StartedAt, &job.CompletedAt, &job.ExecutionTimeMs, &job.ResultRef,
		&job.ErrorMessage, &job.RetryCount, &job.CreatedAt, &job.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return job, nil
}

// List retrieves jobs based on parameters with pagination
func (s *PostgresJobStore) List(params JobListParams) ([]*Job, int, error) {
	// Build WHERE clause
	where := "WHERE user_id = $1"
	args := []interface{}{params.UserID}
	argCount := 1

	if params.Status != "" {
		argCount++
		where += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, params.Status)
	}

	// Count total jobs
	countQuery := "SELECT COUNT(*) FROM quantum_jobs " + where
	var total int
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count jobs: %w", err)
	}

	// Calculate offset
	offset := (params.Page - 1) * params.Limit

	// Get jobs with pagination
	query := `
		SELECT id, user_id, status, code, language, qubits, depth, gate_count,
			   complexity_class, method, priority, assigned_node, assigned_pool,
			   started_at, completed_at, execution_time_ms, result_ref,
			   error_message, retry_count, created_at, updated_at
		FROM quantum_jobs 
	` + where + `
		ORDER BY created_at DESC 
		LIMIT $` + fmt.Sprintf("%d", argCount+1) + ` OFFSET $` + fmt.Sprintf("%d", argCount+2)

	args = append(args, params.Limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job := &Job{}
		err := rows.Scan(
			&job.ID, &job.UserID, &job.Status, &job.Code, &job.Language,
			&job.Qubits, &job.Depth, &job.GateCount, &job.ComplexityClass, &job.Method,
			&job.Priority, &job.AssignedNode, &job.AssignedPool,
			&job.StartedAt, &job.CompletedAt, &job.ExecutionTimeMs, &job.ResultRef,
			&job.ErrorMessage, &job.RetryCount, &job.CreatedAt, &job.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan job: %w", err)
		}
		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate jobs: %w", err)
	}

	return jobs, total, nil
}

// UpdateStatus updates the status of a job
func (s *PostgresJobStore) UpdateStatus(id, userID, status string) error {
	query := `
		UPDATE quantum_jobs 
		SET status = $3, updated_at = $4
		WHERE id = $1 AND user_id = $2
	`

	result, err := s.db.Exec(query, id, userID, status, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// UpdateStatusByID updates job status by ID only (no user check, for K8s sync)
func (s *PostgresJobStore) UpdateStatusByID(id, status string) error {
	query := `UPDATE quantum_jobs SET status = $2, updated_at = $3 WHERE id = $1`
	result, err := s.db.Exec(query, id, status, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateComplexity updates the complexity analysis results
func (s *PostgresJobStore) UpdateComplexity(id string, qubits, depth, gateCount int, complexityClass, method string) error {
	query := `
		UPDATE quantum_jobs 
		SET qubits = $2, depth = $3, gate_count = $4, 
		    complexity_class = $5, method = $6, updated_at = $7
		WHERE id = $1
	`

	_, err := s.db.Exec(query, id, qubits, depth, gateCount, complexityClass, method, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update job complexity: %w", err)
	}

	return nil
}

// UpdateAssignment updates the node assignment
func (s *PostgresJobStore) UpdateAssignment(id, node, pool string) error {
	query := `
		UPDATE quantum_jobs 
		SET assigned_node = $2, assigned_pool = $3, updated_at = $4
		WHERE id = $1
	`

	_, err := s.db.Exec(query, id, node, pool, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update job assignment: %w", err)
	}

	return nil
}

// UpdateExecution updates execution details
func (s *PostgresJobStore) UpdateExecution(id string, startedAt, completedAt *time.Time, executionTimeMs *int64, resultRef, errorMessage string) error {
	query := `
		UPDATE quantum_jobs 
		SET started_at = $2, completed_at = $3, execution_time_ms = $4, 
		    result_ref = $5, error_message = $6, updated_at = $7
		WHERE id = $1
	`

	_, err := s.db.Exec(query, id, startedAt, completedAt, executionTimeMs, resultRef, errorMessage, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update job execution: %w", err)
	}

	return nil
}