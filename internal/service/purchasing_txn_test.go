package service

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"greenpark/finance/internal/domain"
)

func TestTerbilang(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{0, "nol rupiah"},
		{1, "satu rupiah"},
		{11, "sebelas rupiah"},
		{15, "lima belas rupiah"},
		{20, "dua puluh rupiah"},
		{100, "seratus rupiah"},
		{101, "seratus satu rupiah"},
		{2000, "dua ribu rupiah"},
		{1000, "seribu rupiah"},
		{1500, "seribu lima ratus rupiah"},
		{1856000, "satu juta delapan ratus lima puluh enam ribu rupiah"},
		{1000000, "satu juta rupiah"},
		{1250000000, "satu miliar dua ratus lima puluh juta rupiah"},
	}
	for _, c := range cases {
		if got := terbilang(c.n); got != c.want {
			t.Errorf("terbilang(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestComputePO(t *testing.T) {
	cases := []struct {
		name       string
		items      []domain.POItem
		potongan   int64
		ongkir     int64
		wantTotal  int64
		wantTier   string
		wantTanpa  bool
	}{
		{
			name:      "under 500k tanpa po",
			items:     []domain.POItem{{Qty: 2, HargaSatuan: 100_000}},
			wantTotal: 200_000, wantTier: "none", wantTanpa: true,
		},
		{
			name:      "exactly 500k kadep",
			items:     []domain.POItem{{Qty: 1, HargaSatuan: 500_000}},
			wantTotal: 500_000, wantTier: "kadep",
		},
		{
			name:      "1jt boundary kadep",
			items:     []domain.POItem{{Qty: 1, HargaSatuan: 1_000_000}},
			wantTotal: 1_000_000, wantTier: "kadep",
		},
		{
			name:      "over 1jt dirops",
			items:     []domain.POItem{{Qty: 3, HargaSatuan: 400_000}},
			wantTotal: 1_200_000, wantTier: "dirops",
		},
		{
			name:      "potongan + ongkir",
			items:     []domain.POItem{{Qty: 10, HargaSatuan: 150_000}},
			potongan:  100_000, ongkir: 50_000,
			wantTotal: 1_450_000, wantTier: "dirops",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			po := domain.PurchaseOrder{Items: c.items, Potongan: c.potongan, BiayaPengiriman: c.ongkir}
			computePO(&po)
			if po.Total != c.wantTotal {
				t.Errorf("total = %d, want %d", po.Total, c.wantTotal)
			}
			if po.Tier != c.wantTier {
				t.Errorf("tier = %q, want %q", po.Tier, c.wantTier)
			}
			if po.TanpaPo != c.wantTanpa {
				t.Errorf("tanpaPo = %v, want %v", po.TanpaPo, c.wantTanpa)
			}
			// jumlah per item must equal round(qty*harga)
			for i, it := range po.Items {
				if it.No != i+1 {
					t.Errorf("item %d No = %d, want %d", i, it.No, i+1)
				}
			}
		})
	}
}

func TestNextNomorSequence(t *testing.T) {
	svc := newSvc(t).(*financeService)
	// Use the real clock: the seq counts docs created in the SAME calendar year,
	// and CreatePR stamps createdAt with time.Now(), so both must share a year.
	now := time.Now()
	roman := romanMonth(now.Month())
	year := now.Year()
	pr1 := fmt.Sprintf("PR/001/GPG/%s/%d", roman, year)
	pr2 := fmt.Sprintf("PR/002/GPG/%s/%d", roman, year)
	po1 := fmt.Sprintf("PO/001/GPG/%s/%d", roman, year)

	if got := svc.nextNomor("PR", now); got != pr1 {
		t.Fatalf("first PR nomor = %q, want %q", got, pr1)
	}
	// Create one PR so the count advances.
	if _, err := svc.CreatePR(domain.PurchaseRequest{
		RequestBy: "Budi",
		Items:     []domain.PRItem{{Nama: "Semen", Satuan: "sak", Qty: 5}},
	}, true, "tester"); err != nil {
		t.Fatalf("CreatePR: %v", err)
	}
	if got := svc.nextNomor("PR", now); got != pr2 {
		t.Fatalf("second PR nomor = %q, want %q", got, pr2)
	}
	// PO sequence is independent of PR.
	if got := svc.nextNomor("PO", now); got != po1 {
		t.Fatalf("first PO nomor = %q, want %q", got, po1)
	}
}

func TestPRToPOHappyPath(t *testing.T) {
	svc := newSvc(t)

	// 1) Create + submit a PR.
	pr, err := svc.CreatePR(domain.PurchaseRequest{
		RequestBy: "Budi",
		Dept:      "Teknik",
		Proyek:    "VERSAW",
		Items:     []domain.PRItem{{Nama: "Besi", Satuan: "batang", Qty: 20, Tujuan: "cor"}},
	}, true, "staff")
	if err != nil {
		t.Fatalf("CreatePR: %v", err)
	}
	if pr.Status != "pending" || pr.Nomor == "" {
		t.Fatalf("PR after submit: status=%q nomor=%q", pr.Status, pr.Nomor)
	}

	// 2) A staff-role approver must be rejected (403 / ErrForbidden).
	if _, err := svc.ApprovePR(pr.ID, domain.Approver{Name: "Andi", Role: "staff"}, ""); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for staff approving PR, got %v", err)
	}

	// 3) Kadep approves the PR.
	pr, err = svc.ApprovePR(pr.ID, domain.Approver{Name: "Sari", Role: "KADEP"}, "ok lanjut")
	if err != nil {
		t.Fatalf("ApprovePR: %v", err)
	}
	if pr.Status != "approved" || pr.Approval.ApprovedByRole != "KADEP" {
		t.Fatalf("PR not approved correctly: %+v", pr.Approval)
	}

	// 4) Create + submit a PO referencing the approved PR (>1jt → dirops tier).
	po, err := svc.CreatePO(domain.PurchaseOrder{
		PRID:              pr.ID,
		Supplier:          "PT Baja",
		TanggalPengiriman: "2025-11-20",
		Tanggal:           "2025-11-15",
		Items:             []domain.POItem{{Nama: "Besi", Satuan: "batang", Qty: 20, HargaSatuan: 150_000}},
	}, true, "purchasing")
	if err != nil {
		t.Fatalf("CreatePO: %v", err)
	}
	if po.Total != 3_000_000 || po.Tier != "dirops" || po.Status != "pending" {
		t.Fatalf("PO totals/tier wrong: total=%d tier=%q status=%q", po.Total, po.Tier, po.Status)
	}
	if po.PRNomor != pr.Nomor {
		t.Fatalf("PO prNomor = %q, want %q", po.PRNomor, pr.Nomor)
	}
	if po.Terbilang != "tiga juta rupiah" {
		t.Fatalf("PO terbilang = %q", po.Terbilang)
	}

	// 5) Kadep cannot approve a dirops-tier PO.
	if _, err := svc.ApprovePO(po.ID, domain.Approver{Name: "Sari", Role: "kadep"}, ""); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for kadep approving dirops PO, got %v", err)
	}

	// 6) Dirops approves the PO.
	po, err = svc.ApprovePO(po.ID, domain.Approver{Name: "Rudi", Role: "dirops"}, "approved")
	if err != nil {
		t.Fatalf("ApprovePO: %v", err)
	}
	if po.Status != "approved" {
		t.Fatalf("PO status = %q, want approved", po.Status)
	}

	// 7) Receive the goods late vs the promised delivery date.
	po, err = svc.ReceivePO(po.ID, "2025-11-25", true, "")
	if err != nil {
		t.Fatalf("ReceivePO: %v", err)
	}
	if !po.Receiving.Received || po.Status != "received" {
		t.Fatalf("PO not received: %+v", po.Receiving)
	}
	if po.Receiving.Keterangan != "Terlambat" {
		t.Fatalf("keterangan = %q, want Terlambat", po.Receiving.Keterangan)
	}
	if po.Receiving.SLAHari != 10 { // 2025-11-15 → 2025-11-25
		t.Fatalf("slaHari = %d, want 10", po.Receiving.SLAHari)
	}
}

// TestApprovePRDeptScoping verifies that a kadep may only approve/reject PRs
// from their own department, while dirops/ceo/super and legacy callers (empty
// approver.Dept) remain unrestricted.
func TestApprovePRDeptScoping(t *testing.T) {
	svc := newSvc(t)

	newPendingPR := func(dept string) domain.PurchaseRequest {
		pr, err := svc.CreatePR(domain.PurchaseRequest{
			RequestBy: "Budi",
			Dept:      dept,
			Items:     []domain.PRItem{{Nama: "Semen", Satuan: "sak", Qty: 5}},
		}, true, "staff")
		if err != nil {
			t.Fatalf("CreatePR(%q): %v", dept, err)
		}
		return pr
	}

	// A kadep from a different department cannot approve.
	pr := newPendingPR("keuangan")
	if _, err := svc.ApprovePR(pr.ID, domain.Approver{Name: "Sari", Role: "kadep", Dept: "teknik"}, ""); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for cross-dept kadep approve, got %v", err)
	}
	// ...nor reject.
	if _, err := svc.RejectPR(pr.ID, domain.Approver{Name: "Sari", Role: "kadep", Dept: "teknik"}, "tidak sesuai"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for cross-dept kadep reject, got %v", err)
	}

	// A kadep from the SAME department can approve.
	pr = newPendingPR("teknik")
	pr, err := svc.ApprovePR(pr.ID, domain.Approver{Name: "Sari", Role: "kadep", Dept: "teknik"}, "ok")
	if err != nil {
		t.Fatalf("expected same-dept kadep approve to succeed, got %v", err)
	}
	if pr.Status != "approved" {
		t.Fatalf("PR status = %q, want approved", pr.Status)
	}

	// dirops/ceo/super are exempt from the dept check, regardless of their own Dept.
	for _, role := range []string{"dirops", "ceo", "super"} {
		pr := newPendingPR("keuangan")
		pr, err := svc.ApprovePR(pr.ID, domain.Approver{Name: "Rudi", Role: role, Dept: "teknik"}, "ok")
		if err != nil {
			t.Fatalf("expected %s to approve any-dept PR, got %v", role, err)
		}
		if pr.Status != "approved" {
			t.Fatalf("PR status = %q, want approved (role=%s)", pr.Status, role)
		}
	}

	// A legacy caller that omits Dept (empty string) is unaffected — no regression.
	pr = newPendingPR("keuangan")
	pr, err = svc.ApprovePR(pr.ID, domain.Approver{Name: "Sari", Role: "kadep"}, "ok")
	if err != nil {
		t.Fatalf("expected legacy (no Dept) kadep approve to succeed, got %v", err)
	}
	if pr.Status != "approved" {
		t.Fatalf("PR status = %q, want approved", pr.Status)
	}
}

func TestTanpaPoAutoApprove(t *testing.T) {
	svc := newSvc(t)

	// A PO must reference an approved PR even when it lands under the tanpaPo
	// threshold — only the approval STEP is skipped, not the PR-first rule.
	pr, err := svc.CreatePR(domain.PurchaseRequest{
		RequestBy: "Budi",
		Items:     []domain.PRItem{{Nama: "Paku", Satuan: "kg", Qty: 3}},
	}, true, "staff")
	if err != nil {
		t.Fatalf("CreatePR: %v", err)
	}
	pr, err = svc.ApprovePR(pr.ID, domain.Approver{Name: "Sari", Role: "kadep"}, "")
	if err != nil {
		t.Fatalf("ApprovePR: %v", err)
	}

	po, err := svc.CreatePO(domain.PurchaseOrder{
		PRID:     pr.ID,
		Supplier: "Toko Bangunan",
		Items:    []domain.POItem{{Nama: "Paku", Satuan: "kg", Qty: 3, HargaSatuan: 25_000}},
	}, false, "purchasing")
	if err != nil {
		t.Fatalf("CreatePO: %v", err)
	}
	if !po.TanpaPo || po.Tier != "none" {
		t.Fatalf("expected tanpaPo/none, got tanpaPo=%v tier=%q", po.TanpaPo, po.Tier)
	}
	if po.Status != "approved" {
		t.Fatalf("tanpaPo PO should auto-approve, status=%q", po.Status)
	}
	if po.Nomor == "" {
		t.Fatalf("tanpaPo PO should still get a nomor")
	}
	if po.PRNomor != pr.Nomor {
		t.Fatalf("PO prNomor = %q, want %q", po.PRNomor, pr.Nomor)
	}
}

// CreatePO must reject a PO with no PR reference at all — a PO can never be
// created standalone; it always builds on an approved PR (business rule).
func TestCreatePORequiresApprovedPR(t *testing.T) {
	svc := newSvc(t)

	// No PRID at all.
	if _, err := svc.CreatePO(domain.PurchaseOrder{
		Supplier: "Toko Bangunan",
		Items:    []domain.POItem{{Nama: "Paku", Satuan: "kg", Qty: 3, HargaSatuan: 25_000}},
	}, false, "purchasing"); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation for CreatePO without prId, got %v", err)
	}

	// PRID references a PR that is still pending (not yet approved).
	pr, err := svc.CreatePR(domain.PurchaseRequest{
		RequestBy: "Budi",
		Items:     []domain.PRItem{{Nama: "Paku", Satuan: "kg", Qty: 3}},
	}, true, "staff")
	if err != nil {
		t.Fatalf("CreatePR: %v", err)
	}
	if _, err := svc.CreatePO(domain.PurchaseOrder{
		PRID:     pr.ID,
		Supplier: "Toko Bangunan",
		Items:    []domain.POItem{{Nama: "Paku", Satuan: "kg", Qty: 3, HargaSatuan: 25_000}},
	}, false, "purchasing"); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation for CreatePO referencing a non-approved PR, got %v", err)
	}
}
