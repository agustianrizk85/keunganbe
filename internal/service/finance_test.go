package service

import (
	"strings"
	"testing"

	"greenpark/finance/internal/repository"
)

func newSvc(t *testing.T) FinanceService {
	t.Helper()
	repo, err := repository.NewRepository("") // in-memory only
	if err != nil {
		t.Fatalf("repo init: %v", err)
	}
	return New(repo, Options{FocusYear: 2026})
}

func TestEmptyDashboard(t *testing.T) {
	svc := newSvc(t)
	d := svc.Dashboard()
	if d.Summary.AkadCount != 0 {
		t.Fatalf("fresh store should have 0 akad, got %d", d.Summary.AkadCount)
	}
	if !strings.Contains(d.Period, "Belum ada data") {
		t.Errorf("expected empty-state period, got %q", d.Period)
	}
}

// A small akad-only XLSX-like sheet, fed through the Google-Sheets path, should
// produce a non-empty summary after approval.
func TestApproveSheetsFlow(t *testing.T) {
	svc := newSvc(t)
	data := map[string][][]string{
		"Data Akad 2026": {
			{"No.", "GP", "Proyek", "Nama Konsumen", "Blok", "Tgl Booking", "DP", "Plafon KPR", "Cara Bayar", "Bank", "Tgl Akad", "Bulan Akad", "Tahun", "Durasi", "Nama Sales"},
			{"1", "GP 3", "VERSAW", "Budi", "D8", "15-Des-2025", "Rp. 50.000.000", "Rp. 500.000.000", "KPR", "BSI Otista", "3-Jan-2026", "Januari", "2026", "19", "Ayu"},
			{"2", "GP 3", "VERSAW", "Sari", "C7", "10-Des-2025", "Rp. 0", "Rp. 450.000.000", "KPR", "BSI Otista", "5-Feb-2026", "Februari", "2026", "57", "Erwin"},
		},
	}
	if _, err := svc.ApproveSheets(data, "test", "qa"); err != nil {
		t.Fatalf("ApproveSheets: %v", err)
	}
	d := svc.Dashboard()
	if d.Summary.AkadCount != 2 {
		t.Fatalf("expected 2 akad, got %d", d.Summary.AkadCount)
	}
	if d.Summary.NilaiAkad != 950 {
		t.Fatalf("expected nilai 950 juta, got %v", d.Summary.NilaiAkad)
	}
	if svc.Revision() == 0 {
		t.Error("revision should advance after approve")
	}
}
