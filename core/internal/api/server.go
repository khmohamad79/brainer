package api

import (
	"encoding/json"
	"io"
	"net/http"

	"brainer/internal/enrich"
	"brainer/internal/memory"
	"brainer/web"
)

type Server struct {
	Store  *memory.Store
	Enrich *enrich.Runner
}

type captureBody struct {
	Raw string `json:"raw"`
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/captures", s.postCapture)
	mux.HandleFunc("GET /api/tasks", s.listTasks)
	mux.HandleFunc("GET /api/tasks/{id}", s.getTask)
	mux.HandleFunc("POST /api/tasks/{id}/enrich", s.reEnrich)
	mux.Handle("GET /static/", http.FileServer(http.FS(web.Static)))
	mux.HandleFunc("GET /{$}", s.index)
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
	go s.Enrich.Enrich(task)
	writeJSON(w, http.StatusCreated, task)
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.Store.List()
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

func (s *Server) reEnrich(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, err := s.Store.Get(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	task.Enrichment = memory.EnrichmentPending
	task.EnrichmentError = ""
	if err := s.Store.Save(task); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	go s.Enrich.Enrich(task)
	writeJSON(w, http.StatusAccepted, task)
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFileFS(w, r, web.Pages, "index.html")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
