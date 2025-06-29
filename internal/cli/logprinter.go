package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
)

type sessionState struct {
	startTime   int64
	endTime     int64
	initialized bool
	skillsSeen  map[string]bool
}

var sessions = map[string]*sessionState{}
var skillColors = map[string]*color.Color{}

var sessionLabel = color.New(color.FgHiMagenta, color.Bold)

// var timeLabel = color.New(color.FgWhite)
var startLabel = color.New(color.FgGreen).Add(color.Bold)
var endLabel = color.New(color.FgRed).Add(color.Bold)

// Predefined palette of distinct colors for skills
var colorPalette = []*color.Color{
	color.New(color.FgGreen),
	color.New(color.FgCyan),
	color.New(color.FgMagenta),
	color.New(color.FgYellow),
	color.New(color.FgRed),
	color.New(color.FgBlue),
}

var systemColor = color.New(color.FgHiWhite)              // For internal/system messages
var runnerColor = color.New(color.FgHiWhite, color.Faint) // For runner-specific actions
var tansiveColor = color.New(color.FgHiMagenta)           // For core Tansive operations

var _ = tansiveColor

var colorIndex = 0

// PrettyPrintNDJSONLine formats and prints a single NDJSON line with color-coded output
// It handles session tracking, timestamps, and different message types
func PrettyPrintNDJSONLine(line []byte) {
	var m map[string]any
	if err := json.Unmarshal(line, &m); err != nil {
		fmt.Printf("⚠️  Invalid JSON: %s\n", string(line))
		return
	}

	sessionID := str(m["session_id"])
	skill := str(m["skill"])
	msg := str(m["message"])
	source := str(m["source"])
	level := str(m["level"])
	policy := str(m["policy_decision"])
	actor := str(m["actor"])
	runner := str(m["runner"])
	errorMsg := str(m["error"])
	t := int64From(m["time"]) // milliseconds since epoch

	// Initialize session if needed
	sess := sessions[sessionID]
	if sess == nil {
		sess = &sessionState{
			startTime:   t,
			endTime:     t,
			initialized: true,
			skillsSeen:  make(map[string]bool),
		}
		sessions[sessionID] = sess
		startTime := time.UnixMilli(t).Local()

		sessionLabel.Printf("\nSession ID: %s\n", sessionID)
		startLabel.Printf("    Start: %s\n\n", startTime.Format("2006-01-02 15:04:05.000 MST"))
	} else {
		if t > sess.endTime {
			sess.endTime = t
		}
	}

	// Assign color to skill if new
	if skill != "" && skillColors[skill] == nil {
		skillColors[skill] = colorPalette[colorIndex%len(colorPalette)]
		colorIndex++
	}
	skillColor := skillColors[skill]

	// Compute relative timestamp
	relative := time.Duration(t-sess.startTime) * time.Millisecond
	timestamp := fmt.Sprintf("[%02d:%02d.%03d]",
		int(relative.Minutes()),
		int(relative.Seconds())%60,
		relative.Milliseconds()%1000,
	)

	msg = indentMultiline(msg, "                                   ")

	fmt.Print("  " + timestamp + " ")
	switch actor {
	case "system":
		systemColor.Printf("%s", "[tansive]")
	case "runner":
		runnerColor.Printf("[%s]", runner)
	case "skill":
		skillColor.Printf("%s", skill)
	}

	// errors: only ❗ and message in red
	if policy == "true" {
		fmt.Print(" ")
		color.New(color.FgHiRed).Print("🛡️ ")
		if level == "error" {
			color.New(color.FgHiRed).Println(msg)
		} else {
			color.New(color.FgHiGreen).Println(msg)
		}
	} else if source == "stderr" || level == "error" {
		fmt.Print(" ")
		color.New(color.FgHiRed).Print("❗ ")
		color.New(color.FgHiRed).Println(msg)
		if errorMsg != "" {
			color.New(color.FgHiRed).Println("                                   ", errorMsg)
		}
	} else {
		fmt.Print(" ▶ ")
		fmt.Println(msg)
	}

	// Print end time if message indicates session completion. This is quite brittle,
	// we should track by event type. Nothing breaks loose right now.
	if strings.Contains(msg, "Interactive skill completed successfully") {
		endTime := time.UnixMilli(sess.endTime).Local()
		endLabel.Printf("\n    End:   %s\n", endTime.Format("2006-01-02 15:04:05.000 MST"))
	}
}

// str safely converts an interface{} to string, returning empty string if conversion fails
func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// int64From safely converts an interface{} to int64, handling different numeric types
func int64From(v any) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	default:
		return 0
	}
}

// indentMultiline adds indentation to all lines except the first in a multiline string
func indentMultiline(text, indent string) string {
	lines := strings.Split(text, "\n")
	if len(lines) <= 1 {
		return text
	}
	for i := 1; i < len(lines); i++ {
		lines[i] = indent + lines[i]
	}
	return strings.Join(lines, "\n")
}
