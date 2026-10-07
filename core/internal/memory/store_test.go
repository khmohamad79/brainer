package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCaptureWritesRawFirst(t *testing.T) {
	dir := t.TempDir()
	store, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	raw := "Fix login bug by Friday, talk to Sara.\nAlso maybe README."
	task, err := store.Capture(raw)
	if err != nil {
		t.Fatal(err)
	}
	if task.Fill != FillPending {
		t.Fatalf("fill: %s", task.Fill)
	}
	if !strings.Contains(task.ID, "T") || strings.Contains(task.ID, "fix") {
		t.Fatalf("id should be timestamp only: %s", task.ID)
	}
	if task.Raw != raw {
		t.Fatalf("raw mismatch")
	}
	path := filepath.Join(dir, "tasks", task.ID+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Fix login bug") {
		t.Fatalf("file missing raw: %s", data)
	}
	got, err := store.Get(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Raw != raw {
		t.Fatal("expected raw preserved on get")
	}
}

func TestArchiveAndDelete(t *testing.T) {
	dir := t.TempDir()
	store, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	task, err := store.Capture("Archive me later")
	if err != nil {
		t.Fatal(err)
	}
	inbox, err := store.List(false)
	if err != nil || len(inbox) != 1 {
		t.Fatalf("inbox: %+v %v", inbox, err)
	}
	if err := store.Delete(task.ID); err == nil {
		t.Fatal("expected delete of non-archived to fail")
	}
	got, err := store.Archive(task.ID)
	if err != nil || !got.Archived || got.Status != StatusInbox {
		t.Fatalf("archive: %+v %v", got, err)
	}
	inbox, _ = store.List(false)
	arch, _ := store.List(true)
	if len(inbox) != 0 || len(arch) != 1 {
		t.Fatalf("lists inbox=%d arch=%d", len(inbox), len(arch))
	}
	if err := store.Delete(task.ID); err != nil {
		t.Fatal(err)
	}
	arch, _ = store.List(true)
	if len(arch) != 0 {
		t.Fatalf("expected empty archive: %+v", arch)
	}
}

func TestCaptureRejectsEmpty(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Capture("  \n"); err == nil {
		t.Fatal("expected error")
	}
}

func TestOrgTeamsAndEmployees(t *testing.T) {
	dir := t.TempDir()
	store, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	team, err := store.AddTeam(" Platform ", []string{" plat ", "infra", "plat"})
	if err != nil {
		t.Fatal(err)
	}
	if team.ID != "platform" || team.Name != "Platform" {
		t.Fatalf("team: %+v", team)
	}
	if len(team.Nicknames) != 2 || team.Nicknames[0] != "plat" || team.Nicknames[1] != "infra" {
		t.Fatalf("team nicknames: %+v", team.Nicknames)
	}
	emp, err := store.AddEmployee(team.ID, "Sara", []string{"Sari"})
	if err != nil {
		t.Fatal(err)
	}
	if emp.Name != "Sara" || len(emp.Nicknames) != 1 || emp.Nicknames[0] != "Sari" {
		t.Fatalf("employee: %+v", emp)
	}
	nicks := []string{"S"}
	if _, err := store.UpdateEmployee(team.ID, emp.ID, nil, &nicks); err != nil {
		t.Fatal(err)
	}
	platform := "Platform Engineering"
	team, err = store.UpdateTeam(team.ID, &platform, nil)
	if err != nil {
		t.Fatal(err)
	}
	if team.ID != "platform" || team.Name != "Platform Engineering" {
		t.Fatalf("rename team: %+v", team)
	}
	sarah := "Sarah"
	emp, err = store.UpdateEmployee(team.ID, emp.ID, &sarah, nil)
	if err != nil {
		t.Fatal(err)
	}
	if emp.ID != "sara" || emp.Name != "Sarah" {
		t.Fatalf("rename employee: %+v", emp)
	}
	data, err := os.ReadFile(filepath.Join(dir, "org.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Sarah") || !strings.Contains(string(data), "Platform Engineering") || !strings.Contains(string(data), "S") {
		t.Fatalf("org.yaml missing names: %s", data)
	}
	org, err := store.Org()
	if err != nil {
		t.Fatal(err)
	}
	if len(org.Teams) != 1 || len(org.Teams[0].Employees) != 1 {
		t.Fatalf("org: %+v", org)
	}
	if err := store.DeleteEmployee(team.ID, emp.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteTeam(team.ID); err != nil {
		t.Fatal(err)
	}
	org, err = store.Org()
	if err != nil {
		t.Fatal(err)
	}
	if len(org.Teams) != 0 {
		t.Fatalf("expected empty org: %+v", org)
	}
}

func TestStoriesCRUD(t *testing.T) {
	dir := t.TempDir()
	store, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	story, err := store.AddStory(" Payment timeout ", "devpr-5982", "")
	if err != nil {
		t.Fatal(err)
	}
	if story.ID != "payment-timeout" || story.Title != "Payment timeout" || story.JiraKey != "DEVPR-5982" {
		t.Fatalf("story: %+v", story)
	}
	title := "Payment timeouts"
	summary := "Timeouts on checkout"
	story, err = store.UpdateStory(story.ID, &title, nil, &summary)
	if err != nil {
		t.Fatal(err)
	}
	if story.ID != "payment-timeout" || story.Title != "Payment timeouts" || story.Summary != summary {
		t.Fatalf("update: %+v", story)
	}
	key := "DEVPR-6000"
	story, err = store.UpdateStory(story.ID, nil, &key, nil)
	if err != nil {
		t.Fatal(err)
	}
	if story.JiraKey != "DEVPR-6000" {
		t.Fatalf("jira key: %+v", story)
	}
	list, err := store.Stories()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Stories) != 1 {
		t.Fatalf("list: %+v", list)
	}
	task, err := store.Capture("work on payment timeout DEVPR-6000")
	if err != nil {
		t.Fatal(err)
	}
	task.StoryID = story.ID
	if err := store.Save(task); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.StoryID != story.ID {
		t.Fatalf("story_id: %q", got.StoryID)
	}
	if err := store.DeleteStory(story.ID); err != nil {
		t.Fatal(err)
	}
	list, err = store.Stories()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Stories) != 0 {
		t.Fatalf("expected empty stories: %+v", list)
	}
}
