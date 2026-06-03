package service

import (
	"testing"

	"greenpark/finance/internal/domain"
	"greenpark/finance/internal/repository"
)

func newSvc(t *testing.T) FinanceService {
	t.Helper()
	repo, err := repository.NewRepository("") // in-memory only
	if err != nil {
		t.Fatalf("repo init: %v", err)
	}
	return New(repo)
}

func TestSummaryDerivation(t *testing.T) {
	svc := newSvc(t)
	s := svc.Summary()

	if s.TotalRevenue <= 0 {
		t.Fatalf("expected positive total revenue, got %v", s.TotalRevenue)
	}
	if s.CollectionRate < 0 || s.CollectionRate > 100 {
		t.Errorf("collection rate out of range: %d", s.CollectionRate)
	}
	if s.NetMargin <= 0 || s.NetMargin > 100 {
		t.Errorf("net margin out of range: %d", s.NetMargin)
	}
	if s.Runway <= 0 {
		t.Errorf("expected positive runway, got %v", s.Runway)
	}
	// Two receivables are seeded in the >90 day bucket.
	if s.Critical != 2 {
		t.Errorf("expected 2 critical (>90d) receivables, got %d", s.Critical)
	}
	if s.OverdueRisk != "Tinggi" {
		t.Errorf("expected overdue risk Tinggi, got %q", s.OverdueRisk)
	}
}

func TestProjectByIDNotFound(t *testing.T) {
	svc := newSvc(t)
	if _, err := svc.ProjectByID("does-not-exist"); err == nil {
		t.Fatal("expected error for unknown project id")
	}
	if _, err := svc.ProjectByID("aurora"); err != nil {
		t.Fatalf("expected to find seeded project, got %v", err)
	}
}

func TestSaveAndDeleteReceivableFlowsToSummary(t *testing.T) {
	svc := newSvc(t)
	before := svc.Summary().OutstandingAR

	saved, err := svc.SaveReceivable(domain.Receivable{
		ID: "AR-TEST", Project: "Greenpark Aurora", Customer: "Test", Type: "kpr",
		Amount: 500, Bucket: "current", SLA: "ok", Owner: "QA",
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if saved.EntID == "" {
		t.Fatal("expected a generated _id on create")
	}
	if got := svc.Summary().OutstandingAR; got != before+500 {
		t.Errorf("expected AR to grow by 500, before=%v after=%v", before, got)
	}

	ok, err := svc.DeleteReceivable(saved.EntID)
	if err != nil || !ok {
		t.Fatalf("delete: ok=%v err=%v", ok, err)
	}
	if got := svc.Summary().OutstandingAR; got != before {
		t.Errorf("expected AR to return to %v, got %v", before, got)
	}
}
