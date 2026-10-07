package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Story struct {
	ID      string `yaml:"id" json:"id"`
	Title   string `yaml:"title" json:"title"`
	JiraKey string `yaml:"jira_key,omitempty" json:"jira_key,omitempty"`
	Summary string `yaml:"summary" json:"summary"`
}

type StoryList struct {
	Stories []Story `yaml:"stories" json:"stories"`
}

func (s *Store) storiesPath() string {
	return filepath.Join(s.root, "stories.yaml")
}

func (s *Store) Stories() (*StoryList, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.storiesLocked()
}

func (s *Store) AddStory(title, jiraKey, summary string) (*Story, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("title is empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.storiesLocked()
	if err != nil {
		return nil, err
	}
	id := uniqueSlug(title, "story", func(id string) bool {
		for _, st := range list.Stories {
			if st.ID == id {
				return true
			}
		}
		return false
	})
	story := Story{
		ID:      id,
		Title:   title,
		JiraKey: normalizeJiraKey(jiraKey),
		Summary: summary,
	}
	list.Stories = append(list.Stories, story)
	if err := s.writeStoriesLocked(list); err != nil {
		return nil, err
	}
	return &story, nil
}

func (s *Store) DeleteStory(id string) error {
	if !safeID(id) {
		return os.ErrNotExist
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.storiesLocked()
	if err != nil {
		return err
	}
	next := make([]Story, 0, len(list.Stories))
	found := false
	for _, st := range list.Stories {
		if st.ID == id {
			found = true
			continue
		}
		next = append(next, st)
	}
	if !found {
		return os.ErrNotExist
	}
	list.Stories = next
	return s.writeStoriesLocked(list)
}

func (s *Store) UpdateStory(id string, title, jiraKey, summary *string) (*Story, error) {
	if !safeID(id) {
		return nil, os.ErrNotExist
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.storiesLocked()
	if err != nil {
		return nil, err
	}
	i := storyIndex(list, id)
	if i < 0 {
		return nil, os.ErrNotExist
	}
	st := &list.Stories[i]
	if title != nil {
		n := strings.TrimSpace(*title)
		if n == "" {
			return nil, fmt.Errorf("title is empty")
		}
		st.Title = n
	}
	if jiraKey != nil {
		st.JiraKey = normalizeJiraKey(*jiraKey)
	}
	if summary != nil {
		st.Summary = *summary
	}
	if err := s.writeStoriesLocked(list); err != nil {
		return nil, err
	}
	out := list.Stories[i]
	return &out, nil
}

func storyIndex(list *StoryList, id string) int {
	for i, st := range list.Stories {
		if st.ID == id {
			return i
		}
	}
	return -1
}

func normalizeJiraKey(key string) string {
	return strings.ToUpper(strings.TrimSpace(key))
}

func (s *Store) storiesLocked() (*StoryList, error) {
	data, err := os.ReadFile(s.storiesPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &StoryList{Stories: []Story{}}, nil
		}
		return nil, err
	}
	var list StoryList
	if err := yaml.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	if list.Stories == nil {
		list.Stories = []Story{}
	}
	for i := range list.Stories {
		list.Stories[i].JiraKey = normalizeJiraKey(list.Stories[i].JiraKey)
	}
	return &list, nil
}

func (s *Store) writeStoriesLocked(list *StoryList) error {
	if list.Stories == nil {
		list.Stories = []Story{}
	}
	for i := range list.Stories {
		list.Stories[i].JiraKey = normalizeJiraKey(list.Stories[i].JiraKey)
	}
	data, err := yaml.Marshal(list)
	if err != nil {
		return err
	}
	path := s.storiesPath()
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
