package ingest

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"greenpark/finance/internal/domain"
)

// parsed holds the extracted, deduplicated rows from all classified sheets.
type parsed struct {
	akads    []domain.AkadRow      // closed transactions (detail grain, deduped)
	pipeline []pipelineRecord      // live bank-process rows (deduped)
	issues   []string
	info     []SheetInfo
}

// pipelineRecord is the working representation of a booking before it becomes a
// dashboard PipelineRow (carries the computed stage rank + terminal flags).
type pipelineRecord struct {
	row      domain.PipelineRow
	rank     int  // furthest stage reached (see stage ranks below)
	isAkad   bool // terminal: already akad
	isBatal  bool // terminal: cancelled
}

// stage ranks for the booking→akad funnel.
const (
	rankBooking = 0
	rankDP      = 1
	rankBerkas  = 2
	rankBank    = 3
	rankSP3     = 4
	rankAkad    = 5
)

// parseAll classifies every tab and extracts its rows.
func parseAll(sheets map[string][][]string) parsed {
	p := parsed{}
	// Stable order so issues/info are deterministic.
	names := make([]string, 0, len(sheets))
	for n := range sheets {
		names = append(names, n)
	}
	sort.Strings(names)

	seenAkad := map[string]bool{}
	seenPipe := map[string]bool{}

	for _, name := range names {
		rows := sheets[name]
		hi, hdr := findHeader(rows)
		if hi < 0 {
			p.info = append(p.info, SheetInfo{Name: name, Kind: "skipped", Rows: 0})
			continue
		}
		idx := indexHeader(hdr)
		switch classify(idx) {
		case "akad":
			n := p.extractAkad(rows[hi+1:], idx, seenAkad)
			p.info = append(p.info, SheetInfo{Name: name, Kind: "akad", Rows: n})
		case "pipeline":
			n := p.extractPipeline(rows[hi+1:], idx, seenPipe)
			p.info = append(p.info, SheetInfo{Name: name, Kind: "pipeline", Rows: n})
		default:
			p.info = append(p.info, SheetInfo{Name: name, Kind: "skipped", Rows: 0})
		}
	}
	return p
}

/* ----------------------------- header handling ----------------------------- */

// headerTokens are the column names the engine looks for; used to score which
// row is the header (sheets often have title/banner rows above it).
var headerTokens = []string{
	"gp", "proyek", "konsumen", "blok", "plafon", "dp", "bank", "cara bayar",
	"tgl akad", "bulan", "tahun", "sales", "status", "tahap", "booking",
}

// findHeader scans the first rows for the one that looks most like a header.
func findHeader(rows [][]string) (int, []string) {
	limit := 10
	if len(rows) < limit {
		limit = len(rows)
	}
	best, bestScore := -1, 0
	for i := 0; i < limit; i++ {
		score := 0
		for _, c := range rows[i] {
			lc := lower(c)
			for _, t := range headerTokens {
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
	if bestScore < 4 {
		return -1, nil
	}
	return best, rows[best]
}

// colIndex maps a normalized header name to its column position.
type colIndex struct{ hdr []string }

func indexHeader(hdr []string) colIndex { return colIndex{hdr: hdr} }

// first returns the index of the first column whose lowercased name contains all
// of the given substrings and none of the excludes; -1 if none.
func (c colIndex) first(must []string, exclude ...string) int {
	for i, h := range c.hdr {
		lc := lower(h)
		ok := true
		for _, m := range must {
			if !strings.Contains(lc, m) {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		for _, e := range exclude {
			if strings.Contains(lc, e) {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

// last is like first but returns the last matching column (some sheets repeat a
// "Status"/"Platform"-style header; the cleaned one is usually the last).
func (c colIndex) last(must []string, exclude ...string) int {
	res := -1
	for i, h := range c.hdr {
		lc := lower(h)
		ok := true
		for _, m := range must {
			if !strings.Contains(lc, m) {
				ok = false
				break
			}
		}
		for _, e := range exclude {
			if strings.Contains(lc, e) {
				ok = false
				break
			}
		}
		if ok {
			res = i
		}
	}
	return res
}

// classify decides what a sheet is from its header columns. Akad-detail sheets
// carry a "plafon" money column; pipeline sheets carry the KPR bank-process
// columns ("tahap proses kpr" / "status dp" / "pemberkasan"); everything else
// (recaps/dashboards) is recomputed, so it is skipped.
func classify(c colIndex) string {
	hasPlafon := c.first([]string{"plafon"}) >= 0
	hasAkadDate := c.first([]string{"akad"}) >= 0 || c.first([]string{"bulan"}) >= 0
	if hasPlafon && hasAkadDate {
		return "akad"
	}
	if c.first([]string{"tahap"}) >= 0 || c.first([]string{"status", "dp"}) >= 0 ||
		c.first([]string{"pemberkasan"}) >= 0 {
		return "pipeline"
	}
	return ""
}

/* ----------------------------- akad extraction ----------------------------- */

func (p *parsed) extractAkad(rows [][]string, c colIndex, seen map[string]bool) int {
	gp := c.first([]string{"gp"})
	proyek := c.first([]string{"proyek"})
	konsumen := c.first([]string{"konsumen"})
	blok := c.first([]string{"blok"})
	dp := c.first([]string{"dp"}, "tgl", "bulan", "durasi", "status") // money DP, not "Tgl DP"
	plafon := c.first([]string{"plafon"})
	cara := c.first([]string{"cara bayar"})
	bank := c.first([]string{"bank"}, "proses", "tahap")
	bulan := c.first([]string{"bulan"})
	tahun := c.first([]string{"tahun"})
	durasi := c.first([]string{"durasi"})
	sales := c.first([]string{"sales"})
	tglAkad := c.first([]string{"akad"}, "bulan", "tahun", "durasi")

	added := 0
	for _, r := range rows {
		proj := canonProject(get(r, proyek))
		cust := strings.TrimSpace(get(r, konsumen))
		if proj == "" && cust == "" {
			continue // blank/spacer row
		}
		year := atoiSafe(get(r, tahun))
		if year == 0 {
			year = yearFromDate(get(r, tglAkad))
		}
		key := strings.ToLower(get(r, gp) + "|" + proj + "|" + get(r, blok) + "|" + cust + "|" + strconv.Itoa(year))
		if seen[key] {
			continue // duplicate across overlapping detail tabs
		}
		seen[key] = true

		dur := atoiSafe(get(r, durasi))
		if dur > 2000 { // Excel date serial leaked into a duration cell
			p.issues = append(p.issues, fmt.Sprintf("durasi tidak wajar (%d) di %s — diabaikan", dur, proj))
			dur = 0
		}
		p.akads = append(p.akads, domain.AkadRow{
			GP:        normGP(get(r, gp)),
			Project:   proj,
			Customer:  cust,
			Blok:      strings.TrimSpace(get(r, blok)),
			Sales:     canonSales(get(r, sales)),
			Bank:      normBank(get(r, bank)),
			CaraBayar: normCara(get(r, cara)),
			DP:        moneyJuta(get(r, dp)),
			Plafon:    moneyJuta(get(r, plafon)),
			TglAkad:   strings.TrimSpace(get(r, tglAkad)),
			Bulan:     normMonth(get(r, bulan)),
			Tahun:     year,
			Durasi:    dur,
		})
		added++
	}
	return added
}

/* --------------------------- pipeline extraction --------------------------- */

func (p *parsed) extractPipeline(rows [][]string, c colIndex, seen map[string]bool) int {
	gp := c.first([]string{"gp"})
	proyek := c.first([]string{"proyek"})
	konsumen := c.first([]string{"konsumen"})
	blok := c.first([]string{"blok"})
	sales := c.first([]string{"sales"})
	cara := c.first([]string{"cara bayar"})
	status := c.last([]string{"status"}, "dp") // final deal STATUS (not "Status DP")
	statusDP := c.first([]string{"status", "dp"})
	pemberkasan := c.first([]string{"pemberkasan"})
	kirim := c.first([]string{"pengiriman"})
	sp3 := c.first([]string{"sp3"})
	tahap := c.first([]string{"tahap"})
	bank := c.first([]string{"bank"})
	kendala := c.first([]string{"kendala"})
	if kendala < 0 {
		kendala = c.first([]string{"masalah"})
	}

	added := 0
	for _, r := range rows {
		proj := canonProject(get(r, proyek))
		cust := strings.TrimSpace(get(r, konsumen))
		if proj == "" && cust == "" {
			continue
		}
		key := strings.ToLower(get(r, gp) + "|" + proj + "|" + get(r, blok) + "|" + cust)
		if seen[key] {
			continue
		}
		seen[key] = true

		st := upper(get(r, status))
		rec := pipelineRecord{
			row: domain.PipelineRow{
				Project:   proj,
				Customer:  cust,
				Blok:      strings.TrimSpace(get(r, blok)),
				Sales:     canonSales(get(r, sales)),
				Bank:      normBank(get(r, bank)),
				CaraBayar: normCara(get(r, cara)),
				Kendala:   strings.TrimSpace(get(r, kendala)),
			},
		}
		switch {
		case strings.Contains(st, "AKAD"):
			rec.isAkad = true
			rec.rank = rankAkad
		case strings.Contains(st, "BATAL"), strings.Contains(st, "GUGUR"), strings.Contains(st, "CANCEL"):
			rec.isBatal = true
		default:
			rec.rank = pipeRank(r, statusDP, pemberkasan, kirim, sp3, tahap)
		}
		rec.row.StageKey, rec.row.Stage = stageLabel(rec)
		p.pipeline = append(p.pipeline, rec)
		added++
	}
	return added
}

// pipeRank computes the furthest stage an active booking has reached.
func pipeRank(r []string, statusDP, pemberkasan, kirim, sp3, tahap int) int {
	rank := rankBooking
	if has(get(r, statusDP), "SUDAH") {
		rank = max(rank, rankDP)
	}
	if has(get(r, pemberkasan), "LENGKAP") {
		rank = max(rank, rankBerkas)
	}
	if strings.TrimSpace(get(r, kirim)) != "" {
		rank = max(rank, rankBank)
	}
	tp := upper(get(r, tahap))
	if strings.TrimSpace(get(r, sp3)) != "" || strings.Contains(tp, "SP3") {
		rank = max(rank, rankSP3)
	}
	return rank
}

func stageLabel(rec pipelineRecord) (string, string) {
	if rec.isAkad {
		return "akad", "Akad"
	}
	if rec.isBatal {
		return "batal", "Batal"
	}
	switch rec.rank {
	case rankSP3:
		return "sp3", "SP3 Terbit"
	case rankBank:
		return "bank", "Proses Bank"
	case rankBerkas:
		return "berkas", "Berkas Lengkap"
	case rankDP:
		return "dp", "Sudah DP"
	default:
		return "booking", "Booking"
	}
}

/* ------------------------------- cell helpers ------------------------------ */

func get(r []string, i int) string {
	if i < 0 || i >= len(r) {
		return ""
	}
	return r[i]
}

func lower(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
func upper(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

func has(s, sub string) bool { return strings.Contains(upper(s), sub) }

// moneyJuta parses a Rupiah cell ("Rp. 541.800.000", "541800000", "-", "") into
// millions of Rupiah (Rp juta).
func moneyJuta(s string) float64 {
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
	return n / 1_000_000
}

func atoiSafe(s string) int {
	var b strings.Builder
	for _, ch := range strings.TrimSpace(s) {
		if ch >= '0' && ch <= '9' {
			b.WriteRune(ch)
		}
	}
	if b.Len() == 0 {
		return 0
	}
	n, _ := strconv.Atoi(b.String())
	return n
}

// yearFromDate pulls a 20xx year out of a free-text date ("3-Jan-2025", "12-Jan-23").
func yearFromDate(s string) int {
	s = strings.TrimSpace(s)
	// Look for a 4-digit 20xx first.
	for i := 0; i+4 <= len(s); i++ {
		if s[i] == '2' && s[i+1] == '0' && isDigit(s[i+2]) && isDigit(s[i+3]) {
			y, _ := strconv.Atoi(s[i : i+4])
			return y
		}
	}
	// Fallback: trailing 2-digit year ("-23" → 2023).
	if n := len(s); n >= 2 && isDigit(s[n-1]) && isDigit(s[n-2]) {
		yy, _ := strconv.Atoi(s[n-2:])
		if yy >= 0 && yy < 90 {
			return 2000 + yy
		}
	}
	return 0
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }
