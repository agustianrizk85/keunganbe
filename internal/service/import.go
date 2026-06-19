package service

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"time"

	"greenpark/finance/internal/domain"
	"greenpark/finance/internal/ingest"
	"greenpark/finance/internal/repository"
)

// indoMonths renders Updated stamps in Indonesian.
var indoMonths = [...]string{
	"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

// PreviewImport parses an uploaded workbook and returns the validated, mapped
// preview WITHOUT touching the live dashboard.
func (s *financeService) PreviewImport(r io.Reader) (*ingest.Result, error) {
	return ingest.RunReader(r, s.opts.ingest())
}

// PreviewSheets parses sheets fetched from Google Sheets without touching the
// live dashboard.
func (s *financeService) PreviewSheets(data map[string][][]string) (*ingest.Result, error) {
	return ingest.RunSheets(data, s.opts.ingest())
}

// ApproveImport parses the workbook, applies the mapped data to the store,
// records a rollback snapshot + history entry, and returns the new record.
func (s *financeService) ApproveImport(r io.Reader, filename, by string) (domain.ImportRecord, error) {
	res, err := ingest.RunReader(r, s.opts.ingest())
	if err != nil {
		return domain.ImportRecord{}, err
	}
	return s.applyResult(res, filename, by)
}

// ApproveSheets applies a Google-Sheets-sourced import to the store.
func (s *financeService) ApproveSheets(data map[string][][]string, filename, by string) (domain.ImportRecord, error) {
	res, err := ingest.RunSheets(data, s.opts.ingest())
	if err != nil {
		return domain.ImportRecord{}, err
	}
	return s.applyResult(res, filename, by)
}

// applyResult turns an ingest Result into a persisted import.
func (s *financeService) applyResult(res *ingest.Result, filename, by string) (domain.ImportRecord, error) {
	now := time.Now()
	d := res.Preview
	d.Updated = stampDate(now)
	return s.repo.ApplyImport(repository.ImportInput{
		ID:       newImportID(),
		Time:     now.Format(time.RFC3339),
		Filename: filename,
		By:       by,
		Summary:  res.Headline,
		Data:     d,
	})
}

// ResetData clears all dashboard data back to empty (reversible via history).
func (s *financeService) ResetData(by string) (domain.ImportRecord, error) {
	return s.repo.ResetData(by, time.Now().Format(time.RFC3339))
}

// RollbackImport restores the dashboard to its state before the given import.
func (s *financeService) RollbackImport(id string) (domain.ImportRecord, error) {
	return s.repo.Rollback(id)
}

func stampDate(t time.Time) string {
	return t.Format("2") + " " + indoMonths[int(t.Month())] + " " + t.Format("2006")
}

func newImportID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "imp-0000000000"
	}
	return "imp-" + hex.EncodeToString(b)
}
