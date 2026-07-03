// Package domain holds the core business entities of the Finance (keuangan)
// control dashboard. These types are the single source of truth for the data
// shape and carry no dependency on transport or storage concerns.
//
// The dashboard is driven by the akad/KPR closing pipeline: every figure is
// derived from the transaction-grain akad and booking rows ingested from the
// "Keuangan" Google Spreadsheet (down payment + KPR plafond per deal, by
// project, bank, sales and month). All monetary values are expressed in
// millions of Rupiah (Rp juta) unless noted otherwise, keeping the numbers
// compact for the executive war-room view.
package domain

// Status is the common traffic-light health indicator used across the domain.
type Status string

const (
	StatusGreen  Status = "green"  // healthy
	StatusYellow Status = "yellow" // at risk
	StatusRed    Status = "red"    // critical
)

// Summary holds the executive KPIs derived from the akad/booking data set for
// the focused year.
type Summary struct {
	NilaiAkad     float64 `json:"nilaiAkad"`     // Σ plafond of closed akad (Rp juta)
	CashIn        float64 `json:"cashIn"`        // Σ down payment collected (Rp juta)
	PipelineValue float64 `json:"pipelineValue"` // Σ plafond of active bookings not yet akad (Rp juta)
	AkadCount     int     `json:"akadCount"`     // number of akad (closings)
	BookingCount  int     `json:"bookingCount"`  // active bookings still in pipeline
	ProsesCount   int     `json:"prosesCount"`   // deals in bank process
	BatalCount    int     `json:"batalCount"`    // cancelled deals
	CancelRate    int     `json:"cancelRate"`    // batal / (akad + batal + proses) %
	AvgDurasi     int     `json:"avgDurasi"`     // average booking→akad days
	KprShare      int     `json:"kprShare"`      // % of akad financed via KPR
	TargetAkad    int     `json:"targetAkad"`    // akad target for the focus year
	Achievement   int     `json:"achievement"`   // akad / target %
	TopProject    string  `json:"topProject"`    // project with the highest akad value
	TopBank       string  `json:"topBank"`       // bank with the most akad
	BankCount     int     `json:"bankCount"`     // distinct financing banks used
}

// FunnelStage is one step of the booking→akad pipeline funnel (counts only —
// the pipeline sheet carries no amounts).
type FunnelStage struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

// MonthPoint is one month of the akad trend, doubling as the cash-in proxy.
type MonthPoint struct {
	Period string  `json:"period"` // Indonesian month short label (Jan…Des)
	Akad   int     `json:"akad"`   // number of akad in the month
	Nilai  float64 `json:"nilai"`  // Σ plafond (Rp juta)
	DP     float64 `json:"dp"`     // Σ down payment (Rp juta)
}

// ProjectFin is the akad performance of a single project.
type ProjectFin struct {
	Code    string  `json:"code"`
	Name    string  `json:"name"`
	GP      string  `json:"gp"`      // GP group (GP 1..4)
	Akad    int     `json:"akad"`    // closed akad
	Booking int     `json:"booking"` // active bookings
	Batal   int     `json:"batal"`   // cancelled
	Nilai   float64 `json:"nilai"`   // Σ plafond akad (Rp juta)
	DP      float64 `json:"dp"`      // Σ down payment (Rp juta)
	KprPct  int     `json:"kprPct"`  // % akad via KPR
	TopBank string  `json:"topBank"`
	Status  Status  `json:"status"`
	Note    string  `json:"note"`
}

// BankFin is the KPR financing volume routed through one bank (the "Pendanaan"
// view — which bank carries the mortgage book).
type BankFin struct {
	Name   string  `json:"name"`
	Akad   int     `json:"akad"`   // akad financed by this bank
	Plafon float64 `json:"plafon"` // Σ plafond (Rp juta)
	Share  int     `json:"share"`  // % of total KPR plafond
	Status Status  `json:"status"`
}

// SalesRank is the akad contribution of a single sales person or agent.
type SalesRank struct {
	Name    string  `json:"name"`
	Akad    int     `json:"akad"`
	Nilai   float64 `json:"nilai"` // Σ plafond (Rp juta)
	IsAgent bool    `json:"isAgent"`
}

// PayMethod is one payment-scheme bucket (KPR / Cash Keras / Cash Bertahap).
type PayMethod struct {
	Type  string  `json:"type"`
	Count int     `json:"count"`
	Value float64 `json:"value"` // Σ plafond (Rp juta)
}

// PipelineRow is an active deal still moving through the bank process — the
// early-warning list (stuck berkas, KPR problems, slow stages).
type PipelineRow struct {
	Project   string  `json:"project"`
	Customer  string  `json:"customer"`
	Blok      string  `json:"blok"`
	Sales     string  `json:"sales"`
	Bank      string  `json:"bank"`
	CaraBayar string  `json:"caraBayar"`
	Stage     string  `json:"stage"`     // current pipeline stage
	StageKey  string  `json:"stageKey"`  // booking|dp|berkas|bank|sp3
	Plafon    float64 `json:"plafon"`    // expected plafond if known (Rp juta)
	SLA       string  `json:"sla"`       // ok | due | overdue
	Kendala   string  `json:"kendala"`   // free-text KPR problem note
}

// AkadRow is a single closed transaction (akad) at detail grain.
type AkadRow struct {
	GP        string  `json:"gp"`
	Project   string  `json:"project"`
	Customer  string  `json:"customer"`
	Blok      string  `json:"blok"`
	Sales     string  `json:"sales"`
	Bank      string  `json:"bank"`
	CaraBayar string  `json:"caraBayar"`
	DP        float64 `json:"dp"`     // Rp juta
	Plafon    float64 `json:"plafon"` // Rp juta
	TglAkad   string  `json:"tglAkad"`
	Bulan     string  `json:"bulan"`
	Tahun     int     `json:"tahun"`
	Durasi    int     `json:"durasi"` // booking→akad days
}

// PurchaseDoc is one line item of a purchase order (PO) or purchase invoice
// (faktur pembelian), at item grain. Money fields are in Rp juta.
type PurchaseDoc struct {
	Doc     string  `json:"doc"`     // "po" | "invoice"
	Tanggal string  `json:"tanggal"` // document date (raw)
	Nomor   string  `json:"nomor"`   // No PO / No Faktur
	Pemasok string  `json:"pemasok"` // supplier
	Kode    string  `json:"kode"`    // item code
	Barang  string  `json:"barang"`  // item name
	Qty     float64 `json:"qty"`     // quantity (count, not money)
	Satuan  string  `json:"satuan"`  // unit of measure
	Diskon  float64 `json:"diskon"`  // line discount (Rp juta)
	Total   float64 `json:"total"`   // line total (Rp juta)
	Blok    string  `json:"blok"`
	Proyek  string  `json:"proyek"`
	Bulan   string  `json:"bulan"`
	Tahun   int     `json:"tahun"`
}

// PaymentDoc is one purchase payment (pembayaran pembelian) at faktur grain.
// Money fields are in Rp juta.
type PaymentDoc struct {
	Tanggal     string  `json:"tanggal"`     // payment date (raw)
	NoBukti     string  `json:"noBukti"`     // payment voucher number
	Pemasok     string  `json:"pemasok"`     // supplier paid
	Bank        string  `json:"bank"`        // cash/bank account
	NoFaktur    string  `json:"noFaktur"`    // invoice being paid
	TotalFaktur float64 `json:"totalFaktur"` // invoice total (Rp juta)
	Terutang    float64 `json:"terutang"`    // outstanding after this payment (Rp juta)
	Bayar       float64 `json:"bayar"`       // amount paid now (Rp juta)
	Bulan       string  `json:"bulan"`
	Tahun       int     `json:"tahun"`
}

// PurchasingSummary holds the headline procurement (PR) figures.
type PurchasingSummary struct {
	POValue       float64 `json:"poValue"`       // Σ total of purchase orders (Rp juta)
	POCount       int     `json:"poCount"`       // distinct purchase orders
	InvoiceValue  float64 `json:"invoiceValue"`  // Σ total of invoices (Rp juta)
	InvoiceCount  int     `json:"invoiceCount"`  // distinct invoices
	PaidValue     float64 `json:"paidValue"`     // Σ paid (Rp juta)
	PaymentCount  int     `json:"paymentCount"`  // payment vouchers
	Outstanding   float64 `json:"outstanding"`   // Σ outstanding / hutang (Rp juta)
	SupplierCount int     `json:"supplierCount"` // distinct suppliers
	TopSupplier   string  `json:"topSupplier"`   // supplier with the largest PO spend
}

// SupplierSpend is purchase spend grouped by one supplier.
type SupplierSpend struct {
	Name        string  `json:"name"`
	POValue     float64 `json:"poValue"`     // Rp juta
	Invoiced    float64 `json:"invoiced"`    // Rp juta
	Paid        float64 `json:"paid"`        // Rp juta
	Outstanding float64 `json:"outstanding"` // Rp juta (hutang)
	Docs        int     `json:"docs"`        // PO line items
}

// ProjectSpend is purchase spend (PO) grouped by one project.
type ProjectSpend struct {
	Project string  `json:"project"`
	POValue float64 `json:"poValue"` // Rp juta
	Items   int     `json:"items"`   // PO line items
}

// PurchaseMonth is one month of the procurement trend.
type PurchaseMonth struct {
	Period  string  `json:"period"`  // Indonesian month short label
	PO      float64 `json:"po"`      // Σ PO total (Rp juta)
	Invoice float64 `json:"invoice"` // Σ invoice total (Rp juta)
	Paid    float64 `json:"paid"`    // Σ paid (Rp juta)
}

// Purchasing is the procurement (PR) section of the dashboard, fed by the
// "Pembelian (PR)" spreadsheet (PO + faktur + pembayaran tabs). It is synced
// independently of the akad dashboard (its own sync-preview/approve flow), so it
// carries its own Updated stamp.
type Purchasing struct {
	Updated    string            `json:"updated"` // last procurement sync stamp (Indonesian date)
	Summary    PurchasingSummary `json:"summary"`
	BySupplier []SupplierSpend   `json:"bySupplier"`
	ByProject  []ProjectSpend    `json:"byProject"`
	Monthly    []PurchaseMonth   `json:"monthly"`
	Orders     []PurchaseDoc     `json:"orders"`   // recent purchase orders
	Invoices   []PurchaseDoc     `json:"invoices"` // recent invoices
	Payments   []PaymentDoc      `json:"payments"` // recent payments
}

// EmptyPurchasing returns a zero-data Purchasing with non-nil slices, so the JSON
// serialises arrays as "[]" (not null) for safe front-end iteration.
func EmptyPurchasing() Purchasing {
	return Purchasing{
		BySupplier: []SupplierSpend{},
		ByProject:  []ProjectSpend{},
		Monthly:    []PurchaseMonth{},
		Orders:     []PurchaseDoc{},
		Invoices:   []PurchaseDoc{},
		Payments:   []PaymentDoc{},
	}
}

// IsEmpty reports whether the procurement view holds no documents yet (used to
// decide whether a stored view should override the dashboard's own section).
func (p Purchasing) IsEmpty() bool {
	return len(p.Orders) == 0 && len(p.Invoices) == 0 && len(p.Payments) == 0 &&
		len(p.BySupplier) == 0 && p.Summary.POCount == 0 && p.Summary.PaymentCount == 0
}

// Alert is a derived early-warning item for the war-room.
type Alert struct {
	Tone   string `json:"tone"` // red | orange | yellow | green
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Action string `json:"action"`
}

// AIInsight is a generated insight or recommendation for today's war-room.
type AIInsight struct {
	Type string `json:"type"`
	Tone string `json:"tone"`
	Text string `json:"text"`
	Icon string `json:"icon"`
}

// Decision is an action required from a specific role.
type Decision struct {
	Role string `json:"role"`
	Tone string `json:"tone"`
	Text string `json:"text"`
}

// KPI is a reference row describing an indicator, its thresholds and value.
type KPI struct {
	No     int    `json:"no"`
	KPI    string `json:"kpi"`
	Def    string `json:"def"`
	PIC    string `json:"pic"`
	Green  string `json:"green"`
	Yellow string `json:"yellow"`
	Red    string `json:"red"`
	Val    string `json:"val"`
	State  string `json:"state"`
}

// Trigger is an early-warning rule with a threshold, action and escalation path.
type Trigger struct {
	Cond   string `json:"cond"`
	Thr    string `json:"thr"`
	Status string `json:"status"`
	PIC    string `json:"pic"`
	Act    string `json:"act"`
	Esc    string `json:"esc"`
}

// Dashboard is the full payload consumed by the front-end in a single call.
type Dashboard struct {
	Period    string        `json:"period"`    // human label, e.g. "Akad 2026 · Jan–Jun"
	Updated   string        `json:"updated"`   // last data refresh stamp
	FocusYear int           `json:"focusYear"` // year the summary/aggregations focus on
	Years     []int         `json:"years"`     // years available for filtering (desc)
	Summary   Summary       `json:"summary"`
	Funnel    []FunnelStage `json:"funnel"`
	Monthly   []MonthPoint  `json:"monthly"`
	Projects  []ProjectFin  `json:"projects"`
	Banks     []BankFin     `json:"banks"`
	Sales     []SalesRank   `json:"sales"`
	PayMix    []PayMethod   `json:"payMix"`
	Pipeline  []PipelineRow `json:"pipeline"`
	Akads     []AkadRow     `json:"akads"`
	Alerts    []Alert       `json:"alerts"`
	AI        []AIInsight   `json:"ai"`
	Decisions []Decision    `json:"decisions"`
	KPIs      []KPI         `json:"kpis"`
	Triggers  []Trigger     `json:"triggers"`

	// Purchasing (PR) section — procurement spend from the "Pembelian (PR)"
	// spreadsheet. Empty when no PR sheet is configured/synced.
	Purchasing Purchasing `json:"purchasing"`
}

// ImportSummary is the headline of one import, shown in the preview and history.
type ImportSummary struct {
	AkadCount    int     `json:"akadCount"`
	NilaiAkad    float64 `json:"nilaiAkad"` // Rp juta
	CashIn       float64 `json:"cashIn"`    // Rp juta
	BookingCount int     `json:"bookingCount"`
	ProsesCount  int     `json:"prosesCount"`
	BatalCount   int     `json:"batalCount"`
	KprShare     int     `json:"kprShare"`
	Issues       int     `json:"issues"`

	// Procurement (PR) headline — 0 when no PR sheet is part of the import.
	PurchaseValue float64 `json:"purchaseValue"` // Σ PO value (Rp juta)
	Outstanding   float64 `json:"outstanding"`   // Σ hutang (Rp juta)
}

// ImportRecord is one entry of the import history (rollback-able).
type ImportRecord struct {
	ID       string        `json:"id"`
	Time     string        `json:"time"`
	Filename string        `json:"filename"`
	By       string        `json:"by"`
	Summary  ImportSummary `json:"summary"`
}

// Role enumerates the access levels for a dashboard user.
type Role string

const (
	RoleAdmin  Role = "admin"  // full ingest/admin access
	RoleViewer Role = "viewer" // read-only dashboard access
)

// User is a dashboard account. Password material is never serialised to clients
// (json:"-"); it only lives in the persisted store.
type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	Name         string `json:"name"`
	Role         Role   `json:"role"`
	PasswordHash string `json:"-"`
	Salt         string `json:"-"`
}
