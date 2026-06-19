package repository

import (
	"strings"
	"sync"

	"greenpark/finance/internal/domain"
)

// maxHistory caps the rollback-able import history.
const maxHistory = 20

// fileRepository is a mutex-guarded FinanceRepository. The full state is held in
// memory for fast reads and flushed to its persister (file or DB) on every write.
type fileRepository struct {
	mu sync.RWMutex
	p  persister
	st *state
}

// NewRepository returns a FinanceRepository persisted to the given JSON file
// path. An empty path keeps everything in memory only (handy for tests).
func NewRepository(path string) (FinanceRepository, error) {
	return newRepository(filePersister{path: path})
}

// NewPostgresRepository returns a FinanceRepository that persists the whole-state
// snapshot to a single PostgreSQL row.
func NewPostgresRepository(dsn string) (FinanceRepository, error) {
	p, err := newPGPersister(dsn)
	if err != nil {
		return nil, err
	}
	return newRepository(p)
}

func newRepository(p persister) (FinanceRepository, error) {
	st, err := p.load()
	if err != nil {
		return nil, err
	}
	return &fileRepository{p: p, st: st}, nil
}

// persist flushes the current state. Callers must hold the write lock.
func (r *fileRepository) persist() error { return r.p.save(r.st) }

/* ------------------------------- reads ------------------------------- */

func (r *fileRepository) Dashboard() domain.Dashboard {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.st.Data
}

func (r *fileRepository) Revision() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.st.Rev
}

func (r *fileRepository) ImportHistory() []domain.ImportRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.ImportRecord, len(r.st.History))
	for i, e := range r.st.History {
		out[i] = e.Record
	}
	return out
}

/* ------------------------- ingest / lifecycle ------------------------- */

// ApplyImport replaces the live dashboard with the imported data, records a
// rollback snapshot + history entry (newest first, capped) and bumps the rev.
func (r *fileRepository) ApplyImport(in ImportInput) (domain.ImportRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rec := domain.ImportRecord{
		ID: in.ID, Time: in.Time, Filename: in.Filename, By: in.By, Summary: in.Summary,
	}
	entry := importEntry{Record: rec, Prev: r.st.Data}
	r.st.History = append([]importEntry{entry}, r.st.History...)
	if len(r.st.History) > maxHistory {
		r.st.History = r.st.History[:maxHistory]
	}
	r.st.Data = in.Data
	r.st.Rev++
	return rec, r.persist()
}

// ResetData clears the dashboard back to empty (reversible via history).
func (r *fileRepository) ResetData(by, when string) (domain.ImportRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rec := domain.ImportRecord{
		ID: newID("rst"), Time: when, Filename: "Reset data", By: by,
	}
	entry := importEntry{Record: rec, Prev: r.st.Data}
	r.st.History = append([]importEntry{entry}, r.st.History...)
	if len(r.st.History) > maxHistory {
		r.st.History = r.st.History[:maxHistory]
	}
	r.st.Data = emptyDashboard()
	r.st.Rev++
	return rec, r.persist()
}

// Rollback restores the dashboard to its state before the given import id.
func (r *fileRepository) Rollback(id string) (domain.ImportRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	idx := -1
	for i, e := range r.st.History {
		if e.Record.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return domain.ImportRecord{}, ErrNotFound
	}
	undone := r.st.History[idx].Record
	r.st.Data = r.st.History[idx].Prev
	// Drop this entry and everything newer than it (they no longer apply).
	r.st.History = r.st.History[idx+1:]
	r.st.Rev++
	return undone, r.persist()
}

/* ------------------------------- users ------------------------------- */

func (r *fileRepository) Users() []domain.User {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.User, len(r.st.Users))
	for i, u := range r.st.Users {
		out[i] = u.toDomain()
	}
	return out
}

func (r *fileRepository) UserByUsername(username string) (domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	username = strings.ToLower(strings.TrimSpace(username))
	for _, u := range r.st.Users {
		if strings.ToLower(u.Username) == username {
			return u.toDomain(), nil
		}
	}
	return domain.User{}, ErrNotFound
}

func (r *fileRepository) UserByID(id string) (domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.st.Users {
		if u.ID == id {
			return u.toDomain(), nil
		}
	}
	return domain.User{}, ErrNotFound
}
