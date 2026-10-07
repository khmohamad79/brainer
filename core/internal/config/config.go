package config

import (
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	RepoRoot    string
	MemoryDir   string
	HTTPAddr    string
	GPTBaseURL  string
	GPTToken    string
	GPTModel    string
	JiraBaseURL string
	JiraToken   string
}

func Load() (Config, error) {
	root := discoverRoot()
	loadDotEnv(filepath.Join(root, ".env"))
	loadDotEnv(".env")

	cfg := Config{
		RepoRoot:    root,
		MemoryDir:   firstNonEmpty(os.Getenv("MEMORY_DIR"), filepath.Join(root, "memory")),
		HTTPAddr:    firstNonEmpty(os.Getenv("HTTP_ADDR"), ":8080"),
		GPTBaseURL:  firstNonEmpty(os.Getenv("GPT_BASE_URL"), "http://api.hooshyar.systemgroup.net/abramad/gpt/v1"),
		GPTToken:    os.Getenv("GPT_TOKEN"),
		GPTModel:    firstNonEmpty(os.Getenv("GPT_MODEL"), "/gpt-120"),
		JiraBaseURL: strings.TrimRight(os.Getenv("JIRA_BASE_URL"), "/"),
		JiraToken:   os.Getenv("JIRA_TOKEN"),
	}
	if !filepath.IsAbs(cfg.MemoryDir) {
		cfg.MemoryDir = filepath.Join(root, cfg.MemoryDir)
	}
	return cfg, nil
}

func discoverRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	dir := cwd
	for i := 0; i < 6; i++ {
		if isRepoRoot(dir) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return cwd
}

func isRepoRoot(dir string) bool {
	if st, err := os.Stat(filepath.Join(dir, "idea")); err == nil && st.IsDir() {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, "core", "go.mod")); err == nil {
		return true
	}
	return false
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range splitLines(string(data)) {
		if line == "" || line[0] == '#' {
			continue
		}
		key, val, ok := splitKV(line)
		if !ok {
			continue
		}
		// Project .env is the intended config surface; apply it.
		_ = os.Setenv(key, val)
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := trimSpace(s[start:i])
			if line != "" {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		line := trimSpace(s[start:])
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func splitKV(line string) (string, string, bool) {
	i := 0
	for i < len(line) && line[i] != '=' {
		i++
	}
	if i == 0 || i == len(line) {
		return "", "", false
	}
	key := trimSpace(line[:i])
	val := trimSpace(line[i+1:])
	if len(val) >= 2 {
		if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
			val = val[1 : len(val)-1]
		}
	}
	if key == "" {
		return "", "", false
	}
	return key, val, true
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
