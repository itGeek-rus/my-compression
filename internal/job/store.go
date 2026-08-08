package job

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusQueued     Status = "queued"
	StatusProcessing Status = "processing"
	StatusDone       Status = "done"
	StatusFailed     Status = "failed"
)

type Job struct {
	ID           string `json:"id"`
	Action       string `json:"action"` // archive | extract
	Format       string `json:"format"` // zip | tar.gz
	Status       Status `json:"status"`
	Message      string `json:"message"`
	Progress     int    `json:"progress"` // 0%...100 %
	OriginalSize int64  `json:"original_size"`
	ResultSize   int64  `json:"result_size"`
	ResultPath   string `json:"-"`
	DownloadName string `json:"download_name,omitempty"`
	Error        string `json:"error,omitempty"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

var ErrNotFound = errors.New("job not found")

type Store struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

func NewStore() *Store {
	return &Store{jobs: make(map[string]*Job)}
}

func (s *Store) Create(action, format string) *Job {
	now := time.Now().UTC()
	j := &Job{
		ID:        uuid.NewString(),
		Action:    action,
		Format:    format,
		Status:    StatusQueued,
		Message:   "Pending Processing",
		Progress:  0,
		CreatedAt: now.UTC().Unix(),
		UpdatedAt: now.UTC().Unix(),
	}

	s.mu.Lock()
	s.jobs[j.ID] = j
	s.mu.Unlock()
	return clone(j)
}

func (s *Store) Get(id string) (*Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	j, ok := s.jobs[id]
	if !ok {
		return nil, ErrNotFound
	}
	return clone(j), nil
}

func (s *Store) Update(id string, fn func(*Job)) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	j, ok := s.jobs[id]
	if !ok {
		return nil, ErrNotFound
	}
	fn(j)
	j.UpdatedAt = time.Now().UTC().Unix()
	return clone(j), nil
}

func clone(j *Job) *Job {
	cp := *j
	return &cp
}
