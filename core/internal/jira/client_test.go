package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchIssueSummary(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("auth: %q", r.Header.Get("Authorization"))
		}
		if !strings.Contains(r.URL.Path, "/rest/api/2/issue/DEVPR-5982") {
			t.Fatalf("path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"fields": map[string]any{
				"summary": "Payment timeout",
				"status":  map[string]any{"name": "In Progress"},
				"description": "Fix checkout timeouts\n\n<details>more</details>",
			},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token")
	c.HTTPClient = srv.Client()
	got, err := c.FetchIssueSummary("devpr-5982")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "DEVPR-5982: Payment timeout") {
		t.Fatalf("summary line: %q", got)
	}
	if !strings.Contains(got, "Status: In Progress") {
		t.Fatalf("status: %q", got)
	}
	if !strings.Contains(got, "Fix checkout timeouts") {
		t.Fatalf("description: %q", got)
	}
	if strings.Contains(got, "<details>") {
		t.Fatalf("html not stripped: %q", got)
	}
}

func TestFetchIssueSummaryNotConfigured(t *testing.T) {
	c := New("", "")
	if _, err := c.FetchIssueSummary("DEVPR-1"); err == nil {
		t.Fatal("expected error")
	}
}
