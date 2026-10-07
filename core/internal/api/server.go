package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"

	"brainer/internal/fill"
	"brainer/internal/memory"
	"brainer/web"
)

type Server struct {
	Store *memory.Store
	Fill  *fill.Runner
}

type captureBody struct {
	Raw string `json:"raw"`
}

type rosterBody struct {
	Name      *string   `json:"name"`
	Nicknames *[]string `json:"nicknames"`
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/captures", s.postCapture)
	mux.HandleFunc("GET /api/tasks", s.listTasks)
	mux.HandleFunc("GET /api/tasks/{id}", s.getTask)
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
	mux.Handle("GET /static/", http.FileServer(http.FS(web.Static)))
	mux.HandleFunc("GET /{$}", s.index)
	mux.HandleFunc("GET /archive", s.archivePage)
	mux.HandleFunc("GET /teams", s.teamsPage)
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
