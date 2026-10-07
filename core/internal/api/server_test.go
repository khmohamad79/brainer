package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"brainer/internal/jira"
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

func TestPatchTaskRelations(t *testing.T) {
	store, err := memory.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	team, err := store.AddTeam("Platform", nil)
	if err != nil {
		t.Fatal(err)
	}
	emp, err := store.AddEmployee(team.ID, "Sara", nil)
	if err != nil {
		t.Fatal(err)
	}
	story, err := store.AddStory("Payment timeout", "", "")
	if err != nil {
		t.Fatal(err)
	}
	task, err := store.Capture("hello")
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{Store: store}
	h := srv.Handler()

	body := `{"related_teams":["` + team.ID + `","ghost"],"related_employees":["` + emp.ID + `"],"story_id":"` + story.ID + `"}`
	req := httptest.NewRequest("PATCH", "/api/tasks/"+task.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("PATCH: %d %s", res.Code, res.Body.String())
	}
	var got memory.Task
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.RelatedTeams) != 1 || got.RelatedTeams[0] != team.ID {
		t.Fatalf("teams: %+v", got.RelatedTeams)
	}
	if len(got.RelatedEmployees) != 1 || got.RelatedEmployees[0] != emp.ID {
		t.Fatalf("emps: %+v", got.RelatedEmployees)
	}
	if got.StoryID != story.ID {
		t.Fatalf("story: %q", got.StoryID)
	}

	req = httptest.NewRequest("PATCH", "/api/tasks/"+task.ID, strings.NewReader(`{"story_id":""}`))
	req.Header.Set("Content-Type", "application/json")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("clear story: %d %s", res.Code, res.Body.String())
	}
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.StoryID != "" {
		t.Fatalf("expected cleared story, got %q", got.StoryID)
	}
}

func TestStoriesHTTP(t *testing.T) {
	store, err := memory.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	jiraSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"fields": map[string]any{
				"summary":     "Payment timeout",
				"status":      map[string]any{"name": "Open"},
				"description": "Checkout hangs",
			},
		})
	}))
	defer jiraSrv.Close()
	jc := jira.New(jiraSrv.URL, "tok")
	jc.HTTPClient = jiraSrv.Client()
	srv := &Server{Store: store, Jira: jc}
	h := srv.Handler()

	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest("GET", "/stories", nil))
	if res.Code != 200 {
		t.Fatalf("GET /stories: %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "add-story") {
		t.Fatalf("stories page: %s", res.Body.String())
	}

	req := httptest.NewRequest("POST", "/api/stories", strings.NewReader(`{"title":"Payment timeout","jira_key":"DEVPR-5982"}`))
	req.Header.Set("Content-Type", "application/json")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("POST story: %d %s", res.Code, res.Body.String())
	}
	var story memory.Story
	if err := json.Unmarshal(res.Body.Bytes(), &story); err != nil {
		t.Fatal(err)
	}
	if story.ID != "payment-timeout" || story.JiraKey != "DEVPR-5982" {
		t.Fatalf("story: %+v", story)
	}
	if !strings.Contains(story.Summary, "Payment timeout") {
		t.Fatalf("summary not filled: %q", story.Summary)
	}

	req = httptest.NewRequest("PATCH", "/api/stories/"+story.ID, strings.NewReader(`{"summary":"edited"}`))
	req.Header.Set("Content-Type", "application/json")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("PATCH summary: %d %s", res.Code, res.Body.String())
	}

	res = httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest("POST", "/api/stories/"+story.ID+"/jira-refresh", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("jira-refresh: %d %s", res.Code, res.Body.String())
	}
	if err := json.Unmarshal(res.Body.Bytes(), &story); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(story.Summary, "Checkout hangs") {
		t.Fatalf("refreshed summary: %q", story.Summary)
	}

	res = httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest("GET", "/api/stories", nil))
	if res.Code != 200 || !strings.Contains(res.Body.String(), "payment-timeout") {
		t.Fatalf("GET stories: %d %s", res.Code, res.Body.String())
	}

	res = httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest("DELETE", "/api/stories/"+story.ID, nil))
	if res.Code != http.StatusNoContent {
		t.Fatalf("DELETE: %d", res.Code)
	}
}
