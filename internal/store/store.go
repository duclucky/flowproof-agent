package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/ducky/flowproof-agent/internal/model"
)

var ErrNotFound = errors.New("run not found")

type Store interface {
	Create(run model.Run) error
	Get(id string) (model.Run, error)
	Update(id string, mutate func(*model.Run) error) (model.Run, error)
	List(limit int) ([]model.Run, error)
}

type FileStore struct {
	mu   sync.RWMutex
	path string
	runs map[string]model.Run
}

func NewFileStore(path string) (*FileStore, error) {
	s := &FileStore{path: path, runs: map[string]model.Run{}}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *FileStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read run store: %w", err)
	}
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, &s.runs); err != nil {
		return fmt.Errorf("decode run store: %w", err)
	}
	return nil
}

func (s *FileStore) Create(run model.Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.runs[run.ID]; exists {
		return fmt.Errorf("run %s already exists", run.ID)
	}
	s.runs[run.ID] = cloneRun(run)
	return s.persistLocked()
}

func (s *FileStore) Get(id string) (model.Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	run, ok := s.runs[id]
	if !ok {
		return model.Run{}, ErrNotFound
	}
	return cloneRun(run), nil
}

func (s *FileStore) Update(id string, mutate func(*model.Run) error) (model.Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[id]
	if !ok {
		return model.Run{}, ErrNotFound
	}
	working := cloneRun(run)
	if err := mutate(&working); err != nil {
		return model.Run{}, err
	}
	s.runs[id] = working
	if err := s.persistLocked(); err != nil {
		s.runs[id] = run
		return model.Run{}, err
	}
	return cloneRun(working), nil
}

func (s *FileStore) List(limit int) ([]model.Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Run, 0, len(s.runs))
	for _, run := range s.runs {
		out = append(out, cloneRun(run))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *FileStore) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil && filepath.Dir(s.path) != "." {
		return fmt.Errorf("create state directory: %w", err)
	}
	data, err := json.MarshalIndent(s.runs, "", "  ")
	if err != nil {
		return fmt.Errorf("encode run store: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write temporary run store: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace run store: %w", err)
	}
	return nil
}

func cloneRun(in model.Run) model.Run {
	out := in
	out.Events = append([]model.RunEvent(nil), in.Events...)
	out.Evidence = append([]model.Evidence(nil), in.Evidence...)
	if in.FailureAnalysis != nil {
		analysis := *in.FailureAnalysis
		out.FailureAnalysis = &analysis
	}
	return out
}
