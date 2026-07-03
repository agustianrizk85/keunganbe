package http

import (
	"net/http"
	"regexp"
	"strings"

	"greenpark/finance/internal/domain"
)

// ar returns the current AR/piutang view (authenticated read).
func (h *Handler) ar(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.AR())
}

// effectiveARSources returns the AR input sheets: the UI-configured list from the
// store if any, else the env-seeded list (FINANCE_AR_GSHEETS) as a fallback.
func (h *Handler) effectiveARSources() []domain.ARSource {
	if saved := h.svc.ARSources(); len(saved) > 0 {
		return saved
	}
	out := make([]domain.ARSource, 0, len(h.arSheets))
	for _, s := range h.arSheets {
		out = append(out, domain.ARSource{Code: s.Code, ID: s.ID})
	}
	return out
}

// arSheets returns the configured AR input spreadsheets (project code → id).
func (h *Handler) arSheetsGet(w http.ResponseWriter, _ *http.Request) {
	srcs := h.effectiveARSources()
	if srcs == nil {
		srcs = []domain.ARSource{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"sheets": srcs})
}

type arSheetIn struct {
	Code string `json:"code"`
	URL  string `json:"url"` // full Google Sheets URL or a bare spreadsheet ID
}

type arSheetsReq struct {
	Sheets []arSheetIn `json:"sheets"`
}

// arSheetsSet replaces the AR input spreadsheet list (admin). Each entry accepts
// a full Sheets URL or a bare ID; blank rows are dropped.
func (h *Handler) arSheetsSet(w http.ResponseWriter, r *http.Request) {
	req, ok := decode[arSheetsReq](w, r)
	if !ok {
		return
	}
	out := make([]domain.ARSource, 0, len(req.Sheets))
	for _, s := range req.Sheets {
		code := strings.TrimSpace(s.Code)
		id := sheetIDFromURL(strings.TrimSpace(s.URL))
		if code == "" || id == "" {
			continue
		}
		out = append(out, domain.ARSource{Code: strings.ToUpper(code), ID: id})
	}
	if err := h.svc.SetARSources(out); err != nil {
		writeError(w, http.StatusInternalServerError, "gagal menyimpan daftar sheet AR: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sheets": out})
}

// sheetIDFromURL extracts the spreadsheet ID from a Google Sheets URL, or returns
// the input unchanged if it already looks like a bare ID.
var sheetURLRe = regexp.MustCompile(`/spreadsheets/d/([a-zA-Z0-9-_]+)`)

func sheetIDFromURL(s string) string {
	if m := sheetURLRe.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	// Bare ID (or "key=" style) — strip any surrounding url noise.
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "?"); i >= 0 {
		s = s[:i]
	}
	return strings.Trim(s, "/")
}

// fetchARSheets pulls every tab of each configured per-project AR spreadsheet and
// merges them into one map keyed "CODE::TabTitle" so the ingest can attribute
// rows to a project. Writes its own error response; returns ok=false on failure.
func (h *Handler) fetchARSheets(w http.ResponseWriter, r *http.Request) (map[string][][]string, bool) {
	if h.sync == nil {
		writeError(w, http.StatusServiceUnavailable,
			"Sync Google Sheets belum dikonfigurasi — set FINANCE_GOOGLE_CREDENTIALS.")
		return nil, false
	}
	srcs := h.effectiveARSources()
	if len(srcs) == 0 {
		writeError(w, http.StatusServiceUnavailable,
			"Spreadsheet AR per proyek belum diisi — tambahkan URL spreadsheet AR di tab Sync/Import, lalu share tiap sheet ke email service account.")
		return nil, false
	}
	merged := map[string][][]string{}
	for _, src := range srcs {
		data, err := h.sync.FetchAll(r.Context(), src.ID)
		if err != nil {
			writeError(w, http.StatusBadGateway, "gagal ambil sheet AR "+src.Code+": "+err.Error())
			return nil, false
		}
		for tab, rows := range data {
			merged[src.Code+"::"+tab] = rows
		}
	}
	return merged, true
}

// arSyncPreview fetches the AR sheets and returns the derived view (no persist).
func (h *Handler) arSyncPreview(w http.ResponseWriter, r *http.Request) {
	data, ok := h.fetchARSheets(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, h.svc.PreviewAR(data))
}

// arSyncApprove fetches the AR sheets, derives + stores the view, returns it.
func (h *Handler) arSyncApprove(w http.ResponseWriter, r *http.Request) {
	data, ok := h.fetchARSheets(w, r)
	if !ok {
		return
	}
	ar, err := h.svc.ApproveAR(data)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal menyimpan data AR: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ar)
}
