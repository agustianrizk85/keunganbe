package service

import (
	"time"

	"greenpark/finance/internal/domain"
	"greenpark/finance/internal/ingest"
)

// Purchasing returns the current procurement (PR) view.
func (s *financeService) Purchasing() domain.Purchasing { return s.repo.Purchasing() }

// PRSheet returns the UI-configured procurement input spreadsheet ID ("" = unset).
func (s *financeService) PRSheet() string { return s.repo.PRSheet() }

// SetPRSheet stores the procurement input spreadsheet ID.
func (s *financeService) SetPRSheet(id string) error { return s.repo.SetPRSheet(id) }

// PreviewPurchasing builds the procurement view from the input sheet tabs without
// persisting it (used by the sync-preview endpoint).
func (s *financeService) PreviewPurchasing(data map[string][][]string) ingest.PRResult {
	res := ingest.RunPRSheets(data)
	res.Purchasing.Updated = stampDate(time.Now())
	return res
}

// ApprovePurchasing builds the procurement view and stores it (bumping the
// realtime revision so connected dashboards refresh).
func (s *financeService) ApprovePurchasing(data map[string][][]string) (domain.Purchasing, error) {
	res := s.PreviewPurchasing(data)
	if err := s.repo.ApplyPurchasing(res.Purchasing); err != nil {
		return domain.Purchasing{}, err
	}
	return res.Purchasing, nil
}
