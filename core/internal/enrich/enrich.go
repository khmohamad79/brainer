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

Return a JSON object with exactly these keys:
- title: short human label
- requester: who asked, or empty string
- due_at: due date or phrase exactly as stated, or null if not stated. Never invent a calendar date.
- priority: only if stated, else null
- context: array of short strings (project, people, systems)
- open_questions: array of things missing to act
- needs_split: true if the message contains more than one distinct work item
- split_candidates: if needs_split, array of {title, excerpt}; otherwise []
- structured: 1-4 sentence summary of the assignment

Rules:
- Do not invent facts.
- One capture stays one task; only mark split candidates.
- Keep title under 80 characters.
- Output JSON only.`

type result struct {
	Title           string                  `json:"title"`
	Requester       string                  `json:"requester"`
	DueAt           *string                 `json:"due_at"`
	Priority        *string                 `json:"priority"`
	Context         []string                `json:"context"`
	OpenQuestions   []string                `json:"open_questions"`
	NeedsSplit      bool                    `json:"needs_split"`
	SplitCandidates []memory.SplitCandidate `json:"split_candidates"`
	Structured      string                  `json:"structured"`
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

	content, err := r.GPT.ChatJSON(systemPrompt, rawBody)
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
	apply(latest, parsed)
	latest.Enrichment = memory.EnrichmentOK
	latest.EnrichmentError = ""
	_ = r.Store.Save(latest)
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

func apply(t *memory.Task, r result) {
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
