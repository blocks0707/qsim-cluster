package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	_ "github.com/lib/pq"
)

var (
	ErrNotFound = errors.New("record not found")
	ErrConflict = errors.New("record already exists")
)

// Config holds configuration for the store
type Config struct {
	PostgresURL string
	RedisURL    string
}

// Stores holds all store implementations
type Stores struct {
	Jobs  JobStore
	Cache CacheStore
	DB    *sql.DB
	Redis *redis.Client
}

// Job represents a quantum simulation job
type Job struct {
	ID             string    `db:"id"`
	UserID         string    `db:"user_id"`
	Status         string    `db:"status"`
	Code           string    `db:"code"`
	Language       string    `db:"language"`
	Qubits         *int      `db:"qubits"`
	Depth          *int      `db:"depth"`
	GateCount      *int      `db:"gate_count"`
	ComplexityClass *string  `db:"complexity_class"`
	Method         *string   `db:"method"`
	Priority       string    `db:"priority"`
	AssignedNode   *string   `db:"assigned_node"`
	AssignedPool   *string   `db:"assigned_pool"`
	StartedAt      *time.Time `db:"started_at"`
	CompletedAt    *time.Time `db:"completed_at"`
	ExecutionTimeMs *int64    `db:"execution_time_ms"`
	ResultRef      *string   `db:"result_ref"`
	ErrorMessage   string    `db:"error_message"`
	RetryCount     int       `db:"retry_count"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// JobListParams holds parameters for listing jobs
type JobListParams struct {
	UserID string
	Status string
	Page   int
	Limit  int
}

// JobStore interface defines job storage operations
type JobStore interface {
	Create(job *Job) error
	GetByID(id string) (*Job, error)
	List(params JobListParams) ([]*Job, int, error)
	UpdateStatus(id, userID, status string) error
	UpdateStatusByID(id, status string) error
	UpdateComplexity(id string, qubits, depth, gateCount int, complexityClass, method string) error
	UpdateAssignment(id, node, pool string) error
	UpdateExecution(id string, startedAt, completedAt *time.Time, executionTimeMs *int64, resultRef, errorMessage string) error
}

// CacheStore interface defines cache operations
type CacheStore interface {
	Set(key string, value interface{}, expiration time.Duration) error
	Get(key string, dest interface{}) error
	Delete(key string) error
}

// New creates a new Stores instance
func New(config Config, logger *zap.Logger) (*Stores, error) {
	// Initialize PostgreSQL connection
	db, err := sql.Open("postgres", config.PostgresURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	// Initialize Redis connection
	redisClient := redis.NewClient(&redis.Options{
		Addr: config.RedisURL,
	})

	logger.Info("Connected to stores",
		zap.String("postgres", "connected"),
		zap.String("redis", "connected"),
	)

	stores := &Stores{
		Jobs:  NewPostgresJobStore(db),
		Cache: NewRedisCache(redisClient),
		DB:    db,
		Redis: redisClient,
	}

	return stores, nil
}

// Close closes all store connections
func (s *Stores) Close() error {
	if err := s.DB.Close(); err != nil {
		return fmt.Errorf("failed to close postgres connection: %w", err)
	}

	if err := s.Redis.Close(); err != nil {
		return fmt.Errorf("failed to close redis connection: %w", err)
	}

	return nil
}