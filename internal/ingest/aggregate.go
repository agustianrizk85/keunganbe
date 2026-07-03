package ingest

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"greenpark/finance/internal/domain"
)

// assemble turns the parsed rows into the full dashboard for the focus year.
func assemble(p parsed, opts Options, res *Result) domain.Dashboard {
	years := distinctYears(p.akads)
	focus := opts.FocusYear
	if !containsInt(years, focus) && len(years) > 0 {
		focus = years[0] // latest available
	}

	// Akad rows for the focus year drive the money figures.
	var focusAkad []domain.AkadRow
	for _, a := range p.akads {
		if a.Tahun == focus {
			focusAkad = append(focusAkad, a)
		}
	}

	summary := buildSummary(focusAkad, p.pipeline, focus, opts.TargetAkad)
	projects := buildProjects(focusAkad, p.pipeline)
	banks := buildBanks(focusAkad)
	salesRank := buildSales(focusAkad)
	payMix := buildPayMix(focusAkad)
	monthly := buildMonthly(focusAkad)
	funnel := buildFunnel(p.pipeline)
	pipeline := buildPipelineList(p.pipeline)

	if len(banks) > 0 {
		summary.TopBank = banks[0].Name
		summary.BankCount = len(banks)
	}
	if len(projects) > 0 {
		summary.TopProject = projects[0].Name
	}

	d := domain.Dashboard{
		Period:    periodLabel(focus, monthly),
		FocusYear: focus,
		Years:     years,
		Summary:   summary,
		Funnel:    funnel,
		Monthly:   monthly,
		Projects:  projects,
		Banks:     banks,
		Sales:     salesRank,
		PayMix:    payMix,
		Pipeline:  pipeline,
		Akads:     recentAkad(focusAkad),
		Purchasing: buildPurchasing(p),
	}
	deriveAll(&d)
	nonNilSlices(&d)
	return d
}

// nonNilSlices guarantees every collection serialises as a JSON array ("[]")
// rather than null, so the front-end can always call .filter/.map/.length.
func nonNilSlices(d *domain.Dashboard) {
	if d.Years == nil {
		d.Years = []int{}
	}
	if d.Funnel == nil {
		d.Funnel = []domain.FunnelStage{}
	}
	if d.Monthly == nil {
		d.Monthly = []domain.MonthPoint{}
	}
	if d.Projects == nil {
		d.Projects = []domain.ProjectFin{}
	}
	if d.Banks == nil {
		d.Banks = []domain.BankFin{}
	}
	if d.Sales == nil {
		d.Sales = []domain.SalesRank{}
	}
	if d.PayMix == nil {
		d.PayMix = []domain.PayMethod{}
	}
	if d.Pipeline == nil {
		d.Pipeline = []domain.PipelineRow{}
	}
	if d.Akads == nil {
		d.Akads = []domain.AkadRow{}
	}
	if d.Alerts == nil {
		d.Alerts = []domain.Alert{}
	}
	if d.AI == nil {
		d.AI = []domain.AIInsight{}
	}
	if d.Decisions == nil {
		d.Decisions = []domain.Decision{}
	}
	if d.KPIs == nil {
		d.KPIs = []domain.KPI{}
	}
	if d.Triggers == nil {
		d.Triggers = []domain.Trigger{}
	}
	if d.Purchasing.BySupplier == nil {
		d.Purchasing.BySupplier = []domain.SupplierSpend{}
	}
	if d.Purchasing.ByProject == nil {
		d.Purchasing.ByProject = []domain.ProjectSpend{}
	}
	if d.Purchasing.Monthly == nil {
		d.Purchasing.Monthly = []domain.PurchaseMonth{}
	}
	if d.Purchasing.Orders == nil {
		d.Purchasing.Orders = []domain.PurchaseDoc{}
	}
	if d.Purchasing.Invoices == nil {
		d.Purchasing.Invoices = []domain.PurchaseDoc{}
	}
	if d.Purchasing.Payments == nil {
		d.Purchasing.Payments = []domain.PaymentDoc{}
	}
}

func buildSummary(akad []domain.AkadRow, pipe []pipelineRecord, focus, target int) domain.Summary {
	var nilai, cashIn float64
	var kpr, durSum, durN int
	for _, a := range akad {
		nilai += a.Plafon
		cashIn += a.DP
		if a.CaraBayar == "KPR" {
			kpr++
		}
		if a.Durasi > 0 {
			durSum += a.Durasi
			durN++
		}
	}
	akadCount := len(akad)

	var pActive, pProses, pBatal int
	for _, r := range pipe {
		switch {
		case r.isBatal:
			pBatal++
		case r.isAkad:
			// terminal akad in the pipeline sheet — not "open"
		default:
			pActive++
			if r.rank >= rankBerkas {
				pProses++
			}
		}
	}

	s := domain.Summary{
		NilaiAkad:    round1(nilai),
		CashIn:       round1(cashIn),
		AkadCount:    akadCount,
		BookingCount: pActive,
		ProsesCount:  pProses,
		BatalCount:   pBatal,
		TargetAkad:   target,
	}
	if akadCount > 0 {
		s.KprShare = pct(kpr, akadCount)
	}
	if durN > 0 {
		s.AvgDurasi = int(math.Round(float64(durSum) / float64(durN)))
	}
	total := akadCount + pActive + pBatal
	if total > 0 {
		s.CancelRate = pct(pBatal, total)
	}
	// Estimate the value still in the pipeline from the average closed plafond.
	if akadCount > 0 && pActive > 0 {
		s.PipelineValue = round1(nilai / float64(akadCount) * float64(pActive))
	}
	if s.TargetAkad == 0 {
		s.TargetAkad = heuristicTarget(akadCount)
	}
	if s.TargetAkad > 0 {
		s.Achievement = pct(akadCount, s.TargetAkad)
	}
	return s
}

func buildProjects(akad []domain.AkadRow, pipe []pipelineRecord) []domain.ProjectFin {
	type agg struct {
		p     domain.ProjectFin
		kpr   int
		banks map[string]float64
	}
	m := map[string]*agg{}
	order := []string{}
	for _, a := range akad {
		key := a.Project
		g := m[key]
		if g == nil {
			g = &agg{p: domain.ProjectFin{Code: slug(key), Name: titleCase(key), GP: a.GP}, banks: map[string]float64{}}
			m[key] = g
			order = append(order, key)
		}
		g.p.Akad++
		g.p.Nilai += a.Plafon
		g.p.DP += a.DP
		if a.CaraBayar == "KPR" {
			g.kpr++
		}
		if a.Bank != "" {
			g.banks[a.Bank] += a.Plafon
		}
	}
	// Overlay pipeline (active bookings + cancellations) per project.
	for _, r := range pipe {
		g := m[r.row.Project]
		if g == nil {
			continue
		}
		switch {
		case r.isBatal:
			g.p.Batal++
		case !r.isAkad:
			g.p.Booking++
		}
	}

	out := make([]domain.ProjectFin, 0, len(order))
	for _, key := range order {
		g := m[key]
		g.p.Nilai = round1(g.p.Nilai)
		g.p.DP = round1(g.p.DP)
		if g.p.Akad > 0 {
			g.p.KprPct = pct(g.kpr, g.p.Akad)
		}
		g.p.TopBank = topKey(g.banks)
		g.p.Status = projectStatus(g.p)
		g.p.Note = projectNote(g.p)
		out = append(out, g.p)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Nilai > out[j].Nilai })
	return out
}

func buildBanks(akad []domain.AkadRow) []domain.BankFin {
	m := map[string]*domain.BankFin{}
	order := []string{}
	var total float64
	for _, a := range akad {
		if a.Bank == "" {
			continue
		}
		b := m[a.Bank]
		if b == nil {
			b = &domain.BankFin{Name: a.Bank}
			m[a.Bank] = b
			order = append(order, a.Bank)
		}
		b.Akad++
		b.Plafon += a.Plafon
		total += a.Plafon
	}
	out := make([]domain.BankFin, 0, len(order))
	for _, k := range order {
		b := m[k]
		if total > 0 {
			b.Share = pct(int(b.Plafon), int(total))
		}
		b.Plafon = round1(b.Plafon)
		b.Status = domain.StatusGreen
		out = append(out, *b)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Plafon > out[j].Plafon })
	return out
}

func buildSales(akad []domain.AkadRow) []domain.SalesRank {
	m := map[string]*domain.SalesRank{}
	order := []string{}
	for _, a := range akad {
		s := m[a.Sales]
		if s == nil {
			s = &domain.SalesRank{Name: a.Sales, IsAgent: isAgent(a.Sales)}
			m[a.Sales] = s
			order = append(order, a.Sales)
		}
		s.Akad++
		s.Nilai += a.Plafon
	}
	out := make([]domain.SalesRank, 0, len(order))
	for _, k := range order {
		m[k].Nilai = round1(m[k].Nilai)
		out = append(out, *m[k])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Akad > out[j].Akad })
	return out
}

func buildPayMix(akad []domain.AkadRow) []domain.PayMethod {
	m := map[string]*domain.PayMethod{}
	order := []string{"KPR", "Cash Keras", "Cash Bertahap"}
	for _, k := range order {
		m[k] = &domain.PayMethod{Type: k}
	}
	for _, a := range akad {
		pm := m[a.CaraBayar]
		if pm == nil {
			pm = &domain.PayMethod{Type: a.CaraBayar}
			m[a.CaraBayar] = pm
			order = append(order, a.CaraBayar)
		}
		pm.Count++
		pm.Value += a.Plafon
	}
	out := make([]domain.PayMethod, 0, len(order))
	for _, k := range order {
		if m[k].Count == 0 {
			continue
		}
		m[k].Value = round1(m[k].Value)
		out = append(out, *m[k])
	}
	return out
}

func buildMonthly(akad []domain.AkadRow) []domain.MonthPoint {
	pts := make([]domain.MonthPoint, 12)
	for i := range pts {
		pts[i] = domain.MonthPoint{Period: monthShort[i]}
	}
	for _, a := range akad {
		mi := monthIndex(a.Bulan)
		if mi == 0 {
			continue
		}
		pts[mi-1].Akad++
		pts[mi-1].Nilai += a.Plafon
		pts[mi-1].DP += a.DP
	}
	// Trim trailing empty months but keep at least through the last month with data.
	last := 0
	for i, p := range pts {
		if p.Akad > 0 {
			last = i
		}
		pts[i].Nilai = round1(pts[i].Nilai)
		pts[i].DP = round1(pts[i].DP)
	}
	return pts[:last+1]
}

// buildFunnel computes a descending booking→akad funnel from pipeline rows.
func buildFunnel(pipe []pipelineRecord) []domain.FunnelStage {
	stages := []struct {
		key, label string
		rank       int
	}{
		{"booking", "Booking", rankBooking},
		{"dp", "Sudah DP", rankDP},
		{"berkas", "Berkas Lengkap", rankBerkas},
		{"bank", "Proses Bank", rankBank},
		{"sp3", "SP3 Terbit", rankSP3},
		{"akad", "Akad", rankAkad},
	}
	out := make([]domain.FunnelStage, 0, len(stages))
	for _, s := range stages {
		n := 0
		for _, r := range pipe {
			if r.isBatal {
				continue
			}
			if r.rank >= s.rank {
				n++
			}
		}
		out = append(out, domain.FunnelStage{Key: s.key, Label: s.label, Count: n})
	}
	return out
}

// buildPipelineList returns the active (non-terminal) bookings as the
// early-warning list, problems first.
func buildPipelineList(pipe []pipelineRecord) []domain.PipelineRow {
	var out []domain.PipelineRow
	for _, r := range pipe {
		if r.isAkad || r.isBatal {
			continue
		}
		row := r.row
		switch {
		case row.Kendala != "":
			row.SLA = "overdue"
		case r.rank <= rankDP:
			row.SLA = "due"
		default:
			row.SLA = "ok"
		}
		out = append(out, row)
	}
	sort.SliceStable(out, func(i, j int) bool {
		ki, kj := out[i].Kendala != "", out[j].Kendala != ""
		if ki != kj {
			return ki
		}
		return slaWeight(out[i].SLA) > slaWeight(out[j].SLA)
	})
	if len(out) > 50 {
		out = out[:50]
	}
	return out
}

func recentAkad(akad []domain.AkadRow) []domain.AkadRow {
	out := append([]domain.AkadRow(nil), akad...)
	sort.SliceStable(out, func(i, j int) bool {
		mi, mj := monthIndex(out[i].Bulan), monthIndex(out[j].Bulan)
		if mi != mj {
			return mi > mj
		}
		return out[i].Plafon > out[j].Plafon
	})
	if len(out) > 80 {
		out = out[:80]
	}
	return out
}

/* ------------------------------- small helpers ----------------------------- */

func distinctYears(akad []domain.AkadRow) []int {
	set := map[int]bool{}
	for _, a := range akad {
		if a.Tahun > 0 {
			set[a.Tahun] = true
		}
	}
	out := make([]int, 0, len(set))
	for y := range set {
		out = append(out, y)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	return out
}

func containsInt(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func pct(a, b int) int {
	if b == 0 {
		return 0
	}
	return int(math.Round(float64(a) / float64(b) * 100))
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

func heuristicTarget(akad int) int {
	if akad == 0 {
		return 0
	}
	t := int(math.Ceil(float64(akad)*1.3/10) * 10) // ~30% headroom, rounded up to 10
	return t
}

func projectStatus(p domain.ProjectFin) domain.Status {
	denom := p.Akad + p.Batal
	if denom == 0 {
		if p.Booking > 0 {
			return domain.StatusYellow
		}
		return domain.StatusGreen
	}
	rate := float64(p.Batal) / float64(denom)
	switch {
	case rate > 0.30:
		return domain.StatusRed
	case rate > 0.15:
		return domain.StatusYellow
	default:
		return domain.StatusGreen
	}
}

func projectNote(p domain.ProjectFin) string {
	switch p.Status {
	case domain.StatusRed:
		return fmt.Sprintf("Batal tinggi (%d) — tinjau kualitas booking", p.Batal)
	case domain.StatusYellow:
		if p.Booking > 0 {
			return fmt.Sprintf("%d booking aktif menunggu akad", p.Booking)
		}
		return "Cermati rasio batal"
	default:
		return "Akad sehat"
	}
}

func topKey(m map[string]float64) string {
	best, bestV := "", -1.0
	for k, v := range m {
		if v > bestV {
			best, bestV = k, v
		}
	}
	return best
}

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

func slaWeight(s string) int {
	switch s {
	case "overdue":
		return 2
	case "due":
		return 1
	default:
		return 0
	}
}

func periodLabel(year int, monthly []domain.MonthPoint) string {
	first, last := "", ""
	for _, m := range monthly {
		if m.Akad > 0 {
			if first == "" {
				first = m.Period
			}
			last = m.Period
		}
	}
	if first == "" {
		return fmt.Sprintf("Akad %d", year)
	}
	if first == last {
		return fmt.Sprintf("Akad %d · %s", year, first)
	}
	return fmt.Sprintf("Akad %d · %s–%s", year, first, last)
}
