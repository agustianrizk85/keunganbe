package repository

import (
	"os"
	"testing"

	"greenpark/finance/internal/domain"
)

// TestPostgresStateIntegration runs against a real PostgreSQL when
// TEST_DATABASE_URL is set; otherwise it is skipped.
func TestPostgresStateIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping Postgres integration test")
	}

	repo, err := NewPostgresRepository(dsn)
	if err != nil {
		t.Fatalf("NewPostgresRepository: %v", err)
	}
	beforeRev := repo.Revision()

	rec, err := repo.ApplyImport(ImportInput{
		ID: "imp-test", Time: "now", Filename: "t.xlsx", By: "qa",
		Summary: domain.ImportSummary{AkadCount: 3, NilaiAkad: 1500},
		Data:    domain.Dashboard{Period: "Akad 2026", Summary: domain.Summary{AkadCount: 3, NilaiAkad: 1500}},
	})
	if err != nil {
		t.Fatalf("ApplyImport: %v", err)
	}
	if rec.ID != "imp-test" {
		t.Fatalf("unexpected record id %q", rec.ID)
	}

	// Reopen (simulates a restart) → data persisted.
	repo2, err := NewPostgresRepository(dsn)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if got := repo2.Dashboard().Summary.AkadCount; got != 3 {
		t.Fatalf("after restart akad = %d, want 3", got)
	}
	if repo2.Revision() <= beforeRev {
		t.Fatalf("revision did not advance: before=%d after=%d", beforeRev, repo2.Revision())
	}
}
