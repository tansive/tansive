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

func RenderHashedLogToHTML(path string, verificationStatus ...VerificationStatus) error {
	cleanPath := filepath.Clean(path)
	if cleanPath == "" || cleanPath == "." {
		return fmt.Errorf("invalid input path")
	}

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	type SkillNode struct {
		ID        string
		InvokerID string
		Entries   []HashedLogEntry
		Children  []*SkillNode
	}

	// getEntryTime extracts the timestamp from a log entry
	getEntryTime := func(entry HashedLogEntry) time.Time {
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
	getNodeEarliestTime := func(node *SkillNode) time.Time {
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

	f, err := os.Open(absPath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
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
			return fmt.Errorf("failed to read log file: %w", err)
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

	// --- Deterministic node building ---
	nodes := make(map[string]*SkillNode)
	var generalRoot *SkillNode

	// First pass: create all nodes
	for id, entries := range invocationMap {
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
			InvokerID: invokerMap[id],
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

	// --- HTML output ---
	htmlPath := strings.TrimSuffix(absPath, filepath.Ext(absPath)) + ".html"
	if filepath.Dir(htmlPath) != filepath.Dir(absPath) {
		return fmt.Errorf("invalid output path: must be in same directory as input")
	}
	out, err := os.Create(htmlPath)
	if err != nil {
		return fmt.Errorf("failed to create html output: %w", err)
	}
	defer out.Close()

	// BEGIN HTML
	fmt.Fprint(out, `<html><head><meta charset="UTF-8"><title>Tansive™ Session Log</title><style>`)
	fmt.Fprint(out, cssStyle)
	fmt.Fprint(out, `</style></head><body>
<h1>Tansive™ Session Audit Log</h1>
<h2><strong>Session:</strong> `+html.EscapeString(firstSessionID)+`</h2>
<h2><strong>Tangent ID:</strong> `+html.EscapeString(tangentID)+`</h2>
<h2><strong>Tangent URL:</strong> `+html.EscapeString(tangentURL)+`</h2>`)

	if len(verificationStatus) > 0 {
		fmt.Fprintf(out, `<h2><strong>Verification Status:</strong> %s</h2>`, html.EscapeString(str(verificationStatus[0].Verified)))
		fmt.Fprintf(out, `<h2><strong>Key Digest:</strong> %s</h2>`, html.EscapeString(verificationStatus[0].KeyDigest))
		if verificationStatus[0].Error != nil {
			fmt.Fprintf(out, `<h2><strong>Error:</strong> %s</h2>`, html.EscapeString(verificationStatus[0].Error.Error()))
		}
	}
	fmt.Fprint(out, `<br />`)

	var renderNode func(node *SkillNode, depth int)
	renderNode = func(node *SkillNode, depth int) {
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
			renderNode(child, depth+1)
		}
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

	for _, root := range roots {
		renderNode(root, 0)
	}

	fmt.Fprint(out, `</body></html>`)

	if err := out.Sync(); err != nil {
		return fmt.Errorf("failed to sync file to disk: %w", err)
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
	event := strings.ToUpper(str(p["event"]))
	decision := strings.ToUpper(str(p["decision"]))
	level := str(p["level"])
	levelClass := "level"
	switch level {
	case "info":
		levelClass += " level-info"
	case "error":
		levelClass += " level-error"
	}

	fmt.Fprint(out, `<div class="entry">`)
	if rawTime, ok := p["time"]; ok {
		switch v := rawTime.(type) {
		case float64:
			ts := time.UnixMilli(int64(v)).Local().Format("2006-01-02 15:04:05 MST")
			fmt.Fprintf(out, `<div class="time">%s</div>`, html.EscapeString(ts))
		case int64:
			ts := time.UnixMilli(v).Local().Format("2006-01-02 15:04:05 MST")
			fmt.Fprintf(out, `<div class="time">%s</div>`, html.EscapeString(ts))
		case string:
			if parsed, err := time.Parse(time.RFC3339, v); err == nil {
				fmt.Fprintf(out, `<div class="time">%s</div>`, html.EscapeString(parsed.Local().Format("2006-01-02 15:04:05 MST")))
			} else {
				fmt.Fprintf(out, `<div class="time">%s</div>`, html.EscapeString(v))
			}
		default:
			fmt.Fprintf(out, `<div class="time">%s</div>`, html.EscapeString(str(rawTime)))
		}
	}

	if event != "" {
		fmt.Fprintf(out, `<div class="field"><span class="label">Event:</span><span class="value">%s</span></div>`, html.EscapeString(event))
	}
	for _, k := range []string{"actor", "runner", "message", "view"} {
		if v := str(p[k]); v != "" {
			fmt.Fprintf(out, `<div class="field"><span class="label">%s:</span><span class="value">%s</span></div>`, strings.Title(k), html.EscapeString(v))
		}
	}
	if decision != "" {
		fmt.Fprintf(out, `<div class="field"><span class="label">Decision:</span><span class="value">%s</span></div>`, html.EscapeString(decision))
	}
	if level != "" {
		fmt.Fprintf(out, `<span class="%s">%s</span>`, levelClass, html.EscapeString(level))
	}
	if status := str(p["status"]); status != "" {
		fmt.Fprintf(out, `<div class="field"><span class="label">Status:</span><span class="value">%s</span></div>`, html.EscapeString(status))
	}
	if errVal, ok := p["error"]; ok {
		fmt.Fprintf(out, `<div class="error"><strong>Error:</strong> %s</div>`, html.EscapeString(fmt.Sprintf("%v", errVal)))
	}
	if args, ok := p["input_args"]; ok {
		if b, err := json.MarshalIndent(args, "", "  "); err == nil {
			fmt.Fprintf(out, `<div class="input"><strong>Input Args:</strong><br>%s</div>`, html.EscapeString(string(b)))
		}
	}
	if ctx, ok := p["context_name"]; ok {
		fmt.Fprintf(out, `<div class="field"><span class="label">Context Name:</span><span class="value">%s</span></div>`, html.EscapeString(str(ctx)))
	}
	if basis, ok := p["basis"]; ok {
		if b, err := json.MarshalIndent(basis, "", "  "); err == nil {
			fmt.Fprintf(out, `<div class="basis"><strong>Policy Basis:</strong><br><pre>%s</pre></div>`, html.EscapeString(string(b)))
		}
	}
	if acts, ok := p["actions"]; ok {
		if b, err := json.MarshalIndent(acts, "", "  "); err == nil {
			fmt.Fprintf(out, `<div class="actions"><strong>Actions:</strong><br><pre>%s</pre></div>`, html.EscapeString(string(b)))
		}
	}
	fmt.Fprint(out, `</div>`)
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
