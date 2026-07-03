package ingest

import "testing"

// prSheets mirrors the three flat tabs of the "Pembelian (PR)" spreadsheet,
// prefixed as the akad sync merges them ("PR::TabTitle").
func prSheets() map[string][][]string {
	po := [][]string{
		{"Tanggal", "No PO", "Pemasok", "Kode Barang", "Nama Barang", "Kuantitas", "Satuan", "Harga Satuan", "Diskon", "Total Harga", "Blok", "Proyek"},
		{"2026-01-05", "PO-2601-001", "PT Sumber Bangunan", "SMN-40", "Semen 40kg", "500", "Sak", "62000", "0", "31000000", "A1", "Verlim"},
		{"2026-02-03", "PO-2602-001", "PT Cat Indah", "CAT-20", "Cat Tembok 20kg", "80", "Pail", "410000", "0", "32800000", "B1", "Mahaba"},
		{"2026-02-18", "PO-2602-002", "Toko Besi Makmur", "BSI-12", "Besi Beton 12mm", "250", "Btg", "138000", "0", "34500000", "C3", "ZHL"},
	}
	inv := [][]string{
		{"Tanggal", "No Faktur", "No PO", "Pemasok", "Kode Barang", "Nama Barang", "Kuantitas", "Harga Satuan", "Diskon", "Total Harga", "Blok", "Proyek"},
		{"2026-01-10", "FB-2601-001", "PO-2601-001", "PT Sumber Bangunan", "SMN-40", "Semen 40kg", "500", "62000", "0", "31000000", "A1", "Verlim"},
		{"2026-02-09", "FB-2602-001", "PO-2602-001", "PT Cat Indah", "CAT-20", "Cat Tembok 20kg", "80", "410000", "0", "32800000", "B1", "Mahaba"},
	}
	pay := [][]string{
		{"Tgl Bayar", "No Bukti", "Pemasok", "Bank", "No Faktur", "Total Faktur", "Terutang", "Diskon", "Bayar"},
		{"2026-01-20", "BKK-2601-001", "PT Sumber Bangunan", "BCA", "FB-2601-001", "31000000", "0", "0", "31000000"},
		{"2026-02-19", "BKK-2602-001", "PT Cat Indah", "Mandiri", "FB-2602-001", "32800000", "12800000", "0", "20000000"},
	}
	return map[string][][]string{
		"PR::Pesanan Pembelian":    po,
		"PR::Faktur Pembelian":     inv,
		"PR::Pembayaran Pembelian": pay,
	}
}

func TestPurchasingClassifyAndAggregate(t *testing.T) {
	res, err := RunSheets(prSheets(), Options{FocusYear: 2026})
	if err != nil {
		t.Fatalf("RunSheets: %v", err)
	}

	// Each PR tab must be classified to its kind, not skipped.
	kinds := map[string]string{}
	for _, s := range res.Sheets {
		kinds[s.Name] = s.Kind
	}
	want := map[string]string{
		"PR::Pesanan Pembelian":    "pr_po",
		"PR::Faktur Pembelian":     "pr_invoice",
		"PR::Pembayaran Pembelian": "pr_payment",
	}
	for name, k := range want {
		if kinds[name] != k {
			t.Errorf("tab %q classified as %q, want %q", name, kinds[name], k)
		}
	}

	pu := res.Preview.Purchasing.Summary
	if pu.POCount != 3 {
		t.Errorf("POCount = %d, want 3", pu.POCount)
	}
	if got := pu.POValue; got != 98.3 { // 31 + 32.8 + 34.5
		t.Errorf("POValue = %v jt, want 98.3", got)
	}
	if pu.InvoiceCount != 2 {
		t.Errorf("InvoiceCount = %d, want 2", pu.InvoiceCount)
	}
	if got := pu.PaidValue; got != 51 { // 31 + 20
		t.Errorf("PaidValue = %v jt, want 51", got)
	}
	if got := pu.Outstanding; got != 12.8 {
		t.Errorf("Outstanding = %v jt, want 12.8", got)
	}
	if pu.SupplierCount != 3 {
		t.Errorf("SupplierCount = %d, want 3", pu.SupplierCount)
	}
	if pu.TopSupplier != "Toko Besi Makmur" { // PO 34.5, the largest
		t.Errorf("TopSupplier = %q, want Toko Besi Makmur", pu.TopSupplier)
	}

	// Monthly trend must resolve ISO dates to months (Jan + Feb).
	if n := len(res.Preview.Purchasing.Monthly); n < 2 {
		t.Errorf("Monthly months = %d, want >= 2", n)
	}
}

// TestRunPRSheets checks the standalone procurement runner (used by the
// independent /api/purchasing sync) produces the same figures as the akad path
// and reports per-tab classification + non-nil slices.
func TestRunPRSheets(t *testing.T) {
	res := RunPRSheets(prSheets())

	pu := res.Purchasing.Summary
	if pu.POCount != 3 || pu.POValue != 98.3 {
		t.Errorf("PO = {count %d, value %v jt}, want {3, 98.3}", pu.POCount, pu.POValue)
	}
	if pu.Outstanding != 12.8 {
		t.Errorf("Outstanding = %v jt, want 12.8", pu.Outstanding)
	}
	if pu.TopSupplier != "Toko Besi Makmur" {
		t.Errorf("TopSupplier = %q, want Toko Besi Makmur", pu.TopSupplier)
	}
	if len(res.Sheets) != 3 {
		t.Errorf("Sheets = %d, want 3 classified tabs", len(res.Sheets))
	}
	// Slices must serialise as [] not null.
	if res.Issues == nil || res.Purchasing.BySupplier == nil {
		t.Error("nil slice would serialise to null")
	}
}
