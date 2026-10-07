package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"brainer/internal/memory"
)

func TestOrgRosterHTTP(t *testing.T) {
	store, err := memory.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{Store: store}
	h := srv.Handler()

	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest("GET", "/teams", nil))
	if res.Code != 200 {
		t.Fatalf("GET /teams: %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "add-team") || !strings.Contains(res.Body.String(), "memobar") {
		t.Fatalf("teams page missing form: %s", res.Body.String())
	}

	req := httptest.NewRequest("POST", "/api/teams", strings.NewReader(`{"name":"Platform","nicknames":["plat"]}`))
	req.Header.Set("Content-Type", "application/json")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("POST team: %d %s", res.Code, res.Body.String())
	}
	var team memory.Team
	if err := json.Unmarshal(res.Body.Bytes(), &team); err != nil {
		t.Fatal(err)
	}
	if len(team.Nicknames) != 1 || team.Nicknames[0] != "plat" {
		t.Fatalf("team nicknames: %+v", team.Nicknames)
	}

	req = httptest.NewRequest("POST", "/api/teams/"+team.ID+"/employees", strings.NewReader(`{"name":"Sara","nicknames":["Sari"]}`))
	req.Header.Set("Content-Type", "application/json")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("POST employee: %d %s", res.Code, res.Body.String())
	}

	req = httptest.NewRequest("PATCH", "/api/teams/"+team.ID+"/employees/sara", strings.NewReader(`{"nicknames":["S"]}`))
	req.Header.Set("Content-Type", "application/json")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("PATCH employee: %d %s", res.Code, res.Body.String())
	}

	req = httptest.NewRequest("PATCH", "/api/teams/"+team.ID, strings.NewReader(`{"name":"Platform Engineering"}`))
	req.Header.Set("Content-Type", "application/json")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("PATCH team name: %d %s", res.Code, res.Body.String())
	}

	req = httptest.NewRequest("PATCH", "/api/teams/"+team.ID+"/employees/sara", strings.NewReader(`{"name":"Sarah"}`))
	req.Header.Set("Content-Type", "application/json")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("PATCH employee name: %d %s", res.Code, res.Body.String())
	}

	res = httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest("GET", "/api/org", nil))
	if res.Code != 200 {
		t.Fatalf("GET org: %d", res.Code)
	}
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), `"Sarah"`) || !strings.Contains(string(body), `"Platform Engineering"`) || !strings.Contains(string(body), `"S"`) {
		t.Fatalf("org: %s", body)
	}
}
