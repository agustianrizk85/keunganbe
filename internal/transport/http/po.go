package http

import (
	"net/http"

	"greenpark/finance/internal/domain"
)

// poCreateReq is a PurchaseOrder body plus the optional submit flag.
type poCreateReq struct {
	domain.PurchaseOrder
	Submit bool `json:"submit"`
}

// poList returns purchase orders, optionally filtered by ?status=.
func (h *Handler) poList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.POList(r.URL.Query().Get("status")))
}

// poGet returns one purchase order by id.
func (h *Handler) poGet(w http.ResponseWriter, r *http.Request) {
	po, err := h.svc.POByID(r.PathValue("id"))
	if err != nil {
		writeSvcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, po)
}

// poCreate creates a purchase order; the backend computes totals/tier/terbilang.
func (h *Handler) poCreate(w http.ResponseWriter, r *http.Request) {
	in, ok := decode[poCreateReq](w, r)
	if !ok {
		return
	}
	po, err := h.svc.CreatePO(in.PurchaseOrder, in.Submit, actor(r))
	if err != nil {
		writeSvcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, po)
}

// poUpdate edits a draft/pending PO (recomputes totals).
func (h *Handler) poUpdate(w http.ResponseWriter, r *http.Request) {
	in, ok := decode[domain.PurchaseOrder](w, r)
	if !ok {
		return
	}
	po, err := h.svc.UpdatePO(r.PathValue("id"), in)
	if err != nil {
		writeSvcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, po)
}

// poDelete deletes a purchase order.
func (h *Handler) poDelete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeletePO(r.PathValue("id")); err != nil {
		writeSvcErr(w, err)
		return
	}
	okResp(w)
}

// poSubmit transitions a draft PO to pending (or approved when tanpaPo).
func (h *Handler) poSubmit(w http.ResponseWriter, r *http.Request) {
	po, err := h.svc.SubmitPO(r.PathValue("id"))
	if err != nil {
		writeSvcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, po)
}

// poApprove approves a pending PO (validates tier vs approver role).
func (h *Handler) poApprove(w http.ResponseWriter, r *http.Request) {
	req, ok := decode[approveReq](w, r)
	if !ok {
		return
	}
	po, err := h.svc.ApprovePO(r.PathValue("id"), req.Approver, req.Note)
	if err != nil {
		writeSvcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, po)
}

// poReject rejects a pending PO.
func (h *Handler) poReject(w http.ResponseWriter, r *http.Request) {
	req, ok := decode[approveReq](w, r)
	if !ok {
		return
	}
	po, err := h.svc.RejectPO(r.PathValue("id"), req.Approver, req.Note)
	if err != nil {
		writeSvcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, po)
}

// poReceive records goods receipt / BAST and computes the SLA outcome.
func (h *Handler) poReceive(w http.ResponseWriter, r *http.Request) {
	req, ok := decode[receiveReq](w, r)
	if !ok {
		return
	}
	po, err := h.svc.ReceivePO(r.PathValue("id"), req.TanggalDiterima, req.BastSigned, req.Keterangan)
	if err != nil {
		writeSvcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, po)
}

// purchasingRegister returns all POs flattened for the "Laporan PR&PO" table.
func (h *Handler) purchasingRegister(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.POList(""))
}
