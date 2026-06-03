package http

import "net/http"

// NewRouter wires all routes to the handler and applies global middleware.
//
// Access tiers:
//   - public: health check + login
//   - authenticated (any logged-in user): all dashboard reads + me/logout
//   - admin only: every master-data write
func NewRouter(h *Handler, allowOrigin string) http.Handler {
	mux := http.NewServeMux()

	// ---- public ----
	mux.HandleFunc("GET /api/health", h.health)
	mux.HandleFunc("POST /api/auth/login", h.login)

	// ---- authenticated session ----
	mux.HandleFunc("GET /api/auth/me", h.requireAuth(h.me))
	mux.HandleFunc("POST /api/auth/logout", h.requireAuth(h.logout))

	// ---- reads (authenticated) ----
	mux.HandleFunc("GET /api/dashboard", h.requireAuth(h.dashboard))
	mux.HandleFunc("GET /api/summary", h.requireAuth(h.summary))
	mux.HandleFunc("GET /api/projects", h.requireAuth(h.projects))
	mux.HandleFunc("GET /api/projects/{id}", h.requireAuth(h.projectByID))
	mux.HandleFunc("GET /api/receivables", h.requireAuth(h.receivables))
	mux.HandleFunc("GET /api/payables", h.requireAuth(h.payables))
	mux.HandleFunc("GET /api/facilities", h.requireAuth(h.facilities))
	mux.HandleFunc("GET /api/cost-structure", h.requireAuth(h.costStructure))
	mux.HandleFunc("GET /api/treasury", h.requireAuth(h.treasury))
	mux.HandleFunc("GET /api/ai-insights", h.requireAuth(h.aiInsights))
	mux.HandleFunc("GET /api/decisions", h.requireAuth(h.decisions))
	mux.HandleFunc("GET /api/kpis", h.requireAuth(h.kpis))
	mux.HandleFunc("GET /api/triggers", h.requireAuth(h.triggers))

	// ---- singleton / whole-array writes (admin) ----
	mux.HandleFunc("PUT /api/treasury", h.requireAdmin(h.setTreasury))
	mux.HandleFunc("PUT /api/cost-structure", h.requireAdmin(h.setCostStructure))
	mux.HandleFunc("PUT /api/cashflow", h.requireAdmin(h.setCashflow))
	mux.HandleFunc("PUT /api/receivable-type", h.requireAdmin(h.setReceivableType))
	mux.HandleFunc("PUT /api/aging-meta", h.requireAdmin(h.setAgingMeta))
	mux.HandleFunc("PUT /api/priority-meta", h.requireAdmin(h.setPriorityMeta))

	// ---- collection writes (admin) ----
	mux.HandleFunc("POST /api/projects", h.requireAdmin(h.saveProject))
	mux.HandleFunc("DELETE /api/projects/{id}", h.requireAdmin(h.deleteHandler(h.svc.DeleteProject)))
	mux.HandleFunc("POST /api/receivables", h.requireAdmin(h.saveReceivable))
	mux.HandleFunc("DELETE /api/receivables/{id}", h.requireAdmin(h.deleteHandler(h.svc.DeleteReceivable)))
	mux.HandleFunc("POST /api/payables", h.requireAdmin(h.savePayable))
	mux.HandleFunc("DELETE /api/payables/{id}", h.requireAdmin(h.deleteHandler(h.svc.DeletePayable)))
	mux.HandleFunc("POST /api/facilities", h.requireAdmin(h.saveFacility))
	mux.HandleFunc("DELETE /api/facilities/{id}", h.requireAdmin(h.deleteHandler(h.svc.DeleteFacility)))
	mux.HandleFunc("POST /api/ai-insights", h.requireAdmin(h.saveAIInsight))
	mux.HandleFunc("DELETE /api/ai-insights/{id}", h.requireAdmin(h.deleteHandler(h.svc.DeleteAIInsight)))
	mux.HandleFunc("POST /api/decisions", h.requireAdmin(h.saveDecision))
	mux.HandleFunc("DELETE /api/decisions/{id}", h.requireAdmin(h.deleteHandler(h.svc.DeleteDecision)))
	mux.HandleFunc("POST /api/kpis", h.requireAdmin(h.saveKPI))
	mux.HandleFunc("DELETE /api/kpis/{id}", h.requireAdmin(h.deleteHandler(h.svc.DeleteKPI)))
	mux.HandleFunc("POST /api/triggers", h.requireAdmin(h.saveTrigger))
	mux.HandleFunc("DELETE /api/triggers/{id}", h.requireAdmin(h.deleteHandler(h.svc.DeleteTrigger)))

	return chain(mux, logger, cors(allowOrigin))
}
