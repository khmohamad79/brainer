package jira

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const maxSummaryLen = 4000

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		Token:   strings.TrimSpace(token),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Configured() bool {
	return c != nil && c.BaseURL != "" && c.Token != ""
}

var htmlTag = regexp.MustCompile(`(?s)<[^>]*>`)

func (c *Client) FetchIssueSummary(key string) (string, error) {
	if !c.Configured() {
		return "", fmt.Errorf("jira not configured (set JIRA_BASE_URL and JIRA_TOKEN)")
	}
	key = strings.ToUpper(strings.TrimSpace(key))
	if key == "" {
		return "", fmt.Errorf("jira key is empty")
	}
	u := fmt.Sprintf("%s/rest/api/2/issue/%s?fields=summary,status,description",
		c.BaseURL, url.PathEscape(key))
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return "", fmt.Errorf("jira %s: %s", resp.Status, msg)
	}
	var issue issueResponse
	if err := json.Unmarshal(body, &issue); err != nil {
		return "", err
	}
	return composeSummary(key, issue.Fields), nil
}

type issueResponse struct {
	Fields issueFields `json:"fields"`
}

type issueFields struct {
	Summary     string          `json:"summary"`
	Status      *statusField    `json:"status"`
	Description json.RawMessage `json:"description"`
}

type statusField struct {
	Name string `json:"name"`
}

func composeSummary(key string, f issueFields) string {
	var b strings.Builder
	b.WriteString(key)
	if f.Summary != "" {
		b.WriteString(": ")
		b.WriteString(strings.TrimSpace(f.Summary))
	}
	if f.Status != nil && f.Status.Name != "" {
		b.WriteString("\nStatus: ")
		b.WriteString(f.Status.Name)
	}
	desc := descriptionText(f.Description)
	if desc != "" {
		b.WriteString("\n\n")
		b.WriteString(desc)
	}
	out := strings.TrimSpace(b.String())
	if len(out) > maxSummaryLen {
		out = strings.TrimSpace(out[:maxSummaryLen]) + "…"
	}
	return out
}

func descriptionText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return cleanDescription(s)
	}
	// ADF or other structured description — extract string leaves.
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return ""
	}
	var parts []string
	collectText(v, &parts)
	return cleanDescription(strings.Join(parts, " "))
}

func collectText(v any, parts *[]string) {
	switch t := v.(type) {
	case string:
		if s := strings.TrimSpace(t); s != "" {
			*parts = append(*parts, s)
		}
	case []any:
		for _, item := range t {
			collectText(item, parts)
		}
	case map[string]any:
		if text, ok := t["text"].(string); ok {
			if s := strings.TrimSpace(text); s != "" {
				*parts = append(*parts, s)
			}
		}
		for k, child := range t {
			if k == "text" {
				continue
			}
			collectText(child, parts)
		}
	}
}

func cleanDescription(s string) string {
	s = htmlTag.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	var out []string
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		if line != "" {
			out = append(out, line)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
