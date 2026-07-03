package http

import (
	"net/http"
	"strings"
)

// purchasing returns the current procurement (PR) view (authenticated read).
func (h *Handler) purchasing(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.Purchasing())
}

// effectivePRSheet returns the procurement input spreadsheet ID: the
// UI-configured one from the store if set, else the env/default fallback.
func (h *Handler) effectivePRSheet() string {
	if id := h.svc.PRSheet(); id != "" {
		return id
	}
	return h.prSheetID
}

// prSheetGet returns the configured procurement input spreadsheet.
func (h *Handler) prSheetGet(w http.ResponseWriter, _ *http.Request) {
	id := h.effectivePRSheet()
	url := ""
	if id != "" {
		url = "https://docs.google.com/spreadsheets/d/" + id + "/edit"
	}
	writeJSON(w, http.StatusOK, map[string]any{"sheet": id, "url": url})
}

type prSheetReq struct {
	URL string `json:"url"` // full Google Sheets URL or a bare spreadsheet ID
}

// prSheetSet replaces the procurement input spreadsheet (admin). Accepts a full
// Sheets URL or a bare ID; an empty value clears it back to the env default.
func (h *Handler) prSheetSet(w http.ResponseWriter, r *http.Request) {
	req, ok := decode[prSheetReq](w, r)
	if !ok {
		return
	}
	id := sheetIDFromURL(strings.TrimSpace(req.URL))
	if err := h.svc.SetPRSheet(id); err != nil {
		writeError(w, http.StatusInternalServerError, "gagal menyimpan sheet pembelian: "+err.Error())
		return
	}
	h.prSheetGet(w, r)
}

// fetchPRSheet pulls every tab of the configured procurement spreadsheet. It
// writes its own error response and returns ok=false on failure.
func (h *Handler) fetchPRSheet(w http.ResponseWriter, r *http.Request) (map[string][][]string, bool) {
	if h.sync == nil {
		writeError(w, http.StatusServiceUnavailable,
			"Sync Google Sheets belum dikonfigurasi — set FINANCE_GOOGLE_CREDENTIALS (service account JSON) & share spreadsheet ke email service account.")
		return nil, false
	}
	id := h.effectivePRSheet()
	if id == "" {
		writeError(w, http.StatusServiceUnavailable,
			"Spreadsheet Pembelian (PR) belum diisi — tambahkan URL spreadsheet di tab Sync, lalu share ke email service account.")
		return nil, false
	}
	data, err := h.sync.FetchAll(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusBadGateway, "gagal ambil Google Sheets Pembelian: "+err.Error())
		return nil, false
	}
	return data, true
}

// purchasingSyncPreview fetches the PR sheet and returns the derived view (no persist).
func (h *Handler) purchasingSyncPreview(w http.ResponseWriter, r *http.Request) {
	data, ok := h.fetchPRSheet(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, h.svc.PreviewPurchasing(data))
}

// purchasingSyncApprove fetches the PR sheet, derives + stores the view, returns it.
func (h *Handler) purchasingSyncApprove(w http.ResponseWriter, r *http.Request) {
	data, ok := h.fetchPRSheet(w, r)
	if !ok {
		return
	}
	pur, err := h.svc.ApprovePurchasing(data)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal menyimpan data pembelian: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pur)
}
