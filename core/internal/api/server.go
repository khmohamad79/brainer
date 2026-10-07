package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"brainer/internal/fill"
	"brainer/internal/jira"
	"brainer/internal/memory"
	"brainer/web"
)

type Server struct {
	Store *memory.Store
	Fill  *fill.Runner
	Jira  *jira.Client
}

type captureBody struct {
	Raw string `json:"raw"`
}

type rosterBody struct {
	Name      *string   `json:"name"`
	Nicknames *[]string `json:"nicknames"`
}

type storyBody struct {
	Title   *string `json:"title"`
	JiraKey *string `json:"jira_key"`
	Summary *string `json:"summary"`
}

type taskPatchBody struct {
	RelatedTeams     *[]string `json:"related_teams"`
	RelatedEmployees *[]string `json:"related_employees"`
	StoryID          *string   `json:"story_id"`
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/captures", s.postCapture)
	mux.HandleFunc("GET /api/tasks", s.listTasks)
	mux.HandleFunc("GET /api/tasks/{id}", s.getTask)
	mux.HandleFunc("PATCH /api/tasks/{id}", s.patchTask)
	mux.HandleFunc("POST /api/tasks/{id}/fill", s.reFill)
	mux.HandleFunc("POST /api/tasks/{id}/archive", s.archiveTask)
	mux.HandleFunc("DELETE /api/tasks/{id}", s.deleteTask)
	mux.HandleFunc("GET /api/org", s.getOrg)
	mux.HandleFunc("POST /api/teams", s.postTeam)
	mux.HandleFunc("PATCH /api/teams/{id}", s.patchTeam)
	mux.HandleFunc("DELETE /api/teams/{id}", s.deleteTeam)
	mux.HandleFunc("POST /api/teams/{id}/employees", s.postEmployee)
	mux.HandleFunc("PATCH /api/teams/{id}/employees/{eid}", s.patchEmployee)
	mux.HandleFunc("DELETE /api/teams/{id}/employees/{eid}", s.deleteEmployee)
	mux.HandleFunc("GET /api/stories", s.getStories)
	mux.HandleFunc("POST /api/stories", s.postStory)
	mux.HandleFunc("PATCH /api/stories/{id}", s.patchStory)
	mux.HandleFunc("DELETE /api/stories/{id}", s.deleteStory)
	mux.HandleFunc("POST /api/stories/{id}/jira-refresh", s.refreshStoryJira)
	mux.Handle("GET /static/", http.FileServer(http.FS(web.Static)))
	mux.HandleFunc("GET /{$}", s.index)
	mux.HandleFunc("GET /archive", s.archivePage)
	mux.HandleFunc("GET /teams", s.teamsPage)
	mux.HandleFunc("GET /stories", s.storiesPage)
	return mux
}

func (s *Server) postCapture(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	var in captureBody
	if err := json.Unmarshal(body, &in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	task, err := s.Store.Capture(in.Raw)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	go s.Fill.Fill(task)
	writeJSON(w, http.StatusCreated, task)
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	archived := r.URL.Query().Get("archived") == "1"
	tasks, err := s.Store.List(archived)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tasks == nil {
		tasks = []*memory.Task{}
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, err := s.Store.Get(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) patchTask(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	var in taskPatchBody
	if err := json.Unmarshal(body, &in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if in.RelatedTeams == nil && in.RelatedEmployees == nil && in.StoryID == nil {
		http.Error(w, "no fields to patch", http.StatusBadRequest)
		return
	}
	id := r.PathValue("id")
	task, err := s.Store.Get(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	org, err := s.Store.Org()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	stories, err := s.Store.Stories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if in.RelatedTeams != nil || in.RelatedEmployees != nil {
		teams := task.RelatedTeams
		emps := task.RelatedEmployees
		if in.RelatedTeams != nil {
			teams = *in.RelatedTeams
		}
		if in.RelatedEmployees != nil {
			emps = *in.RelatedEmployees
		}
		task.RelatedTeams, task.RelatedEmployees = filterTaskRelated(teams, emps, org)
	}
	if in.StoryID != nil {
		task.StoryID = filterTaskStory(*in.StoryID, stories)
	}
	if err := s.Store.Save(task); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func filterTaskRelated(teamIDs, empIDs []string, org *memory.Org) ([]string, []string) {
	knownTeams := map[string]struct{}{}
	knownEmps := map[string]struct{}{}
	if org != nil {
		for _, t := range org.Teams {
			knownTeams[t.ID] = struct{}{}
			for _, e := range t.Employees {
				knownEmps[e.ID] = struct{}{}
			}
		}
	}
	return keepKnownIDs(teamIDs, knownTeams), keepKnownIDs(empIDs, knownEmps)
}

func filterTaskStory(id string, stories *memory.StoryList) string {
	id = strings.TrimSpace(id)
	if id == "" || stories == nil {
		return ""
	}
	for _, st := range stories.Stories {
		if st.ID == id {
			return id
		}
	}
	return ""
}

func keepKnownIDs(ids []string, known map[string]struct{}) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := known[id]; !ok {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func (s *Server) reFill(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, err := s.Store.Get(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	task.Fill = memory.FillPending
	task.FillError = ""
	if err := s.Store.Save(task); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	go s.Fill.Fill(task)
	writeJSON(w, http.StatusAccepted, task)
}

func (s *Server) archiveTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, err := s.Store.Archive(id)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.Store.Delete(id); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFileFS(w, r, web.Pages, "index.html")
}

func (s *Server) archivePage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/?list=archive", http.StatusFound)
}

func (s *Server) teamsPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFileFS(w, r, web.Pages, "teams.html")
}

func (s *Server) storiesPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFileFS(w, r, web.Pages, "stories.html")
}

func (s *Server) getStories(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.Stories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) postStory(w http.ResponseWriter, r *http.Request) {
	in, ok := readStory(w, r)
	if !ok {
		return
	}
	if in.Title == nil || strings.TrimSpace(*in.Title) == "" {
		http.Error(w, "title is empty", http.StatusBadRequest)
		return
	}
	jiraKey := ""
	if in.JiraKey != nil {
		jiraKey = strings.TrimSpace(*in.JiraKey)
	}
	summary := ""
	if jiraKey != "" {
		sum, err := s.fetchJiraSummary(jiraKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		summary = sum
	}
	story, err := s.Store.AddStory(*in.Title, jiraKey, summary)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, story)
}

func (s *Server) patchStory(w http.ResponseWriter, r *http.Request) {
	in, ok := readStory(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	list, err := s.Store.Stories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var prev *memory.Story
	for i := range list.Stories {
		if list.Stories[i].ID == id {
			prev = &list.Stories[i]
			break
		}
	}
	if prev == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	summary := in.Summary
	if in.JiraKey != nil {
		newKey := strings.ToUpper(strings.TrimSpace(*in.JiraKey))
		oldKey := prev.JiraKey
		if newKey != oldKey {
			if newKey == "" {
				empty := ""
				summary = &empty
			} else {
				sum, err := s.fetchJiraSummary(newKey)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadGateway)
					return
				}
				summary = &sum
			}
		}
	}
	story, err := s.Store.UpdateStory(id, in.Title, in.JiraKey, summary)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, story)
}

func (s *Server) deleteStory(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.DeleteStory(r.PathValue("id")); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) refreshStoryJira(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	list, err := s.Store.Stories()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var prev *memory.Story
	for i := range list.Stories {
		if list.Stories[i].ID == id {
			prev = &list.Stories[i]
			break
		}
	}
	if prev == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if prev.JiraKey == "" {
		http.Error(w, "jira_key is empty", http.StatusBadRequest)
		return
	}
	sum, err := s.fetchJiraSummary(prev.JiraKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	story, err := s.Store.UpdateStory(id, nil, nil, &sum)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, story)
}

func (s *Server) fetchJiraSummary(key string) (string, error) {
	if s.Jira == nil || !s.Jira.Configured() {
		return "", errors.New("jira not configured (set JIRA_BASE_URL and JIRA_TOKEN)")
	}
	return s.Jira.FetchIssueSummary(key)
}

func readStory(w http.ResponseWriter, r *http.Request) (storyBody, bool) {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return storyBody{}, false
	}
	var in storyBody
	if err := json.Unmarshal(body, &in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return storyBody{}, false
	}
	return in, true
}

func (s *Server) getOrg(w http.ResponseWriter, r *http.Request) {
	org, err := s.Store.Org()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, org)
}

func (s *Server) postTeam(w http.ResponseWriter, r *http.Request) {
	in, ok := readRoster(w, r)
	if !ok {
		return
	}
	if in.Name == nil {
		http.Error(w, "name is empty", http.StatusBadRequest)
		return
	}
	team, err := s.Store.AddTeam(*in.Name, derefNicks(in.Nicknames))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, team)
}

func (s *Server) patchTeam(w http.ResponseWriter, r *http.Request) {
	in, ok := readRoster(w, r)
	if !ok {
		return
	}
	team, err := s.Store.UpdateTeam(r.PathValue("id"), in.Name, in.Nicknames)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, team)
}

func (s *Server) deleteTeam(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.DeleteTeam(r.PathValue("id")); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) postEmployee(w http.ResponseWriter, r *http.Request) {
	in, ok := readRoster(w, r)
	if !ok {
		return
	}
	if in.Name == nil {
		http.Error(w, "name is empty", http.StatusBadRequest)
		return
	}
	emp, err := s.Store.AddEmployee(r.PathValue("id"), *in.Name, derefNicks(in.Nicknames))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, emp)
}

func (s *Server) patchEmployee(w http.ResponseWriter, r *http.Request) {
	in, ok := readRoster(w, r)
	if !ok {
		return
	}
	emp, err := s.Store.UpdateEmployee(r.PathValue("id"), r.PathValue("eid"), in.Name, in.Nicknames)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, emp)
}

func (s *Server) deleteEmployee(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.DeleteEmployee(r.PathValue("id"), r.PathValue("eid")); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func readRoster(w http.ResponseWriter, r *http.Request) (rosterBody, bool) {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return rosterBody{}, false
	}
	var in rosterBody
	if err := json.Unmarshal(body, &in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return rosterBody{}, false
	}
	return in, true
}

func derefNicks(nicks *[]string) []string {
	if nicks == nil {
		return nil
	}
	return *nicks
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
