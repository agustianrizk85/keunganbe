package ingest

import "testing"

func TestRunARSheets(t *testing.T) {
	data := map[string][][]string{
		"LHL::REKAP DANA MASUK": {
			{"No.", "Waktu Transaksi", "Nama Pengirim", "Bank Pengirim", "Nama Penerima", "No. Rekening Penerima", "Bank Penerima", "Blok", "No. Unit", "Deskripsi", "Nominal", "Keterangan"},
			{"1", "2026-01-12 00:00:00", "BUDI", "BCA", "PT", "123", "BSI", "B", "1", "PEMBAYARAN BOOKING FEE", "3000000", ""},
			{"2", "2026-01-20 00:00:00", "BUDI", "BSI", "PT", "123", "BTNS", "B", "1", "PENCAIRAN KPR TAHAP 1 (AKAD)", "200000000", ""},
			{"3", "2026-02-05 00:00:00", "BUDI", "BSI", "PT", "123", "BTNS", "B", "1", "PENCAIRAN KPR TAHAP 2 (PONDASI)", "130000000", ""},
			{"4", "45901", "BUDI", "BSI", "PT", "123", "BTNS", "B", "1", "PENCAIRAN KPR TAHAP 3 (ATAP)", "270000000", ""}, // serial date in 2025
			{"", "", "", "", "", "", "", "", "", "", "", ""}, // spacer
		},
		"LHL::DP": {
			{"NO.", "NAMA KONSUMEN", "BLOK", "NO. UNIT", "TYPE", "NAMA SALES", "TGL BOOKING", "DOWN PAYMENT", "PEMBAYARAN DOWN PAYMENT", "TOTAL PEMBAYARAN KONSUMEN", "SISA PEMBAYARAN KONSUMEN", "STATUS", "TGL JATUH TEMPO", "UMUR PIUTANG DP (HARI)", "KET"},
			{"1", "BUDI", "B", "1", "JADE", "ARDAN", "x", "10000000", "2000000", "2000000", "8000000", "SUDAH JATUH TEMPO", "x", "45", ""},
			{"2", "SITI", "B", "2", "JADE", "ARDAN", "x", "10000000", "10000000", "10000000", "0", "BELUM JATUH TEMPO", "x", "0", ""}, // lunas → skip
		},
		"LHL::LAPORAN": {
			{"NO", "BLOK", "NO. UNIT", "TYPE", "NAMA KONSUMEN", "STATUS UNIT", "KET", "TGL BOOKING", "NAMA SALES", "CARA BAYAR", "NAMA BANK", "TGL DP", "TOTAL DP", "DP DIBAYARKAN", "PIUTANG DP", "PLAFON KPR", "HARGA JUAL", "TGL", "TAHAP 1", "TOTAL PEMBAYARAN", "SISA PIUTANG", "STATUS", "TGL AKAD", "STATUS AKAD", "TGL BAST", "STATUS BAST"},
			{"1", "B", "1", "JADE", "BUDI", "SOLD", "", "x", "ARDAN", "KPR", "BSI", "x", "10000000", "10000000", "0", "700000000", "710000000", "x", "200000000", "630000000", "80000000", "BELUM LUNAS", "x", "SUDAH AKAD", "", "BELUM BAST"},
			{"2", "B", "2", "JADE", "SITI", "SOLD", "", "x", "ARDAN", "CASH", "CASH", "x", "0", "0", "0", "0", "510000000", "x", "510000000", "510000000", "0", "LUNAS", "x", "SUDAH AKAD", "", "BELUM BAST"},
			{"", "", "TOTAL", "", "", "", "", "", "", "", "", "", "", "", "", "", "1210000000", "", "", "1140000000", "80000000", "", "", "", "", ""}, // total row → skip
		},
		"LHL::SPK":            {{"No.", "No. SPK", "Tanggal SPK", "Blok"}, {"1", "x", "x", "B"}}, // skipped (no AR signature)
		"LHL::DATA PENJUALAN": {{"NO", "UNIT", "BLOK", "TYPE", "PLAFON KPR", "TGL AKAD"}, {"1", "x", "B", "JADE", "700000000", "x"}}, // skipped for AR
	}

	d := RunARSheets(data, 2026)

	// Summary from LAPORAN
	if got := d.Summary.NilaiKontrak; got != 1_220_000_000 {
		t.Errorf("NilaiKontrak = %.0f, want 1220000000", got)
	}
	if got := d.Summary.SisaPiutang; got != 80_000_000 {
		t.Errorf("SisaPiutang = %.0f, want 80000000", got)
	}
	if d.Summary.UnitLunas != 1 || d.Summary.UnitBelumLunas != 1 || d.Summary.UnitTotal != 2 {
		t.Errorf("units lunas/belum/total = %d/%d/%d, want 1/1/2", d.Summary.UnitLunas, d.Summary.UnitBelumLunas, d.Summary.UnitTotal)
	}
	// Cash-in focus year 2026 = 3M + 200M + 130M (serial row is 2025, excluded from focus-year cashIn but added as undated? no—it parses ok as 2025)
	// 45901 → 2025-09-xx, so it's a dated non-focus row → NOT added to cashIn. Focus cashIn = 333M.
	if got := d.Summary.CashIn; got != 333_000_000 {
		t.Errorf("CashIn(2026) = %.0f, want 333000000", got)
	}
	// PencairanKpr: focus-year pencairan only (200M akad + 130M pondasi); the 270M atap is 2025.
	if got := d.Summary.PencairanKpr; got != 330_000_000 {
		t.Errorf("PencairanKpr(2026) = %.0f, want 330000000", got)
	}
	// Tahapan accumulates across all years (akad 200M, pondasi 130M, atap 270M).
	if len(d.Tahapan) != 3 {
		t.Errorf("Tahapan stages = %d, want 3 (akad,pondasi,atap)", len(d.Tahapan))
	}
	// DP aging: one overdue row, 45 days → "31–90", sisa 8M; lunas row skipped.
	if d.Summary.DpJatuhTempo != 1 || d.Summary.DpSisa != 8_000_000 {
		t.Errorf("DP jatuhTempo/sisa = %d/%.0f, want 1/8000000", d.Summary.DpJatuhTempo, d.Summary.DpSisa)
	}
	if len(d.Aging) != 1 || d.Aging[0].Count != 1 {
		t.Errorf("Aging = %+v, want 1 bucket count 1", d.Aging)
	}
	// Monthly: Jan 2026 = 203M, Feb 2026 = 130M.
	if len(d.Monthly) != 2 {
		t.Fatalf("Monthly months = %d, want 2", len(d.Monthly))
	}
	if d.Monthly[0].Period != "2026-01" || d.Monthly[0].CashIn != 203_000_000 {
		t.Errorf("Monthly[0] = %+v, want 2026-01 / 203000000", d.Monthly[0])
	}
	// Watch-list: one row with sisa>0 (Budi).
	if len(d.Piutang) != 1 || d.Piutang[0].Customer != "BUDI" {
		t.Errorf("Piutang = %+v, want 1 row Budi", d.Piutang)
	}
	// Project rollup present for LHL.
	if len(d.Projects) != 1 || d.Projects[0].Code != "LHL" {
		t.Errorf("Projects = %+v, want 1 (LHL)", d.Projects)
	}
}
