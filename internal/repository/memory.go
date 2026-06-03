package repository

import (
	"strings"
	"sync"

	"greenpark/finance/internal/domain"
)

// fileRepository is a mutex-guarded, file-backed FinanceRepository. The full
// state is held in memory for fast reads and flushed to disk on every write.
type fileRepository struct {
	mu   sync.RWMutex
	path string
	st   *state
}

// NewRepository returns a FinanceRepository persisted to the given JSON file
// path. An empty path keeps everything in memory only (handy for tests).
func NewRepository(path string) (FinanceRepository, error) {
	st, err := load(path)
	if err != nil {
		return nil, err
	}
	return &fileRepository{path: path, st: st}, nil
}

// persist flushes the current state to disk. Callers must hold the write lock.
func (r *fileRepository) persist() error { return save(r.path, r.st) }

/* ---------------------------- reads ---------------------------- */

func (r *fileRepository) Projects() []domain.Project {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return clone(r.st.Projects)
}

func (r *fileRepository) ProjectByID(id string) (domain.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.st.Projects {
		if p.ID == id {
			return p, nil
		}
	}
	return domain.Project{}, ErrNotFound
}

func (r *fileRepository) Receivables() []domain.Receivable {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return clone(r.st.Receivables)
}

func (r *fileRepository) ReceivableType() []domain.MetaItem {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return clone(r.st.ReceivableType)
}

func (r *fileRepository) AgingMeta() []domain.MetaItem {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return clone(r.st.AgingMeta)
}

func (r *fileRepository) Payables() []domain.Payable {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return clone(r.st.Payables)
}

func (r *fileRepository) PriorityMeta() []domain.MetaItem {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return clone(r.st.PriorityMeta)
}

func (r *fileRepository) Facilities() []domain.Facility {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return clone(r.st.Facilities)
}

func (r *fileRepository) CostStructure() []domain.CostCategory {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return clone(r.st.CostStructure)
}

func (r *fileRepository) Treasury() domain.Treasury {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.st.Treasury
}

func (r *fileRepository) AIInsights() []domain.AIInsight {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return clone(r.st.AIInsights)
}

func (r *fileRepository) Decisions() []domain.Decision {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return clone(r.st.Decisions)
}

func (r *fileRepository) CashflowTrend() []domain.CashflowPoint {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return clone(r.st.CashflowTrend)
}

func (r *fileRepository) KPITable() []domain.KPI {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return clone(r.st.KPITable)
}

func (r *fileRepository) Triggers() []domain.Trigger {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return clone(r.st.Triggers)
}

/* ---------------------------- singleton / whole-array writes ---------------------------- */

func (r *fileRepository) SetTreasury(t domain.Treasury) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.st.Treasury = t
	return r.persist()
}

func (r *fileRepository) SetCostStructure(c []domain.CostCategory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.st.CostStructure = c
	return r.persist()
}

func (r *fileRepository) SetCashflowTrend(c []domain.CashflowPoint) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.st.CashflowTrend = c
	return r.persist()
}

func (r *fileRepository) SetReceivableType(m []domain.MetaItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.st.ReceivableType = m
	return r.persist()
}

func (r *fileRepository) SetAgingMeta(m []domain.MetaItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.st.AgingMeta = m
	return r.persist()
}

func (r *fileRepository) SetPriorityMeta(m []domain.MetaItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.st.PriorityMeta = m
	return r.persist()
}

/* ---------------------------- collection writes ---------------------------- */

func (r *fileRepository) SaveProject(p domain.Project) (domain.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p.EntID == "" {
		p.EntID = newID("prj")
	}
	r.st.Projects = upsertEntity(r.st.Projects, p)
	return p, r.persist()
}

func (r *fileRepository) DeleteProject(id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	next, ok := deleteEntity(r.st.Projects, id)
	r.st.Projects = next
	if !ok {
		return false, nil
	}
	return true, r.persist()
}

func (r *fileRepository) SaveReceivable(v domain.Receivable) (domain.Receivable, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v.EntID == "" {
		v.EntID = newID("ar")
	}
	r.st.Receivables = upsertEntity(r.st.Receivables, v)
	return v, r.persist()
}

func (r *fileRepository) DeleteReceivable(id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	next, ok := deleteEntity(r.st.Receivables, id)
	r.st.Receivables = next
	if !ok {
		return false, nil
	}
	return true, r.persist()
}

func (r *fileRepository) SavePayable(v domain.Payable) (domain.Payable, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v.EntID == "" {
		v.EntID = newID("ap")
	}
	r.st.Payables = upsertEntity(r.st.Payables, v)
	return v, r.persist()
}

func (r *fileRepository) DeletePayable(id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	next, ok := deleteEntity(r.st.Payables, id)
	r.st.Payables = next
	if !ok {
		return false, nil
	}
	return true, r.persist()
}

func (r *fileRepository) SaveFacility(v domain.Facility) (domain.Facility, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v.EntID == "" {
		v.EntID = newID("fac")
	}
	r.st.Facilities = upsertEntity(r.st.Facilities, v)
	return v, r.persist()
}

func (r *fileRepository) DeleteFacility(id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	next, ok := deleteEntity(r.st.Facilities, id)
	r.st.Facilities = next
	if !ok {
		return false, nil
	}
	return true, r.persist()
}

func (r *fileRepository) SaveAIInsight(v domain.AIInsight) (domain.AIInsight, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v.EntID == "" {
		v.EntID = newID("ai")
	}
	r.st.AIInsights = upsertEntity(r.st.AIInsights, v)
	return v, r.persist()
}

func (r *fileRepository) DeleteAIInsight(id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	next, ok := deleteEntity(r.st.AIInsights, id)
	r.st.AIInsights = next
	if !ok {
		return false, nil
	}
	return true, r.persist()
}

func (r *fileRepository) SaveDecision(v domain.Decision) (domain.Decision, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v.EntID == "" {
		v.EntID = newID("dec")
	}
	r.st.Decisions = upsertEntity(r.st.Decisions, v)
	return v, r.persist()
}

func (r *fileRepository) DeleteDecision(id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	next, ok := deleteEntity(r.st.Decisions, id)
	r.st.Decisions = next
	if !ok {
		return false, nil
	}
	return true, r.persist()
}

func (r *fileRepository) SaveKPI(k domain.KPI) (domain.KPI, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if k.EntID == "" {
		k.EntID = newID("kpi")
	}
	r.st.KPITable = upsertEntity(r.st.KPITable, k)
	return k, r.persist()
}

func (r *fileRepository) DeleteKPI(id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	next, ok := deleteEntity(r.st.KPITable, id)
	r.st.KPITable = next
	if !ok {
		return false, nil
	}
	return true, r.persist()
}

func (r *fileRepository) SaveTrigger(t domain.Trigger) (domain.Trigger, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t.EntID == "" {
		t.EntID = newID("trg")
	}
	r.st.Triggers = upsertEntity(r.st.Triggers, t)
	return t, r.persist()
}

func (r *fileRepository) DeleteTrigger(id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	next, ok := deleteEntity(r.st.Triggers, id)
	r.st.Triggers = next
	if !ok {
		return false, nil
	}
	return true, r.persist()
}

/* ---------------------------- users ---------------------------- */

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
