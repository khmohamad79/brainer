package fill

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"brainer/internal/gpt"
	"brainer/internal/memory"
)

const systemPrompt = `You extract a few fields from an unstructured work assignment.

You are also given a company roster (teams and employees with ids, names, nicknames)
and a story roster (stories with ids, titles, optional jira keys).
Link the assignment to roster entries that are named or clearly implied in the text.

Return a JSON object with exactly these keys:
- requester: who asked, or empty string (free text; do not invent a roster id for this)
- open_questions: array of things missing to act
- related_teams: array of team ids from the roster that the text relates to, or []
- related_employees: array of employee ids from the roster that the text relates to, or []
- story_id: at most one story id from the story roster, or empty string

Rules:
- Do not invent facts.
- Write requester and open_questions in the same language as the assignment text. If the assignment is Persian, respond in Persian; if English, respond in English; match mixed language the same way.
- related_teams and related_employees must use only ids listed in the roster. Prefer nickname matches. If nothing matches, use [].
- story_id must be a story id from the story roster, matching title or jira key when clearly implied. If none or unclear, use "".
- Related means mentions or clear implication only — not an assignee role.
- Output JSON only.`

type result struct {
	Requester        string   `json:"requester"`
	OpenQuestions    []string `json:"open_questions"`
	RelatedTeams     []string `json:"related_teams"`
	RelatedEmployees []string `json:"related_employees"`
	StoryID          string   `json:"story_id"`
}

type Runner struct {
	GPT   *gpt.Client
	Store *memory.Store
}

func (r *Runner) Fill(task *memory.Task) {
	rawID := task.ID
	rawBody := task.Raw

	latest, getErr := r.Store.Get(rawID)
	if getErr != nil {
		return
	}
	latest.Fill = memory.FillPending
	latest.FillError = ""
	if err := r.Store.Save(latest); err != nil {
		log.Printf("fill mark pending %s: %v", rawID, err)
		return
	}

	org, orgErr := r.Store.Org()
	if orgErr != nil {
		log.Printf("fill org %s: %v", rawID, orgErr)
		org = &memory.Org{Teams: []memory.Team{}}
	}
	stories, storiesErr := r.Store.Stories()
	if storiesErr != nil {
		log.Printf("fill stories %s: %v", rawID, storiesErr)
		stories = &memory.StoryList{Stories: []memory.Story{}}
	}
	userMsg := buildUserMessage(rawBody, org, stories)

	content, err := r.GPT.ChatJSON(systemPrompt, userMsg)
	latest, getErr = r.Store.Get(rawID)
	if getErr != nil {
		return
	}
	if latest.Raw != rawBody {
		latest.Raw = rawBody
	}
	if err != nil {
		log.Printf("fill failed %s: %v", rawID, err)
		latest.Fill = memory.FillFailed
		latest.FillError = err.Error()
		_ = r.Store.Save(latest)
		return
	}
	parsed, parseErr := parseResult(content)
	if parseErr != nil {
		log.Printf("fill parse %s: %v", rawID, parseErr)
		latest.Fill = memory.FillFailed
		latest.FillError = parseErr.Error()
		_ = r.Store.Save(latest)
		return
	}
	apply(latest, parsed, org, stories)
	latest.Fill = memory.FillOK
	latest.FillError = ""
	_ = r.Store.Save(latest)
}

func buildUserMessage(raw string, org *memory.Org, stories *memory.StoryList) string {
	var b strings.Builder
	b.WriteString("Assignment:\n")
	b.WriteString(raw)
	b.WriteString("\n\nCompany roster (use only these ids for related_teams / related_employees):\n")
	if org == nil || len(org.Teams) == 0 {
		b.WriteString("(empty — leave related_teams and related_employees as [])\n")
	} else {
		for _, t := range org.Teams {
			b.WriteString(fmt.Sprintf("- team id=%s name=%q nicknames=%s\n", t.ID, t.Name, formatNicks(t.Nicknames)))
			for _, e := range t.Employees {
				b.WriteString(fmt.Sprintf("  - employee id=%s team=%s name=%q nicknames=%s\n", e.ID, t.ID, e.Name, formatNicks(e.Nicknames)))
			}
		}
	}
	b.WriteString("\nStory roster (use only these ids for story_id):\n")
	if stories == nil || len(stories.Stories) == 0 {
		b.WriteString("(empty — leave story_id as \"\")\n")
		return b.String()
	}
	for _, st := range stories.Stories {
		key := st.JiraKey
		if key == "" {
			key = "(none)"
		}
		b.WriteString(fmt.Sprintf("- story id=%s title=%q jira_key=%s\n", st.ID, st.Title, key))
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
		return result{}, fmt.Errorf("fill json: %w", err)
	}
	return parsed, nil
}

func apply(t *memory.Task, r result, org *memory.Org, stories *memory.StoryList) {
	t.Requester = strings.TrimSpace(r.Requester)
	t.OpenQuestions = r.OpenQuestions
	teams, emps := filterRelated(r.RelatedTeams, r.RelatedEmployees, org)
	t.RelatedTeams = teams
	t.RelatedEmployees = emps
	t.StoryID = keepKnownStory(r.StoryID, stories)
}

func keepKnownStory(id string, stories *memory.StoryList) string {
	id = strings.TrimSpace(id)
	if id == "" || stories == nil {
		return ""
	}
	for _, st := range stories.Stories {
		if st.ID == id {
			return id
		}
	}
	return ""
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
