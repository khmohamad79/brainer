package fill

import (
	"strings"
	"testing"

	"brainer/internal/memory"
)

func TestFilterRelatedDropsUnknown(t *testing.T) {
	org := &memory.Org{
		Teams: []memory.Team{
			{
				ID:   "platform",
				Name: "Platform",
				Employees: []memory.Employee{
					{ID: "sara", Name: "Sara"},
				},
			},
		},
	}
	teams, emps := filterRelated(
		[]string{"platform", "ghost", "platform", " "},
		[]string{"sara", "bob", "sara"},
		org,
	)
	if len(teams) != 1 || teams[0] != "platform" {
		t.Fatalf("teams: %+v", teams)
	}
	if len(emps) != 1 || emps[0] != "sara" {
		t.Fatalf("emps: %+v", emps)
	}
}

func TestFilterRelatedEmptyOrg(t *testing.T) {
	teams, emps := filterRelated([]string{"platform"}, []string{"sara"}, &memory.Org{})
	if len(teams) != 0 || len(emps) != 0 {
		t.Fatalf("expected empty, got teams=%+v emps=%+v", teams, emps)
	}
	teams, emps = filterRelated([]string{"platform"}, []string{"sara"}, nil)
	if len(teams) != 0 || len(emps) != 0 {
		t.Fatalf("nil org: teams=%+v emps=%+v", teams, emps)
	}
}

func TestBuildUserMessageIncludesRoster(t *testing.T) {
	org := &memory.Org{
		Teams: []memory.Team{
			{
				ID:        "platform",
				Name:      "Platform",
				Nicknames: []string{"plat"},
				Employees: []memory.Employee{
					{ID: "sara", Name: "Sara", Nicknames: []string{"sari"}},
				},
			},
		},
	}
	stories := &memory.StoryList{
		Stories: []memory.Story{
			{ID: "payment-timeout", Title: "Payment timeout", JiraKey: "DEVPR-5982"},
		},
	}
	msg := buildUserMessage("Talk to Sara about plat login DEVPR-5982", org, stories)
	for _, want := range []string{"id=platform", "id=sara", "plat", "sari", "Talk to Sara", "id=payment-timeout", "DEVPR-5982"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing %q in:\n%s", want, msg)
		}
	}
}

func TestBuildUserMessageEmptyRoster(t *testing.T) {
	msg := buildUserMessage("Fix login", &memory.Org{}, &memory.StoryList{})
	if !strings.Contains(msg, "empty") {
		t.Fatalf("expected empty roster note: %s", msg)
	}
	if !strings.Contains(msg, `leave story_id as ""`) {
		t.Fatalf("expected empty story note: %s", msg)
	}
}

func TestApplySetsRelated(t *testing.T) {
	org := &memory.Org{
		Teams: []memory.Team{
			{ID: "platform", Name: "Platform", Employees: []memory.Employee{{ID: "sara", Name: "Sara"}}},
		},
	}
	stories := &memory.StoryList{
		Stories: []memory.Story{{ID: "payment-timeout", Title: "Payment timeout"}},
	}
	task := &memory.Task{}
	apply(task, result{
		Requester:        "Sara",
		RelatedTeams:     []string{"platform", "nope"},
		RelatedEmployees: []string{"sara"},
		StoryID:          "payment-timeout",
	}, org, stories)
	if task.Requester != "Sara" {
		t.Fatalf("requester: %s", task.Requester)
	}
	if len(task.RelatedTeams) != 1 || task.RelatedTeams[0] != "platform" {
		t.Fatalf("teams: %+v", task.RelatedTeams)
	}
	if len(task.RelatedEmployees) != 1 || task.RelatedEmployees[0] != "sara" {
		t.Fatalf("emps: %+v", task.RelatedEmployees)
	}
	if task.StoryID != "payment-timeout" {
		t.Fatalf("story_id: %q", task.StoryID)
	}
}

func TestApplyDropsUnknownStory(t *testing.T) {
	task := &memory.Task{}
	apply(task, result{StoryID: "ghost"}, &memory.Org{}, &memory.StoryList{
		Stories: []memory.Story{{ID: "payment-timeout", Title: "Payment timeout"}},
	})
	if task.StoryID != "" {
		t.Fatalf("expected empty story_id, got %q", task.StoryID)
	}
}
