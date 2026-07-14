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

	// transactional purchasing — master data (create assigns id + timestamps,
	// update preserves id + createdAt, all writes bump the revision).
	Vendors() []domain.Vendor
	CreateVendor(v domain.Vendor) (domain.Vendor, error)
	UpdateVendor(id string, v domain.Vendor) (domain.Vendor, error)
	DeleteVendor(id string) error

	ProdukList() []domain.Produk
	CreateProduk(p domain.Produk) (domain.Produk, error)
	UpdateProduk(id string, p domain.Produk) (domain.Produk, error)
	DeleteProduk(id string) error

	KaryawanList() []domain.Karyawan
	CreateKaryawan(k domain.Karyawan) (domain.Karyawan, error)
	UpdateKaryawan(id string, k domain.Karyawan) (domain.Karyawan, error)
	DeleteKaryawan(id string) error

	SLAList() []domain.SLAItem
	CreateSLA(s domain.SLAItem) (domain.SLAItem, error)
	UpdateSLA(id string, s domain.SLAItem) (domain.SLAItem, error)
	DeleteSLA(id string) error

	// transactional purchasing — PR/PO workflow
	PRList() []domain.PurchaseRequest
	PRByID(id string) (domain.PurchaseRequest, error)
	CreatePR(pr domain.PurchaseRequest) (domain.PurchaseRequest, error)
	UpdatePR(id string, pr domain.PurchaseRequest) (domain.PurchaseRequest, error)
	DeletePR(id string) error

	POListAll() []domain.PurchaseOrder
	POByID(id string) (domain.PurchaseOrder, error)
	CreatePO(po domain.PurchaseOrder) (domain.PurchaseOrder, error)
	UpdatePO(id string, po domain.PurchaseOrder) (domain.PurchaseOrder, error)
	DeletePO(id string) error
}
