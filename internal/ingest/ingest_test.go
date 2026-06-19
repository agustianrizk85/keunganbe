package ingest

import "testing"

// Representative rows mirroring the real "Keuangan" spreadsheet structure:
// overlapping akad-detail tabs (deduped), a bank-process pipeline tab, and a
// recap tab that must be skipped.
func sampleSheets() map[string][][]string {
	akadHdr := []string{"No.", "GP", "Proyek", "Nama Konsumen", "Blok", "Tgl Booking", "Tgl DP", "DP", "Plafon KPR", "Cara Bayar", "Nama Bank", "Tgl Akad", "Bulan Akad", "Tahun Akad", "Durasi Booking - Akad", "Nama Sales"}
	rowVersaw := []string{"1", "GP 3", "VERSAW", "Sri Kartika Aji", "D8", "15-Des-2025", "-", "Rp. 0", "Rp. 426.600.000", "KPR", "BSI Otista", "3-Jan-2026", "Januari", "2026", "19", "Erwin"}
	return map[string][][]string{
		// Tab E (GP3 detail)
		"Data Akad GP3 2026": {
			akadHdr,
			rowVersaw,
			{"2", "GP 3", "VERLIM 3 EXT", "Avan Ary", "N1", "8-Jan-2026", "20-Jan-2026", "Rp. 57.000.000", "Rp. 394.600.000", "Cash Keras", "Cash", "23-Jan-2026", "Januari", "2026", "44946", "Ardan"},
		},
		// Tab G (overlapping GP3 detail — rowVersaw duplicated, must dedup)
		"Data Akad GP3 2026 (alt)": {
			akadHdr,
			rowVersaw,
			{"3", "GP 3", "Z HAUZ LIMO", "Budi", "F4", "1-Feb-2026", "10-Feb-2026", "Rp. 30.000.000", "Rp. 600.000.000", "KPR", "BSN Harmoni", "20-Feb-2026", "Februari", "2026", "19", "Doni"},
		},
		// Pipeline tab (booking → akad bank-process flow)
		"GP 2026": {
			{"No", "GP", "Nama Proyek", "Nama Konsumen", "Blok", "Nama Sales", "Cara Bayar", "Status", "Tgl Booking", "Status DP", "Pemberkasan", "Tgl Pengiriman Berkas Ke Bank", "Bank Proses KPR", "Tahap Proses KPR", "Kendala / Masalah Proses KPR", "Tgl Terbit SP3", "Tgl Akad", "Bulan Akad", "STATUS", "Keterangan"},
			{"1", "GP 3", "VERLIM 3 EXT", "Muhtadibillah", "N11", "Oyi", "KPR", "CLOSED", "1-Jan-2026", "SUDAH DP", "Sudah Lengkap", "14-Jan-2026", "BSN Harmoni", "SP3", "", "23-Jan-2026", "23-Jan-2026", "Januari", "Akad", "Sudah AKAD"},
			{"2", "GP 3", "Z HAUZ LIMO", "Aang Abdul", "F4", "Agent Rafli", "KPR", "BOOKING", "8-Mei-2026", "SUDAH DP", "Sudah Lengkap", "", "BSN Harmoni", "Berkas", "Berkas tertahan, BI checking", "", "", "", "Berkas Lengkap", ""},
			{"3", "GP 3", "VERSAW", "Citra", "C2", "Seto", "KPR", "BOOKING", "10-Mei-2026", "BELUM", "", "", "BSI Otista", "", "", "", "", "", "Booking", ""},
			{"4", "GP 3", "VERSAW", "Dewi", "C3", "Seto", "KPR", "Batal", "2-Jan-2026", "BELUM", "", "", "", "", "", "", "", "", "BATAL", "Batal"},
		},
		// Recap/dashboard tab — must be classified as skipped.
		"Rekap 2026": {
			{"PROYEK", "STATUS/AKAD", "STATUS/BOOKING", "TOTAL"},
			{"VERSAW", "10", "2", "12"},
		},
	}
}

func TestIngestClassifyDedupAndFunnel(t *testing.T) {
	res, err := RunSheets(sampleSheets(), Options{FocusYear: 2026})
	if err != nil {
		t.Fatalf("RunSheets: %v", err)
	}
	d := res.Preview

	// 3 distinct akad after deduping the repeated VERSAW row across two tabs.
	if d.Summary.AkadCount != 3 {
		t.Fatalf("akad count = %d, want 3 (deduped)", d.Summary.AkadCount)
	}
	// Plafond: 426.6 + 394.6 + 600 = 1421.2 juta.
	if d.Summary.NilaiAkad != 1421.2 {
		t.Errorf("nilai akad = %v, want 1421.2", d.Summary.NilaiAkad)
	}
	// The 44946 serial-leak duration must be dropped → flagged as an issue.
	if len(res.Issues) == 0 {
		t.Error("expected a data-quality issue for the leaked duration serial")
	}

	// Recap tab skipped; two detail tabs + one pipeline tab classified.
	kinds := map[string]int{}
	for _, s := range res.Sheets {
		kinds[s.Kind]++
	}
	if kinds["akad"] != 2 || kinds["pipeline"] != 1 || kinds["skipped"] != 1 {
		t.Errorf("classification = %+v", kinds)
	}

	// Funnel is descending: Booking ≥ DP ≥ … ≥ Akad, batal excluded.
	if len(d.Funnel) != 6 {
		t.Fatalf("funnel stages = %d, want 6", len(d.Funnel))
	}
	booking, akad := d.Funnel[0].Count, d.Funnel[5].Count
	if booking != 3 { // 4 pipeline rows minus 1 batal
		t.Errorf("booking funnel = %d, want 3", booking)
	}
	if akad != 1 {
		t.Errorf("akad funnel = %d, want 1", akad)
	}
	for i := 1; i < len(d.Funnel); i++ {
		if d.Funnel[i].Count > d.Funnel[i-1].Count {
			t.Errorf("funnel not descending at %s", d.Funnel[i].Label)
		}
	}

	// Active pipeline list excludes the akad + batal rows (2 active bookings).
	if d.Summary.BookingCount != 2 {
		t.Errorf("active bookings = %d, want 2", d.Summary.BookingCount)
	}
	// The booking with a kendala should surface as overdue and sort first.
	if len(d.Pipeline) == 0 || d.Pipeline[0].Kendala == "" {
		t.Errorf("expected the kendala booking first, got %+v", d.Pipeline)
	}
	if d.Summary.BatalCount != 1 {
		t.Errorf("batal = %d, want 1", d.Summary.BatalCount)
	}
}
