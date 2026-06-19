// Package ingest turns the raw "Keuangan" workbook (an XLSX upload or a live
// Google Sheet) into the derived finance Dashboard. The workbook is an akad/KPR
// closing tracker: per-transaction down payment + KPR plafond, by project, bank,
// sales and month, plus a booking→akad bank-process pipeline.
//
// The engine is source-agnostic: both RunReader (XLSX) and RunSheets (Google
// Sheets) produce a map of tab-title → rows, then classify each tab by its
// header signature (no fixed tab names required) and assemble the dashboard.
package ingest

import (
	"io"

	"greenpark/finance/internal/domain"
)

// Options tunes the ingest for the executive focus.
type Options struct {
	FocusYear  int // year the summary/aggregations focus on (0 → latest in data)
	TargetAkad int // akad target for the focus year (0 → derived heuristic)
}

func (o Options) withDefaults() Options {
	if o.FocusYear == 0 {
		o.FocusYear = 2026
	}
	return o
}

// Result is the outcome of an ingest: the assembled dashboard, the headline
// figures and any data-quality issues found while mapping.
type Result struct {
	Preview  domain.Dashboard     `json:"preview"`
	Headline domain.ImportSummary `json:"headline"`
	Issues   []string             `json:"issues"`
	Sheets   []SheetInfo          `json:"sheets"` // per-tab classification (for the preview)
}

// SheetInfo reports how one tab was classified and how many rows it contributed.
type SheetInfo struct {
	Name string `json:"name"`
	Kind string `json:"kind"` // akad | pipeline | skipped
	Rows int    `json:"rows"`
}

// RunReader parses an uploaded XLSX workbook.
func RunReader(r io.Reader, opts Options) (*Result, error) {
	sheets, err := readXLSX(r)
	if err != nil {
		return nil, err
	}
	return run(sheets, opts), nil
}

// RunSheets processes tabs fetched from Google Sheets (title → rows).
func RunSheets(data map[string][][]string, opts Options) (*Result, error) {
	return run(data, opts.withDefaults()), nil
}

// run is the shared pipeline: classify → extract → dedup → aggregate → derive.
func run(sheets map[string][][]string, opts Options) *Result {
	opts = opts.withDefaults()
	p := parseAll(sheets)
	res := &Result{Issues: p.issues, Sheets: p.info}
	res.Preview = assemble(p, opts, res)
	s := res.Preview.Summary
	res.Headline = domain.ImportSummary{
		AkadCount:    s.AkadCount,
		NilaiAkad:    s.NilaiAkad,
		CashIn:       s.CashIn,
		BookingCount: s.BookingCount,
		ProsesCount:  s.ProsesCount,
		BatalCount:   s.BatalCount,
		KprShare:     s.KprShare,
		Issues:       len(res.Issues),
	}
	return res
}
