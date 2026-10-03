package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/wastingnotime/wishlist/apps/api/internal/application"
	"github.com/wastingnotime/wishlist/apps/api/internal/domain"
)

const visitorCookie = "wishlist_visitor"

type Server struct {
	board         *application.Board
	visitor       *application.Visitor
	secureCookies bool
	adminToken    string
	adminOIDC     *AdminOIDC
	casdoorAdmin  bool
	admin         AdminStore
}

type AdminStore interface {
	ListAllApps(context.Context) ([]domain.App, error)
	CreateApp(context.Context, domain.App) error
	UpdateApp(context.Context, string, string, string, string, bool) error
	CreateFeature(context.Context, domain.Feature) error
	UpdateFeature(context.Context, string, string, string, domain.Status, string, time.Time) error
	PendingSuggestions(context.Context) ([]map[string]any, error)
	ReviewSuggestion(context.Context, string, string, string, string, string, string, time.Time) error
	PendingPrivacyReviews(context.Context) ([]application.PublicPrivacyReview, error)
	ResolvePrivacyReview(context.Context, string, time.Time) error
	OperatorPrivacyData(context.Context, string, time.Time) (application.PrivacyData, error)
	OperatorErasePrivacyData(context.Context, string, time.Time) error
	OperatorCorrectEmail(context.Context, string, string) error
}

func New(board *application.Board, visitor *application.Visitor, secureCookies bool, admin ...AdminStore) *Server {
	casdoorAdmin := getEnv("APP_ENV") == "production" || getEnv("WISHLIST_ADMIN_AUTH_MODE") == "casdoor"
	server := &Server{board: board, visitor: visitor, secureCookies: secureCookies, casdoorAdmin: casdoorAdmin}
	if !casdoorAdmin {
		server.adminToken = strings.TrimSpace(getEnv("WISHLIST_ADMIN_TOKEN"))
	}
	if len(admin) > 0 {
		server.admin = admin[0]
	}
	return server
}

func (server *Server) UseAdminOIDC(auth *AdminOIDC) { server.adminOIDC = auth }

func getEnv(key string) string { return os.Getenv(key) }

func (server *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.health)
	mux.HandleFunc("GET /readyz", server.ready)
	mux.HandleFunc("GET /v1/apps", server.apps)
	mux.HandleFunc("GET /v1/features", server.features)
	mux.HandleFunc("POST /v1/otp", server.requestOTP)
	mux.HandleFunc("POST /v1/otp/verify", server.verifyOTP)
	mux.HandleFunc("GET /v1/session", server.session)
	mux.HandleFunc("DELETE /v1/session", server.logout)
	mux.HandleFunc("POST /v1/features/{id}/vote", server.toggleVote)
	mux.HandleFunc("POST /v1/suggestions", server.submitSuggestion)
	mux.HandleFunc("GET /v1/privacy/data", server.privacyData)
	mux.HandleFunc("DELETE /v1/privacy/data", server.erasePrivacyData)
	mux.HandleFunc("POST /v1/privacy/email/request", server.requestEmailCorrection)
	mux.HandleFunc("POST /v1/privacy/email/verify", server.completeEmailCorrection)
	mux.HandleFunc("POST /v1/admin/apps", server.adminCreateApp)
	mux.HandleFunc("GET /v1/admin/session", server.adminSession)
	mux.HandleFunc("GET /v1/admin/apps", server.adminListApps)
	mux.HandleFunc("PATCH /v1/admin/apps/{id}", server.adminUpdateApp)
	mux.HandleFunc("POST /v1/admin/features", server.adminCreateFeature)
	mux.HandleFunc("PATCH /v1/admin/features/{id}", server.adminUpdateFeature)
	mux.HandleFunc("GET /v1/admin/suggestions", server.adminSuggestions)
	mux.HandleFunc("POST /v1/admin/suggestions/{id}/accept", server.adminAcceptSuggestion)
	mux.HandleFunc("POST /v1/admin/suggestions/{id}/reject", server.adminRejectSuggestion)
	mux.HandleFunc("POST /v1/admin/suggestions/{id}/merge", server.adminMergeSuggestion)
	mux.HandleFunc("GET /v1/admin/privacy/reviews", server.adminPrivacyReviews)
	mux.HandleFunc("POST /v1/admin/privacy/reviews/{id}/resolve", server.adminResolvePrivacyReview)
	mux.HandleFunc("POST /v1/admin/privacy/data", server.adminPrivacyData)
	mux.HandleFunc("POST /v1/admin/privacy/erase", server.adminErasePrivacyData)
	mux.HandleFunc("POST /v1/admin/privacy/correct-email", server.adminCorrectEmail)
	return mux
}

func (server *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (server *Server) ready(w http.ResponseWriter, r *http.Request) {
	if err := server.board.Ready(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "not_ready")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (server *Server) apps(w http.ResponseWriter, r *http.Request) {
	apps, err := server.board.Apps(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"apps": apps})
}

func (server *Server) features(w http.ResponseWriter, r *http.Request) {
	view := r.URL.Query().Get("view")
	if view == "" {
		view = string(domain.Voting)
	}
	rows, err := server.board.Features(r.Context(), domain.Status(view), r.URL.Query().Get("app"))
	if err != nil {
		if errors.Is(err, domain.ErrInvalidRequest) {
			writeError(w, http.StatusBadRequest, "invalid_request")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"features": rows})
}

func (server *Server) requestOTP(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	var command struct {
		Email string `json:"email"`
	}
	if !decode(w, r, &command) {
		return
	}
	if err := server.visitor.RequestOTP(r.Context(), command.Email); err != nil {
		if errors.Is(err, domain.ErrInvalidRequest) {
			writeError(w, http.StatusBadRequest, "invalid_request")
			return
		}
		writeError(w, http.StatusServiceUnavailable, "otp_unavailable")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"requested": true})
}

func (server *Server) verifyOTP(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	var command struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if !decode(w, r, &command) {
		return
	}
	session, err := server.visitor.VerifyOTP(r.Context(), command.Email, command.Code)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCode) {
			writeError(w, http.StatusBadRequest, "invalid_or_expired_code")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: visitorCookie, Value: session, Path: "/", HttpOnly: true, Secure: server.secureCookies, SameSite: http.SameSiteLaxMode, MaxAge: int(application.SessionValidity.Seconds())})
	writeJSON(w, http.StatusOK, map[string]bool{"verified": true})
}

func (server *Server) session(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(visitorCookie)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"verified": false, "vote_feature_ids": []string{}})
		return
	}
	votes, err := server.visitor.Session(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthenticated) {
			writeJSON(w, http.StatusOK, map[string]any{"verified": false, "vote_feature_ids": []string{}})
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"verified": true, "vote_feature_ids": votes})
}

func (server *Server) logout(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if cookie, err := r.Cookie(visitorCookie); err == nil {
		if err := server.visitor.Logout(r.Context(), cookie.Value); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
	}
	http.SetCookie(w, &http.Cookie{Name: visitorCookie, Value: "", Path: "/", HttpOnly: true, Secure: server.secureCookies, SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0)})
	w.WriteHeader(http.StatusNoContent)
}

func (server *Server) toggleVote(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	cookie, err := r.Cookie(visitorCookie)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	voted, count, err := server.visitor.ToggleVote(r.Context(), cookie.Value, r.PathValue("id"))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUnauthenticated):
			writeError(w, http.StatusUnauthorized, "unauthenticated")
		case errors.Is(err, domain.ErrFeatureNotFound):
			writeError(w, http.StatusNotFound, "not_found")
		case errors.Is(err, domain.ErrFeatureNotVoting), errors.Is(err, domain.ErrVoteConflict):
			writeError(w, http.StatusConflict, "vote_unavailable")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"voted": voted, "vote_count": count})
}

func (server *Server) submitSuggestion(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	cookie, err := r.Cookie(visitorCookie)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var command struct {
		AppID       string `json:"app_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if !decode(w, r, &command) {
		return
	}
	id, err := server.visitor.SubmitSuggestion(r.Context(), cookie.Value, command.AppID, command.Title, command.Description)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUnauthenticated):
			writeError(w, http.StatusUnauthorized, "unauthenticated")
		case errors.Is(err, domain.ErrRateLimited):
			writeError(w, http.StatusTooManyRequests, "rate_limited")
		case errors.Is(err, domain.ErrInvalidRequest), errors.Is(err, domain.ErrAppNotFound):
			writeError(w, http.StatusBadRequest, "invalid_request")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"submitted": true, "suggestion_id": id})
}

func visitorSessionCookie(r *http.Request) string {
	cookie, err := r.Cookie(visitorCookie)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func writePrivacyError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrUnauthenticated):
		writeError(w, http.StatusUnauthorized, "unauthenticated")
	case errors.Is(err, domain.ErrFreshVerification):
		writeError(w, http.StatusForbidden, "fresh_verification_required")
	case errors.Is(err, domain.ErrEmailInUse):
		writeError(w, http.StatusConflict, "email_in_use")
	case errors.Is(err, domain.ErrInvalidCode):
		writeError(w, http.StatusBadRequest, "invalid_or_expired_code")
	case errors.Is(err, domain.ErrInvalidRequest):
		writeError(w, http.StatusBadRequest, "invalid_request")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error")
	}
}

func (server *Server) privacyData(w http.ResponseWriter, r *http.Request) {
	data, err := server.visitor.PrivacyData(r.Context(), visitorSessionCookie(r))
	if err != nil {
		writePrivacyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (server *Server) requestEmailCorrection(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	var command struct {
		NewEmail string `json:"new_email"`
	}
	if !decode(w, r, &command) {
		return
	}
	err := server.visitor.RequestEmailCorrection(r.Context(), visitorSessionCookie(r), command.NewEmail)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidRequest) || errors.Is(err, domain.ErrUnauthenticated) || errors.Is(err, domain.ErrFreshVerification) {
			writePrivacyError(w, err)
		} else {
			writeError(w, http.StatusServiceUnavailable, "otp_unavailable")
		}
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"requested": true})
}

func (server *Server) completeEmailCorrection(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	var command struct {
		NewEmail string `json:"new_email"`
		Code     string `json:"code"`
	}
	if !decode(w, r, &command) {
		return
	}
	if err := server.visitor.CompleteEmailCorrection(r.Context(), visitorSessionCookie(r), command.NewEmail, command.Code); err != nil {
		writePrivacyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"corrected": true})
}

func (server *Server) erasePrivacyData(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err := server.visitor.ErasePrivacyData(r.Context(), visitorSessionCookie(r)); err != nil {
		writePrivacyError(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: visitorCookie, Value: "", Path: "/", HttpOnly: true, Secure: server.secureCookies, SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0)})
	w.WriteHeader(http.StatusNoContent)
}

func sameOrigin(r *http.Request) bool { return r.Header.Get("X-Wishlist-Same-Origin") == "1" }

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code}})
}

func (server *Server) authorizeAdmin(w http.ResponseWriter, r *http.Request) bool {
	if server.casdoorAdmin {
		if server.adminOIDC == nil {
			writeError(w, http.StatusServiceUnavailable, "admin_unavailable")
			return false
		}
		if _, err := server.adminOIDC.Authorize(r); err != nil {
			if errors.Is(err, errAdminForbidden) {
				writeError(w, http.StatusForbidden, "forbidden")
			} else {
				writeError(w, http.StatusUnauthorized, "unauthorized")
			}
			return false
		}
	} else {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if server.adminToken == "" || len(got) != len(server.adminToken) || subtle.ConstantTimeCompare([]byte(got), []byte(server.adminToken)) != 1 {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return false
		}
	}
	if server.admin == nil {
		writeError(w, http.StatusServiceUnavailable, "admin_unavailable")
		return false
	}
	return true
}
func (server *Server) adminSession(w http.ResponseWriter, r *http.Request) {
	if !server.authorizeAdmin(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"authorized": true})
}
func (server *Server) adminCreateApp(w http.ResponseWriter, r *http.Request) {
	if !server.authorizeAdmin(w, r) {
		return
	}
	var x struct {
		Slug        string `json:"slug"`
		Name        string `json:"name"`
		Description string `json:"description"`
		URL         string `json:"url"`
	}
	if !decode(w, r, &x) {
		return
	}
	if !domain.ValidSlug(x.Slug) || strings.TrimSpace(x.Name) == "" || len(x.Name) > 100 || len(x.Description) > 500 || len(x.URL) > 500 || (x.URL != "" && !strings.HasPrefix(x.URL, "https://")) {
		writeError(w, 400, "invalid_request")
		return
	}
	id := newID()
	err := server.admin.CreateApp(r.Context(), domain.App{ID: id, Slug: x.Slug, Name: strings.TrimSpace(x.Name), Description: strings.TrimSpace(x.Description), URL: strings.TrimSpace(x.URL)})
	if err != nil {
		writeError(w, 409, "conflict")
		return
	}
	writeJSON(w, 201, map[string]string{"id": id})
}
func (server *Server) adminListApps(w http.ResponseWriter, r *http.Request) {
	if !server.authorizeAdmin(w, r) {
		return
	}
	apps, e := server.admin.ListAllApps(r.Context())
	if e != nil {
		writeError(w, 500, "internal_error")
		return
	}
	type row struct {
		ID          string `json:"id"`
		Slug        string `json:"slug"`
		Name        string `json:"name"`
		Description string `json:"description"`
		URL         string `json:"url"`
		Active      bool   `json:"active"`
	}
	out := make([]row, 0, len(apps))
	for _, a := range apps {
		out = append(out, row{a.ID, a.Slug, a.Name, a.Description, a.URL, a.Active})
	}
	writeJSON(w, 200, map[string]any{"apps": out})
}
func (server *Server) adminUpdateApp(w http.ResponseWriter, r *http.Request) {
	if !server.authorizeAdmin(w, r) {
		return
	}
	var x struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		URL         string `json:"url"`
		Active      bool   `json:"active"`
	}
	if !decode(w, r, &x) {
		return
	}
	if strings.TrimSpace(x.Name) == "" || len(x.Name) > 100 || len(x.Description) > 500 || len(x.URL) > 500 || (x.URL != "" && !strings.HasPrefix(x.URL, "https://")) {
		writeError(w, 400, "invalid_request")
		return
	}
	if e := server.admin.UpdateApp(r.Context(), r.PathValue("id"), strings.TrimSpace(x.Name), strings.TrimSpace(x.Description), strings.TrimSpace(x.URL), x.Active); e != nil {
		writeError(w, 404, "not_found")
		return
	}
	w.WriteHeader(204)
}
func (server *Server) adminCreateFeature(w http.ResponseWriter, r *http.Request) {
	if !server.authorizeAdmin(w, r) {
		return
	}
	var x struct {
		AppID       string `json:"app_id"`
		Slug        string `json:"slug"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if !decode(w, r, &x) {
		return
	}
	if !domain.ValidSlug(x.Slug) || x.AppID == "" || strings.TrimSpace(x.Title) == "" || len(x.Title) > 160 || len(x.Description) > 2000 {
		writeError(w, 400, "invalid_request")
		return
	}
	id := newID()
	e := server.admin.CreateFeature(r.Context(), domain.Feature{ID: id, AppID: x.AppID, Slug: x.Slug, Title: strings.TrimSpace(x.Title), Description: strings.TrimSpace(x.Description), Status: domain.Voting, PublishedAt: time.Now().UTC()})
	if e != nil {
		writeError(w, 400, "invalid_request")
		return
	}
	writeJSON(w, 201, map[string]string{"id": id})
}
func (server *Server) adminUpdateFeature(w http.ResponseWriter, r *http.Request) {
	if !server.authorizeAdmin(w, r) {
		return
	}
	var x struct {
		Title       string        `json:"title"`
		Description string        `json:"description"`
		Status      domain.Status `json:"status"`
		DeliveryURL string        `json:"delivery_url"`
	}
	if !decode(w, r, &x) {
		return
	}
	if !x.Status.Valid() || strings.TrimSpace(x.Title) == "" || len(x.Title) > 160 || len(x.Description) > 2000 || len(x.DeliveryURL) > 1000 || (x.Status == domain.Delivered && (!strings.HasPrefix(x.DeliveryURL, "https://") || len(x.DeliveryURL) > 1000)) || (x.DeliveryURL != "" && !strings.HasPrefix(x.DeliveryURL, "https://")) {
		writeError(w, 400, "invalid_request")
		return
	}
	if e := server.admin.UpdateFeature(r.Context(), r.PathValue("id"), strings.TrimSpace(x.Title), strings.TrimSpace(x.Description), x.Status, strings.TrimSpace(x.DeliveryURL), time.Now().UTC()); e != nil {
		if errors.Is(e, domain.ErrInvalidTransition) {
			writeError(w, 409, "invalid_transition")
		} else if errors.Is(e, domain.ErrFeatureNotFound) {
			writeError(w, 404, "not_found")
		} else {
			writeError(w, 400, "invalid_request")
		}
		return
	}
	w.WriteHeader(204)
}
func (server *Server) adminSuggestions(w http.ResponseWriter, r *http.Request) {
	if !server.authorizeAdmin(w, r) {
		return
	}
	rows, e := server.admin.PendingSuggestions(r.Context())
	if e != nil {
		writeError(w, 500, "internal_error")
		return
	}
	writeJSON(w, 200, map[string]any{"suggestions": rows})
}
func (server *Server) adminPrivacyReviews(w http.ResponseWriter, r *http.Request) {
	if !server.authorizeAdmin(w, r) {
		return
	}
	rows, err := server.admin.PendingPrivacyReviews(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reviews": rows})
}
func (server *Server) adminPrivacyData(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if !server.authorizeAdmin(w, r) {
		return
	}
	var command struct {
		Email         string `json:"email"`
		CaseReference string `json:"case_reference"`
	}
	if !decode(w, r, &command) {
		return
	}
	email, ok := privacyOperatorInput(command.Email, command.CaseReference)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	data, err := server.admin.OperatorPrivacyData(r.Context(), email, time.Now().UTC())
	if err != nil {
		writeOperatorPrivacyError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}
func (server *Server) adminErasePrivacyData(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if !server.authorizeAdmin(w, r) {
		return
	}
	var command struct {
		Email         string `json:"email"`
		CaseReference string `json:"case_reference"`
	}
	if !decode(w, r, &command) {
		return
	}
	email, ok := privacyOperatorInput(command.Email, command.CaseReference)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if err := server.admin.OperatorErasePrivacyData(r.Context(), email, time.Now().UTC()); err != nil {
		writeOperatorPrivacyError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (server *Server) adminCorrectEmail(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if !server.authorizeAdmin(w, r) {
		return
	}
	var command struct {
		Email         string `json:"email"`
		NewEmail      string `json:"new_email"`
		CaseReference string `json:"case_reference"`
	}
	if !decode(w, r, &command) {
		return
	}
	email, ok := privacyOperatorInput(command.Email, command.CaseReference)
	newEmail := strings.ToLower(strings.TrimSpace(command.NewEmail))
	if !ok || len(newEmail) > 254 || !emailPatternForOperator(newEmail) || newEmail == email {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if err := server.admin.OperatorCorrectEmail(r.Context(), email, newEmail); err != nil {
		if errors.Is(err, domain.ErrEmailInUse) {
			writeError(w, http.StatusConflict, "email_in_use")
		} else {
			writeOperatorPrivacyError(w, err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func privacyOperatorInput(email, caseReference string) (string, bool) {
	email = strings.ToLower(strings.TrimSpace(email))
	caseReference = strings.TrimSpace(caseReference)
	return email, len(email) <= 254 && emailPatternForOperator(email) && len(caseReference) >= 3 && len(caseReference) <= 120
}
func emailPatternForOperator(email string) bool {
	parts := strings.Split(email, "@")
	return len(parts) == 2 && parts[0] != "" && strings.Contains(parts[1], ".") && !strings.ContainsAny(email, " \t\r\n")
}
func writeOperatorPrivacyError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrInvalidRequest) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	writeError(w, http.StatusInternalServerError, "internal_error")
}
func (server *Server) adminResolvePrivacyReview(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if !server.authorizeAdmin(w, r) {
		return
	}
	err := server.admin.ResolvePrivacyReview(r.Context(), r.PathValue("id"), time.Now().UTC())
	if err != nil {
		if errors.Is(err, domain.ErrInvalidRequest) {
			writeError(w, http.StatusNotFound, "not_found")
		} else {
			writeError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (server *Server) adminAcceptSuggestion(w http.ResponseWriter, r *http.Request) {
	server.reviewSuggestion(w, r, "accept")
}
func (server *Server) adminRejectSuggestion(w http.ResponseWriter, r *http.Request) {
	server.reviewSuggestion(w, r, "reject")
}
func (server *Server) adminMergeSuggestion(w http.ResponseWriter, r *http.Request) {
	server.reviewSuggestion(w, r, "merge")
}
func (server *Server) reviewSuggestion(w http.ResponseWriter, r *http.Request, action string) {
	if !server.authorizeAdmin(w, r) {
		return
	}
	var x struct {
		Slug        string `json:"slug"`
		Title       string `json:"title"`
		Description string `json:"description"`
		FeatureID   string `json:"feature_id"`
	}
	if !decode(w, r, &x) {
		return
	}
	if (action == "accept" && (!domain.ValidSlug(x.Slug) || strings.TrimSpace(x.Title) == "" || len(x.Title) > 160 || len(x.Description) > 2000)) || (action == "merge" && x.FeatureID == "") {
		writeError(w, 400, "invalid_request")
		return
	}
	id := newID()
	if action == "merge" {
		id = x.FeatureID
	}
	e := server.admin.ReviewSuggestion(r.Context(), r.PathValue("id"), action, id, x.Slug, strings.TrimSpace(x.Title), strings.TrimSpace(x.Description), time.Now().UTC())
	if e != nil {
		writeError(w, 404, "not_found")
		return
	}
	w.WriteHeader(204)
}
func newID() string {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
