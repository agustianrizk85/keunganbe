package service

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"greenpark/finance/internal/domain"
	"greenpark/finance/internal/repository"
)

// This file holds the transactional purchasing business logic: document-number
// generation, terbilang (rupiah-to-words), PO total/tier computation, the PR/PO
// lifecycle transitions and role validation on approval. The repository stays a
// thin CRUD store; all rules live here.

// Sentinel errors mapped to HTTP status codes by the transport layer.
var (
	// ErrValidation → 400 Bad Request.
	ErrValidation = errors.New("validasi gagal")
	// ErrForbidden → 403 Forbidden (role not allowed for this approval tier).
	ErrForbidden = errors.New("akses ditolak")
	// ErrNotFound → 404 Not Found (re-exported for the transport layer).
	ErrNotFound = repository.ErrNotFound
)

/* ------------------------------ helpers ------------------------------- */

var angka = []string{
	"", "satu", "dua", "tiga", "empat", "lima", "enam", "tujuh",
	"delapan", "sembilan", "sepuluh", "sebelas",
}

// spell returns the Indonesian words for a non-negative integer (no "rupiah").
func spell(n int64) string {
	switch {
	case n < 0:
		return "minus " + spell(-n)
	case n < 12:
		return angka[n]
	case n < 20:
		return spell(n-10) + " belas"
	case n < 100:
		return spell(n/10) + " puluh" + tail(n%10)
	case n < 200:
		return "seratus" + tail(n-100)
	case n < 1000:
		return spell(n/100) + " ratus" + tail(n%100)
	case n < 2000:
		return "seribu" + tail(n-1000)
	case n < 1_000_000:
		return spell(n/1000) + " ribu" + tail(n%1000)
	case n < 1_000_000_000:
		return spell(n/1_000_000) + " juta" + tail(n%1_000_000)
	case n < 1_000_000_000_000:
		return spell(n/1_000_000_000) + " miliar" + tail(n%1_000_000_000)
	default:
		return spell(n/1_000_000_000_000) + " triliun" + tail(n%1_000_000_000_000)
	}
}

// tail spells a remainder, prefixed with a space, or "" when zero.
func tail(n int64) string {
	if n == 0 {
		return ""
	}
	return " " + spell(n)
}

// terbilang renders a Rupiah amount as Indonesian words with a " rupiah" suffix.
func terbilang(n int64) string {
	if n == 0 {
		return "nol rupiah"
	}
	if n < 0 {
		return "minus " + terbilang(-n)
	}
	return strings.TrimSpace(spell(n)) + " rupiah"
}

var romanNumerals = []string{
	"", "I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII",
}

func romanMonth(m time.Month) string { return romanNumerals[int(m)] }

// yearOf parses an RFC3339 createdAt stamp and returns its year (0 on failure).
func yearOf(stamp string) int {
	if t, err := time.Parse(time.RFC3339, stamp); err == nil {
		return t.Year()
	}
	return 0
}

// nextNomor builds the next human document number for the given kind ("PR" or
// "PO"): {kind}/{seq:03}/GPG/{ROMAN_MONTH}/{YYYY}, where seq = 1 + count of that
// doc type already created in the calendar year of `now`.
func (s *financeService) nextNomor(kind string, now time.Time) string {
	year := now.Year()
	count := 0
	if kind == "PR" {
		for _, p := range s.repo.PRList() {
			if yearOf(p.CreatedAt) == year {
				count++
			}
		}
	} else {
		for _, p := range s.repo.POListAll() {
			if yearOf(p.CreatedAt) == year {
				count++
			}
		}
	}
	return fmt.Sprintf("%s/%03d/GPG/%s/%d", kind, count+1, romanMonth(now.Month()), year)
}

// computePO fills each item's No + Jumlah and rolls up subTotal/total/terbilang
// plus the approval tier from the total. It is the single source of truth for PO
// money and must run on every create/update.
func computePO(po *domain.PurchaseOrder) {
	var sub int64
	for i := range po.Items {
		it := &po.Items[i]
		it.No = i + 1
		it.Jumlah = int64(math.Round(it.Qty * float64(it.HargaSatuan)))
		sub += it.Jumlah
	}
	po.SubTotal = sub
	po.Total = sub - po.Potongan + po.BiayaPengiriman
	po.Terbilang = terbilang(po.Total)
	switch {
	case po.Total < 500_000:
		po.TanpaPo = true
		po.Tier = "none"
	case po.Total <= 1_000_000:
		po.TanpaPo = false
		po.Tier = "kadep"
	default:
		po.TanpaPo = false
		po.Tier = "dirops"
	}
}

// roleAllowed reports whether role (case-insensitive) is in the allowed set.
func roleAllowed(role string, allowed ...string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	for _, a := range allowed {
		if role == a {
			return true
		}
	}
	return false
}

// today returns the current date as YYYY-MM-DD (used to default requestDate).
func today() string { return time.Now().Format("2006-01-02") }

// nowStamp returns the current instant as RFC3339 (used for approval/reject
// timestamps, matching the repository's createdAt/updatedAt convention).
func nowStamp() string { return time.Now().Format(time.RFC3339) }

/* ---------------------------- master data ----------------------------- */

func (s *financeService) Vendors() []domain.Vendor { return s.repo.Vendors() }
func (s *financeService) CreateVendor(v domain.Vendor) (domain.Vendor, error) {
	if strings.TrimSpace(v.Nama) == "" {
		return domain.Vendor{}, fmt.Errorf("%w: nama vendor wajib diisi", ErrValidation)
	}
	return s.repo.CreateVendor(v)
}
func (s *financeService) UpdateVendor(id string, v domain.Vendor) (domain.Vendor, error) {
	return s.repo.UpdateVendor(id, v)
}
func (s *financeService) DeleteVendor(id string) error { return s.repo.DeleteVendor(id) }

func (s *financeService) ProdukList() []domain.Produk { return s.repo.ProdukList() }
func (s *financeService) CreateProduk(p domain.Produk) (domain.Produk, error) {
	if strings.TrimSpace(p.Nama) == "" {
		return domain.Produk{}, fmt.Errorf("%w: nama produk wajib diisi", ErrValidation)
	}
	s.denormProduk(&p)
	return s.repo.CreateProduk(p)
}
func (s *financeService) UpdateProduk(id string, p domain.Produk) (domain.Produk, error) {
	s.denormProduk(&p)
	return s.repo.UpdateProduk(id, p)
}
func (s *financeService) DeleteProduk(id string) error { return s.repo.DeleteProduk(id) }

// denormProduk fills vendorNama from the referenced vendor for display.
func (s *financeService) denormProduk(p *domain.Produk) {
	if p.VendorID == "" {
		return
	}
	for _, v := range s.repo.Vendors() {
		if v.ID == p.VendorID {
			p.VendorNama = v.Nama
			return
		}
	}
}

func (s *financeService) KaryawanList() []domain.Karyawan { return s.repo.KaryawanList() }
func (s *financeService) CreateKaryawan(k domain.Karyawan) (domain.Karyawan, error) {
	if strings.TrimSpace(k.Nama) == "" {
		return domain.Karyawan{}, fmt.Errorf("%w: nama karyawan wajib diisi", ErrValidation)
	}
	return s.repo.CreateKaryawan(k)
}
func (s *financeService) UpdateKaryawan(id string, k domain.Karyawan) (domain.Karyawan, error) {
	return s.repo.UpdateKaryawan(id, k)
}
func (s *financeService) DeleteKaryawan(id string) error { return s.repo.DeleteKaryawan(id) }

func (s *financeService) SLAList() []domain.SLAItem { return s.repo.SLAList() }
func (s *financeService) CreateSLA(item domain.SLAItem) (domain.SLAItem, error) {
	return s.repo.CreateSLA(item)
}
func (s *financeService) UpdateSLA(id string, item domain.SLAItem) (domain.SLAItem, error) {
	return s.repo.UpdateSLA(id, item)
}
func (s *financeService) DeleteSLA(id string) error { return s.repo.DeleteSLA(id) }

/* ------------------------- purchase request --------------------------- */

func (s *financeService) PRList(status string) []domain.PurchaseRequest {
	all := s.repo.PRList()
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return all
	}
	out := make([]domain.PurchaseRequest, 0, len(all))
	for _, p := range all {
		if strings.ToLower(p.Status) == status {
			out = append(out, p)
		}
	}
	return out
}

func (s *financeService) PRByID(id string) (domain.PurchaseRequest, error) {
	return s.repo.PRByID(id)
}

func (s *financeService) CreatePR(pr domain.PurchaseRequest, submit bool, by string) (domain.PurchaseRequest, error) {
	if strings.TrimSpace(pr.RequestBy) == "" {
		return domain.PurchaseRequest{}, fmt.Errorf("%w: pemohon (requestBy) wajib diisi", ErrValidation)
	}
	if len(pr.Items) == 0 {
		return domain.PurchaseRequest{}, fmt.Errorf("%w: minimal satu item", ErrValidation)
	}
	if pr.RequestDate == "" {
		pr.RequestDate = today()
	}
	numberPRItems(pr.Items)
	pr.CreatedBy = by
	pr.Approval = domain.Approval{}
	if submit {
		pr.Status = "pending"
		pr.Nomor = s.nextNomor("PR", time.Now())
	} else {
		pr.Status = "draft"
	}
	return s.repo.CreatePR(pr)
}

func (s *financeService) UpdatePR(id string, in domain.PurchaseRequest) (domain.PurchaseRequest, error) {
	cur, err := s.repo.PRByID(id)
	if err != nil {
		return domain.PurchaseRequest{}, err
	}
	if cur.Status != "draft" && cur.Status != "pending" {
		return domain.PurchaseRequest{}, fmt.Errorf("%w: PR %s tidak bisa diubah", ErrValidation, cur.Status)
	}
	// Preserve identity + workflow fields; only editable content is replaced.
	in.ID = cur.ID
	in.Nomor = cur.Nomor
	in.Status = cur.Status
	in.Approval = cur.Approval
	in.CreatedBy = cur.CreatedBy
	if in.RequestDate == "" {
		in.RequestDate = cur.RequestDate
	}
	numberPRItems(in.Items)
	return s.repo.UpdatePR(id, in)
}

func (s *financeService) DeletePR(id string) error { return s.repo.DeletePR(id) }

func (s *financeService) SubmitPR(id string) (domain.PurchaseRequest, error) {
	pr, err := s.repo.PRByID(id)
	if err != nil {
		return domain.PurchaseRequest{}, err
	}
	if pr.Status != "draft" {
		return domain.PurchaseRequest{}, fmt.Errorf("%w: hanya PR draft yang bisa disubmit", ErrValidation)
	}
	pr.Status = "pending"
	if strings.TrimSpace(pr.Nomor) == "" {
		pr.Nomor = s.nextNomor("PR", time.Now())
	}
	return s.repo.UpdatePR(id, pr)
}

func (s *financeService) ApprovePR(id string, approver domain.Approver, note string) (domain.PurchaseRequest, error) {
	pr, err := s.repo.PRByID(id)
	if err != nil {
		return domain.PurchaseRequest{}, err
	}
	if pr.Status != "pending" {
		return domain.PurchaseRequest{}, fmt.Errorf("%w: PR harus berstatus pending untuk di-approve", ErrValidation)
	}
	if !roleAllowed(approver.Role, "kadep", "dirops", "ceo", "super") {
		return domain.PurchaseRequest{}, fmt.Errorf("%w: role %q tidak berwenang menyetujui PR", ErrForbidden, approver.Role)
	}
	if strings.EqualFold(approver.Role, "kadep") && approver.Dept != "" && pr.Dept != "" && !strings.EqualFold(approver.Dept, pr.Dept) {
		return domain.PurchaseRequest{}, fmt.Errorf("%w: kadep divisi lain tidak berwenang menyetujui PR divisi %s", ErrForbidden, pr.Dept)
	}
	pr.Status = "approved"
	pr.Approval.ApprovedBy = approver.Name
	pr.Approval.ApprovedByRole = approver.Role
	pr.Approval.ApprovedAt = nowStamp()
	pr.Approval.Note = note
	return s.repo.UpdatePR(id, pr)
}

func (s *financeService) RejectPR(id string, approver domain.Approver, note string) (domain.PurchaseRequest, error) {
	pr, err := s.repo.PRByID(id)
	if err != nil {
		return domain.PurchaseRequest{}, err
	}
	if pr.Status != "pending" {
		return domain.PurchaseRequest{}, fmt.Errorf("%w: PR harus berstatus pending untuk ditolak", ErrValidation)
	}
	if !roleAllowed(approver.Role, "kadep", "dirops", "ceo", "super") {
		return domain.PurchaseRequest{}, fmt.Errorf("%w: role %q tidak berwenang menolak PR", ErrForbidden, approver.Role)
	}
	if strings.EqualFold(approver.Role, "kadep") && approver.Dept != "" && pr.Dept != "" && !strings.EqualFold(approver.Dept, pr.Dept) {
		return domain.PurchaseRequest{}, fmt.Errorf("%w: kadep divisi lain tidak berwenang menolak PR divisi %s", ErrForbidden, pr.Dept)
	}
	pr.Status = "rejected"
	pr.Approval.RejectedBy = approver.Name
	pr.Approval.RejectedByRole = approver.Role
	pr.Approval.RejectedAt = nowStamp()
	pr.Approval.RejectNote = note
	return s.repo.UpdatePR(id, pr)
}

func numberPRItems(items []domain.PRItem) {
	for i := range items {
		items[i].No = i + 1
	}
}

/* --------------------------- purchase order --------------------------- */

func (s *financeService) POList(status string) []domain.PurchaseOrder {
	all := s.repo.POListAll()
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return all
	}
	out := make([]domain.PurchaseOrder, 0, len(all))
	for _, p := range all {
		if strings.ToLower(p.Status) == status {
			out = append(out, p)
		}
	}
	return out
}

func (s *financeService) POByID(id string) (domain.PurchaseOrder, error) {
	return s.repo.POByID(id)
}

func (s *financeService) CreatePO(po domain.PurchaseOrder, submit bool, by string) (domain.PurchaseOrder, error) {
	if len(po.Items) == 0 {
		return domain.PurchaseOrder{}, fmt.Errorf("%w: minimal satu item", ErrValidation)
	}
	// A PO must always reference an approved PR (kadep/dirops approval happens at
	// the PR stage; Purchasing then builds the PO from it) — prefill prNomor.
	if strings.TrimSpace(po.PRID) == "" {
		return domain.PurchaseOrder{}, fmt.Errorf("%w: PR referensi wajib diisi dan harus berstatus approved", ErrValidation)
	}
	pr, err := s.repo.PRByID(po.PRID)
	if err != nil {
		return domain.PurchaseOrder{}, fmt.Errorf("%w: PR referensi tidak ditemukan", ErrValidation)
	}
	if pr.Status != "approved" {
		return domain.PurchaseOrder{}, fmt.Errorf("%w: PR referensi belum di-approve", ErrValidation)
	}
	po.PRNomor = pr.Nomor
	if po.Tanggal == "" {
		po.Tanggal = today()
	}
	po.CreatedBy = by
	po.Approval = domain.Approval{}
	po.Receiving = domain.Receiving{}
	computePO(&po)

	switch {
	case po.TanpaPo:
		// Pembelian langsung < Rp500.000 — auto-approved, no approval needed.
		po.Status = "approved"
		po.Nomor = s.nextNomor("PO", time.Now())
	case submit:
		po.Status = "pending"
		po.Nomor = s.nextNomor("PO", time.Now())
	default:
		po.Status = "draft"
	}
	return s.repo.CreatePO(po)
}

func (s *financeService) UpdatePO(id string, in domain.PurchaseOrder) (domain.PurchaseOrder, error) {
	cur, err := s.repo.POByID(id)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	if cur.Status != "draft" && cur.Status != "pending" {
		return domain.PurchaseOrder{}, fmt.Errorf("%w: PO %s tidak bisa diubah", ErrValidation, cur.Status)
	}
	// Preserve identity + workflow fields; recompute money from the new items.
	in.ID = cur.ID
	in.Nomor = cur.Nomor
	in.Status = cur.Status
	in.PRID = cur.PRID
	in.PRNomor = cur.PRNomor
	in.Approval = cur.Approval
	in.Receiving = cur.Receiving
	in.CreatedBy = cur.CreatedBy
	if in.Tanggal == "" {
		in.Tanggal = cur.Tanggal
	}
	computePO(&in)
	return s.repo.UpdatePO(id, in)
}

func (s *financeService) DeletePO(id string) error { return s.repo.DeletePO(id) }

func (s *financeService) SubmitPO(id string) (domain.PurchaseOrder, error) {
	po, err := s.repo.POByID(id)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	if po.Status != "draft" {
		return domain.PurchaseOrder{}, fmt.Errorf("%w: hanya PO draft yang bisa disubmit", ErrValidation)
	}
	computePO(&po)
	if po.TanpaPo {
		po.Status = "approved" // < Rp500.000 auto-approved on submit
	} else {
		po.Status = "pending"
	}
	if strings.TrimSpace(po.Nomor) == "" {
		po.Nomor = s.nextNomor("PO", time.Now())
	}
	return s.repo.UpdatePO(id, po)
}

func (s *financeService) ApprovePO(id string, approver domain.Approver, note string) (domain.PurchaseOrder, error) {
	po, err := s.repo.POByID(id)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	if po.Status != "pending" {
		return domain.PurchaseOrder{}, fmt.Errorf("%w: PO harus berstatus pending untuk di-approve", ErrValidation)
	}
	if err := validatePOTier(po.Tier, approver.Role); err != nil {
		return domain.PurchaseOrder{}, err
	}
	po.Status = "approved"
	po.Approval.ApprovedBy = approver.Name
	po.Approval.ApprovedByRole = approver.Role
	po.Approval.ApprovedAt = nowStamp()
	po.Approval.Note = note
	return s.repo.UpdatePO(id, po)
}

func (s *financeService) RejectPO(id string, approver domain.Approver, note string) (domain.PurchaseOrder, error) {
	po, err := s.repo.POByID(id)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	if po.Status != "pending" {
		return domain.PurchaseOrder{}, fmt.Errorf("%w: PO harus berstatus pending untuk ditolak", ErrValidation)
	}
	if err := validatePOTier(po.Tier, approver.Role); err != nil {
		return domain.PurchaseOrder{}, err
	}
	po.Status = "rejected"
	po.Approval.RejectedBy = approver.Name
	po.Approval.RejectedByRole = approver.Role
	po.Approval.RejectedAt = nowStamp()
	po.Approval.RejectNote = note
	return s.repo.UpdatePO(id, po)
}

// validatePOTier enforces the approver role required for a PO's tier.
func validatePOTier(tier, role string) error {
	switch tier {
	case "kadep":
		if !roleAllowed(role, "kadep", "dirops", "ceo", "super") {
			return fmt.Errorf("%w: role %q tidak berwenang menyetujui PO tier kadep", ErrForbidden, role)
		}
	case "dirops":
		if !roleAllowed(role, "dirops", "ceo", "super") {
			return fmt.Errorf("%w: role %q tidak berwenang menyetujui PO tier dirops", ErrForbidden, role)
		}
	default: // "none" — tanpa PO, no approval endpoint applies
		return fmt.Errorf("%w: PO tanpa-PO tidak memerlukan approval", ErrValidation)
	}
	return nil
}

func (s *financeService) ReceivePO(id, tanggalDiterima string, bastSigned bool, keterangan string) (domain.PurchaseOrder, error) {
	po, err := s.repo.POByID(id)
	if err != nil {
		return domain.PurchaseOrder{}, err
	}
	if po.Status != "approved" && po.Status != "received" {
		return domain.PurchaseOrder{}, fmt.Errorf("%w: PO harus di-approve sebelum penerimaan", ErrValidation)
	}
	if strings.TrimSpace(tanggalDiterima) == "" {
		tanggalDiterima = today()
	}
	po.Receiving.Received = true
	po.Receiving.TanggalDiterima = tanggalDiterima
	po.Receiving.BastSigned = bastSigned

	// SLA days measured from the order date (fallback: delivery date) to receipt.
	base := po.Tanggal
	if base == "" {
		base = po.TanggalPengiriman
	}
	po.Receiving.SLAHari = dateDiffDays(base, tanggalDiterima)

	// Derive on-time / late vs the promised delivery date unless caller set it.
	if strings.TrimSpace(keterangan) != "" {
		po.Receiving.Keterangan = keterangan
	} else if po.TanggalPengiriman != "" {
		if !parseDate(tanggalDiterima).After(parseDate(po.TanggalPengiriman)) {
			po.Receiving.Keterangan = "Tepat Waktu"
		} else {
			po.Receiving.Keterangan = "Terlambat"
		}
	} else {
		po.Receiving.Keterangan = ""
	}
	po.Status = "received"
	return s.repo.UpdatePO(id, po)
}

// parseDate parses a YYYY-MM-DD date; the zero Time on failure.
func parseDate(s string) time.Time {
	t, _ := time.Parse("2006-01-02", strings.TrimSpace(s))
	return t
}

// dateDiffDays returns the whole-day difference from → to (0 if either unparses
// or the result would be negative).
func dateDiffDays(from, to string) int {
	f := parseDate(from)
	t := parseDate(to)
	if f.IsZero() || t.IsZero() {
		return 0
	}
	d := int(t.Sub(f).Hours() / 24)
	if d < 0 {
		return 0
	}
	return d
}
