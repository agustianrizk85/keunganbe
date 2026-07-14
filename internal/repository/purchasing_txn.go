package repository

import (
	"time"

	"greenpark/finance/internal/domain"
)

// This file implements the transactional purchasing storage on the file-backed
// fileRepository. Every method follows the existing write pattern: take the
// write lock, mutate r.st, bump r.st.Rev on writes, and flush via r.persist().
// Create assigns the id + createdAt/updatedAt; Update preserves id + createdAt
// and refreshes updatedAt. Timestamps are RFC3339 (sortable, parseable).

func nowStamp() string { return time.Now().Format(time.RFC3339) }

/* ------------------------------- vendors ------------------------------- */

func (r *fileRepository) Vendors() []domain.Vendor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.Vendor, len(r.st.Vendors))
	copy(out, r.st.Vendors)
	return out
}

func (r *fileRepository) CreateVendor(v domain.Vendor) (domain.Vendor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v.ID = newID("ven")
	v.CreatedAt = nowStamp()
	v.UpdatedAt = v.CreatedAt
	r.st.Vendors = append(r.st.Vendors, v)
	r.st.Rev++
	return v, r.persist()
}

func (r *fileRepository) UpdateVendor(id string, v domain.Vendor) (domain.Vendor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.st.Vendors {
		if r.st.Vendors[i].ID == id {
			v.ID = id
			v.CreatedAt = r.st.Vendors[i].CreatedAt
			v.UpdatedAt = nowStamp()
			r.st.Vendors[i] = v
			r.st.Rev++
			return v, r.persist()
		}
	}
	return domain.Vendor{}, ErrNotFound
}

func (r *fileRepository) DeleteVendor(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.st.Vendors {
		if r.st.Vendors[i].ID == id {
			r.st.Vendors = append(r.st.Vendors[:i], r.st.Vendors[i+1:]...)
			r.st.Rev++
			return r.persist()
		}
	}
	return ErrNotFound
}

/* ------------------------------- produk -------------------------------- */

func (r *fileRepository) ProdukList() []domain.Produk {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.Produk, len(r.st.Produk))
	copy(out, r.st.Produk)
	return out
}

func (r *fileRepository) CreateProduk(p domain.Produk) (domain.Produk, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p.ID = newID("prd")
	p.CreatedAt = nowStamp()
	p.UpdatedAt = p.CreatedAt
	r.st.Produk = append(r.st.Produk, p)
	r.st.Rev++
	return p, r.persist()
}

func (r *fileRepository) UpdateProduk(id string, p domain.Produk) (domain.Produk, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.st.Produk {
		if r.st.Produk[i].ID == id {
			p.ID = id
			p.CreatedAt = r.st.Produk[i].CreatedAt
			p.UpdatedAt = nowStamp()
			r.st.Produk[i] = p
			r.st.Rev++
			return p, r.persist()
		}
	}
	return domain.Produk{}, ErrNotFound
}

func (r *fileRepository) DeleteProduk(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.st.Produk {
		if r.st.Produk[i].ID == id {
			r.st.Produk = append(r.st.Produk[:i], r.st.Produk[i+1:]...)
			r.st.Rev++
			return r.persist()
		}
	}
	return ErrNotFound
}

/* ------------------------------ karyawan ------------------------------- */

func (r *fileRepository) KaryawanList() []domain.Karyawan {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.Karyawan, len(r.st.Karyawan))
	copy(out, r.st.Karyawan)
	return out
}

func (r *fileRepository) CreateKaryawan(k domain.Karyawan) (domain.Karyawan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k.ID = newID("kar")
	k.CreatedAt = nowStamp()
	k.UpdatedAt = k.CreatedAt
	r.st.Karyawan = append(r.st.Karyawan, k)
	r.st.Rev++
	return k, r.persist()
}

func (r *fileRepository) UpdateKaryawan(id string, k domain.Karyawan) (domain.Karyawan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.st.Karyawan {
		if r.st.Karyawan[i].ID == id {
			k.ID = id
			k.CreatedAt = r.st.Karyawan[i].CreatedAt
			k.UpdatedAt = nowStamp()
			r.st.Karyawan[i] = k
			r.st.Rev++
			return k, r.persist()
		}
	}
	return domain.Karyawan{}, ErrNotFound
}

func (r *fileRepository) DeleteKaryawan(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.st.Karyawan {
		if r.st.Karyawan[i].ID == id {
			r.st.Karyawan = append(r.st.Karyawan[:i], r.st.Karyawan[i+1:]...)
			r.st.Rev++
			return r.persist()
		}
	}
	return ErrNotFound
}

/* -------------------------------- sla ---------------------------------- */

func (r *fileRepository) SLAList() []domain.SLAItem {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.SLAItem, len(r.st.SLA))
	copy(out, r.st.SLA)
	return out
}

func (r *fileRepository) CreateSLA(s domain.SLAItem) (domain.SLAItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s.ID = newID("sla")
	s.CreatedAt = nowStamp()
	s.UpdatedAt = s.CreatedAt
	r.st.SLA = append(r.st.SLA, s)
	r.st.Rev++
	return s, r.persist()
}

func (r *fileRepository) UpdateSLA(id string, s domain.SLAItem) (domain.SLAItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.st.SLA {
		if r.st.SLA[i].ID == id {
			s.ID = id
			s.CreatedAt = r.st.SLA[i].CreatedAt
			s.UpdatedAt = nowStamp()
			r.st.SLA[i] = s
			r.st.Rev++
			return s, r.persist()
		}
	}
	return domain.SLAItem{}, ErrNotFound
}

func (r *fileRepository) DeleteSLA(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.st.SLA {
		if r.st.SLA[i].ID == id {
			r.st.SLA = append(r.st.SLA[:i], r.st.SLA[i+1:]...)
			r.st.Rev++
			return r.persist()
		}
	}
	return ErrNotFound
}

/* --------------------------- purchase request -------------------------- */

func (r *fileRepository) PRList() []domain.PurchaseRequest {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.PurchaseRequest, len(r.st.PRs))
	copy(out, r.st.PRs)
	return out
}

func (r *fileRepository) PRByID(id string) (domain.PurchaseRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.st.PRs {
		if p.ID == id {
			return p, nil
		}
	}
	return domain.PurchaseRequest{}, ErrNotFound
}

func (r *fileRepository) CreatePR(pr domain.PurchaseRequest) (domain.PurchaseRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	pr.ID = newID("pr")
	pr.CreatedAt = nowStamp()
	pr.UpdatedAt = pr.CreatedAt
	if pr.Items == nil {
		pr.Items = []domain.PRItem{}
	}
	r.st.PRs = append(r.st.PRs, pr)
	r.st.Rev++
	return pr, r.persist()
}

func (r *fileRepository) UpdatePR(id string, pr domain.PurchaseRequest) (domain.PurchaseRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.st.PRs {
		if r.st.PRs[i].ID == id {
			pr.ID = id
			pr.CreatedAt = r.st.PRs[i].CreatedAt
			pr.UpdatedAt = nowStamp()
			if pr.Items == nil {
				pr.Items = []domain.PRItem{}
			}
			r.st.PRs[i] = pr
			r.st.Rev++
			return pr, r.persist()
		}
	}
	return domain.PurchaseRequest{}, ErrNotFound
}

func (r *fileRepository) DeletePR(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.st.PRs {
		if r.st.PRs[i].ID == id {
			r.st.PRs = append(r.st.PRs[:i], r.st.PRs[i+1:]...)
			r.st.Rev++
			return r.persist()
		}
	}
	return ErrNotFound
}

/* ---------------------------- purchase order --------------------------- */

func (r *fileRepository) POListAll() []domain.PurchaseOrder {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.PurchaseOrder, len(r.st.POs))
	copy(out, r.st.POs)
	return out
}

func (r *fileRepository) POByID(id string) (domain.PurchaseOrder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.st.POs {
		if p.ID == id {
			return p, nil
		}
	}
	return domain.PurchaseOrder{}, ErrNotFound
}

func (r *fileRepository) CreatePO(po domain.PurchaseOrder) (domain.PurchaseOrder, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	po.ID = newID("po")
	po.CreatedAt = nowStamp()
	po.UpdatedAt = po.CreatedAt
	if po.Items == nil {
		po.Items = []domain.POItem{}
	}
	r.st.POs = append(r.st.POs, po)
	r.st.Rev++
	return po, r.persist()
}

func (r *fileRepository) UpdatePO(id string, po domain.PurchaseOrder) (domain.PurchaseOrder, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.st.POs {
		if r.st.POs[i].ID == id {
			po.ID = id
			po.CreatedAt = r.st.POs[i].CreatedAt
			po.UpdatedAt = nowStamp()
			if po.Items == nil {
				po.Items = []domain.POItem{}
			}
			r.st.POs[i] = po
			r.st.Rev++
			return po, r.persist()
		}
	}
	return domain.PurchaseOrder{}, ErrNotFound
}

func (r *fileRepository) DeletePO(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.st.POs {
		if r.st.POs[i].ID == id {
			r.st.POs = append(r.st.POs[:i], r.st.POs[i+1:]...)
			r.st.Rev++
			return r.persist()
		}
	}
	return ErrNotFound
}
