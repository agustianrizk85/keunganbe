package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"greenpark/finance/internal/auth"
	"greenpark/finance/internal/domain"
	"greenpark/finance/internal/repository"
	"greenpark/finance/internal/service"
)

// Handler holds the dependencies for the HTTP handlers.
type Handler struct {
	svc  service.FinanceService
	auth *auth.Service
}

// NewHandler creates a Handler bound to the service and auth service.
func NewHandler(svc service.FinanceService, authSvc *auth.Service) *Handler {
	return &Handler{svc: svc, auth: authSvc}
}

/* ---------------------------- auth plumbing ---------------------------- */

type ctxKey int

const userCtxKey ctxKey = 0

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[len("Bearer "):])
	}
	return ""
}

// requireAuth wraps a handler, rejecting requests without a valid session.
func (h *Handler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := h.auth.Validate(bearer(r))
		if err != nil {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userCtxKey, u)))
	}
}

// requireAdmin wraps a handler, requiring a valid session with the admin role.
func (h *Handler) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return h.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if u, ok := r.Context().Value(userCtxKey).(domain.User); !ok || u.Role != domain.RoleAdmin {
			writeError(w, http.StatusForbidden, "butuh akses admin")
			return
		}
		next(w, r)
	})
}

// decode reads the JSON request body into a value of type T.
func decode[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		writeError(w, http.StatusBadRequest, "body JSON tidak valid: "+err.Error())
		return v, false
	}
	return v, true
}

/* ---------------------------- auth handlers ---------------------------- */

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	req, ok := decode[loginReq](w, r)
	if !ok {
		return
	}
	token, user, err := h.auth.Login(req.Username, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": user})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	h.auth.Logout(bearer(r))
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	u, _ := r.Context().Value(userCtxKey).(domain.User)
	writeJSON(w, http.StatusOK, u)
}

/* ---------------------------- read handlers ---------------------------- */

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "finance"})
}

func (h *Handler) dashboard(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.Dashboard())
}

func (h *Handler) summary(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.Summary())
}

func (h *Handler) projects(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.Projects())
}

func (h *Handler) projectByID(w http.ResponseWriter, r *http.Request) {
	project, err := h.svc.ProjectByID(r.PathValue("id"))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load project")
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (h *Handler) receivables(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.Receivables())
}

func (h *Handler) payables(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.Payables())
}

func (h *Handler) facilities(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.Facilities())
}

func (h *Handler) costStructure(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.CostStructure())
}

func (h *Handler) treasury(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.Treasury())
}

func (h *Handler) aiInsights(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.AIInsights())
}

func (h *Handler) decisions(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.Decisions())
}

func (h *Handler) kpis(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.KPITable())
}

func (h *Handler) triggers(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.Triggers())
}

/* ---------------------------- singleton / whole-array write handlers ---------------------------- */

func (h *Handler) setTreasury(w http.ResponseWriter, r *http.Request) {
	v, ok := decode[domain.Treasury](w, r)
	if !ok {
		return
	}
	if err := h.svc.SetTreasury(v); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (h *Handler) setCostStructure(w http.ResponseWriter, r *http.Request) {
	v, ok := decode[[]domain.CostCategory](w, r)
	if !ok {
		return
	}
	if err := h.svc.SetCostStructure(v); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (h *Handler) setCashflow(w http.ResponseWriter, r *http.Request) {
	v, ok := decode[[]domain.CashflowPoint](w, r)
	if !ok {
		return
	}
	if err := h.svc.SetCashflowTrend(v); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (h *Handler) setReceivableType(w http.ResponseWriter, r *http.Request) {
	v, ok := decode[[]domain.MetaItem](w, r)
	if !ok {
		return
	}
	if err := h.svc.SetReceivableType(v); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (h *Handler) setAgingMeta(w http.ResponseWriter, r *http.Request) {
	v, ok := decode[[]domain.MetaItem](w, r)
	if !ok {
		return
	}
	if err := h.svc.SetAgingMeta(v); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (h *Handler) setPriorityMeta(w http.ResponseWriter, r *http.Request) {
	v, ok := decode[[]domain.MetaItem](w, r)
	if !ok {
		return
	}
	if err := h.svc.SetPriorityMeta(v); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, v)
}

/* ---------------------------- collection write handlers ---------------------------- */

func (h *Handler) saveProject(w http.ResponseWriter, r *http.Request) {
	v, ok := decode[domain.Project](w, r)
	if !ok {
		return
	}
	saved, err := h.svc.SaveProject(v)
	respondSave(w, saved, err)
}

func (h *Handler) saveReceivable(w http.ResponseWriter, r *http.Request) {
	v, ok := decode[domain.Receivable](w, r)
	if !ok {
		return
	}
	saved, err := h.svc.SaveReceivable(v)
	respondSave(w, saved, err)
}

func (h *Handler) savePayable(w http.ResponseWriter, r *http.Request) {
	v, ok := decode[domain.Payable](w, r)
	if !ok {
		return
	}
	saved, err := h.svc.SavePayable(v)
	respondSave(w, saved, err)
}

func (h *Handler) saveFacility(w http.ResponseWriter, r *http.Request) {
	v, ok := decode[domain.Facility](w, r)
	if !ok {
		return
	}
	saved, err := h.svc.SaveFacility(v)
	respondSave(w, saved, err)
}

func (h *Handler) saveAIInsight(w http.ResponseWriter, r *http.Request) {
	v, ok := decode[domain.AIInsight](w, r)
	if !ok {
		return
	}
	saved, err := h.svc.SaveAIInsight(v)
	respondSave(w, saved, err)
}

func (h *Handler) saveDecision(w http.ResponseWriter, r *http.Request) {
	v, ok := decode[domain.Decision](w, r)
	if !ok {
		return
	}
	saved, err := h.svc.SaveDecision(v)
	respondSave(w, saved, err)
}

func (h *Handler) saveKPI(w http.ResponseWriter, r *http.Request) {
	v, ok := decode[domain.KPI](w, r)
	if !ok {
		return
	}
	saved, err := h.svc.SaveKPI(v)
	respondSave(w, saved, err)
}

func (h *Handler) saveTrigger(w http.ResponseWriter, r *http.Request) {
	v, ok := decode[domain.Trigger](w, r)
	if !ok {
		return
	}
	saved, err := h.svc.SaveTrigger(v)
	respondSave(w, saved, err)
}

func respondSave[T any](w http.ResponseWriter, saved T, err error) {
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

// deleteHandler adapts a repository delete to an HTTP handler keyed on {id}.
func (h *Handler) deleteHandler(del func(id string) (bool, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok, err := del(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeError(w, http.StatusNotFound, "data tidak ditemukan")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}
