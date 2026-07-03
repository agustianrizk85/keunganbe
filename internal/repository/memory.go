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
	d := r.st.Data
	// Procurement is synced independently of the akad dashboard, so overlay the
	// separately-stored Purchasing view onto the dashboard payload (keeping the
	// single-call FE contract intact).
	if !r.st.Purchasing.IsEmpty() {
		d.Purchasing = r.st.Purchasing
	}
	return d
}

func (r *fileRepository) Purchasing() domain.Purchasing {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.st.Purchasing.IsEmpty() {
		return domain.EmptyPurchasing()
	}
	return r.st.Purchasing
}

// PRSheet returns the UI-configured procurement input spreadsheet ID ("" when
// none is set, so the handler falls back to the env default).
func (r *fileRepository) PRSheet() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.st.PRSheet
}

func (r *fileRepository) AR() domain.ARData {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.st.AR.Sheets == nil && r.st.AR.Period == "" {
		return domain.EmptyARData()
	}
	return r.st.AR
}

func (r *fileRepository) Revision() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.st.Rev
}

// ARSources returns the UI-configured per-project AR input sheets.
func (r *fileRepository) ARSources() []domain.ARSource {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.ARSource, len(r.st.ARSources))
	copy(out, r.st.ARSources)
	return out
}

// SetARSources replaces the AR input sheet list and persists it. It does not
// bump the data revision — saving config does not change the dashboard until a
// sync runs.
func (r *fileRepository) SetARSources(src []domain.ARSource) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.st.ARSources = append([]domain.ARSource(nil), src...)
	return r.persist()
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

// ApplyAR replaces the AR/piutang view and bumps the shared revision so the
// realtime watcher pushes an update to connected dashboards. AR has no rollback
// history (it is a derived read-only view, re-synced from the input sheets).
func (r *fileRepository) ApplyAR(ar domain.ARData) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.st.AR = ar
	r.st.Rev++
	return r.persist()
}

// ApplyPurchasing replaces the procurement (PR) view and bumps the shared
// revision so connected dashboards refresh. Like AR, it has no rollback history
// (it is a derived view, re-synced from the procurement input sheet).
func (r *fileRepository) ApplyPurchasing(p domain.Purchasing) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.st.Purchasing = p
	r.st.Rev++
	return r.persist()
}

// SetPRSheet stores the procurement input spreadsheet ID. It does not bump the
// data revision — saving config does not change the dashboard until a sync runs.
func (r *fileRepository) SetPRSheet(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.st.PRSheet = id
	return r.persist()
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
