package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wastingnotime/wishlist/apps/api/internal/application"
	"github.com/wastingnotime/wishlist/apps/api/internal/infrastructure"
)

func testServer(t *testing.T) http.Handler {
	t.Helper()
	board, err := infrastructure.SampleBoard()
	if err != nil {
		t.Fatal(err)
	}
	return New(application.NewBoard(board)).Handler()
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
	if len(body.Features) != 3 || body.Features[0].ID != "feature-family-sharing" || body.Features[0].VoteCount != 184 {
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
