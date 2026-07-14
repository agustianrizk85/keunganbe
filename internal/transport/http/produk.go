package http

import (
	"net/http"

	"greenpark/finance/internal/domain"
)

// produkList returns all produk master records.
func (h *Handler) produkList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.ProdukList())
}

// produkCreate creates a produk.
func (h *Handler) produkCreate(w http.ResponseWriter, r *http.Request) {
	in, ok := decode[domain.Produk](w, r)
	if !ok {
		return
	}
	p, err := h.svc.CreateProduk(in)
	if err != nil {
		writeSvcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// produkUpdate updates a produk by id.
func (h *Handler) produkUpdate(w http.ResponseWriter, r *http.Request) {
	in, ok := decode[domain.Produk](w, r)
	if !ok {
		return
	}
	p, err := h.svc.UpdateProduk(r.PathValue("id"), in)
	if err != nil {
		writeSvcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// produkDelete deletes a produk by id.
func (h *Handler) produkDelete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteProduk(r.PathValue("id")); err != nil {
		writeSvcErr(w, err)
		return
	}
	okResp(w)
}
