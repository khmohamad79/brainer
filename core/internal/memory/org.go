package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Employee struct {
	ID        string   `yaml:"id" json:"id"`
	Name      string   `yaml:"name" json:"name"`
	Nicknames []string `yaml:"nicknames" json:"nicknames"`
}

type Team struct {
	ID        string     `yaml:"id" json:"id"`
	Name      string     `yaml:"name" json:"name"`
	Nicknames []string   `yaml:"nicknames" json:"nicknames"`
	Employees []Employee `yaml:"employees" json:"employees"`
}

type Org struct {
	Teams []Team `yaml:"teams" json:"teams"`
}

func (s *Store) orgPath() string {
	return filepath.Join(s.root, "org.yaml")
}

func (s *Store) Org() (*Org, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.orgLocked()
}

func (s *Store) AddTeam(name string, nicknames []string) (*Team, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name is empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	org, err := s.orgLocked()
	if err != nil {
		return nil, err
	}
	id := uniqueSlug(name, "team", func(id string) bool {
		for _, t := range org.Teams {
			if t.ID == id {
				return true
			}
		}
		return false
	})
	team := Team{ID: id, Name: name, Nicknames: cleanNicknames(nicknames), Employees: []Employee{}}
	org.Teams = append(org.Teams, team)
	if err := s.writeOrgLocked(org); err != nil {
		return nil, err
	}
	return &team, nil
}

func (s *Store) DeleteTeam(id string) error {
	if !safeID(id) {
		return os.ErrNotExist
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	org, err := s.orgLocked()
	if err != nil {
		return err
	}
	next := make([]Team, 0, len(org.Teams))
	found := false
	for _, t := range org.Teams {
		if t.ID == id {
			found = true
			continue
		}
		next = append(next, t)
	}
	if !found {
		return os.ErrNotExist
	}
	org.Teams = next
	return s.writeOrgLocked(org)
}

func (s *Store) UpdateTeam(id string, name *string, nicknames *[]string) (*Team, error) {
	if !safeID(id) {
		return nil, os.ErrNotExist
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	org, err := s.orgLocked()
	if err != nil {
		return nil, err
	}
	i := teamIndex(org, id)
	if i < 0 {
		return nil, os.ErrNotExist
	}
	if err := applyRosterEdit(&org.Teams[i].Name, &org.Teams[i].Nicknames, name, nicknames); err != nil {
		return nil, err
	}
	if err := s.writeOrgLocked(org); err != nil {
		return nil, err
	}
	t := org.Teams[i]
	return &t, nil
}

func (s *Store) AddEmployee(teamID, name string, nicknames []string) (*Employee, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name is empty")
	}
	if !safeID(teamID) {
		return nil, os.ErrNotExist
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	org, err := s.orgLocked()
	if err != nil {
		return nil, err
	}
	i := teamIndex(org, teamID)
	if i < 0 {
		return nil, os.ErrNotExist
	}
	id := uniqueSlug(name, "person", func(id string) bool {
		for _, e := range org.Teams[i].Employees {
			if e.ID == id {
				return true
			}
		}
		return false
	})
	emp := Employee{ID: id, Name: name, Nicknames: cleanNicknames(nicknames)}
	org.Teams[i].Employees = append(org.Teams[i].Employees, emp)
	if err := s.writeOrgLocked(org); err != nil {
		return nil, err
	}
	return &emp, nil
}

func (s *Store) DeleteEmployee(teamID, empID string) error {
	if !safeID(teamID) || !safeID(empID) {
		return os.ErrNotExist
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	org, err := s.orgLocked()
	if err != nil {
		return err
	}
	i := teamIndex(org, teamID)
	if i < 0 {
		return os.ErrNotExist
	}
	emps := org.Teams[i].Employees
	next := make([]Employee, 0, len(emps))
	found := false
	for _, e := range emps {
		if e.ID == empID {
			found = true
			continue
		}
		next = append(next, e)
	}
	if !found {
		return os.ErrNotExist
	}
	org.Teams[i].Employees = next
	return s.writeOrgLocked(org)
}

func (s *Store) UpdateEmployee(teamID, empID string, name *string, nicknames *[]string) (*Employee, error) {
	if !safeID(teamID) || !safeID(empID) {
		return nil, os.ErrNotExist
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	org, err := s.orgLocked()
	if err != nil {
		return nil, err
	}
	i := teamIndex(org, teamID)
	if i < 0 {
		return nil, os.ErrNotExist
	}
	for j := range org.Teams[i].Employees {
		if org.Teams[i].Employees[j].ID == empID {
			e := &org.Teams[i].Employees[j]
			if err := applyRosterEdit(&e.Name, &e.Nicknames, name, nicknames); err != nil {
				return nil, err
			}
			if err := s.writeOrgLocked(org); err != nil {
				return nil, err
			}
			out := org.Teams[i].Employees[j]
			return &out, nil
		}
	}
	return nil, os.ErrNotExist
}

func applyRosterEdit(dstName *string, dstNicks *[]string, name *string, nicknames *[]string) error {
	if name != nil {
		n := strings.TrimSpace(*name)
		if n == "" {
			return fmt.Errorf("name is empty")
		}
		*dstName = n
	}
	if nicknames != nil {
		*dstNicks = cleanNicknames(*nicknames)
	}
	return nil
}

func teamIndex(org *Org, id string) int {
	for i, t := range org.Teams {
		if t.ID == id {
			return i
		}
	}
	return -1
}

func uniqueSlug(name, fallback string, taken func(string) bool) string {
	base := slug(name)
	if base == "task" {
		base = fallback
	}
	id := base
	for n := 2; taken(id); n++ {
		id = fmt.Sprintf("%s_%d", base, n)
	}
	return id
}

func (s *Store) orgLocked() (*Org, error) {
	data, err := os.ReadFile(s.orgPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &Org{Teams: []Team{}}, nil
		}
		return nil, err
	}
	var org Org
	if err := yaml.Unmarshal(data, &org); err != nil {
		return nil, err
	}
	if org.Teams == nil {
		org.Teams = []Team{}
	}
	for i := range org.Teams {
		if org.Teams[i].Employees == nil {
			org.Teams[i].Employees = []Employee{}
		}
		org.Teams[i].Nicknames = cleanNicknames(org.Teams[i].Nicknames)
		for j := range org.Teams[i].Employees {
			org.Teams[i].Employees[j].Nicknames = cleanNicknames(org.Teams[i].Employees[j].Nicknames)
		}
	}
	return &org, nil
}

func (s *Store) writeOrgLocked(org *Org) error {
	if org.Teams == nil {
		org.Teams = []Team{}
	}
	for i := range org.Teams {
		if org.Teams[i].Employees == nil {
			org.Teams[i].Employees = []Employee{}
		}
		org.Teams[i].Nicknames = cleanNicknames(org.Teams[i].Nicknames)
		for j := range org.Teams[i].Employees {
			org.Teams[i].Employees[j].Nicknames = cleanNicknames(org.Teams[i].Employees[j].Nicknames)
		}
	}
	data, err := yaml.Marshal(org)
	if err != nil {
		return err
	}
	path := s.orgPath()
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

func cleanNicknames(in []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, n := range in {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		key := strings.ToLower(n)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, n)
	}
	if out == nil {
		return []string{}
	}
	return out
}
