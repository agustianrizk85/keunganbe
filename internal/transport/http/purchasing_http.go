package http

import (
	"errors"
	"net/http"

	"greenpark/finance/internal/domain"
	"greenpark/finance/internal/service"
)

// This file holds shared plumbing for the transactional purchasing handlers:
// service-error → HTTP-status mapping and the approve/reject/receive request
// bodies used by both PR and PO endpoints.

// writeSvcErr maps a service/repository error to the right HTTP status.
func writeSvcErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// approveReq is the body for approve/reject endpoints: the acting approver
// (name+role captured from the FE SSO user) plus an optional note.
type approveReq struct {
	Approver domain.Approver `json:"approver"`
	Note     string          `json:"note"`
}

// receiveReq is the body for the PO receive/BAST endpoint.
type receiveReq struct {
	TanggalDiterima string `json:"tanggalDiterima"`
	BastSigned      bool   `json:"bastSigned"`
	Keterangan      string `json:"keterangan"`
}

// okResp is the {"ok":true} envelope returned by DELETE endpoints.
func okResp(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
