package ingest

import (
	"sort"
	"strconv"
	"strings"

	"greenpark/finance/internal/domain"
)

/* ------------------------- procurement (PR) extraction ------------------------ */

// extractPurchase reads a purchase-order ("po") or purchase-invoice ("invoice")
// tab into line-item rows. The two share a near-identical layout (an invoice
// references a faktur number, a PO a PO number; only the PO carries a Satuan).
func (p *parsed) extractPurchase(rows [][]string, c colIndex, doc string) int {
	tanggal := c.first([]string{"tanggal"})
	nomor := c.first([]string{"faktur"})
	if doc == "po" {
		nomor = c.first([]string{"po"}, "proyek", "pemasok")
	}
	pemasok := c.first([]string{"pemasok"})
	kode := c.first([]string{"kode"})
	barang := c.first([]string{"nama"}) // "Nama Barang" (not "Kode Barang")
	qty := c.first([]string{"kuantitas"})
	satuan := c.first([]string{"satuan"})
	diskon := c.first([]string{"diskon"})
	total := c.first([]string{"total"})
	blok := c.first([]string{"blok"})
	proyek := c.first([]string{"proyek"})

	added := 0
	for _, r := range rows {
		nm := strings.TrimSpace(get(r, nomor))
		barangName := strings.TrimSpace(get(r, barang))
		if nm == "" && barangName == "" {
			continue // blank/spacer row
		}
		row := domain.PurchaseDoc{
			Doc:     doc,
			Tanggal: strings.TrimSpace(get(r, tanggal)),
			Nomor:   nm,
			Pemasok: strings.TrimSpace(get(r, pemasok)),
			Kode:    strings.TrimSpace(get(r, kode)),
			Barang:  barangName,
			Qty:     numFloat(get(r, qty)),
			Satuan:  strings.TrimSpace(get(r, satuan)),
			Diskon:  moneyJuta(get(r, diskon)),
			Total:   moneyJuta(get(r, total)),
			Blok:    strings.TrimSpace(get(r, blok)),
			Proyek:  canonProject(get(r, proyek)),
			Bulan:   monthLabel(get(r, tanggal)),
			Tahun:   yearFromDate(get(r, tanggal)),
		}
		if doc == "po" {
			p.orders = append(p.orders, row)
		} else {
			p.invoices = append(p.invoices, row)
		}
		added++
	}
	return added
}

// extractPayment reads a purchase-payment tab into faktur-grain payment rows.
func (p *parsed) extractPayment(rows [][]string, c colIndex) int {
	tanggal := c.first([]string{"tgl"})
	bukti := c.first([]string{"bukti"})
	pemasok := c.first([]string{"pemasok"})
	bank := c.first([]string{"bank"})
	faktur := c.first([]string{"faktur"}, "total")
	totalFaktur := c.first([]string{"total faktur"})
	terutang := c.first([]string{"terutang"})
	bayar := c.first([]string{"bayar"}, "tgl")

	added := 0
	for _, r := range rows {
		nf := strings.TrimSpace(get(r, faktur))
		pem := strings.TrimSpace(get(r, pemasok))
		if nf == "" && pem == "" {
			continue
		}
		p.payments = append(p.payments, domain.PaymentDoc{
			Tanggal:     strings.TrimSpace(get(r, tanggal)),
			NoBukti:     strings.TrimSpace(get(r, bukti)),
			Pemasok:     pem,
			Bank:        strings.TrimSpace(get(r, bank)),
			NoFaktur:    nf,
			TotalFaktur: moneyJuta(get(r, totalFaktur)),
			Terutang:    moneyJuta(get(r, terutang)),
			Bayar:       moneyJuta(get(r, bayar)),
			Bulan:       monthLabel(get(r, tanggal)),
			Tahun:       yearFromDate(get(r, tanggal)),
		})
		added++
	}
	return added
}

// PRResult is the outcome of a standalone procurement (PR) sync: the assembled
// Purchasing section plus the per-tab classification and any data-quality notes,
// mirroring ingest.Result for the akad path so the sync-preview UI can show how
// each tab was read.
type PRResult struct {
	Purchasing domain.Purchasing `json:"purchasing"`
	Sheets     []SheetInfo       `json:"sheets"`
	Issues     []string          `json:"issues"`
}

// RunPRSheets builds the Purchasing section from the "Pembelian (PR)" spreadsheet
// tabs alone — fully independent of the akad/KPR ingest, so a procurement sync
// refreshes the purchasing view without touching the akad dashboard (the AR path
// is decoupled the same way). Tabs are classified by their header signature, so
// no fixed tab names are required.
func RunPRSheets(data map[string][][]string) PRResult {
	p := parseAll(data)
	res := PRResult{
		Purchasing: buildPurchasing(p),
		Sheets:     p.info,
		Issues:     p.issues,
	}
	if res.Sheets == nil {
		res.Sheets = []SheetInfo{}
	}
	if res.Issues == nil {
		res.Issues = []string{}
	}
	return res
}

/* ------------------------- procurement (PR) aggregation ----------------------- */

// buildPurchasing assembles the procurement section from the parsed PR rows.
func buildPurchasing(p parsed) domain.Purchasing {
	out := domain.Purchasing{
		BySupplier: []domain.SupplierSpend{},
		ByProject:  []domain.ProjectSpend{},
		Monthly:    []domain.PurchaseMonth{},
		Orders:     []domain.PurchaseDoc{},
		Invoices:   []domain.PurchaseDoc{},
		Payments:   []domain.PaymentDoc{},
	}
	if len(p.orders) == 0 && len(p.invoices) == 0 && len(p.payments) == 0 {
		return out
	}

	// Summary headline.
	poDocs := map[string]bool{}
	var poVal float64
	for _, o := range p.orders {
		poVal += o.Total
		if o.Nomor != "" {
			poDocs[o.Nomor] = true
		}
	}
	invDocs := map[string]bool{}
	var invVal float64
	for _, in := range p.invoices {
		invVal += in.Total
		if in.Nomor != "" {
			invDocs[in.Nomor] = true
		}
	}
	var paid, outstanding float64
	for _, pay := range p.payments {
		paid += pay.Bayar
		outstanding += pay.Terutang
	}

	// Per-supplier roll-up.
	sup := map[string]*domain.SupplierSpend{}
	supOrder := []string{}
	supGet := func(name string) *domain.SupplierSpend {
		name = strings.TrimSpace(name)
		if name == "" {
			name = "—"
		}
		s := sup[name]
		if s == nil {
			s = &domain.SupplierSpend{Name: name}
			sup[name] = s
			supOrder = append(supOrder, name)
		}
		return s
	}
	for _, o := range p.orders {
		s := supGet(o.Pemasok)
		s.POValue += o.Total
		s.Docs++
	}
	for _, in := range p.invoices {
		supGet(in.Pemasok).Invoiced += in.Total
	}
	for _, pay := range p.payments {
		s := supGet(pay.Pemasok)
		s.Paid += pay.Bayar
		s.Outstanding += pay.Terutang
	}
	for _, name := range supOrder {
		s := sup[name]
		s.POValue = round1(s.POValue)
		s.Invoiced = round1(s.Invoiced)
		s.Paid = round1(s.Paid)
		s.Outstanding = round1(s.Outstanding)
		out.BySupplier = append(out.BySupplier, *s)
	}
	sort.SliceStable(out.BySupplier, func(i, j int) bool {
		return out.BySupplier[i].POValue > out.BySupplier[j].POValue
	})

	// Per-project roll-up (PO spend).
	prj := map[string]*domain.ProjectSpend{}
	prjOrder := []string{}
	for _, o := range p.orders {
		name := o.Proyek
		if name == "" {
			name = "—"
		}
		g := prj[name]
		if g == nil {
			g = &domain.ProjectSpend{Project: name}
			prj[name] = g
			prjOrder = append(prjOrder, name)
		}
		g.POValue += o.Total
		g.Items++
	}
	for _, name := range prjOrder {
		g := prj[name]
		g.POValue = round1(g.POValue)
		out.ByProject = append(out.ByProject, *g)
	}
	sort.SliceStable(out.ByProject, func(i, j int) bool {
		return out.ByProject[i].POValue > out.ByProject[j].POValue
	})

	// Monthly trend.
	months := make([]domain.PurchaseMonth, 12)
	for i := range months {
		months[i] = domain.PurchaseMonth{Period: monthShort[i]}
	}
	for _, o := range p.orders {
		if mi := monthIndex(o.Bulan); mi > 0 {
			months[mi-1].PO += o.Total
		}
	}
	for _, in := range p.invoices {
		if mi := monthIndex(in.Bulan); mi > 0 {
			months[mi-1].Invoice += in.Total
		}
	}
	for _, pay := range p.payments {
		if mi := monthIndex(pay.Bulan); mi > 0 {
			months[mi-1].Paid += pay.Bayar
		}
	}
	last := 0
	for i := range months {
		if months[i].PO > 0 || months[i].Invoice > 0 || months[i].Paid > 0 {
			last = i
		}
		months[i].PO = round1(months[i].PO)
		months[i].Invoice = round1(months[i].Invoice)
		months[i].Paid = round1(months[i].Paid)
	}
	out.Monthly = months[:last+1]

	// Recent document lists (capped).
	out.Orders = recentDocs(p.orders)
	out.Invoices = recentDocs(p.invoices)
	out.Payments = recentPayments(p.payments)

	out.Summary = domain.PurchasingSummary{
		POValue:       round1(poVal),
		POCount:       len(poDocs),
		InvoiceValue:  round1(invVal),
		InvoiceCount:  len(invDocs),
		PaidValue:     round1(paid),
		PaymentCount:  len(p.payments),
		Outstanding:   round1(outstanding),
		SupplierCount: len(supOrder),
	}
	if len(out.BySupplier) > 0 {
		out.Summary.TopSupplier = out.BySupplier[0].Name
	}
	return out
}

func recentDocs(docs []domain.PurchaseDoc) []domain.PurchaseDoc {
	out := append([]domain.PurchaseDoc(nil), docs...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Tahun != out[j].Tahun {
			return out[i].Tahun > out[j].Tahun
		}
		mi, mj := monthIndex(out[i].Bulan), monthIndex(out[j].Bulan)
		if mi != mj {
			return mi > mj
		}
		return out[i].Total > out[j].Total
	})
	if len(out) > 60 {
		out = out[:60]
	}
	return out
}

func recentPayments(docs []domain.PaymentDoc) []domain.PaymentDoc {
	out := append([]domain.PaymentDoc(nil), docs...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Tahun != out[j].Tahun {
			return out[i].Tahun > out[j].Tahun
		}
		mi, mj := monthIndex(out[i].Bulan), monthIndex(out[j].Bulan)
		if mi != mj {
			return mi > mj
		}
		return out[i].Bayar > out[j].Bayar
	})
	if len(out) > 60 {
		out = out[:60]
	}
	return out
}

/* ------------------------------- small helpers ------------------------------- */

// numFloat parses a plain numeric cell (quantity) — keeps decimals, ignores
// thousands separators and stray text.
func numFloat(s string) float64 {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, ch := range s {
		if (ch >= '0' && ch <= '9') || ch == '.' || ch == ',' {
			b.WriteRune(ch)
		}
	}
	v := b.String()
	// If both separators appear, assume "," is thousands; drop it.
	if strings.Contains(v, ".") && strings.Contains(v, ",") {
		v = strings.ReplaceAll(v, ",", "")
	} else {
		v = strings.ReplaceAll(v, ",", ".")
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}
	return n
}

// monthLabel resolves the short Indonesian month label ("Jan") from an ISO or
// loose date. Handles numeric months ("2026-01-05" → "Jan") and named months
// ("5-Jan-2026" → "Jan"); returns "" when unknown.
func monthLabel(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 7 && s[4] == '-' { // YYYY-MM-DD
		if mi, err := strconv.Atoi(s[5:7]); err == nil && mi >= 1 && mi <= 12 {
			return monthShort[mi-1]
		}
	}
	if lbl := normMonth(s); lbl != "" {
		return lbl
	}
	// Fall back to the second token of a d-m-y / d/m/y date.
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '-' || r == '/' || r == ' ' })
	if len(parts) >= 2 {
		if mi, err := strconv.Atoi(parts[1]); err == nil && mi >= 1 && mi <= 12 {
			return monthShort[mi-1]
		}
		return normMonth(parts[1])
	}
	return ""
}
