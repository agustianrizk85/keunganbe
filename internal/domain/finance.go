// Package domain holds the core business entities of the Finance (keuangan)
// control dashboard. These types are the single source of truth for the data
// shape and carry no dependency on transport or storage concerns.
//
// All monetary values are expressed in millions of Rupiah (Rp juta) unless
// noted otherwise, keeping the numbers compact for the executive war-room view.
//
// Collection entities carry a synthetic _id (EntID) — the stable handle the
// master-data admin uses to update/delete a row, independent of any editable
// business field.
package domain

// Status is the common traffic-light health indicator used across the domain.
type Status string

const (
	StatusGreen  Status = "green"  // healthy
	StatusYellow Status = "yellow" // at risk
	StatusRed    Status = "red"    // critical
)

// Project is the financial view of a construction project: budget vs cost,
// sales revenue, cash collected and the resulting margin.
type Project struct {
	EntID     string  `json:"_id"`
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Units     int     `json:"units"`
	Budget    float64 `json:"budget"`    // RAB / planned cost (Rp juta)
	Spent     float64 `json:"spent"`     // realised cost to date (Rp juta)
	Revenue   float64 `json:"revenue"`   // contracted sales value (Rp juta)
	Collected float64 `json:"collected"` // cash collected from buyers (Rp juta)
	Margin    float64 `json:"margin"`    // projected gross margin %
	Status    Status  `json:"status"`
	PIC       string  `json:"pic"`
	CashNote  string  `json:"cashNote"` // cash-flow note / recovery action
	Decision  string  `json:"decision"`
}

// Receivable is an outstanding customer collection (piutang) tracked by aging.
type Receivable struct {
	EntID    string  `json:"_id"`
	ID       string  `json:"id"`
	Project  string  `json:"project"`
	Customer string  `json:"customer"`
	Type     string  `json:"type"`   // kpr | cash | dp — key into ReceivableTypeMeta
	Amount   float64 `json:"amount"` // outstanding amount (Rp juta)
	Aging    int     `json:"aging"`  // days overdue (0 = current)
	Bucket   string  `json:"bucket"` // current | d30 | d60 | d90 — key into AgingMeta
	SLA      string  `json:"sla"`    // ok | due | overdue
	Owner    string  `json:"owner"`
	Next     string  `json:"next"`
}

// Payable is an outstanding vendor/contractor obligation (hutang) due for payment.
type Payable struct {
	EntID    string  `json:"_id"`
	ID       string  `json:"id"`
	Vendor   string  `json:"vendor"`
	Project  string  `json:"project"`
	Category string  `json:"category"` // termin | material | upah | overhead
	Amount   float64 `json:"amount"`   // amount due (Rp juta)
	DueDays  int     `json:"dueDays"`  // days until due (negative = overdue)
	Priority string  `json:"priority"` // high | med | low — key into PriorityMeta
	Status   string  `json:"status"`   // ok | due | overdue
	Note     string  `json:"note"`
}

// Facility is a funding source: a bank facility or KPR (mortgage) pipeline.
type Facility struct {
	EntID   string  `json:"_id"`
	Name    string  `json:"name"`
	Type    string  `json:"type"`    // KI | KMK | KPR | Equity
	Plafond float64 `json:"plafond"` // facility ceiling (Rp juta)
	Used    float64 `json:"used"`    // drawn / utilised (Rp juta)
	Rate    float64 `json:"rate"`    // interest rate %
	Tenor   string  `json:"tenor"`
	Status  Status  `json:"status"`
}

// MetaItem describes a classification (receivable type, aging bucket, priority)
// and its tone/notes. Modelled as an ordered slice element so the JSON order is
// stable. Edited as a whole array by the admin.
type MetaItem struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Tone  string `json:"tone"`
	Note  string `json:"note,omitempty"`
	SLA   string `json:"sla,omitempty"`
}

// CostCategory is a budget-vs-actual breakdown of a cost line.
type CostCategory struct {
	Name   string  `json:"name"`
	Budget float64 `json:"budget"` // Rp juta
	Actual float64 `json:"actual"` // Rp juta
}

// Treasury holds the base cash position figures used to derive liquidity KPIs.
type Treasury struct {
	CashOnHand     float64 `json:"cashOnHand"`     // unrestricted cash (Rp juta)
	RestrictedCash float64 `json:"restrictedCash"` // escrow / restricted (Rp juta)
	MonthlyBurn    float64 `json:"monthlyBurn"`    // average monthly opex burn (Rp juta)
}

// AIInsight is a generated insight or recommendation for today's war-room.
type AIInsight struct {
	EntID string `json:"_id"`
	Type  string `json:"type"`
	Tone  string `json:"tone"`
	Text  string `json:"text"`
	Icon  string `json:"icon"`
}

// Decision is an action required from a specific role.
type Decision struct {
	EntID string `json:"_id"`
	Role  string `json:"role"`
	Tone  string `json:"tone"`
	Text  string `json:"text"`
}

// CashflowPoint is one period of the inflow-vs-outflow cash trend.
type CashflowPoint struct {
	Period  string  `json:"period"`
	Inflow  float64 `json:"inflow"`  // cash in (Rp juta)
	Outflow float64 `json:"outflow"` // cash out (Rp juta)
}

// KPI is a reference row describing an indicator, its thresholds and current value.
type KPI struct {
	EntID  string `json:"_id"`
	No     int    `json:"no"`
	KPI    string `json:"kpi"`
	Def    string `json:"def"`
	PIC    string `json:"pic"`
	Upd    string `json:"upd"`
	Green  string `json:"green"`
	Yellow string `json:"yellow"`
	Red    string `json:"red"`
	Val    string `json:"val"`
	State  string `json:"state"`
}

// Trigger is an early-warning rule with a threshold, mandatory action and escalation path.
type Trigger struct {
	EntID  string `json:"_id"`
	Cond   string `json:"cond"`
	Thr    string `json:"thr"`
	Status string `json:"status"`
	PIC    string `json:"pic"`
	Act    string `json:"act"`
	Esc    string `json:"esc"`
}

// Summary holds the executive KPIs derived from the rest of the data set.
type Summary struct {
	TotalRevenue     float64 `json:"totalRevenue"`     // contracted sales (Rp juta)
	CashPosition     float64 `json:"cashPosition"`     // cash on hand (Rp juta)
	Collected        float64 `json:"collected"`        // cash collected (Rp juta)
	CollectionRate   int     `json:"collectionRate"`   // collected / revenue %
	OutstandingAR    float64 `json:"outstandingAR"`    // total receivables (Rp juta)
	OutstandingAP    float64 `json:"outstandingAP"`    // total payables (Rp juta)
	NetMargin        int     `json:"netMargin"`        // revenue-weighted gross margin %
	BudgetAbsorption int     `json:"budgetAbsorption"` // spent / budget %
	Runway           float64 `json:"runway"`           // cash runway in months
	OverdueRisk      string  `json:"overdueRisk"`      // qualitative collection-risk label
	Critical         int     `json:"critical"`         // receivables aged >90 days
}

// Dashboard is the full payload consumed by the front-end in a single call.
type Dashboard struct {
	Projects       []Project       `json:"projects"`
	Receivables    []Receivable    `json:"receivables"`
	ReceivableType []MetaItem      `json:"receivableType"`
	AgingMeta      []MetaItem      `json:"agingMeta"`
	Payables       []Payable       `json:"payables"`
	PriorityMeta   []MetaItem      `json:"priorityMeta"`
	Facilities     []Facility      `json:"facilities"`
	CostStructure  []CostCategory  `json:"costStructure"`
	Treasury       Treasury        `json:"treasury"`
	AIInsights     []AIInsight     `json:"aiInsights"`
	Decisions      []Decision      `json:"decisions"`
	CashflowTrend  []CashflowPoint `json:"cashflowTrend"`
	KPITable       []KPI           `json:"kpiTable"`
	Triggers       []Trigger       `json:"triggers"`
	Summary        Summary         `json:"summary"`
}

// Entity is implemented by every CRUD collection element. The synthetic _id is
// the stable handle the master-data admin uses to update/delete a row.
type Entity interface {
	GetID() string
}

func (p Project) GetID() string    { return p.EntID }
func (r Receivable) GetID() string { return r.EntID }
func (p Payable) GetID() string    { return p.EntID }
func (f Facility) GetID() string   { return f.EntID }
func (a AIInsight) GetID() string  { return a.EntID }
func (d Decision) GetID() string   { return d.EntID }
func (k KPI) GetID() string        { return k.EntID }
func (t Trigger) GetID() string    { return t.EntID }

// Role enumerates the access levels for a dashboard user.
type Role string

const (
	RoleAdmin  Role = "admin"  // full master-data access
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
