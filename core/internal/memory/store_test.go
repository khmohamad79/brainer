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
	if task.Enrichment != EnrichmentPending {
		t.Fatalf("enrichment: %s", task.Enrichment)
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
	if got.Title == "" {
		t.Fatal("expected fallback title")
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
