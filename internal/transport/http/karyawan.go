package http

import (
	"net/http"

	"greenpark/finance/internal/domain"
)

// karyawanList returns all karyawan master records.
func (h *Handler) karyawanList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.KaryawanList())
}

// karyawanCreate creates a karyawan.
func (h *Handler) karyawanCreate(w http.ResponseWriter, r *http.Request) {
	in, ok := decode[domain.Karyawan](w, r)
	if !ok {
		return
	}
	k, err := h.svc.CreateKaryawan(in)
	if err != nil {
		writeSvcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, k)
}

// karyawanUpdate updates a karyawan by id.
func (h *Handler) karyawanUpdate(w http.ResponseWriter, r *http.Request) {
	in, ok := decode[domain.Karyawan](w, r)
	if !ok {
		return
	}
	k, err := h.svc.UpdateKaryawan(r.PathValue("id"), in)
	if err != nil {
		writeSvcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, k)
}

// karyawanDelete deletes a karyawan by id.
func (h *Handler) karyawanDelete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteKaryawan(r.PathValue("id")); err != nil {
		writeSvcErr(w, err)
		return
	}
	okResp(w)
}
