package http

import (
	"net/http"

	"greenpark/finance/internal/domain"
)

// prCreateReq is a PurchaseRequest body plus the optional submit flag (create +
// submit in one call). Embedded fields are inlined in the JSON.
type prCreateReq struct {
	domain.PurchaseRequest
	Submit bool `json:"submit"`
}

// prList returns purchase requests, optionally filtered by ?status=.
func (h *Handler) prList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.PRList(r.URL.Query().Get("status")))
}

// prGet returns one purchase request by id.
func (h *Handler) prGet(w http.ResponseWriter, r *http.Request) {
	pr, err := h.svc.PRByID(r.PathValue("id"))
	if err != nil {
		writeSvcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pr)
}

// prCreate creates a purchase request (draft, or pending when submit:true).
func (h *Handler) prCreate(w http.ResponseWriter, r *http.Request) {
	in, ok := decode[prCreateReq](w, r)
	if !ok {
		return
	}
	pr, err := h.svc.CreatePR(in.PurchaseRequest, in.Submit, actor(r))
	if err != nil {
		writeSvcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pr)
}

// prUpdate edits a draft/pending purchase request.
func (h *Handler) prUpdate(w http.ResponseWriter, r *http.Request) {
	in, ok := decode[domain.PurchaseRequest](w, r)
	if !ok {
		return
	}
	pr, err := h.svc.UpdatePR(r.PathValue("id"), in)
	if err != nil {
		writeSvcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pr)
}

// prDelete deletes a purchase request.
func (h *Handler) prDelete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeletePR(r.PathValue("id")); err != nil {
		writeSvcErr(w, err)
		return
	}
	okResp(w)
}

// prSubmit transitions a draft PR to pending (assigns nomor).
func (h *Handler) prSubmit(w http.ResponseWriter, r *http.Request) {
	pr, err := h.svc.SubmitPR(r.PathValue("id"))
	if err != nil {
		writeSvcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pr)
}

// prApprove approves a pending PR (validates approver role).
func (h *Handler) prApprove(w http.ResponseWriter, r *http.Request) {
	req, ok := decode[approveReq](w, r)
	if !ok {
		return
	}
	pr, err := h.svc.ApprovePR(r.PathValue("id"), req.Approver, req.Note)
	if err != nil {
		writeSvcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pr)
}

// prReject rejects a pending PR.
func (h *Handler) prReject(w http.ResponseWriter, r *http.Request) {
	req, ok := decode[approveReq](w, r)
	if !ok {
		return
	}
	pr, err := h.svc.RejectPR(r.PathValue("id"), req.Approver, req.Note)
	if err != nil {
		writeSvcErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pr)
}
