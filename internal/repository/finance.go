// Package repository defines storage access for the Finance dashboard and ships
// a file-backed, in-memory implementation seeded with representative data.
// Writes are mutex-guarded and persisted to a JSON file so master-data edits
// survive restarts. Swapping in a database-backed store only requires
// satisfying the FinanceRepository interface.
package repository

import (
	"errors"

	"greenpark/finance/internal/domain"
)

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("resource not found")

// FinanceRepository is the persistence boundary for the dashboard data set.
type FinanceRepository interface {
	// ---- reads ----
	Projects() []domain.Project
	ProjectByID(id string) (domain.Project, error)
	Receivables() []domain.Receivable
	ReceivableType() []domain.MetaItem
	AgingMeta() []domain.MetaItem
	Payables() []domain.Payable
	PriorityMeta() []domain.MetaItem
	Facilities() []domain.Facility
	CostStructure() []domain.CostCategory
	Treasury() domain.Treasury
	AIInsights() []domain.AIInsight
	Decisions() []domain.Decision
	CashflowTrend() []domain.CashflowPoint
	KPITable() []domain.KPI
	Triggers() []domain.Trigger

	// ---- singleton / whole-array writes ----
	SetTreasury(domain.Treasury) error
	SetCostStructure([]domain.CostCategory) error
	SetCashflowTrend([]domain.CashflowPoint) error
	SetReceivableType([]domain.MetaItem) error
	SetAgingMeta([]domain.MetaItem) error
	SetPriorityMeta([]domain.MetaItem) error

	// ---- collection writes (Save = create when _id empty, else update) ----
	SaveProject(domain.Project) (domain.Project, error)
	DeleteProject(id string) (bool, error)
	SaveReceivable(domain.Receivable) (domain.Receivable, error)
	DeleteReceivable(id string) (bool, error)
	SavePayable(domain.Payable) (domain.Payable, error)
	DeletePayable(id string) (bool, error)
	SaveFacility(domain.Facility) (domain.Facility, error)
	DeleteFacility(id string) (bool, error)
	SaveAIInsight(domain.AIInsight) (domain.AIInsight, error)
	DeleteAIInsight(id string) (bool, error)
	SaveDecision(domain.Decision) (domain.Decision, error)
	DeleteDecision(id string) (bool, error)
	SaveKPI(domain.KPI) (domain.KPI, error)
	DeleteKPI(id string) (bool, error)
	SaveTrigger(domain.Trigger) (domain.Trigger, error)
	DeleteTrigger(id string) (bool, error)

	// ---- users (auth) ----
	Users() []domain.User
	UserByUsername(username string) (domain.User, error)
	UserByID(id string) (domain.User, error)
}
