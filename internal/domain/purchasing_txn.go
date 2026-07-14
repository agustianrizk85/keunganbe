package domain

// This file holds the TRANSACTIONAL purchasing types (master data + PR/PO
// workflow) that back the keuangan Purchasing sub-module. They are entirely
// separate from the analytics `Purchasing` view (finance.go) which is derived
// from a synced spreadsheet — these are user-created records stored in the JSON
// state blob. All money fields are FULL RUPIAH int64 (not millions).

// Vendor is a supplier master record.
type Vendor struct {
	ID        string `json:"id"`
	Nama      string `json:"nama"`
	Alamat    string `json:"alamat"`
	Telepon   string `json:"telepon"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// Produk is a purchasable item / material master record.
type Produk struct {
	ID         string `json:"id"`
	Nama       string `json:"nama"`
	VendorID   string `json:"vendorId"`
	VendorNama string `json:"vendorNama"` // denormalized for display
	Harga      int64  `json:"harga"`      // Rupiah
	Satuan     string `json:"satuan"`
	Negotiable bool   `json:"negotiable"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

// Karyawan is an employee master record (used for pemohon / approver pickers).
type Karyawan struct {
	ID        string `json:"id"`
	Nama      string `json:"nama"`
	Jabatan   string `json:"jabatan"`
	Divisi    string `json:"divisi"`
	Role      string `json:"role"` // staff | kadep | dirops | purchasing | ...
	Telepon   string `json:"telepon"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// SLAItem is one row of the procurement SLA master (SOP 5.2).
type SLAItem struct {
	ID            string `json:"id"`
	No            int    `json:"no"`
	Aktivitas     string `json:"aktivitas"`
	PIC           string `json:"pic"`
	SLAHari       string `json:"slaHari"`       // free text e.g. "1 Hari", "Sesuai tempo"
	SLATargetHari int    `json:"slaTargetHari"` // numeric target for keterangan; 0 if N/A
	Output        string `json:"output"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

// Approver identifies who is acting on an approve/reject (captured from the FE
// SSO user, since the backend authenticates with a shared service account).
type Approver struct {
	Name string `json:"name"`
	Role string `json:"role"`
	Dept string `json:"dept"` // FE dashboard Division string (e.g. "teknik","keuangan") of the approver — used to scope kadep approval to their own department. Empty = not supplied (legacy caller), skip the department check.
}

// Approval is the embedded approval/rejection record on a PR or PO.
type Approval struct {
	ApprovedBy     string `json:"approvedBy"`
	ApprovedByRole string `json:"approvedByRole"`
	ApprovedAt     string `json:"approvedAt"`
	Note           string `json:"note"`
	RejectedBy     string `json:"rejectedBy"`
	RejectedByRole string `json:"rejectedByRole"`
	RejectedAt     string `json:"rejectedAt"`
	RejectNote     string `json:"rejectNote"`
}

// Receiving is the embedded goods-receipt / BAST record on a PO.
type Receiving struct {
	Received        bool   `json:"received"`
	TanggalDiterima string `json:"tanggalDiterima"`
	BastSigned      bool   `json:"bastSigned"`
	Keterangan      string `json:"keterangan"` // "Tepat Waktu" | "Terlambat" | ""
	SLAHari         int    `json:"slaHari"`    // actual days from tanggal to diterima
}

// PRItem is one requested line of a Purchase Request.
type PRItem struct {
	No     int     `json:"no"`
	Nama   string  `json:"nama"`
	Satuan string  `json:"satuan"`
	Qty    float64 `json:"qty"`
	Tujuan string  `json:"tujuan"` // keperluan / tujuan item
}

// PurchaseRequest is a purchase request (stage 1 of the flow).
type PurchaseRequest struct {
	ID               string   `json:"id"`
	Nomor            string   `json:"nomor"`
	Status           string   `json:"status"`       // draft | pending | approved | rejected
	RequestDate      string   `json:"requestDate"`  // YYYY-MM-DD
	DateRequired     string   `json:"dateRequired"` // perkiraan material sampai, YYYY-MM-DD
	RequestBy        string   `json:"requestBy"`    // nama karyawan pemohon
	Dept             string   `json:"dept"`         // divisi pemohon
	Proyek           string   `json:"proyek"`
	Supplier         string   `json:"supplier"` // vendor nama (optional at PR)
	AlamatPengiriman string   `json:"alamatPengiriman"`
	PIC              string   `json:"pic"`
	Items            []PRItem `json:"items"`
	Catatan          string   `json:"catatan"`
	DiajukanOleh     string   `json:"diajukanOleh"`
	DiketahuiOleh    string   `json:"diketahuiOleh"`
	Approval         Approval `json:"approval"`
	CreatedAt        string   `json:"createdAt"`
	UpdatedAt        string   `json:"updatedAt"`
	CreatedBy        string   `json:"createdBy"`
}

// POItem is one line of a Purchase Order. Jumlah = round(Qty*HargaSatuan).
type POItem struct {
	No          int     `json:"no"`
	Nama        string  `json:"nama"`
	Satuan      string  `json:"satuan"`
	Qty         float64 `json:"qty"`
	HargaSatuan int64   `json:"hargaSatuan"`
	Jumlah      int64   `json:"jumlah"`
}

// PurchaseOrder is a purchase order (stage 2 of the flow).
type PurchaseOrder struct {
	ID                string    `json:"id"`
	Nomor             string    `json:"nomor"`
	PRID              string    `json:"prId"`    // ref PurchaseRequest
	PRNomor           string    `json:"prNomor"` // denormalized
	Status            string    `json:"status"`  // draft | pending | approved | rejected | received | completed
	Tanggal           string    `json:"tanggal"`
	TanggalPengiriman string    `json:"tanggalPengiriman"`
	SyaratPembayaran  string    `json:"syaratPembayaran"` // Tempo | Cash | free text
	CaraBayar         string    `json:"caraBayar"`        // Tempo | Cash
	Purchaser         string    `json:"purchaser"`        // staff purchasing nama
	Supplier          string    `json:"supplier"`         // vendor nama
	VendorID          string    `json:"vendorId"`
	AlamatPengiriman  string    `json:"alamatPengiriman"`
	PIC               string    `json:"pic"`
	Proyek            string    `json:"proyek"`
	Items             []POItem  `json:"items"`
	SubTotal          int64     `json:"subTotal"`
	Potongan          int64     `json:"potongan"`
	BiayaPengiriman   int64     `json:"biayaPengiriman"`
	Total             int64     `json:"total"`
	Terbilang         string    `json:"terbilang"`
	TanpaPo           bool      `json:"tanpaPo"` // total < 500000
	Tier              string    `json:"tier"`    // none | kadep | dirops
	DisiapkanOleh     string    `json:"disiapkanOleh"`
	DiketahuiOleh     string    `json:"diketahuiOleh"`
	DisetujuiOleh     string    `json:"disetujuiOleh"`
	Approval          Approval  `json:"approval"`
	Receiving         Receiving `json:"receiving"`
	Catatan           string    `json:"catatan"`
	CreatedAt         string    `json:"createdAt"`
	UpdatedAt         string    `json:"updatedAt"`
	CreatedBy         string    `json:"createdBy"`
}

// EmptyPurchaseRequest returns a PR with a non-nil (empty) items slice so the
// JSON serialises "items":[] not null.
func EmptyPurchaseRequest() PurchaseRequest {
	return PurchaseRequest{Status: "draft", Items: []PRItem{}}
}

// EmptyPurchaseOrder returns a PO with a non-nil (empty) items slice.
func EmptyPurchaseOrder() PurchaseOrder {
	return PurchaseOrder{Status: "draft", Items: []POItem{}}
}
