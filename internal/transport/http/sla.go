package http

import (
	"net/http"

	"greenpark/finance/internal/domain"
)

// slaList returns all SLA master rows.
func (h *Handler) slaList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.SLAList())
}

// slaCreate creates an SLA row.
func (h *Handler) slaCreate(w http.ResponseWriter, r *http.Request) {
	in, ok := decode[domain.SLAItem](w, r)
	if !ok {
		return
	}
	item, err := h.svc.CreateSLA(in)
	if err != nil {
		writeSvcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// slaUpdate updates an SLA row by id.
func (h *Handler) slaUpdate(w http.ResponseWriter, r *http.Request) {
	in, ok := decode[domain.SLAItem](w, r)
	if !ok {
		return
	}
	item, err := h.svc.UpdateSLA(r.PathValue("id"), in)
	if err != nil {
		writeSvcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// slaDelete deletes an SLA row by id.
func (h *Handler) slaDelete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteSLA(r.PathValue("id")); err != nil {
		writeSvcErr(w, err)
		return
	}
	okResp(w)
}
