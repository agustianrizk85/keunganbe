// Package service holds the business logic of the Finance dashboard. It exposes
// the dashboard read and the ingest use-cases (preview/approve from an XLSX
// upload or a live Google Sheet, plus reset and rollback), keeping transport
// handlers thin. The dashboard is assembled by the ingest engine and stored
// whole, so reads are a straight pass-through.
package service

import (
	"io"

	"greenpark/finance/internal/domain"
	"greenpark/finance/internal/ingest"
	"greenpark/finance/internal/repository"
)

// Options configures how imports are interpreted (executive focus).
type Options struct {
	FocusYear  int
	TargetAkad int
}

func (o Options) ingest() ingest.Options {
	return ingest.Options{FocusYear: o.FocusYear, TargetAkad: o.TargetAkad}
}

// FinanceService exposes the read and ingest use-cases of the dashboard.
type FinanceService interface {
	// reads
	Dashboard() domain.Dashboard
	AR() domain.ARData
	Purchasing() domain.Purchasing
	Summary() domain.Summary
	Revision() int64
	ImportHistory() []domain.ImportRecord

	// ingest (no side effects)
	PreviewImport(r io.Reader) (*ingest.Result, error)
	PreviewSheets(data map[string][][]string) (*ingest.Result, error)

	// AR (piutang) ingest from the per-project input sheets
	PreviewAR(data map[string][][]string) domain.ARData
	ApproveAR(data map[string][][]string) (domain.ARData, error)
	ARSources() []domain.ARSource
	SetARSources(src []domain.ARSource) error

	// procurement (PR) ingest from the "Pembelian (PR)" spreadsheet
	PreviewPurchasing(data map[string][][]string) ingest.PRResult
	ApprovePurchasing(data map[string][][]string) (domain.Purchasing, error)
	PRSheet() string
	SetPRSheet(id string) error

	// ingest (apply + record rollback snapshot)
	ApproveImport(r io.Reader, filename, by string) (domain.ImportRecord, error)
	ApproveSheets(data map[string][][]string, filename, by string) (domain.ImportRecord, error)

	// lifecycle
	ResetData(by string) (domain.ImportRecord, error)
	RollbackImport(id string) (domain.ImportRecord, error)

	// transactional purchasing — master data
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
	PRList(status string) []domain.PurchaseRequest
	PRByID(id string) (domain.PurchaseRequest, error)
	CreatePR(pr domain.PurchaseRequest, submit bool, by string) (domain.PurchaseRequest, error)
	UpdatePR(id string, pr domain.PurchaseRequest) (domain.PurchaseRequest, error)
	DeletePR(id string) error
	SubmitPR(id string) (domain.PurchaseRequest, error)
	ApprovePR(id string, approver domain.Approver, note string) (domain.PurchaseRequest, error)
	RejectPR(id string, approver domain.Approver, note string) (domain.PurchaseRequest, error)

	POList(status string) []domain.PurchaseOrder
	POByID(id string) (domain.PurchaseOrder, error)
	CreatePO(po domain.PurchaseOrder, submit bool, by string) (domain.PurchaseOrder, error)
	UpdatePO(id string, po domain.PurchaseOrder) (domain.PurchaseOrder, error)
	DeletePO(id string) error
	SubmitPO(id string) (domain.PurchaseOrder, error)
	ApprovePO(id string, approver domain.Approver, note string) (domain.PurchaseOrder, error)
	RejectPO(id string, approver domain.Approver, note string) (domain.PurchaseOrder, error)
	ReceivePO(id, tanggalDiterima string, bastSigned bool, keterangan string) (domain.PurchaseOrder, error)
}

type financeService struct {
	repo repository.FinanceRepository
	opts Options
}

// New returns a FinanceService backed by the given repository and ingest options.
func New(repo repository.FinanceRepository, opts Options) FinanceService {
	if opts.FocusYear == 0 {
		opts.FocusYear = 2026
	}
	return &financeService{repo: repo, opts: opts}
}

func (s *financeService) Dashboard() domain.Dashboard          { return s.repo.Dashboard() }
func (s *financeService) AR() domain.ARData                    { return s.repo.AR() }
func (s *financeService) Summary() domain.Summary              { return s.repo.Dashboard().Summary }
func (s *financeService) Revision() int64                      { return s.repo.Revision() }
func (s *financeService) ImportHistory() []domain.ImportRecord { return s.repo.ImportHistory() }
func (s *financeService) ARSources() []domain.ARSource         { return s.repo.ARSources() }
func (s *financeService) SetARSources(src []domain.ARSource) error {
	return s.repo.SetARSources(src)
}
