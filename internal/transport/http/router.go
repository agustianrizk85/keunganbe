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

	// ---- transactional purchasing: master data (authenticated) ----
	mux.HandleFunc("GET /api/vendors", h.requireAuth(h.vendorsList))
	mux.HandleFunc("POST /api/vendors", h.requireAuth(h.vendorCreate))
	mux.HandleFunc("PUT /api/vendors/{id}", h.requireAuth(h.vendorUpdate))
	mux.HandleFunc("DELETE /api/vendors/{id}", h.requireAuth(h.vendorDelete))

	mux.HandleFunc("GET /api/produk", h.requireAuth(h.produkList))
	mux.HandleFunc("POST /api/produk", h.requireAuth(h.produkCreate))
	mux.HandleFunc("PUT /api/produk/{id}", h.requireAuth(h.produkUpdate))
	mux.HandleFunc("DELETE /api/produk/{id}", h.requireAuth(h.produkDelete))

	mux.HandleFunc("GET /api/karyawan", h.requireAuth(h.karyawanList))
	mux.HandleFunc("POST /api/karyawan", h.requireAuth(h.karyawanCreate))
	mux.HandleFunc("PUT /api/karyawan/{id}", h.requireAuth(h.karyawanUpdate))
	mux.HandleFunc("DELETE /api/karyawan/{id}", h.requireAuth(h.karyawanDelete))

	mux.HandleFunc("GET /api/sla", h.requireAuth(h.slaList))
	mux.HandleFunc("POST /api/sla", h.requireAuth(h.slaCreate))
	mux.HandleFunc("PUT /api/sla/{id}", h.requireAuth(h.slaUpdate))
	mux.HandleFunc("DELETE /api/sla/{id}", h.requireAuth(h.slaDelete))

	// ---- transactional purchasing: Purchase Request (any division may submit/
	// approve; department-scoped authorization enforced in the service layer) ----
	mux.HandleFunc("GET /api/pr", h.requireAnyDivisionAuth(h.prList))
	mux.HandleFunc("GET /api/pr/{id}", h.requireAnyDivisionAuth(h.prGet))
	mux.HandleFunc("POST /api/pr", h.requireAnyDivisionAuth(h.prCreate))
	mux.HandleFunc("PUT /api/pr/{id}", h.requireAnyDivisionAuth(h.prUpdate))
	mux.HandleFunc("DELETE /api/pr/{id}", h.requireAuth(h.prDelete)) // finance-only, conservative
	mux.HandleFunc("POST /api/pr/{id}/submit", h.requireAnyDivisionAuth(h.prSubmit))
	mux.HandleFunc("POST /api/pr/{id}/approve", h.requireAnyDivisionAuth(h.prApprove))
	mux.HandleFunc("POST /api/pr/{id}/reject", h.requireAnyDivisionAuth(h.prReject))

	// ---- transactional purchasing: Purchase Order (authenticated) ----
	mux.HandleFunc("GET /api/po", h.requireAuth(h.poList))
	mux.HandleFunc("GET /api/po/{id}", h.requireAuth(h.poGet))
	mux.HandleFunc("POST /api/po", h.requireAuth(h.poCreate))
	mux.HandleFunc("PUT /api/po/{id}", h.requireAuth(h.poUpdate))
	mux.HandleFunc("DELETE /api/po/{id}", h.requireAuth(h.poDelete))
	mux.HandleFunc("POST /api/po/{id}/submit", h.requireAuth(h.poSubmit))
	mux.HandleFunc("POST /api/po/{id}/approve", h.requireAuth(h.poApprove))
	mux.HandleFunc("POST /api/po/{id}/reject", h.requireAuth(h.poReject))
	mux.HandleFunc("POST /api/po/{id}/receive", h.requireAuth(h.poReceive))

	// ---- laporan register (authenticated) ----
	mux.HandleFunc("GET /api/purchasing/register", h.requireAuth(h.purchasingRegister))

	return chain(mux, logger, cors(allowOrigin))
}
