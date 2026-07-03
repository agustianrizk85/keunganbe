package domain

// AR (Accounts Receivable / Piutang) is a separate view from the akad/KPR closing
// tracker. It is fed by the per-project AR input spreadsheets (DATA PENJUALAN /
// REKAP DANA MASUK / DP / LAPORAN tabs) and answers the finance team's piutang
// questions: how much KPR has disbursed per stage, DP aging, remaining receivable
// vs. contract value, and monthly cash-in by bank.
//
// Unlike the akad dashboard (Rp juta), AR figures are kept in full Rupiah and
// formatted on the front-end, since piutang reconciliation is done to the rupiah.

// ARSource is one project's AR input spreadsheet (project code → spreadsheet ID).
// These are configurable at runtime from the UI and persisted in the store, so
// the finance team can add a project's sheet without an env change/restart.
type ARSource struct {
	Code string `json:"code"`
	ID   string `json:"id"`
}

// ARSummary holds the headline AR KPIs for the focused year.
type ARSummary struct {
	NilaiKontrak   float64 `json:"nilaiKontrak"`   // Σ harga jual (LAPORAN)
	TotalTerbayar  float64 `json:"totalTerbayar"`  // Σ total pembayaran konsumen
	SisaPiutang    float64 `json:"sisaPiutang"`    // Σ sisa piutang
	ProgresPct     int     `json:"progresPct"`     // terbayar / kontrak %
	UnitTotal      int     `json:"unitTotal"`      // unit terjual (baris LAPORAN berisi)
	UnitLunas      int     `json:"unitLunas"`      // unit dengan sisa piutang 0
	UnitBelumLunas int     `json:"unitBelumLunas"` // unit dengan sisa piutang > 0
	CashIn         float64 `json:"cashIn"`         // Σ dana masuk (tahun fokus)
	PencairanKpr   float64 `json:"pencairanKpr"`   // Σ pencairan KPR (tahun fokus)
	DpJatuhTempo   int     `json:"dpJatuhTempo"`   // jumlah DP sudah jatuh tempo & belum lunas
	DpSisa         float64 `json:"dpSisa"`         // Σ sisa pembayaran DP
}

// ARTahap is one KPR disbursement stage (Akad / Pondasi / Atap / BAST).
type ARTahap struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	Count int     `json:"count"`
	Nilai float64 `json:"nilai"`
}

// ARAging is one DP aging bucket.
type ARAging struct {
	Bucket string  `json:"bucket"`
	Count  int     `json:"count"`
	Nilai  float64 `json:"nilai"` // Σ sisa DP in the bucket
}

// ARMonth is one month of cash-in (all dana masuk) and KPR disbursement.
type ARMonth struct {
	Period string  `json:"period"` // YYYY-MM
	CashIn float64 `json:"cashIn"`
	Kpr    float64 `json:"kpr"`
}

// ARBank is cash-in volume routed to one receiving bank account.
type ARBank struct {
	Name  string  `json:"name"`
	Count int     `json:"count"`
	Nilai float64 `json:"nilai"`
}

// ARProject is the receivable roll-up for one project.
type ARProject struct {
	Code          string  `json:"code"`
	NilaiKontrak  float64 `json:"nilaiKontrak"`
	TotalTerbayar float64 `json:"totalTerbayar"`
	SisaPiutang   float64 `json:"sisaPiutang"`
	CashIn        float64 `json:"cashIn"`
	Unit          int     `json:"unit"`
	Lunas         int     `json:"lunas"`
	ProgresPct    int     `json:"progresPct"`
}

// ARPiutangRow is one unit's outstanding receivable (for the watch-list).
type ARPiutangRow struct {
	Project   string  `json:"project"`
	Customer  string  `json:"customer"`
	Blok      string  `json:"blok"`
	Bank      string  `json:"bank"`
	HargaJual float64 `json:"hargaJual"`
	Terbayar  float64 `json:"terbayar"`
	Sisa      float64 `json:"sisa"`
	Status    string  `json:"status"`
}

// ARSheetInfo reports how one project tab was classified during ingest.
type ARSheetInfo struct {
	Project string `json:"project"`
	Tab     string `json:"tab"`
	Kind    string `json:"kind"` // danamasuk | dp | laporan | skipped
	Rows    int    `json:"rows"`
}

// ARData is the full AR payload consumed by the front-end in one call.
type ARData struct {
	Period    string         `json:"period"`
	Updated   string         `json:"updated"`
	FocusYear int            `json:"focusYear"`
	Summary   ARSummary      `json:"summary"`
	Tahapan   []ARTahap      `json:"tahapan"`
	Aging     []ARAging      `json:"aging"`
	Monthly   []ARMonth      `json:"monthly"`
	Banks     []ARBank       `json:"banks"`
	Projects  []ARProject    `json:"projects"`
	Piutang   []ARPiutangRow `json:"piutang"`
	Sheets    []ARSheetInfo  `json:"sheets"`
}

// EmptyARData is the fresh, no-data AR payload (non-nil slices so JSON stays
// "[]" not "null").
func EmptyARData() ARData {
	return ARData{
		Period:    "Belum ada data AR — sync spreadsheet input per proyek",
		FocusYear: 2026,
		Tahapan:   []ARTahap{},
		Aging:     []ARAging{},
		Monthly:   []ARMonth{},
		Banks:     []ARBank{},
		Projects:  []ARProject{},
		Piutang:   []ARPiutangRow{},
		Sheets:    []ARSheetInfo{},
	}
}
