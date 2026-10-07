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
	StatusInbox = "inbox"
	FillPending = "pending"
	FillOK      = "ok"
	FillFailed  = "failed"
	SourceWeb   = "web"
)

type Task struct {
	ID               string    `yaml:"id" json:"id"`
	Status           string    `yaml:"status" json:"status"`
	Archived         bool      `yaml:"archived" json:"archived"`
	CreatedAt        time.Time `yaml:"created_at" json:"created_at"`
	UpdatedAt        time.Time `yaml:"updated_at" json:"updated_at"`
	Source           string    `yaml:"source" json:"source"`
	Fill             string    `yaml:"fill" json:"fill"`
	FillError        string    `yaml:"fill_error,omitempty" json:"fill_error,omitempty"`
	Requester        string    `yaml:"requester" json:"requester"`
	OpenQuestions    []string  `yaml:"open_questions" json:"open_questions"`
	RelatedTeams     []string  `yaml:"related_teams" json:"related_teams"`
	RelatedEmployees []string  `yaml:"related_employees" json:"related_employees"`
	StoryID          string    `yaml:"story_id" json:"story_id"`
	Raw              string    `yaml:"raw" json:"raw"`
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
	id := s.uniqueID(now)
	t := &Task{
		ID:               id,
		Status:           StatusInbox,
		CreatedAt:        now,
		UpdatedAt:        now,
		Source:           SourceWeb,
		Fill:             FillPending,
		OpenQuestions:    []string{},
		RelatedTeams:     []string{},
		RelatedEmployees: []string{},
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

func (s *Store) List(archived bool) ([]*Task, error) {
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
		if t.Archived != archived {
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

func (s *Store) Archive(id string) (*Task, error) {
	t, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	t.Archived = true
	if err := s.Save(t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Store) Delete(id string) error {
	if !safeID(id) {
		return os.ErrNotExist
	}
	t, err := s.Get(id)
	if err != nil {
		return err
	}
	if !t.Archived {
		return fmt.Errorf("delete only allowed for archived tasks")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.Remove(s.path(id))
}

func (s *Store) writeLocked(t *Task) error {
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

func (s *Store) uniqueID(now time.Time) string {
	base := now.Format("20060102T150405Z")
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
