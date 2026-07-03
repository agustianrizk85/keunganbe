// Package repository defines storage access for the Finance dashboard and ships
// a file-backed, in-memory implementation. The dashboard is ingest-driven: the
// whole derived payload is replaced on each import, with a rollback-able history
// and a monotonic revision for realtime push. Swapping in a database-backed
// store only requires satisfying the FinanceRepository interface.
package repository

import (
	"errors"

	"greenpark/finance/internal/domain"
)

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("resource not found")

// ImportInput is the assembled dashboard plus its import metadata, handed to
// ApplyImport to become the new live state.
type ImportInput struct {
	ID       string
	Time     string
	Filename string
	By       string
	Summary  domain.ImportSummary
	Data     domain.Dashboard
}

// FinanceRepository is the persistence boundary for the dashboard data set.
type FinanceRepository interface {
	// reads
	Dashboard() domain.Dashboard
	AR() domain.ARData
	ARSources() []domain.ARSource
	Purchasing() domain.Purchasing
	PRSheet() string
	Revision() int64
	ImportHistory() []domain.ImportRecord

	// ingest / lifecycle writes
	ApplyImport(in ImportInput) (domain.ImportRecord, error)
	ApplyAR(ar domain.ARData) error
	SetARSources(src []domain.ARSource) error
	ApplyPurchasing(p domain.Purchasing) error
	SetPRSheet(id string) error
	ResetData(by, when string) (domain.ImportRecord, error)
	Rollback(id string) (domain.ImportRecord, error)

	// users (auth)
	Users() []domain.User
	UserByUsername(username string) (domain.User, error)
	UserByID(id string) (domain.User, error)
}
