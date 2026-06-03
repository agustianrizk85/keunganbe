// Package service holds the business logic of the Finance dashboard. It composes
// the repository data and computes the derived executive summary, keeping
// transport handlers thin. Write use-cases delegate to the repository (which
// persists), so master-data edits flow straight back into the dashboard read.
package service

import (
	"math"

	"greenpark/finance/internal/domain"
	"greenpark/finance/internal/repository"
)

// FinanceService exposes the read and write use-cases of the dashboard.
type FinanceService interface {
	// reads
	Dashboard() domain.Dashboard
	Summary() domain.Summary
	Projects() []domain.Project
	ProjectByID(id string) (domain.Project, error)
	Receivables() []domain.Receivable
	Payables() []domain.Payable
	Facilities() []domain.Facility
	CostStructure() []domain.CostCategory
	Treasury() domain.Treasury
	AIInsights() []domain.AIInsight
	Decisions() []domain.Decision
	KPITable() []domain.KPI
	Triggers() []domain.Trigger

	// singleton / whole-array writes
	SetTreasury(domain.Treasury) error
	SetCostStructure([]domain.CostCategory) error
	SetCashflowTrend([]domain.CashflowPoint) error
	SetReceivableType([]domain.MetaItem) error
	SetAgingMeta([]domain.MetaItem) error
	SetPriorityMeta([]domain.MetaItem) error

	// collection writes
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
}

type financeService struct {
	repo repository.FinanceRepository
}

// New returns a FinanceService backed by the given repository.
func New(repo repository.FinanceRepository) FinanceService {
	return &financeService{repo: repo}
}

// Dashboard assembles the full payload including the derived summary.
func (s *financeService) Dashboard() domain.Dashboard {
	return domain.Dashboard{
		Projects:       s.repo.Projects(),
		Receivables:    s.repo.Receivables(),
		ReceivableType: s.repo.ReceivableType(),
		AgingMeta:      s.repo.AgingMeta(),
		Payables:       s.repo.Payables(),
		PriorityMeta:   s.repo.PriorityMeta(),
		Facilities:     s.repo.Facilities(),
		CostStructure:  s.repo.CostStructure(),
		Treasury:       s.repo.Treasury(),
		AIInsights:     s.repo.AIInsights(),
		Decisions:      s.repo.Decisions(),
		CashflowTrend:  s.repo.CashflowTrend(),
		KPITable:       s.repo.KPITable(),
		Triggers:       s.repo.Triggers(),
		Summary:        s.Summary(),
	}
}

// Summary computes the executive KPIs from projects, receivables, payables and
// the treasury position.
func (s *financeService) Summary() domain.Summary {
	projects := s.repo.Projects()
	receivables := s.repo.Receivables()
	payables := s.repo.Payables()
	treasury := s.repo.Treasury()

	var totalRevenue, totalCollected, totalBudget, totalSpent, weightedMargin float64
	for _, p := range projects {
		totalRevenue += p.Revenue
		totalCollected += p.Collected
		totalBudget += p.Budget
		totalSpent += p.Spent
		weightedMargin += p.Margin * p.Revenue
	}

	var outstandingAR, outstandingAP float64
	var critical int
	for _, r := range receivables {
		outstandingAR += r.Amount
		if r.Bucket == "d90" {
			critical++
		}
	}
	for _, p := range payables {
		outstandingAP += p.Amount
	}

	collectionRate := 0
	netMargin := 0
	if totalRevenue > 0 {
		collectionRate = int(math.Round(totalCollected / totalRevenue * 100))
		netMargin = int(math.Round(weightedMargin / totalRevenue))
	}
	budgetAbsorption := 0
	if totalBudget > 0 {
		budgetAbsorption = int(math.Round(totalSpent / totalBudget * 100))
	}
	runway := 0.0
	if treasury.MonthlyBurn > 0 {
		runway = math.Round(treasury.CashOnHand/treasury.MonthlyBurn*10) / 10
	}

	return domain.Summary{
		TotalRevenue:     totalRevenue,
		CashPosition:     treasury.CashOnHand,
		Collected:        totalCollected,
		CollectionRate:   collectionRate,
		OutstandingAR:    outstandingAR,
		OutstandingAP:    outstandingAP,
		NetMargin:        netMargin,
		BudgetAbsorption: budgetAbsorption,
		Runway:           runway,
		OverdueRisk:      overdueRisk(critical, outstandingAR),
		Critical:         critical,
	}
}

// overdueRisk derives a qualitative collection-risk label.
func overdueRisk(critical int, outstandingAR float64) string {
	switch {
	case critical >= 2:
		return "Tinggi"
	case critical == 1 || outstandingAR >= 3000:
		return "Sedang"
	default:
		return "Rendah"
	}
}

/* ---- reads ---- */

func (s *financeService) Projects() []domain.Project { return s.repo.Projects() }

func (s *financeService) ProjectByID(id string) (domain.Project, error) {
	return s.repo.ProjectByID(id)
}

func (s *financeService) Receivables() []domain.Receivable     { return s.repo.Receivables() }
func (s *financeService) Payables() []domain.Payable           { return s.repo.Payables() }
func (s *financeService) Facilities() []domain.Facility        { return s.repo.Facilities() }
func (s *financeService) CostStructure() []domain.CostCategory { return s.repo.CostStructure() }
func (s *financeService) Treasury() domain.Treasury            { return s.repo.Treasury() }
func (s *financeService) AIInsights() []domain.AIInsight       { return s.repo.AIInsights() }
func (s *financeService) Decisions() []domain.Decision         { return s.repo.Decisions() }
func (s *financeService) KPITable() []domain.KPI               { return s.repo.KPITable() }
func (s *financeService) Triggers() []domain.Trigger           { return s.repo.Triggers() }

/* ---- singleton / whole-array writes ---- */

func (s *financeService) SetTreasury(t domain.Treasury) error { return s.repo.SetTreasury(t) }
func (s *financeService) SetCostStructure(c []domain.CostCategory) error {
	return s.repo.SetCostStructure(c)
}
func (s *financeService) SetCashflowTrend(c []domain.CashflowPoint) error {
	return s.repo.SetCashflowTrend(c)
}
func (s *financeService) SetReceivableType(m []domain.MetaItem) error {
	return s.repo.SetReceivableType(m)
}
func (s *financeService) SetAgingMeta(m []domain.MetaItem) error    { return s.repo.SetAgingMeta(m) }
func (s *financeService) SetPriorityMeta(m []domain.MetaItem) error { return s.repo.SetPriorityMeta(m) }

/* ---- collection writes ---- */

func (s *financeService) SaveProject(p domain.Project) (domain.Project, error) {
	return s.repo.SaveProject(p)
}
func (s *financeService) DeleteProject(id string) (bool, error) { return s.repo.DeleteProject(id) }
func (s *financeService) SaveReceivable(v domain.Receivable) (domain.Receivable, error) {
	return s.repo.SaveReceivable(v)
}
func (s *financeService) DeleteReceivable(id string) (bool, error) {
	return s.repo.DeleteReceivable(id)
}
func (s *financeService) SavePayable(v domain.Payable) (domain.Payable, error) {
	return s.repo.SavePayable(v)
}
func (s *financeService) DeletePayable(id string) (bool, error) { return s.repo.DeletePayable(id) }
func (s *financeService) SaveFacility(v domain.Facility) (domain.Facility, error) {
	return s.repo.SaveFacility(v)
}
func (s *financeService) DeleteFacility(id string) (bool, error) { return s.repo.DeleteFacility(id) }
func (s *financeService) SaveAIInsight(v domain.AIInsight) (domain.AIInsight, error) {
	return s.repo.SaveAIInsight(v)
}
func (s *financeService) DeleteAIInsight(id string) (bool, error) { return s.repo.DeleteAIInsight(id) }
func (s *financeService) SaveDecision(v domain.Decision) (domain.Decision, error) {
	return s.repo.SaveDecision(v)
}
func (s *financeService) DeleteDecision(id string) (bool, error) { return s.repo.DeleteDecision(id) }
func (s *financeService) SaveKPI(k domain.KPI) (domain.KPI, error) {
	return s.repo.SaveKPI(k)
}
func (s *financeService) DeleteKPI(id string) (bool, error) { return s.repo.DeleteKPI(id) }
func (s *financeService) SaveTrigger(t domain.Trigger) (domain.Trigger, error) {
	return s.repo.SaveTrigger(t)
}
func (s *financeService) DeleteTrigger(id string) (bool, error) { return s.repo.DeleteTrigger(id) }
