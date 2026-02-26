package session

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	bRoleUser      = []byte(`"role":"user"`)
	bRoleUserS     = []byte(`"role": "user"`)
	bIsMeta        = []byte(`"isMeta":true`)
	bIsMetaSpaced  = []byte(`"isMeta": true`)
	bCwd           = []byte(`"cwd"`)
	bGitBranch     = []byte(`"gitBranch"`)
	bTodosNonEmpty = []byte(`"todos":[{`)
)

func ScanSessions() ([]Session, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	projectsDir := filepath.Join(home, ".claude", "projects")
	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		return nil, nil
	}

	type fileEntry struct {
		path    string
		modTime time.Time
		size    int64
	}
	var files []fileEntry
	err = filepath.Walk(projectsDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if info.IsDir() {
			if info.Name() == "subagents" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".jsonl") || strings.HasPrefix(info.Name(), "agent-") {
			return nil
		}
		files = append(files, fileEntry{path: path, modTime: info.ModTime(), size: info.Size()})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk projects dir: %w", err)
	}

	const numWorkers = 12
	fileCh := make(chan fileEntry, len(files))
	resultCh := make(chan Session, len(files))

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fe := range fileCh {
				sess := scanSessionStream(fe.path, fe.modTime, home)
				if sess.MsgCount > 0 {
					resultCh <- sess
				}
			}
		}()
	}

	for _, f := range files {
		fileCh <- f
	}
	close(fileCh)

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	sessions := make([]Session, 0, len(files))
	for sess := range resultCh {
		sessions = append(sessions, sess)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ModTime.After(sessions[j].ModTime)
	})

	return sessions, nil
}

// scanSessionStream uses buffered line-by-line scanning for large files to avoid OOM.
func scanSessionStream(path string, modTime time.Time, home string) Session {
	id := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	shortID := id
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}

	sess := Session{
		ID:       id,
		ShortID:  shortID,
		FilePath: path,
		ModTime:  modTime,
		IsLive:   time.Since(modTime) < 60*time.Second,
	}

	dir := filepath.Dir(path)
	dirName := filepath.Base(dir)
	sess.ProjectName = decodeDirName(dirName, home)

	f, err := os.Open(path)
	if err != nil {
		return sess
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 256*1024), 10*1024*1024)

	userMsgCount := 0
	gotPrompt := false

	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}

		isMeta := bytes.Contains(line, bIsMeta) || bytes.Contains(line, bIsMetaSpaced)
		if isMeta {
			cwd, branch := extractMetadataFast(line)
			if cwd != "" {
				sess.ProjectPath = cwd
				sess.ProjectName = shortenPath(cwd, home)
			}
			if branch != "" {
				sess.GitBranch = branch
			}
			continue
		}

		if bytes.Contains(line, bRoleUser) || bytes.Contains(line, bRoleUserS) {
			userMsgCount++
			if !gotPrompt {
				prompt, ts := extractFirstPromptFast(line)
				if !ts.IsZero() && sess.Created.IsZero() {
					sess.Created = ts
				}
				if prompt != "" {
					sess.FirstPrompt = prompt
					gotPrompt = true
				}
			}
		}

		// Track latest non-empty todos snapshot
		if bytes.Contains(line, bTodosNonEmpty) {
			if todos := extractTodos(line); len(todos) > 0 {
				sess.Todos = todos
			}
		}
	}

	sess.MsgCount = userMsgCount
	if sess.ProjectPath != "" {
		sess.IsWorktree = isGitWorktree(sess.ProjectPath)
		sess.HasMemory = hasProjectMemory(sess.ProjectPath, home)
	}
	return sess
}

func extractFirstPromptFast(line []byte) (prompt string, ts time.Time) {
	// Extract timestamp
	ts = extractTimestamp(line)

	// Try simple string content: "content":"text..."
	prompt = extractSimpleContent(line)
	if prompt != "" {
		return prompt, ts
	}

	// Array content: fall back to full parse
	if bytes.Contains(line, []byte(`"content":[`)) || bytes.Contains(line, []byte(`"content": [`)) {
		entry, parseErr := ParseEntry(string(line))
		if parseErr != nil {
			return "", ts
		}
		preview := EntryPreview(entry)
		if preview != "" && preview != "(no content)" && !isSystemPrompt(preview) {
			return preview, ts
		}
	}

	return "", ts
}

func extractTimestamp(line []byte) time.Time {
	markers := [][]byte{[]byte(`"timestamp":"`), []byte(`"timestamp": "`)}
	for _, marker := range markers {
		idx := bytes.Index(line, marker)
		if idx < 0 {
			continue
		}
		start := idx + len(marker)
		if start+40 > len(line) {
			continue
		}
		end := bytes.IndexByte(line[start:], '"')
		if end <= 0 || end > 40 {
			continue
		}
		tsStr := string(line[start : start+end])
		if t, err := time.Parse(time.RFC3339Nano, tsStr); err == nil {
			return t
		}
		if t, err := time.Parse(time.RFC3339, tsStr); err == nil {
			return t
		}
	}
	return time.Time{}
}

func extractSimpleContent(line []byte) string {
	markers := [][]byte{[]byte(`"content":"`), []byte(`"content": "`)}
	for _, marker := range markers {
		idx := bytes.Index(line, marker)
		if idx < 0 {
			continue
		}
		start := idx + len(marker)
		if start >= len(line) {
			continue
		}
		// Make sure this isn't array content
		// The marker already ends with `"` so we're inside the string value
		text := extractJSONString(line[start:])
		if text == "" {
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		text = strings.ReplaceAll(text, "\n", " ")
		if isSystemPrompt(text) {
			return ""
		}
		if len(text) > 100 {
			text = text[:97] + "..."
		}
		return text
	}
	return ""
}

func extractJSONString(b []byte) string {
	var buf []byte
	limit := min(len(b), 200)
	for i := 0; i < limit; i++ {
		if b[i] == '\\' && i+1 < limit {
			next := b[i+1]
			switch next {
			case '"':
				buf = append(buf, '"')
			case '\\':
				buf = append(buf, '\\')
			case 'n':
				buf = append(buf, '\n')
			case 't':
				buf = append(buf, '\t')
			case 'r':
				buf = append(buf, '\r')
			default:
				buf = append(buf, '\\', next)
			}
			i++
			continue
		}
		if b[i] == '"' {
			return string(buf)
		}
		buf = append(buf, b[i])
	}
	if len(buf) > 0 {
		return string(buf)
	}
	return ""
}

func extractMetadataFast(line []byte) (cwd, gitBranch string) {
	if idx := bytes.Index(line, bCwd); idx >= 0 {
		cwd = extractJSONFieldValue(line[idx+len(bCwd):])
	}
	if idx := bytes.Index(line, bGitBranch); idx >= 0 {
		gitBranch = extractJSONFieldValue(line[idx+len(bGitBranch):])
	}
	return cwd, gitBranch
}

func extractJSONFieldValue(b []byte) string {
	i := 0
	for i < len(b) && b[i] != ':' {
		i++
	}
	i++ // skip colon
	for i < len(b) && (b[i] == ' ' || b[i] == '\t') {
		i++
	}
	if i >= len(b) || b[i] != '"' {
		return ""
	}
	i++ // skip opening quote
	start := i
	for i < len(b) {
		if b[i] == '\\' {
			i += 2
			continue
		}
		if b[i] == '"' {
			return string(b[start:i])
		}
		i++
	}
	return ""
}

func extractTodos(line []byte) []TodoItem {
	idx := bytes.Index(line, bTodosNonEmpty)
	if idx < 0 {
		return nil
	}
	start := idx + 8 // skip `"todos":`
	depth := 0
	for i := start; i < len(line); i++ {
		if line[i] == '[' {
			depth++
		} else if line[i] == ']' {
			depth--
			if depth == 0 {
				var todos []TodoItem
				if json.Unmarshal(line[start:i+1], &todos) == nil {
					return todos
				}
				return nil
			}
		}
	}
	return nil
}

// EncodeProjectPath converts an absolute path to the Claude projects directory name.
// Claude replaces both '/' and '.' with '-'.
func EncodeProjectPath(path string) string {
	s := strings.ReplaceAll(path, "/", "-")
	s = strings.ReplaceAll(s, ".", "-")
	return s
}

func hasProjectMemory(projectPath, home string) bool {
	encoded := EncodeProjectPath(projectPath)
	memDir := filepath.Join(home, ".claude", "projects", encoded, "memory")
	entries, err := os.ReadDir(memDir)
	return err == nil && len(entries) > 0
}

func isGitWorktree(projectPath string) bool {
	gitPath := filepath.Join(projectPath, ".git")
	info, err := os.Lstat(gitPath)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func isSystemPrompt(s string) bool {
	prefixes := []string{"<command-", "[Request interrupted", "{\"type\""}
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func decodeDirName(dirName, home string) string {
	if !strings.HasPrefix(dirName, "-") {
		return dirName
	}
	decoded := strings.ReplaceAll(dirName, "-", "/")
	if strings.HasPrefix(decoded, "/Users/") {
		return shortenPath(decoded, home)
	}
	return decoded
}

func shortenPath(path, home string) string {
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

func LoadMessages(filePath string) ([]Entry, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	var entries []Entry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}

		entry, parseErr := ParseEntry(line)
		if parseErr != nil {
			continue
		}

		if entry.IsMeta || entry.Type == "progress" || entry.Type == "file-history-snapshot" {
			continue
		}
		if entry.Role == "user" || entry.Role == "assistant" {
			entries = append(entries, entry)
		}
	}
	return entries, sc.Err()
}

// LoadMessagesSummary loads only the first headN and last tailN messages from a
// session file, returning them along with the total message count. This avoids
// parsing the entire file for preview purposes.
func LoadMessagesSummary(filePath string, headN, tailN int) (head []Entry, tail []Entry, total int, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	ringIdx := 0

	// Keep raw lines for tail in a string ring buffer to defer parsing
	rawRing := make([]string, tailN)

	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}

		// Fast skip: meta and non-message lines without full JSON parse
		if bytes.Contains(line, bIsMeta) || bytes.Contains(line, bIsMetaSpaced) {
			continue
		}
		hasRole := bytes.Contains(line, bRoleUser) || bytes.Contains(line, bRoleUserS) ||
			bytes.Contains(line, []byte(`"role":"assistant"`)) || bytes.Contains(line, []byte(`"role": "assistant"`))
		if !hasRole {
			continue
		}

		total++

		// Fully parse only head entries
		if total <= headN {
			entry, parseErr := ParseEntry(string(line))
			if parseErr != nil {
				total--
				continue
			}
			head = append(head, entry)
		}

		// Store raw line for tail (cheap - no parsing)
		rawRing[ringIdx%tailN] = string(line)
		ringIdx++
	}

	if err := sc.Err(); err != nil {
		return nil, nil, 0, err
	}

	// Extract tail from ring buffer (avoid duplicating head entries)
	if total <= headN {
		return head, nil, total, nil
	}
	tailStart := max(total-tailN, headN)
	tailCount := total - tailStart
	tail = make([]Entry, 0, tailCount)
	for i := total - tailCount; i < total; i++ {
		raw := rawRing[i%tailN]
		if entry, parseErr := ParseEntry(raw); parseErr == nil {
			tail = append(tail, entry)
		}
	}
	return head, tail, total, nil
}

func FindSubagents(sessionFile string) ([]Subagent, error) {
	dir := filepath.Dir(sessionFile)
	sessID := strings.TrimSuffix(filepath.Base(sessionFile), ".jsonl")
	agentDir := filepath.Join(dir, sessID, "subagents")

	if _, err := os.Stat(agentDir); os.IsNotExist(err) {
		return nil, nil
	}

	matches, err := filepath.Glob(filepath.Join(agentDir, "agent-*.jsonl"))
	if err != nil {
		return nil, err
	}

	var agents []Subagent
	for _, p := range matches {
		agents = append(agents, scanSubagentFile(p))
	}

	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Timestamp.After(agents[j].Timestamp)
	})
	return agents, nil
}

func scanSubagentFile(path string) Subagent {
	name := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	id := strings.TrimPrefix(name, "agent-")
	shortID := id
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}

	agent := Subagent{ID: id, ShortID: shortID, FilePath: path}

	f, err := os.Open(path)
	if err != nil {
		return agent
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		entry, parseErr := ParseEntry(line)
		if parseErr != nil {
			continue
		}
		if entry.IsMeta {
			continue
		}
		if entry.Role == "user" || entry.Role == "assistant" {
			agent.MsgCount++
			if agent.Timestamp.IsZero() && !entry.Timestamp.IsZero() {
				agent.Timestamp = entry.Timestamp
			}
			if agent.FirstPrompt == "" && entry.Role == "user" {
				preview := EntryPreview(entry)
				if preview != "" && preview != "(no content)" {
					agent.FirstPrompt = preview
				}
			}
		}
	}
	return agent
}
