package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"gopkg.in/yaml.v3"
)

const (
	StatusInbox       = "inbox"
	EnrichmentPending = "pending"
	EnrichmentOK      = "ok"
	EnrichmentFailed  = "failed"
	SourceWeb         = "web"
)

type SplitCandidate struct {
	Title   string `yaml:"title" json:"title"`
	Excerpt string `yaml:"excerpt" json:"excerpt"`
}

type Task struct {
	ID               string           `yaml:"id" json:"id"`
	Status           string           `yaml:"status" json:"status"`
	CreatedAt        time.Time        `yaml:"created_at" json:"created_at"`
	UpdatedAt        time.Time        `yaml:"updated_at" json:"updated_at"`
	Source           string           `yaml:"source" json:"source"`
	Enrichment       string           `yaml:"enrichment" json:"enrichment"`
	EnrichmentError  string           `yaml:"enrichment_error,omitempty" json:"enrichment_error,omitempty"`
	NeedsSplit       bool             `yaml:"needs_split" json:"needs_split"`
	SplitCandidates  []SplitCandidate `yaml:"split_candidates" json:"split_candidates"`
	Title            string           `yaml:"title" json:"title"`
	Requester        string           `yaml:"requester" json:"requester"`
	DueAt            *string          `yaml:"due_at" json:"due_at"`
	Priority         *string          `yaml:"priority" json:"priority"`
	Context          []string         `yaml:"context" json:"context"`
	OpenQuestions    []string         `yaml:"open_questions" json:"open_questions"`
	RelatedTeams     []string         `yaml:"related_teams" json:"related_teams"`
	RelatedEmployees []string         `yaml:"related_employees" json:"related_employees"`
	Raw              string           `yaml:"raw" json:"raw"`
	Structured       string           `yaml:"structured" json:"structured"`
}

type Store struct {
	root string
	dir  string
	mu   sync.Mutex
}

func New(root string) (*Store, error) {
	dir := filepath.Join(root, "tasks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{root: root, dir: dir}, nil
}

func (s *Store) Capture(raw string) (*Task, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("raw is empty")
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.uniqueID(raw, now)
	t := &Task{
		ID:               id,
		Status:           StatusInbox,
		CreatedAt:        now,
		UpdatedAt:        now,
		Source:           SourceWeb,
		Enrichment:       EnrichmentPending,
		SplitCandidates:  []SplitCandidate{},
		Context:          []string{},
		OpenQuestions:    []string{},
		RelatedTeams:     []string{},
		RelatedEmployees: []string{},
		Title:            fallbackTitle(raw),
		Raw:              raw,
	}
	if err := s.writeLocked(t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Store) Get(id string) (*Task, error) {
	if !safeID(id) {
		return nil, os.ErrNotExist
	}
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		return nil, err
	}
	var t Task
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) List() ([]*Task, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var tasks []*Task
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".yaml")
		t, err := s.Get(id)
		if err != nil {
			continue
		}
		tasks = append(tasks, t)
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
	})
	return tasks, nil
}

func (s *Store) Save(t *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t.UpdatedAt = time.Now().UTC()
	return s.writeLocked(t)
}

func (s *Store) writeLocked(t *Task) error {
	if t.SplitCandidates == nil {
		t.SplitCandidates = []SplitCandidate{}
	}
	if t.Context == nil {
		t.Context = []string{}
	}
	if t.OpenQuestions == nil {
		t.OpenQuestions = []string{}
	}
	if t.RelatedTeams == nil {
		t.RelatedTeams = []string{}
	}
	if t.RelatedEmployees == nil {
		t.RelatedEmployees = []string{}
	}
	data, err := yaml.Marshal(t)
	if err != nil {
		return err
	}
	path := s.path(t.ID)
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) path(id string) string {
	return filepath.Join(s.dir, id+".yaml")
}

func (s *Store) uniqueID(raw string, now time.Time) string {
	base := now.Format("20060102T150405Z") + "_" + slug(raw)
	id := base
	for n := 2; fileExists(s.path(id)); n++ {
		id = fmt.Sprintf("%s_%d", base, n)
	}
	return id
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slug(raw string) string {
	line := firstLine(raw)
	var b strings.Builder
	for _, r := range strings.ToLower(line) {
		if unicode.IsLetter(r) && r > unicode.MaxASCII {
			continue
		}
		b.WriteRune(r)
	}
	s := nonSlug.ReplaceAllString(b.String(), "-")
	s = strings.Trim(s, "-")
	if len(s) > 40 {
		s = strings.Trim(s[:40], "-")
	}
	if s == "" {
		return "task"
	}
	return s
}

func fallbackTitle(raw string) string {
	line := firstLine(raw)
	if line == "" {
		return "Untitled assignment"
	}
	if len(line) > 80 {
		return strings.TrimSpace(line[:80]) + "…"
	}
	return line
}

func firstLine(raw string) string {
	line, _, _ := strings.Cut(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	return strings.TrimSpace(line)
}

func safeID(id string) bool {
	if id == "" || strings.Contains(id, "/") || strings.Contains(id, "..") {
		return false
	}
	for _, r := range id {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}
