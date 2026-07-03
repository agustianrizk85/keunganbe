package service

import (
	"time"

	"greenpark/finance/internal/domain"
	"greenpark/finance/internal/ingest"
)

// PreviewAR builds the AR/piutang view from the per-project input sheets without
// persisting it (used by the sync-preview endpoint).
func (s *financeService) PreviewAR(data map[string][][]string) domain.ARData {
	ar := ingest.RunARSheets(data, s.opts.FocusYear)
	ar.Updated = stampDate(time.Now())
	return ar
}

// ApproveAR builds the AR view and stores it (bumping the realtime revision).
func (s *financeService) ApproveAR(data map[string][][]string) (domain.ARData, error) {
	ar := s.PreviewAR(data)
	if err := s.repo.ApplyAR(ar); err != nil {
		return domain.ARData{}, err
	}
	return ar, nil
}
