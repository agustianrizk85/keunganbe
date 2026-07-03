package repository

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"greenpark/finance/internal/domain"
)

// state is the full persisted snapshot: the derived dashboard, the rollback-able
// import history, a monotonic data revision and the user accounts. The dashboard
// is assembled at ingest time and stored whole (no master-data CRUD).
type state struct {
	Data       domain.Dashboard  `json:"data"`
	AR         domain.ARData     `json:"ar"`         // AR/piutang view (separate ingest, same revision)
	ARSources  []domain.ARSource `json:"arSources"`  // per-project AR input sheets (UI-configured)
	Purchasing domain.Purchasing `json:"purchasing"` // procurement (PR) view (separate ingest, same revision)
	PRSheet    string            `json:"prSheet"`    // procurement input spreadsheet ID (UI-configured; "" = use env default)
	History    []importEntry     `json:"history"`
	Rev        int64             `json:"rev"`
	Users      []storeUser       `json:"users"`
}

// importEntry is one history record plus the snapshot of the dashboard as it was
// BEFORE the import — applying that snapshot rolls the import back.
type importEntry struct {
	Record domain.ImportRecord `json:"record"`
	Prev   domain.Dashboard    `json:"prev"`
}

// storeUser is the persisted user shape (serialises salt + hash, unlike
// domain.User). It never leaves the store.
type storeUser struct {
	ID           string      `json:"id"`
	Username     string      `json:"username"`
	Name         string      `json:"name"`
	Role         domain.Role `json:"role"`
	PasswordHash string      `json:"passwordHash"`
	Salt         string      `json:"salt"`
}

func (u storeUser) toDomain() domain.User {
	return domain.User{
		ID: u.ID, Username: u.Username, Name: u.Name, Role: u.Role,
		PasswordHash: u.PasswordHash, Salt: u.Salt,
	}
}

// emptyDashboard is the fresh, no-data dashboard (non-nil empty slices so the
// JSON stays "[]" not "null").
func emptyDashboard() domain.Dashboard {
	return domain.Dashboard{
		Period:    "Belum ada data — silakan Sync Google Sheets / upload Excel",
		FocusYear: 2026,
		Years:     []int{},
		Funnel:    []domain.FunnelStage{},
		Monthly:   []domain.MonthPoint{},
		Projects:  []domain.ProjectFin{},
		Banks:     []domain.BankFin{},
		Sales:     []domain.SalesRank{},
		PayMix:    []domain.PayMethod{},
		Pipeline:  []domain.PipelineRow{},
		Akads:     []domain.AkadRow{},
		Alerts:    []domain.Alert{},
		AI:        []domain.AIInsight{},
		Decisions: []domain.Decision{},
		KPIs:      []domain.KPI{},
		Triggers:   []domain.Trigger{},
		Purchasing: domain.EmptyPurchasing(),
	}
}

// seedState builds a fresh store: empty dashboard + the default accounts.
func seedState() *state {
	return &state{
		Data:       emptyDashboard(),
		AR:         domain.EmptyARData(),
		Purchasing: domain.EmptyPurchasing(),
		History:    []importEntry{},
		Users:      seedUsers(),
	}
}

// load reads the state from disk; a missing file seeds a fresh state and writes
// it. An empty path means in-memory only (used by tests).
func load(path string) (*state, error) {
	if path == "" {
		return seedState(), nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			s := seedState()
			if err := save(path, s); err != nil {
				return nil, err
			}
			return s, nil
		}
		return nil, err
	}
	s := &state{}
	if err := json.Unmarshal(b, s); err != nil {
		return nil, err
	}
	// Migration: procurement used to live inside the dashboard (Data.Purchasing),
	// fed by the akad sync. It now has its own slot + independent sync flow — lift
	// the existing view across once so the dashboard keeps showing it pre-resync.
	if s.Purchasing.IsEmpty() && !s.Data.Purchasing.IsEmpty() {
		s.Purchasing = s.Data.Purchasing
	}
	return s, nil
}

// save atomically writes the state to disk (write temp + rename).
func save(path string, s *state) error {
	if path == "" {
		return nil
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// persister abstracts where the whole-state snapshot lives (JSON file or DB row).
type persister interface {
	load() (*state, error)
	save(*state) error
}

// filePersister stores the state as an atomic JSON file on disk.
type filePersister struct{ path string }

func (f filePersister) load() (*state, error) { return load(f.path) }
func (f filePersister) save(st *state) error  { return save(f.path, st) }

// newID returns a short, collision-resistant identifier with the given prefix.
func newID(prefix string) string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return prefix + "-0000000000"
	}
	return prefix + "-" + hex.EncodeToString(b)
}
