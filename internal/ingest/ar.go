package ingest

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"greenpark/finance/internal/domain"
)

// RunARSheets builds the AR (piutang) dashboard from the per-project input
// spreadsheets. The data map is keyed "CODE::TabTitle" (CODE = project code, set
// by the transport layer when fetching each project sheet), so each row can be
// attributed to its project. This path is fully independent of the akad/KPR
// classifier — AR tabs (DATA PENJUALAN etc.) never pollute the akad dashboard.
func RunARSheets(data map[string][][]string, focusYear int) domain.ARData {
	if focusYear == 0 {
		focusYear = 2026
	}
	ag := newARAgg(focusYear)

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		code, tab := splitARKey(key)
		rows := data[key]
		hi, hdr := findARHeader(rows)
		if hi < 0 {
			ag.info = append(ag.info, domain.ARSheetInfo{Project: code, Tab: tab, Kind: "skipped"})
			continue
		}
		idx := indexHeader(hdr)
		switch classifyAR(idx) {
		case "danamasuk":
			n := ag.danaMasuk(code, rows[hi+1:], idx)
			ag.info = append(ag.info, domain.ARSheetInfo{Project: code, Tab: tab, Kind: "danamasuk", Rows: n})
		case "dp":
			n := ag.dp(code, rows[hi+1:], idx)
			ag.info = append(ag.info, domain.ARSheetInfo{Project: code, Tab: tab, Kind: "dp", Rows: n})
		case "laporan":
			n := ag.laporan(code, rows[hi+1:], idx)
			ag.info = append(ag.info, domain.ARSheetInfo{Project: code, Tab: tab, Kind: "laporan", Rows: n})
		default:
			ag.info = append(ag.info, domain.ARSheetInfo{Project: code, Tab: tab, Kind: "skipped"})
		}
	}
	return ag.finish()
}

// arHeaderTokens score which row is the header for AR input tabs (DANA MASUK /
// DP / LAPORAN). Kept separate from the akad header tokens so AR layouts — whose
// columns differ — are detected reliably.
var arHeaderTokens = []string{
	"waktu transaksi", "deskripsi", "nominal", "bank", "blok", "konsumen",
	"umur piutang", "jatuh tempo", "down payment", "sisa piutang", "sisa pembayaran",
	"harga jual", "total pembayaran", "no. unit", "status", "nama penerima",
}

// findARHeader scans the first rows for the one that looks most like an AR header
// (threshold 3 — AR tabs have fewer of the akad-style tokens).
func findARHeader(rows [][]string) (int, []string) {
	limit := 12
	if len(rows) < limit {
		limit = len(rows)
	}
	best, bestScore := -1, 0
	for i := 0; i < limit; i++ {
		score := 0
		for _, c := range rows[i] {
			lc := lower(c)
			for _, t := range arHeaderTokens {
				if strings.Contains(lc, t) {
					score++
					break
				}
			}
		}
		if score > bestScore {
			best, bestScore = i, score
		}
	}
	if bestScore < 3 {
		return -1, nil
	}
	return best, rows[best]
}

func splitARKey(k string) (code, tab string) {
	if i := strings.Index(k, "::"); i >= 0 {
		return strings.TrimSpace(k[:i]), strings.TrimSpace(k[i+2:])
	}
	return "", strings.TrimSpace(k)
}

// classifyAR decides what an AR tab is from its header columns.
func classifyAR(c colIndex) string {
	if c.first([]string{"waktu transaksi"}) >= 0 ||
		(c.first([]string{"deskripsi"}) >= 0 && c.first([]string{"nominal"}) >= 0) {
		return "danamasuk"
	}
	if c.first([]string{"umur piutang"}) >= 0 ||
		(c.first([]string{"down payment"}) >= 0 && c.first([]string{"jatuh tempo"}) >= 0) {
		return "dp"
	}
	if c.first([]string{"sisa piutang"}) >= 0 {
		return "laporan"
	}
	return ""
}

/* ------------------------------- aggregator ------------------------------- */

type arAgg struct {
	focusYear int

	// summary accumulators
	nilaiKontrak, terbayar, sisaPiutang float64
	unitLunas, unitBelum                int
	cashIn, pencairanKpr                float64
	dpJatuhTempo                        int
	dpSisa                              float64

	tahap   map[string]*domain.ARTahap   // key → stage
	aging   map[string]*domain.ARAging   // bucket → aging
	monthly map[string]*domain.ARMonth   // YYYY-MM → month
	banks   map[string]*domain.ARBank    // bank → cash-in
	proj    map[string]*domain.ARProject // code → project rollup
	piutang []domain.ARPiutangRow

	info []domain.ARSheetInfo
}

func newARAgg(focusYear int) *arAgg {
	return &arAgg{
		focusYear: focusYear,
		tahap:     map[string]*domain.ARTahap{},
		aging:     map[string]*domain.ARAging{},
		monthly:   map[string]*domain.ARMonth{},
		banks:     map[string]*domain.ARBank{},
		proj:      map[string]*domain.ARProject{},
		info:      []domain.ARSheetInfo{},
	}
}

func (a *arAgg) projOf(code string) *domain.ARProject {
	p := a.proj[code]
	if p == nil {
		p = &domain.ARProject{Code: code}
		a.proj[code] = p
	}
	return p
}

/* ------------------------------- dana masuk ------------------------------- */

func (a *arAgg) danaMasuk(code string, rows [][]string, c colIndex) int {
	deskripsi := c.first([]string{"deskripsi"})
	nominal := c.first([]string{"nominal"})
	bankPenerima := c.first([]string{"bank penerima"})
	waktu := c.first([]string{"waktu transaksi"})

	added := 0
	for _, r := range rows {
		desc := strings.TrimSpace(get(r, deskripsi))
		val := rupiah(get(r, nominal))
		if desc == "" && val == 0 {
			continue
		}
		y, m, ok := parseARDate(get(r, waktu))
		isKpr := strings.Contains(upper(desc), "PENCAIRAN")
		// focus-year aggregations
		if ok && y == a.focusYear {
			a.cashIn += val
			if isKpr {
				a.pencairanKpr += val
			}
			a.projOf(code).CashIn += val
			key := fmt.Sprintf("%04d-%02d", y, m)
			mp := a.monthly[key]
			if mp == nil {
				mp = &domain.ARMonth{Period: key}
				a.monthly[key] = mp
			}
			mp.CashIn += val
			if isKpr {
				mp.Kpr += val
			}
		} else if !ok {
			// undated rows still count to project cash-in total
			a.cashIn += val
			a.projOf(code).CashIn += val
			if isKpr {
				a.pencairanKpr += val
			}
		}
		// bank split (all years)
		if bn := normBank(get(r, bankPenerima)); bn != "" {
			b := a.banks[bn]
			if b == nil {
				b = &domain.ARBank{Name: bn}
				a.banks[bn] = b
			}
			b.Count++
			b.Nilai += val
		}
		// KPR stage split (all years)
		if isKpr {
			if k, lbl := tahapOf(desc); k != "" {
				t := a.tahap[k]
				if t == nil {
					t = &domain.ARTahap{Key: k, Label: lbl}
					a.tahap[k] = t
				}
				t.Count++
				t.Nilai += val
			}
		}
		added++
	}
	return added
}

// tahapOf maps a "PENCAIRAN KPR …" description to its stage.
func tahapOf(desc string) (string, string) {
	u := upper(desc)
	switch {
	case strings.Contains(u, "BAST"):
		return "bast", "Tahap 4 · BAST"
	case strings.Contains(u, "ATAP"):
		return "atap", "Tahap 3 · Atap"
	case strings.Contains(u, "PONDASI"):
		return "pondasi", "Tahap 2 · Pondasi"
	case strings.Contains(u, "AKAD"):
		return "akad", "Tahap 1 · Akad"
	default:
		return "lain", "Lainnya"
	}
}

/* ---------------------------------- DP ----------------------------------- */

func (a *arAgg) dp(_ string, rows [][]string, c colIndex) int {
	konsumen := c.first([]string{"konsumen"})
	sisa := c.first([]string{"sisa pembayaran"})
	if sisa < 0 {
		sisa = c.first([]string{"sisa"})
	}
	umur := c.first([]string{"umur piutang"})
	status := c.first([]string{"status"})

	added := 0
	for _, r := range rows {
		cust := strings.TrimSpace(get(r, konsumen))
		if cust == "" || upper(cust) == "TOTAL" {
			continue
		}
		s := rupiah(get(r, sisa))
		if s <= 0 {
			added++
			continue // lunas DP — not a receivable
		}
		a.dpSisa += s
		days := atoiSafe(get(r, umur))
		st := upper(get(r, status))
		overdue := strings.Contains(st, "SUDAH")
		if overdue {
			a.dpJatuhTempo++
		}
		bucket := agingBucket(overdue, days)
		ab := a.aging[bucket.key]
		if ab == nil {
			ab = &domain.ARAging{Bucket: bucket.label}
			a.aging[bucket.key] = ab
		}
		ab.Count++
		ab.Nilai += s
		added++
	}
	return added
}

type agingDef struct{ key, label string }

func agingBucket(overdue bool, days int) agingDef {
	if !overdue {
		return agingDef{"0belum", "Belum jatuh tempo"}
	}
	switch {
	case days <= 30:
		return agingDef{"1_0_30", "Lewat 1–30 hari"}
	case days <= 90:
		return agingDef{"2_31_90", "Lewat 31–90 hari"}
	default:
		return agingDef{"3_90", "Lewat > 90 hari"}
	}
}

/* -------------------------------- LAPORAN -------------------------------- */

func (a *arAgg) laporan(code string, rows [][]string, c colIndex) int {
	konsumen := c.first([]string{"konsumen"})
	blok := c.first([]string{"blok"})
	bank := c.first([]string{"bank"}, "pengirim")
	harga := c.first([]string{"harga jual"})
	if harga < 0 {
		harga = c.first([]string{"harga"})
	}
	terbayar := c.first([]string{"total pembayaran"})
	sisa := c.first([]string{"sisa piutang"})
	status := c.first([]string{"status"}, "unit", "akad", "bast")

	p := a.projOf(code)
	added := 0
	for _, r := range rows {
		cust := strings.TrimSpace(get(r, konsumen))
		if cust == "" || upper(cust) == "TOTAL" {
			continue
		}
		hj := rupiah(get(r, harga))
		tb := rupiah(get(r, terbayar))
		ss := rupiah(get(r, sisa))
		if hj == 0 && tb == 0 && ss == 0 {
			continue
		}
		a.nilaiKontrak += hj
		a.terbayar += tb
		a.sisaPiutang += ss
		p.NilaiKontrak += hj
		p.TotalTerbayar += tb
		p.SisaPiutang += ss
		p.Unit++
		if ss <= 0 {
			a.unitLunas++
			p.Lunas++
		} else {
			a.unitBelum++
		}
		st := strings.TrimSpace(get(r, status))
		if ss > 0 {
			a.piutang = append(a.piutang, domain.ARPiutangRow{
				Project:   code,
				Customer:  cust,
				Blok:      strings.TrimSpace(get(r, blok)),
				Bank:      normBank(get(r, bank)),
				HargaJual: hj,
				Terbayar:  tb,
				Sisa:      ss,
				Status:    st,
			})
		}
		added++
	}
	return added
}

/* -------------------------------- finish --------------------------------- */

func (a *arAgg) finish() domain.ARData {
	d := domain.EmptyARData()
	d.FocusYear = a.focusYear
	d.Period = fmt.Sprintf("AR / Piutang · %d", a.focusYear)

	prog := 0
	if a.nilaiKontrak > 0 {
		prog = int(a.terbayar / a.nilaiKontrak * 100)
	}
	d.Summary = domain.ARSummary{
		NilaiKontrak:   a.nilaiKontrak,
		TotalTerbayar:  a.terbayar,
		SisaPiutang:    a.sisaPiutang,
		ProgresPct:     prog,
		UnitTotal:      a.unitLunas + a.unitBelum,
		UnitLunas:      a.unitLunas,
		UnitBelumLunas: a.unitBelum,
		CashIn:         a.cashIn,
		PencairanKpr:   a.pencairanKpr,
		DpJatuhTempo:   a.dpJatuhTempo,
		DpSisa:         a.dpSisa,
	}

	// stages in fixed order
	for _, k := range []string{"akad", "pondasi", "atap", "bast", "lain"} {
		if t := a.tahap[k]; t != nil {
			d.Tahapan = append(d.Tahapan, *t)
		}
	}
	// aging in fixed order
	for _, k := range []string{"0belum", "1_0_30", "2_31_90", "3_90"} {
		if b := a.aging[k]; b != nil {
			d.Aging = append(d.Aging, *b)
		}
	}
	// monthly sorted by period
	mk := make([]string, 0, len(a.monthly))
	for k := range a.monthly {
		mk = append(mk, k)
	}
	sort.Strings(mk)
	for _, k := range mk {
		d.Monthly = append(d.Monthly, *a.monthly[k])
	}
	// banks sorted by nilai desc
	for _, b := range a.banks {
		d.Banks = append(d.Banks, *b)
	}
	sort.Slice(d.Banks, func(i, j int) bool { return d.Banks[i].Nilai > d.Banks[j].Nilai })
	// projects sorted by sisa piutang desc
	for _, p := range a.proj {
		if p.NilaiKontrak > 0 {
			p.ProgresPct = int(p.TotalTerbayar / p.NilaiKontrak * 100)
		}
		d.Projects = append(d.Projects, *p)
	}
	sort.Slice(d.Projects, func(i, j int) bool { return d.Projects[i].SisaPiutang > d.Projects[j].SisaPiutang })
	// piutang watch-list: top by sisa, cap 60
	sort.Slice(a.piutang, func(i, j int) bool { return a.piutang[i].Sisa > a.piutang[j].Sisa })
	if len(a.piutang) > 60 {
		a.piutang = a.piutang[:60]
	}
	d.Piutang = a.piutang
	if d.Piutang == nil {
		d.Piutang = []domain.ARPiutangRow{}
	}
	d.Sheets = a.info
	return d
}

/* ------------------------------- helpers --------------------------------- */

// rupiah parses a money cell into full Rupiah (no juta scaling). Accepts
// "3.000.000", "3000000", "Rp 3.000.000", "-", "".
func rupiah(s string) float64 {
	var b strings.Builder
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			b.WriteRune(ch)
		}
	}
	if b.Len() == 0 {
		return 0
	}
	n, err := strconv.ParseFloat(b.String(), 64)
	if err != nil {
		return 0
	}
	return n
}

var idnMonths = map[string]int{
	"jan": 1, "feb": 2, "mar": 3, "apr": 4, "mei": 5, "may": 5, "jun": 6,
	"jul": 7, "agu": 8, "ags": 8, "aug": 8, "sep": 9, "okt": 10, "oct": 10,
	"nov": 11, "des": 12, "dec": 12,
}

// parseARDate extracts (year, month) from the many date encodings AR sheets use:
// an Excel/Sheets serial number ("45901"), an ISO timestamp ("2025-08-23 00:00:00")
// or a free-text Indonesian date ("1-Mei-2026").
func parseARDate(s string) (year, month int, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0, false
	}
	// Serial number (all digits, plausible Excel range).
	if isAllDigits(s) {
		if n, err := strconv.Atoi(s); err == nil && n > 20000 && n < 80000 {
			t := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC).AddDate(0, 0, n)
			return t.Year(), int(t.Month()), true
		}
	}
	// ISO "YYYY-MM-DD…"
	if len(s) >= 7 && s[4] == '-' {
		y, e1 := strconv.Atoi(s[0:4])
		m, e2 := strconv.Atoi(strings.TrimLeft(s[5:7], "0"))
		if e1 == nil && e2 == nil && m >= 1 && m <= 12 {
			return y, m, true
		}
	}
	// "d-Mon-YYYY" / "d Mon yyyy"
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '-' || r == '/' || r == ' ' })
	if len(parts) >= 3 {
		mon := strings.ToLower(parts[1])
		if len(mon) >= 3 {
			if m, found := idnMonths[mon[:3]]; found {
				if y, err := strconv.Atoi(parts[2]); err == nil {
					if y < 100 {
						y += 2000
					}
					return y, m, true
				}
			}
		}
	}
	// Fallback: any 20xx year, month unknown.
	if y := yearFromDate(s); y != 0 {
		return y, 0, false
	}
	return 0, 0, false
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
