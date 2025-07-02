package hashlog

import (
	"bufio"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type VerificationStatus struct {
	Verified  bool
	Error     error
	KeyDigest string
}

type SkillNode struct {
	ID        string
	InvokerID string
	Entries   []HashedLogEntry
	Children  []*SkillNode
}

type LogData struct {
	InvocationMap map[string][]HashedLogEntry
	InvokerMap    map[string]string
	SessionID     string
	TangentID     string
	TangentURL    string
}

// getEntryTime extracts the timestamp from a log entry
func getEntryTime(entry HashedLogEntry) time.Time {
	if rawTime, ok := entry.Payload["time"]; ok {
		switch v := rawTime.(type) {
		case float64:
			return time.UnixMilli(int64(v))
		case int64:
			return time.UnixMilli(v)
		case string:
			if parsed, err := time.Parse(time.RFC3339, v); err == nil {
				return parsed
			}
		}
	}
	return time.Time{}
}

// getNodeEarliestTime returns the earliest timestamp from all entries in a node
func getNodeEarliestTime(node *SkillNode) time.Time {
	if len(node.Entries) == 0 {
		return time.Time{}
	}
	earliest := getEntryTime(node.Entries[0])
	for _, entry := range node.Entries {
		if entryTime := getEntryTime(entry); !entryTime.IsZero() && (earliest.IsZero() || entryTime.Before(earliest)) {
			earliest = entryTime
		}
	}
	return earliest
}

// parseLogFile reads and parses the log file, returning structured data
func parseLogFile(absPath string) (*LogData, error) {
	f, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	invocationMap := make(map[string][]HashedLogEntry)
	invokerMap := make(map[string]string)
	firstSessionID := ""
	tangentID := ""
	tangentURL := ""

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to read log file: %w", err)
		}

		var entry HashedLogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		p := entry.Payload
		invID := str(p["invocation_id"])
		invokerID := str(p["invoker_id"])
		if invID == "" {
			invID = "__session__"
		}
		invocationMap[invID] = append(invocationMap[invID], entry)
		if _, ok := invokerMap[invID]; !ok {
			invokerMap[invID] = invokerID
		}
		if firstSessionID == "" {
			firstSessionID = str(p["session_id"])
		}
		if tangentID == "" {
			tangentID = str(p["tangent_id"])
		}
		if tangentURL == "" {
			tangentURL = str(p["tangent_url"])
		}
	}

	return &LogData{
		InvocationMap: invocationMap,
		InvokerMap:    invokerMap,
		SessionID:     firstSessionID,
		TangentID:     tangentID,
		TangentURL:    tangentURL,
	}, nil
}

// buildSkillTree creates the skill invocation tree from parsed log data
func buildSkillTree(logData *LogData) []*SkillNode {
	nodes := make(map[string]*SkillNode)
	var generalRoot *SkillNode

	// First pass: create all nodes
	for id, entries := range logData.InvocationMap {
		// Sort entries by time ascending
		sort.SliceStable(entries, func(i, j int) bool {
			timeI := getEntryTime(entries[i])
			timeJ := getEntryTime(entries[j])
			if timeI.IsZero() && timeJ.IsZero() {
				return false
			}
			if timeI.IsZero() {
				return false
			}
			if timeJ.IsZero() {
				return true
			}
			return timeI.Before(timeJ)
		})

		node := &SkillNode{
			ID:        id,
			InvokerID: logData.InvokerMap[id],
			Entries:   entries,
		}
		nodes[id] = node
	}

	// Second pass: attach children
	var unattached []*SkillNode
	for _, node := range nodes {
		if node.ID == "__session__" {
			generalRoot = node
			continue
		}
		if node.InvokerID == "" {
			if generalRoot != nil {
				generalRoot.Children = append(generalRoot.Children, node)
			} else {
				unattached = append(unattached, node)
			}
		} else {
			if parent, ok := nodes[node.InvokerID]; ok {
				parent.Children = append(parent.Children, node)
			} else {
				unattached = append(unattached, node)
			}
		}
	}

	// Third pass: build roots
	var roots []*SkillNode
	if generalRoot != nil {
		roots = append([]*SkillNode{generalRoot}, unattached...)
	} else {
		roots = unattached
	}

	// Sort roots deterministically
	sort.SliceStable(roots, func(i, j int) bool {
		if roots[i].ID == "__session__" {
			return true
		}
		if roots[j].ID == "__session__" {
			return false
		}
		return roots[i].ID < roots[j].ID
	})

	return roots
}

// writeHTMLHeader writes the HTML document header
func writeHTMLHeader(out io.Writer, logData *LogData, verificationStatus []VerificationStatus) {
	fmt.Fprint(out, `<html><head><meta charset="UTF-8"><title>Tansive™ Session Log</title><style>`)
	fmt.Fprint(out, cssStyle)
	fmt.Fprint(out, `</style></head><body>
<h1>Tansive™ Session Audit Log</h1>
<h2><strong>Session:</strong> `+html.EscapeString(logData.SessionID)+`</h2>
<h2><strong>Tangent ID:</strong> `+html.EscapeString(logData.TangentID)+`</h2>
<h2><strong>Tangent URL:</strong> `+html.EscapeString(logData.TangentURL)+`</h2>`)

	if len(verificationStatus) > 0 {
		fmt.Fprintf(out, `<h2><strong>Verification Status:</strong> %s</h2>`, html.EscapeString(str(verificationStatus[0].Verified)))
		fmt.Fprintf(out, `<h2><strong>Key Digest:</strong> %s</h2>`, html.EscapeString(verificationStatus[0].KeyDigest))
		if verificationStatus[0].Error != nil {
			fmt.Fprintf(out, `<h2><strong>Error:</strong> %s</h2>`, html.EscapeString(verificationStatus[0].Error.Error()))
		}
	}
	fmt.Fprint(out, `<br />`)
}

// renderSkillNode recursively renders a skill node and its children
func renderSkillNode(out io.Writer, node *SkillNode, depth int) {
	// Sort children by earliest time
	sort.SliceStable(node.Children, func(i, j int) bool {
		timeI := getNodeEarliestTime(node.Children[i])
		timeJ := getNodeEarliestTime(node.Children[j])
		if timeI.IsZero() && timeJ.IsZero() {
			return false
		}
		if timeI.IsZero() {
			return false
		}
		if timeJ.IsZero() {
			return true
		}
		return timeI.Before(timeJ)
	})

	skillName := "Skill Invocation"
	for _, e := range node.Entries {
		if name := str(e.Payload["skill"]); name != "" {
			skillName = name
			break
		}
	}
	prefix := strings.Repeat("↳ ", depth)
	fmt.Fprintf(out, `<details open><summary>%s🧠 %s</summary><div class="indent">`, prefix, html.EscapeString(skillName))

	for _, entry := range node.Entries {
		renderLogEntry(out, entry)
	}

	fmt.Fprint(out, `</div></details>`)
	for _, child := range node.Children {
		renderSkillNode(out, child, depth+1)
	}
}

func RenderHashedLogToHTML(path string, verificationStatus ...VerificationStatus) error {
	cleanPath := filepath.Clean(path)
	if cleanPath == "" || cleanPath == "." {
		return fmt.Errorf("invalid input path")
	}

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Parse log file
	logData, err := parseLogFile(absPath)
	if err != nil {
		return err
	}

	// Build skill tree
	roots := buildSkillTree(logData)

	// Create output file
	htmlPath := strings.TrimSuffix(absPath, filepath.Ext(absPath)) + ".html"
	if filepath.Dir(htmlPath) != filepath.Dir(absPath) {
		return fmt.Errorf("invalid output path: must be in same directory as input")
	}
	out, err := os.Create(htmlPath)
	if err != nil {
		return fmt.Errorf("failed to create html output: %w", err)
	}

	// Write HTML content
	writeHTMLHeader(out, logData, verificationStatus)

	for _, root := range roots {
		renderSkillNode(out, root, 0)
	}

	fmt.Fprint(out, `</body></html>`)

	if err := out.Sync(); err != nil {
		out.Close()
		return fmt.Errorf("failed to sync file to disk: %w", err)
	}

	// Close the file handle before returning to avoid Windows file locking issues
	if err := out.Close(); err != nil {
		return fmt.Errorf("failed to close html file: %w", err)
	}

	return nil
}

func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func renderLogEntry(out io.Writer, entry HashedLogEntry) {
	p := entry.Payload

	// Extract and process basic fields
	event := strings.ToUpper(str(p["event"]))
	decision := strings.ToUpper(str(p["decision"]))
	level := str(p["level"])
	levelClass := getLevelClass(level)

	fmt.Fprint(out, `<div class="entry">`)

	// Render time
	renderTime(out, p)

	// Render basic fields
	renderBasicFields(out, p, event, decision, level, levelClass)

	// Render complex fields
	renderComplexFields(out, p)

	fmt.Fprint(out, `</div>`)
}

// getLevelClass returns the CSS class for the log level
func getLevelClass(level string) string {
	levelClass := "level"
	switch level {
	case "info":
		levelClass += " level-info"
	case "error":
		levelClass += " level-error"
	}
	return levelClass
}

// renderTime renders the timestamp field
func renderTime(out io.Writer, p map[string]any) {
	if rawTime, ok := p["time"]; ok {
		formattedTime := formatTime(rawTime)
		fmt.Fprintf(out, `<div class="time">%s</div>`, html.EscapeString(formattedTime))
	}
}

// formatTime formats a time value into a readable string
func formatTime(rawTime any) string {
	switch v := rawTime.(type) {
	case float64:
		return time.UnixMilli(int64(v)).Local().Format("2006-01-02 15:04:05 MST")
	case int64:
		return time.UnixMilli(v).Local().Format("2006-01-02 15:04:05 MST")
	case string:
		if parsed, err := time.Parse(time.RFC3339, v); err == nil {
			return parsed.Local().Format("2006-01-02 15:04:05 MST")
		}
		return v
	default:
		return str(rawTime)
	}
}

// renderBasicFields renders simple string fields
func renderBasicFields(out io.Writer, p map[string]any, event, decision, level, levelClass string) {
	// Render event
	if event != "" {
		fmt.Fprintf(out, `<div class="field"><span class="label">Event:</span><span class="value">%s</span></div>`, html.EscapeString(event))
	}

	// Render standard fields
	standardFields := []string{"actor", "runner", "message", "view"}
	for _, k := range standardFields {
		if v := str(p[k]); v != "" {
			fmt.Fprintf(out, `<div class="field"><span class="label">%s:</span><span class="value">%s</span></div>`, strings.Title(k), html.EscapeString(v))
		}
	}

	// Render decision
	if decision != "" {
		fmt.Fprintf(out, `<div class="field"><span class="label">Decision:</span><span class="value">%s</span></div>`, html.EscapeString(decision))
	}

	// Render level
	if level != "" {
		fmt.Fprintf(out, `<span class="%s">%s</span>`, levelClass, html.EscapeString(level))
	}

	// Render status
	if status := str(p["status"]); status != "" {
		fmt.Fprintf(out, `<div class="field"><span class="label">Status:</span><span class="value">%s</span></div>`, html.EscapeString(status))
	}

	// Render context name
	if ctx, ok := p["context_name"]; ok {
		fmt.Fprintf(out, `<div class="field"><span class="label">Context Name:</span><span class="value">%s</span></div>`, html.EscapeString(str(ctx)))
	}
}

// renderComplexFields renders complex fields that require special formatting
func renderComplexFields(out io.Writer, p map[string]any) {
	// Render error
	renderErrorField(out, p)

	// Render JSON fields
	renderJSONField(out, p, "input_args", "Input Args", "input")
	renderJSONField(out, p, "basis", "Policy Basis", "basis")
	renderJSONField(out, p, "actions", "Actions", "actions")
}

// renderErrorField renders the error field
func renderErrorField(out io.Writer, p map[string]any) {
	if errVal, ok := p["error"]; ok {
		fmt.Fprintf(out, `<div class="error"><strong>Error:</strong> %s</div>`, html.EscapeString(fmt.Sprintf("%v", errVal)))
	}
}

// renderJSONField renders a JSON field with proper formatting
func renderJSONField(out io.Writer, p map[string]any, fieldKey, fieldLabel, cssClass string) {
	if fieldValue, ok := p[fieldKey]; ok {
		if b, err := json.MarshalIndent(fieldValue, "", "  "); err == nil {
			if cssClass == "basis" || cssClass == "actions" {
				fmt.Fprintf(out, `<div class="%s"><strong>%s:</strong><br><pre>%s</pre></div>`, cssClass, fieldLabel, html.EscapeString(string(b)))
			} else {
				fmt.Fprintf(out, `<div class="%s"><strong>%s:</strong><br>%s</div>`, cssClass, fieldLabel, html.EscapeString(string(b)))
			}
		}
	}
}

const cssStyle = `
:root {
  --entry-bg: #fefefe;
}
@media (prefers-color-scheme: dark) {
  :root {
    --entry-bg: #1b1b1b;
  }
}
body {
  font-family: sans-serif;
  margin: 2em;
  background: #fff;
  color: #000;
}
@media (prefers-color-scheme: dark) {
  body { background: #111; color: #ccc; }
}
h1 { font-size: 1.6em; margin-bottom: 0.3em; }
h2 { font-weight: normal; color: #aaa; font-size: 1em; margin-bottom: 0em; }
.entry {
  border-left: 4px solid #ccc;
  margin: 1em 0;
  padding: 1em;
  background: var(--entry-bg);
}
.time { font-size: 0.9em; color: #888; margin-bottom: 0.5em; }
.label { font-weight: 600; margin-top: 0.3em; }
.value { margin-left: 0.5em; color: #bbb; font-weight: normal; }
.level {
  font-size: 0.75em;
  padding: 2px 6px;
  border-radius: 4px;
  margin-left: 6px;
  display: inline-block;
  font-weight: bold;
}
.level-info { background: #eaf5ff; color: #0366d6; }
.level-error { background: #ffeef0; color: #d73a49; }
.input, .actions, .error, .basis {
  margin-top: 0.75em;
  font-family: monospace;
  white-space: pre-wrap;
}
pre {
  background: #222;
  color: #ddd;
  padding: 0.5em;
  border-radius: 4px;
  overflow-x: auto;
}
@media (prefers-color-scheme: light) {
  pre { background: #f6f8fa; color: #333; }
}
details summary {
  font-size: 1em;
  font-weight: bold;
  cursor: pointer;
  margin-bottom: 0.5em;
}
.indent { margin-left: 2em; }
`
