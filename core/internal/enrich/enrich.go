package enrich

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"brainer/internal/gpt"
	"brainer/internal/memory"
)

const systemPrompt = `You extract a structured work assignment from unstructured text.

You are also given a company roster (teams and employees with ids, names, nicknames).
Link the assignment to roster entries that are named or clearly implied in the text.

Return a JSON object with exactly these keys:
- title: short human label
- requester: who asked, or empty string (free text; do not invent a roster id for this)
- due_at: due date or phrase exactly as stated, or null if not stated. Never invent a calendar date.
- priority: only if stated, else null
- context: array of short strings (project, people, systems)
- open_questions: array of things missing to act
- needs_split: true if the message contains more than one distinct work item
- split_candidates: if needs_split, array of {title, excerpt}; otherwise []
- structured: 1-4 sentence summary of the assignment
- related_teams: array of team ids from the roster that the text relates to, or []
- related_employees: array of employee ids from the roster that the text relates to, or []

Rules:
- Do not invent facts.
- One capture stays one task; only mark split candidates.
- Keep title under 80 characters.
- related_teams and related_employees must use only ids listed in the roster. Prefer nickname matches. If nothing matches, use [].
- Related means mentions or clear implication only — not an assignee role.
- Output JSON only.`

type result struct {
	Title            string                  `json:"title"`
	Requester        string                  `json:"requester"`
	DueAt            *string                 `json:"due_at"`
	Priority         *string                 `json:"priority"`
	Context          []string                `json:"context"`
	OpenQuestions    []string                `json:"open_questions"`
	NeedsSplit       bool                    `json:"needs_split"`
	SplitCandidates  []memory.SplitCandidate `json:"split_candidates"`
	Structured       string                  `json:"structured"`
	RelatedTeams     []string                `json:"related_teams"`
	RelatedEmployees []string                `json:"related_employees"`
}

type Runner struct {
	GPT   *gpt.Client
	Store *memory.Store
}

func (r *Runner) Enrich(task *memory.Task) {
	rawID := task.ID
	rawBody := task.Raw

	latest, getErr := r.Store.Get(rawID)
	if getErr != nil {
		return
	}
	latest.Enrichment = memory.EnrichmentPending
	latest.EnrichmentError = ""
	if err := r.Store.Save(latest); err != nil {
		log.Printf("enrich mark pending %s: %v", rawID, err)
		return
	}

	org, orgErr := r.Store.Org()
	if orgErr != nil {
		log.Printf("enrich org %s: %v", rawID, orgErr)
		org = &memory.Org{Teams: []memory.Team{}}
	}
	userMsg := buildUserMessage(rawBody, org)

	content, err := r.GPT.ChatJSON(systemPrompt, userMsg)
	latest, getErr = r.Store.Get(rawID)
	if getErr != nil {
		return
	}
	if latest.Raw != rawBody {
		latest.Raw = rawBody
	}
	if err != nil {
		log.Printf("enrich failed %s: %v", rawID, err)
		latest.Enrichment = memory.EnrichmentFailed
		latest.EnrichmentError = err.Error()
		if strings.TrimSpace(latest.Title) == "" {
			latest.Title = task.Title
		}
		_ = r.Store.Save(latest)
		return
	}
	parsed, parseErr := parseResult(content)
	if parseErr != nil {
		log.Printf("enrich parse %s: %v", rawID, parseErr)
		latest.Enrichment = memory.EnrichmentFailed
		latest.EnrichmentError = parseErr.Error()
		_ = r.Store.Save(latest)
		return
	}
	apply(latest, parsed, org)
	latest.Enrichment = memory.EnrichmentOK
	latest.EnrichmentError = ""
	_ = r.Store.Save(latest)
}

func buildUserMessage(raw string, org *memory.Org) string {
	var b strings.Builder
	b.WriteString("Assignment:\n")
	b.WriteString(raw)
	b.WriteString("\n\nCompany roster (use only these ids for related_teams / related_employees):\n")
	if org == nil || len(org.Teams) == 0 {
		b.WriteString("(empty — leave related_teams and related_employees as [])\n")
		return b.String()
	}
	for _, t := range org.Teams {
		b.WriteString(fmt.Sprintf("- team id=%s name=%q nicknames=%s\n", t.ID, t.Name, formatNicks(t.Nicknames)))
		for _, e := range t.Employees {
			b.WriteString(fmt.Sprintf("  - employee id=%s team=%s name=%q nicknames=%s\n", e.ID, t.ID, e.Name, formatNicks(e.Nicknames)))
		}
	}
	return b.String()
}

func formatNicks(nicks []string) string {
	if len(nicks) == 0 {
		return "[]"
	}
	parts := make([]string, len(nicks))
	for i, n := range nicks {
		parts[i] = fmt.Sprintf("%q", n)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func parseResult(content string) (result, error) {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}
	var parsed result
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return result{}, fmt.Errorf("enrich json: %w", err)
	}
	return parsed, nil
}

func apply(t *memory.Task, r result, org *memory.Org) {
	if strings.TrimSpace(r.Title) != "" {
		t.Title = strings.TrimSpace(r.Title)
	}
	t.Requester = strings.TrimSpace(r.Requester)
	t.DueAt = emptyToNil(r.DueAt)
	t.Priority = emptyToNil(r.Priority)
	t.Context = r.Context
	t.OpenQuestions = r.OpenQuestions
	t.NeedsSplit = r.NeedsSplit
	t.SplitCandidates = r.SplitCandidates
	t.Structured = strings.TrimSpace(r.Structured)
	if !t.NeedsSplit {
		t.SplitCandidates = []memory.SplitCandidate{}
	}
	teams, emps := filterRelated(r.RelatedTeams, r.RelatedEmployees, org)
	t.RelatedTeams = teams
	t.RelatedEmployees = emps
}

func filterRelated(teamIDs, empIDs []string, org *memory.Org) ([]string, []string) {
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
	return keepKnown(teamIDs, knownTeams), keepKnown(empIDs, knownEmps)
}

func keepKnown(ids []string, known map[string]struct{}) []string {
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

func emptyToNil(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" || v == "null" {
		return nil
	}
	return &v
}
