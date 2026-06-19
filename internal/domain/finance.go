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
