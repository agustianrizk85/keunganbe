package http

import (
	"net/http"

	"greenpark/finance/internal/domain"
)

// vendorsList returns all vendor master records.
func (h *Handler) vendorsList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.Vendors())
}

// vendorCreate creates a vendor.
func (h *Handler) vendorCreate(w http.ResponseWriter, r *http.Request) {
	in, ok := decode[domain.Vendor](w, r)
	if !ok {
		return
	}
	v, err := h.svc.CreateVendor(in)
	if err != nil {
		writeSvcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

// vendorUpdate updates a vendor by id.
func (h *Handler) vendorUpdate(w http.ResponseWriter, r *http.Request) {
	in, ok := decode[domain.Vendor](w, r)
	if !ok {
		return
	}
	v, err := h.svc.UpdateVendor(r.PathValue("id"), in)
	if err != nil {
		writeSvcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

// vendorDelete deletes a vendor by id.
func (h *Handler) vendorDelete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteVendor(r.PathValue("id")); err != nil {
		writeSvcErr(w, err)
		return
	}
	okResp(w)
}
