package http

import "net/http"

// NewRouter wires all routes to the handler and applies global middleware.
//
// Access tiers:
//   - public: health check + login + ws (token in query)
//   - authenticated (any logged-in user): dashboard reads + me/logout + version
//   - admin only: every ingest / sync / lifecycle write
func NewRouter(h *Handler, allowOrigin string) http.Handler {
	mux := http.NewServeMux()

	// ---- public ----
	mux.HandleFunc("GET /api/health", h.health)
	mux.HandleFunc("POST /api/auth/login", h.login)
	mux.HandleFunc("GET /api/ws", h.ws) // token validated inside (query param)

	// ---- authenticated session ----
	mux.HandleFunc("GET /api/auth/me", h.requireAuth(h.me))
	mux.HandleFunc("POST /api/auth/logout", h.requireAuth(h.logout))

	// ---- reads (authenticated) ----
	mux.HandleFunc("GET /api/dashboard", h.requireAuth(h.dashboard))
	mux.HandleFunc("GET /api/summary", h.requireAuth(h.summary))
	mux.HandleFunc("GET /api/version", h.requireAuth(h.version))
	mux.HandleFunc("GET /api/ar", h.requireAuth(h.ar))
	mux.HandleFunc("GET /api/ar/sheets", h.requireAuth(h.arSheetsGet))
	mux.HandleFunc("GET /api/purchasing", h.requireAuth(h.purchasing))
	mux.HandleFunc("GET /api/purchasing/sheet", h.requireAuth(h.prSheetGet))

	// ---- AR (piutang) sync from per-project sheets (admin) ----
	mux.HandleFunc("PUT /api/ar/sheets", h.requireAdmin(h.arSheetsSet))
	mux.HandleFunc("POST /api/ar/sync-preview", h.requireAdmin(h.arSyncPreview))
	mux.HandleFunc("POST /api/ar/sync-approve", h.requireAdmin(h.arSyncApprove))

	// ---- procurement (PR/pembelian) sync from the "Pembelian (PR)" sheet (admin) ----
	mux.HandleFunc("PUT /api/purchasing/sheet", h.requireAdmin(h.prSheetSet))
	mux.HandleFunc("POST /api/purchasing/sync-preview", h.requireAdmin(h.purchasingSyncPreview))
	mux.HandleFunc("POST /api/purchasing/sync-approve", h.requireAdmin(h.purchasingSyncApprove))

	// ---- ingest: upload XLSX (admin) ----
	mux.HandleFunc("POST /api/import/preview", h.requireAdmin(h.importPreview))
	mux.HandleFunc("POST /api/import/approve", h.requireAdmin(h.importApprove))

	// ---- ingest: Google Sheets sync (admin) ----
	mux.HandleFunc("POST /api/import/sync-preview", h.requireAdmin(h.importSyncPreview))
	mux.HandleFunc("POST /api/import/sync-approve", h.requireAdmin(h.importSyncApprove))

	// ---- auto-sync control (admin) ----
	mux.HandleFunc("GET /api/import/auto", h.requireAdmin(h.autoStatus))
	mux.HandleFunc("PUT /api/import/auto", h.requireAdmin(h.autoSet))

	// ---- history / lifecycle (admin) ----
	mux.HandleFunc("GET /api/import/history", h.requireAdmin(h.importHistory))
	mux.HandleFunc("POST /api/import/reset", h.requireAdmin(h.importReset))
	mux.HandleFunc("POST /api/import/rollback/{id}", h.requireAdmin(h.importRollback))

	return chain(mux, logger, cors(allowOrigin))
}
