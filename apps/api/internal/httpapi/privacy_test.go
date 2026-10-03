package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wastingnotime/wishlist/apps/api/internal/application"
	"github.com/wastingnotime/wishlist/apps/api/internal/infrastructure"
)

type mutablePrivacyClock struct{ now time.Time }

func (clock *mutablePrivacyClock) Now() time.Time { return clock.now }

func TestPrivacyAccessCorrectionAndErasure(t *testing.T) {
	t.Setenv("WISHLIST_ADMIN_TOKEN", "privacy-admin-token")
	board, err := infrastructure.SampleBoard()
	if err != nil {
		t.Fatal(err)
	}
	store := testPostgresStore(t, board)
	clock := &mutablePrivacyClock{now: time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)}
	sender := &testSender{}
	visitor := application.NewVisitor(store, sender, clock, "privacy-test-secret")
	handler := New(application.NewBoard(store), visitor, false, store).Handler()
	call := func(method, path, body string, cookie *http.Cookie, admin, sameOrigin bool) *httptest.ResponseRecorder {
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		if sameOrigin {
			request.Header.Set("X-Wishlist-Same-Origin", "1")
		}
		if cookie != nil {
			request.AddCookie(cookie)
		}
		if admin {
			request.Header.Set("Authorization", "Bearer privacy-admin-token")
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}
	verify := func(email string) *http.Cookie {
		t.Helper()
		if result := call(http.MethodPost, "/v1/otp", `{"email":"`+email+`"}`, nil, false, true); result.Code != http.StatusAccepted {
			t.Fatalf("request code: %d %s", result.Code, result.Body)
		}
		result := call(http.MethodPost, "/v1/otp/verify", `{"email":"`+email+`","code":"`+sender.code+`"}`, nil, false, true)
		if result.Code != http.StatusOK {
			t.Fatalf("verify code: %d %s", result.Code, result.Body)
		}
		return result.Result().Cookies()[0]
	}
	if result := call(http.MethodGet, "/v1/privacy/data", "", nil, false, false); result.Code != http.StatusUnauthorized {
		t.Fatalf("unverified access = %d", result.Code)
	}
	if result := call(http.MethodDelete, "/v1/privacy/data", "", nil, false, true); result.Code != http.StatusUnauthorized {
		t.Fatalf("unverified erasure = %d", result.Code)
	}
	first := verify("person@example.com")
	if result := call(http.MethodPost, "/v1/features/feature-family-sharing/vote", `{}`, first, false, true); result.Code != http.StatusOK {
		t.Fatalf("vote: %d %s", result.Code, result.Body)
	}
	if result := call(http.MethodPost, "/v1/suggestions", `{"app_id":"app-cat-care","title":"Private request","description":"My private details"}`, first, false, true); result.Code != http.StatusCreated {
		t.Fatalf("suggestion: %d %s", result.Code, result.Body)
	}
	other := verify("other@example.com")
	if result := call(http.MethodGet, "/v1/privacy/data", "", other, false, false); result.Code != http.StatusOK || strings.Contains(result.Body.String(), "Private request") {
		t.Fatalf("cross-account access: %d %s", result.Code, result.Body)
	}
	result := call(http.MethodGet, "/v1/privacy/data", "", first, false, false)
	var data application.PrivacyData
	if err := json.Unmarshal(result.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	if result.Code != http.StatusOK || data.Email != "person@example.com" || len(data.Votes) != 1 || len(data.Suggestions) != 1 || data.Suggestions[0].Description != "My private details" {
		t.Fatalf("privacy access: %d %+v", result.Code, data)
	}
	operatorBody := `{"email":"person@example.com","case_reference":"RIGHTS-001"}`
	if result := call(http.MethodPost, "/v1/admin/privacy/data", operatorBody, nil, false, true); result.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized operator access = %d", result.Code)
	}
	if result := call(http.MethodPost, "/v1/admin/privacy/data", `{"email":"person@example.com"}`, nil, true, true); result.Code != http.StatusBadRequest {
		t.Fatalf("operator access without case = %d", result.Code)
	}
	if result := call(http.MethodPost, "/v1/admin/privacy/data", operatorBody, nil, true, true); result.Code != http.StatusOK || !strings.Contains(result.Body.String(), "Private request") {
		t.Fatalf("operator access: %d %s", result.Code, result.Body)
	}
	if result := call(http.MethodDelete, "/v1/privacy/data", "", first, false, false); result.Code != http.StatusForbidden {
		t.Fatalf("cross-origin erasure = %d", result.Code)
	}
	clock.now = clock.now.Add(11 * time.Minute)
	if result := call(http.MethodGet, "/v1/privacy/data", "", first, false, false); result.Code != http.StatusForbidden {
		t.Fatalf("stale access = %d", result.Code)
	}
	if result := call(http.MethodDelete, "/v1/privacy/data", "", first, false, true); result.Code != http.StatusForbidden {
		t.Fatalf("stale erasure = %d", result.Code)
	}
	if result := call(http.MethodGet, "/v1/session", "", first, false, false); !strings.Contains(result.Body.String(), `"verified":true`) {
		t.Fatalf("ordinary session lost: %s", result.Body)
	}
	fresh := verify("person@example.com")
	if result := call(http.MethodPost, "/v1/privacy/email/request", `{"new_email":"other@example.com"}`, fresh, false, true); result.Code != http.StatusAccepted {
		t.Fatalf("existing email code request: %d %s", result.Code, result.Body)
	}
	if result := call(http.MethodPost, "/v1/privacy/email/verify", `{"new_email":"other@example.com","code":"`+sender.code+`"}`, fresh, false, true); result.Code != http.StatusConflict {
		t.Fatalf("existing identity correction = %d %s", result.Code, result.Body)
	}
	if result := call(http.MethodPost, "/v1/privacy/email/request", `{"new_email":"new@example.com"}`, fresh, false, true); result.Code != http.StatusAccepted {
		t.Fatalf("new email code request: %d %s", result.Code, result.Body)
	}
	wrong := "000000"
	if sender.code == wrong {
		wrong = "111111"
	}
	if result := call(http.MethodPost, "/v1/privacy/email/verify", `{"new_email":"new@example.com","code":"`+wrong+`"}`, fresh, false, true); result.Code != http.StatusBadRequest {
		t.Fatalf("wrong correction code = %d", result.Code)
	}
	if result := call(http.MethodPost, "/v1/privacy/email/verify", `{"new_email":"new@example.com","code":"`+sender.code+`"}`, fresh, false, true); result.Code != http.StatusOK {
		t.Fatalf("correct email: %d %s", result.Code, result.Body)
	}
	if result := call(http.MethodGet, "/v1/privacy/data", "", fresh, false, false); result.Code != http.StatusOK || !strings.Contains(result.Body.String(), `"email":"new@example.com"`) {
		t.Fatalf("corrected access: %d %s", result.Code, result.Body)
	}
	if result := call(http.MethodGet, "/v1/session", "", first, false, false); !strings.Contains(result.Body.String(), `"verified":false`) {
		t.Fatalf("older session survived correction: %s", result.Body)
	}
	queue := call(http.MethodGet, "/v1/admin/suggestions", "", nil, true, false)
	var suggestions struct {
		Suggestions []struct {
			ID string `json:"id"`
		} `json:"suggestions"`
	}
	if err := json.Unmarshal(queue.Body.Bytes(), &suggestions); err != nil || len(suggestions.Suggestions) != 1 {
		t.Fatalf("admin suggestions: %s (%v)", queue.Body, err)
	}
	accepted := call(http.MethodPost, "/v1/admin/suggestions/"+suggestions.Suggestions[0].ID+"/accept", `{"slug":"private-request","title":"Published request","description":"A reviewed public idea"}`, nil, true, true)
	if accepted.Code != http.StatusNoContent {
		t.Fatalf("accept suggestion: %d %s", accepted.Code, accepted.Body)
	}
	deleted := call(http.MethodDelete, "/v1/privacy/data", "", fresh, false, true)
	if deleted.Code != http.StatusNoContent || len(deleted.Result().Cookies()) == 0 {
		t.Fatalf("erasure: %d %s", deleted.Code, deleted.Body)
	}
	if result := call(http.MethodDelete, "/v1/privacy/data", "", fresh, false, true); result.Code != http.StatusUnauthorized {
		t.Fatalf("repeated erasure = %d", result.Code)
	}
	if result := call(http.MethodGet, "/v1/session", "", fresh, false, false); !strings.Contains(result.Body.String(), `"verified":false`) {
		t.Fatalf("session survived erasure: %s", result.Body)
	}
	public := call(http.MethodGet, "/v1/features?view=voting&app=cat-care", "", nil, false, false)
	if strings.Contains(public.Body.String(), `"vote_count":1`) || !strings.Contains(public.Body.String(), "Published request") {
		t.Fatalf("vote count or public feature after erasure: %s", public.Body)
	}
	if result := call(http.MethodGet, "/v1/admin/privacy/reviews", "", nil, false, false); result.Code != http.StatusUnauthorized {
		t.Fatalf("review queue public = %d", result.Code)
	}
	reviews := call(http.MethodGet, "/v1/admin/privacy/reviews", "", nil, true, false)
	var reviewQueue struct {
		Reviews []struct {
			FeatureID string `json:"feature_id"`
		} `json:"reviews"`
	}
	if err := json.Unmarshal(reviews.Body.Bytes(), &reviewQueue); err != nil || len(reviewQueue.Reviews) != 1 {
		t.Fatalf("review queue: %s (%v)", reviews.Body, err)
	}
	if result := call(http.MethodPost, "/v1/admin/privacy/reviews/"+reviewQueue.Reviews[0].FeatureID+"/resolve", `{}`, nil, true, true); result.Code != http.StatusNoContent {
		t.Fatalf("resolve review: %d %s", result.Code, result.Body)
	}
	if result := call(http.MethodGet, "/v1/admin/privacy/reviews", "", nil, true, false); !strings.Contains(result.Body.String(), `"reviews":[]`) {
		t.Fatalf("resolved review remains: %s", result.Body)
	}
	clock.now = clock.now.Add(time.Second)
	reregistered := verify("new@example.com")
	result = call(http.MethodGet, "/v1/privacy/data", "", reregistered, false, false)
	if err := json.Unmarshal(result.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	if result.Code != http.StatusOK || data.Email != "new@example.com" || len(data.Votes) != 0 || len(data.Suggestions) != 0 {
		t.Fatalf("re-registration reused erased data: %d %+v", result.Code, data)
	}
	if result := call(http.MethodPost, "/v1/admin/privacy/correct-email", `{"email":"new@example.com","new_email":"corrected@example.com","case_reference":"RIGHTS-002"}`, nil, true, true); result.Code != http.StatusNoContent {
		t.Fatalf("operator correction: %d %s", result.Code, result.Body)
	}
	if result := call(http.MethodPost, "/v1/admin/privacy/erase", `{"email":"corrected@example.com","case_reference":"RIGHTS-002"}`, nil, true, true); result.Code != http.StatusNoContent {
		t.Fatalf("operator erasure: %d %s", result.Code, result.Body)
	}
	if result := call(http.MethodGet, "/v1/session", "", reregistered, false, false); !strings.Contains(result.Body.String(), `"verified":false`) {
		t.Fatalf("operator erasure retained session: %s", result.Body)
	}
	if result := call(http.MethodPost, "/v1/admin/privacy/erase", `{"email":"corrected@example.com","case_reference":"RIGHTS-002"}`, nil, true, true); result.Code != http.StatusNotFound {
		t.Fatalf("repeated operator erasure = %d", result.Code)
	}
}
