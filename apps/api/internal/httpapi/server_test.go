package httpapi

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/wastingnotime/wishlist/apps/api/internal/application"
	"github.com/wastingnotime/wishlist/apps/api/internal/domain"
	"github.com/wastingnotime/wishlist/apps/api/internal/infrastructure"
)

func testPostgresStore(t *testing.T, board domain.Board) *infrastructure.PostgresStore {
	t.Helper()
	databaseURL := os.Getenv("WISHLIST_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set WISHLIST_TEST_DATABASE_URL to run PostgreSQL API tests")
	}
	suffix := make([]byte, 8)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatal(err)
	}
	schema := "wishlist_test_" + hex.EncodeToString(suffix)
	admin, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = admin.Exec(`CREATE SCHEMA ` + schema); err != nil {
		_ = admin.Close()
		t.Fatal(err)
	}
	var store *infrastructure.PostgresStore
	t.Cleanup(func() {
		if store != nil {
			_ = store.Close()
		}
		_, _ = admin.Exec(`DROP SCHEMA ` + schema + ` CASCADE`)
		_ = admin.Close()
	})
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	store, err = infrastructure.OpenPostgres(parsed.String(), board)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

type testClock struct{}

func (testClock) Now() time.Time { return time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC) }

type testSender struct{ code string }

func (s *testSender) Send(_ context.Context, _ string, code string) error { s.code = code; return nil }

func testServer(t *testing.T) http.Handler {
	t.Helper()
	board, err := infrastructure.SampleBoard()
	if err != nil {
		t.Fatal(err)
	}
	store := testPostgresStore(t, board)
	visitor := application.NewVisitor(store, &testSender{}, testClock{}, "test-secret")
	return New(application.NewBoard(store), visitor, false).Handler()
}

func TestPublicViewsFilterAndRankFeatures(t *testing.T) {
	handler := testServer(t)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/features?view=voting&app=cat-care", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	var body struct {
		Features []struct {
			ID        string `json:"id"`
			VoteCount int    `json:"vote_count"`
			AppSlug   string `json:"app_slug"`
		} `json:"features"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Features) != 3 || body.Features[0].ID != "feature-family-sharing" || body.Features[0].VoteCount != 0 {
		t.Fatalf("unexpected ranking: %+v", body.Features)
	}
	for _, feature := range body.Features {
		if feature.AppSlug != "cat-care" {
			t.Fatalf("app filter leaked feature: %+v", feature)
		}
	}
}

func TestAppFilterIsConsistentAcrossAllViews(t *testing.T) {
	handler := testServer(t)
	for _, view := range []string{"voting", "producing", "delivered"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/features?view="+view+"&app=cat-care", nil))
		var body struct {
			Features []struct {
				AppSlug string `json:"app_slug"`
			} `json:"features"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusOK || len(body.Features) == 0 {
			t.Fatalf("%s returned status %d and %d rows", view, response.Code, len(body.Features))
		}
		for _, feature := range body.Features {
			if feature.AppSlug != "cat-care" {
				t.Fatalf("%s leaked app %q", view, feature.AppSlug)
			}
		}
	}
}

func TestViewsDefaultErrorsAndHealth(t *testing.T) {
	handler := testServer(t)
	cases := []struct {
		path string
		want int
	}{
		{"/v1/apps", http.StatusOK},
		{"/v1/features", http.StatusOK},
		{"/v1/features?view=upcoming", http.StatusBadRequest},
		{"/v1/features?app=bad%20slug", http.StatusBadRequest},
		{"/healthz", http.StatusOK},
	}
	for _, test := range cases {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
		if response.Code != test.want {
			t.Errorf("GET %s status = %d, want %d", test.path, response.Code, test.want)
		}
	}
}

func TestPublicResponsesContainNoIdentityOrSuggestionFields(t *testing.T) {
	response := httptest.NewRecorder()
	testServer(t).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/features", nil))
	body := response.Body.String()
	for _, forbidden := range []string{"email", "identity_id", "session_id", "suggestion"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("public response contains %q: %s", forbidden, body)
		}
	}
}

func TestOTPVoteAndPrivateSuggestionFlow(t *testing.T) {
	board, _ := infrastructure.SampleBoard()
	store := testPostgresStore(t, board)
	sender := &testSender{}
	visitor := application.NewVisitor(store, sender, testClock{}, "test-secret")
	handler := New(application.NewBoard(store), visitor, false).Handler()
	call := func(method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("X-Wishlist-Same-Origin", "1")
		if cookie != nil {
			req.AddCookie(cookie)
		}
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		return res
	}
	if got := call(http.MethodPost, "/v1/otp", `{"email":"person@example.com"}`, nil); got.Code != http.StatusAccepted {
		t.Fatalf("request OTP: %d %s", got.Code, got.Body)
	}
	badCode := call(http.MethodPost, "/v1/otp/verify", `{"email":"person@example.com","code":"000000"}`, nil)
	if badCode.Code != http.StatusBadRequest || strings.Contains(badCode.Body.String(), "email") {
		t.Fatalf("invalid OTP response was not generic: %d %s", badCode.Code, badCode.Body)
	}
	verified := call(http.MethodPost, "/v1/otp/verify", `{"email":"person@example.com","code":"`+sender.code+`"}`, nil)
	if verified.Code != http.StatusOK {
		t.Fatalf("verify OTP: %d %s", verified.Code, verified.Body)
	}
	cookies := verified.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly {
		t.Fatalf("session cookie not private: %+v", cookies)
	}
	vote := call(http.MethodPost, "/v1/features/feature-family-sharing/vote", `{}`, cookies[0])
	if vote.Code != http.StatusOK || !strings.Contains(vote.Body.String(), `"vote_count":1`) {
		t.Fatalf("vote: %d %s", vote.Code, vote.Body)
	}
	removed := call(http.MethodPost, "/v1/features/feature-family-sharing/vote", `{}`, cookies[0])
	if removed.Code != http.StatusOK || !strings.Contains(removed.Body.String(), `"vote_count":0`) {
		t.Fatalf("vote toggle did not remove vote: %d %s", removed.Code, removed.Body)
	}
	vote = call(http.MethodPost, "/v1/features/feature-family-sharing/vote", `{}`, cookies[0])
	session := call(http.MethodGet, "/v1/session", "", cookies[0])
	if session.Code != http.StatusOK || !strings.Contains(session.Body.String(), "feature-family-sharing") {
		t.Fatalf("session: %d %s", session.Code, session.Body)
	}
	suggestion := call(http.MethodPost, "/v1/suggestions", `{"app_id":"app-cat-care","title":"A private suggestion","description":"Only reviewers should see this."}`, cookies[0])
	if suggestion.Code != http.StatusCreated || strings.Contains(suggestion.Body.String(), "Only reviewers") {
		t.Fatalf("suggestion leaked or failed: %d %s", suggestion.Code, suggestion.Body)
	}
	if got := call(http.MethodPost, "/v1/features/feature-family-sharing-shipped/vote", `{}`, cookies[0]); got.Code != http.StatusConflict {
		t.Fatalf("delivered item accepted vote: %d", got.Code)
	}
}

func TestAdminModerationAndLifecycleKeepVotes(t *testing.T) {
	t.Setenv("WISHLIST_ADMIN_TOKEN", "test-admin-token")
	board, _ := infrastructure.SampleBoard()
	store := testPostgresStore(t, board)
	sender := &testSender{}
	visitor := application.NewVisitor(store, sender, testClock{}, "secret")
	handler := New(application.NewBoard(store), visitor, false, store).Handler()
	request := func(method, path, body string, cookie *http.Cookie, admin bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("X-Wishlist-Same-Origin", "1")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		if admin {
			r.Header.Set("Authorization", "Bearer test-admin-token")
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if unauthorized := request(http.MethodGet, "/v1/admin/suggestions", "", nil, false); unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("admin route was not protected: %d", unauthorized.Code)
	}
	request(http.MethodPost, "/v1/otp", `{"email":"review@example.com"}`, nil, false)
	verified := request(http.MethodPost, "/v1/otp/verify", `{"email":"review@example.com","code":"`+sender.code+`"}`, nil, false)
	cookie := verified.Result().Cookies()[0]
	request(http.MethodPost, "/v1/suggestions", `{"app_id":"app-cat-care","title":"Care timeline export","description":"Let me print a visit-ready timeline."}`, cookie, false)
	public := request(http.MethodGet, "/v1/features?view=voting", "", nil, false)
	if strings.Contains(public.Body.String(), "Care timeline export") {
		t.Fatal("pending suggestion appeared publicly")
	}
	queue := request(http.MethodGet, "/v1/admin/suggestions", "", nil, true)
	var q struct {
		Suggestions []struct {
			ID string `json:"id"`
		} `json:"suggestions"`
	}
	if err := json.Unmarshal(queue.Body.Bytes(), &q); err != nil || len(q.Suggestions) != 1 {
		t.Fatalf("pending queue: %s (%v)", queue.Body, err)
	}
	accepted := request(http.MethodPost, "/v1/admin/suggestions/"+q.Suggestions[0].ID+"/accept", `{"slug":"care-timeline-export","title":"Care timeline export","description":"Let me print a visit-ready timeline."}`, nil, true)
	if accepted.Code != http.StatusNoContent {
		t.Fatalf("accept: %d %s", accepted.Code, accepted.Body)
	}
	listed := request(http.MethodGet, "/v1/features?view=voting&app=cat-care", "", nil, false)
	if !strings.Contains(listed.Body.String(), "Care timeline export") {
		t.Fatalf("accepted feature not public: %s", listed.Body)
	}
	featureID := ""
	var rows struct {
		Features []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"features"`
	}
	_ = json.Unmarshal(listed.Body.Bytes(), &rows)
	for _, row := range rows.Features {
		if row.Title == "Care timeline export" {
			featureID = row.ID
		}
	}
	if featureID == "" {
		t.Fatal("new feature id missing")
	}
	vote := request(http.MethodPost, "/v1/features/"+featureID+"/vote", `{}`, cookie, false)
	if vote.Code != http.StatusOK {
		t.Fatalf("vote accepted feature: %d %s", vote.Code, vote.Body)
	}
	request(http.MethodPost, "/v1/suggestions", `{"app_id":"app-cat-care","title":"A related timeline","description":"Combine it with the export."}`, cookie, false)
	queue = request(http.MethodGet, "/v1/admin/suggestions", "", nil, true)
	q.Suggestions = nil
	_ = json.Unmarshal(queue.Body.Bytes(), &q)
	if len(q.Suggestions) != 1 {
		t.Fatalf("merge suggestion queue: %s", queue.Body)
	}
	merged := request(http.MethodPost, "/v1/admin/suggestions/"+q.Suggestions[0].ID+"/merge", `{"feature_id":"`+featureID+`"}`, nil, true)
	if merged.Code != http.StatusNoContent {
		t.Fatalf("merge: %d %s", merged.Code, merged.Body)
	}
	skipped := request(http.MethodPatch, "/v1/admin/features/"+featureID, `{"title":"Care timeline export","description":"","status":"delivered","delivery_url":"https://example.org/release"}`, nil, true)
	if skipped.Code != http.StatusConflict {
		t.Fatalf("lifecycle allowed a stage skip: %d %s", skipped.Code, skipped.Body)
	}
	for _, step := range []struct {
		status string
		url    string
	}{{"producing", ""}, {"delivered", "https://wastingnotime.org/releases/care-timeline"}} {
		body, _ := json.Marshal(map[string]string{"title": "Care timeline export", "description": "Let me print a visit-ready timeline.", "status": step.status, "delivery_url": step.url})
		res := request(http.MethodPatch, "/v1/admin/features/"+featureID, string(body), nil, true)
		if res.Code != http.StatusNoContent {
			t.Fatalf("move to %s: %d %s", step.status, res.Code, res.Body)
		}
	}
	delivered := request(http.MethodGet, "/v1/features?view=delivered&app=cat-care", "", nil, false)
	if !strings.Contains(delivered.Body.String(), `"vote_count":1`) || !strings.Contains(delivered.Body.String(), "https://wastingnotime.org/releases/care-timeline") {
		t.Fatalf("historical vote or delivery link missing: %s", delivered.Body)
	}
}
