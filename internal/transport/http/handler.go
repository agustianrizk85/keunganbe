package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"greenpark/finance/internal/auth"
	"greenpark/finance/internal/domain"
	"greenpark/finance/internal/gsheets"
	"greenpark/finance/internal/service"
)

// Handler holds the dependencies for the HTTP handlers.
type Handler struct {
	svc     service.FinanceService
	auth    *auth.Service
	sync    *gsheets.Client // nil when Google Sheets sync is not configured
	sheetID string
	auto    *autoSync
	hub     *wsHub
}

// NewHandler creates a Handler bound to the service, auth, and (optional) Google
// Sheets sync client.
func NewHandler(svc service.FinanceService, authSvc *auth.Service, sync *gsheets.Client, sheetID string, intervalSec int) *Handler {
	return &Handler{
		svc:     svc,
		auth:    authSvc,
		sync:    sync,
		sheetID: sheetID,
		auto:    newAutoSync(intervalSec),
		hub:     newWSHub(),
	}
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

func actor(r *http.Request) string {
	if u, ok := r.Context().Value(userCtxKey).(domain.User); ok {
		return u.Username
	}
	return "-"
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

func (h *Handler) version(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]int64{"rev": h.svc.Revision()})
}
