package actions

import (
	"regexp"
	"strings"
)

// actionRE matches "ACTION: COMMAND args" with tolerance for markdown formatting,
// list prefixes, heading markers, and inconsistent spacing.
var actionRE = regexp.MustCompile(
	`(?im)^[\t >*\-` + "`" + `]*(?:#{1,6}\s*)?ACTION\s*[:\-]\s*([A-Z_]+)\s*(.*)$`,
)

// blockRE extracts <<<...>>> delimited blocks.
var blockRE = regexp.MustCompile(`(?s)<<<\s*\n?(.*?)\n?\s*>>>`)

// ParseTextAction parses a text-based agent response into an ActionEnvelope.
// Returns nil if no action is found (the model just wrote analysis text).
func ParseTextAction(response string) (*ActionEnvelope, error) {
	matches := actionRE.FindStringSubmatch(response)
	if matches == nil {
		// Try inline fallback — find last ACTION anywhere in the response
		allMatches := actionRE.FindAllStringSubmatch(response, -1)
		if len(allMatches) == 0 {
			return nil, &ValidationError{Err: ErrNoAction}
		}
		matches = allMatches[len(allMatches)-1]
	}

	command := strings.ToUpper(strings.TrimSpace(matches[1]))
	args := strings.TrimSpace(matches[2])

	// Find the text after the ACTION line for block content
	actionIdx := strings.Index(response, matches[0])
	afterAction := ""
	if actionIdx >= 0 {
		afterAction = response[actionIdx+len(matches[0]):]
	}

	action, err := buildActionFromText(command, args, afterAction)
	if err != nil {
		return nil, err
	}

	// Extract notes from any text before the ACTION line
	notes := ""
	if actionIdx > 0 {
		notes = strings.TrimSpace(response[:actionIdx])
	}

	return &ActionEnvelope{
		Actions: []Action{action},
		Notes:   notes,
	}, nil
}

func buildActionFromText(command, args, body string) (Action, error) {
	switch command {
	case "SCOPE":
		path := args
		if path == "" {
			path = "."
		}
		return Action{Type: ActionReadTree, Path: path, MaxDepth: 2}, nil

	case "TREE":
		path := args
		if path == "" {
			path = "."
		}
		return Action{Type: ActionReadTree, Path: path, MaxDepth: 3}, nil

	case "READ":
		if args == "" {
			return Action{}, &ValidationError{Err: errMissing("READ", "file path")}
		}
		return Action{Type: ActionReadFile, Path: args}, nil

	case "SEARCH":
		parts := splitArgs(args, 2)
		if len(parts) == 0 || parts[0] == "" {
			return Action{}, &ValidationError{Err: errMissing("SEARCH", "query")}
		}
		a := Action{Type: ActionSearchText, Query: parts[0]}
		if len(parts) > 1 {
			a.Path = parts[1]
		}
		return a, nil

	case "EDIT":
		if args == "" {
			return Action{}, &ValidationError{Err: errMissing("EDIT", "file path")}
		}
		blocks := blockRE.FindAllStringSubmatch(body, -1)
		if len(blocks) < 2 {
			return Action{}, &ValidationError{Err: errMissing("EDIT", "OLD and NEW blocks delimited by <<< and >>>")}
		}
		oldText := blocks[0][1]
		newText := blocks[1][1]
		// Build a unified diff-style patch
		patch := buildUnifiedPatch(args, oldText, newText)
		return Action{Type: ActionEditCode, Path: args, Patch: patch, OldText: oldText, NewText: newText}, nil

	case "WRITE":
		if args == "" {
			return Action{}, &ValidationError{Err: errMissing("WRITE", "file path")}
		}
		blocks := blockRE.FindAllStringSubmatch(body, -1)
		if len(blocks) == 0 {
			// Try to use everything after the ACTION line as content
			content := strings.TrimSpace(body)
			if content == "" {
				return Action{}, &ValidationError{Err: errMissing("WRITE", "file content in <<< >>> block")}
			}
			return Action{Type: ActionWriteFile, Path: args, Content: content}, nil
		}
		return Action{Type: ActionWriteFile, Path: args, Content: blocks[0][1]}, nil

	case "BUILD":
		return Action{Type: ActionBuildProject}, nil

	case "TEST":
		a := Action{Type: ActionRunTests}
		if args != "" {
			a.TestPattern = args
		}
		return a, nil

	case "BASH":
		if args == "" {
			return Action{}, &ValidationError{Err: errMissing("BASH", "command")}
		}
		// Command might continue on the next line
		cmd := args
		if strings.TrimSpace(body) != "" {
			cmd = args + "\n" + strings.TrimSpace(body)
		}
		return Action{Type: ActionRunCommand, Command: cmd}, nil

	case "DONE":
		return Action{Type: ActionDone, Reason: args}, nil

	case "CLOSE_BEAD":
		return Action{Type: ActionCloseBead, Reason: args}, nil

	case "ESCALATE":
		return Action{Type: ActionEscalateCEO, Reason: args}, nil

	case "GIT_COMMIT":
		return Action{Type: ActionGitCommit, CommitMessage: args}, nil

	case "GIT_PUSH":
		return Action{Type: ActionGitPush}, nil

	case "GIT_STATUS":
		return Action{Type: ActionGitStatus}, nil

	default:
		return Action{}, &ValidationError{Err: errUnknown(command)}
	}
}

// splitArgs splits on whitespace but respects quoted strings.
func splitArgs(s string, max int) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Fields(s)
	if max > 0 && len(parts) > max {
		// Rejoin the tail
		tail := strings.Join(parts[max-1:], " ")
		result := make([]string, max)
		copy(result, parts[:max-1])
		result[max-1] = tail
		return result
	}
	return parts
}

func buildUnifiedPatch(file, oldText, newText string) string {
	var sb strings.Builder
	sb.WriteString("--- a/" + file + "\n")
	sb.WriteString("+++ b/" + file + "\n")
	sb.WriteString("@@ -1,1 +1,1 @@\n")
	for _, line := range strings.Split(oldText, "\n") {
		sb.WriteString("-" + line + "\n")
	}
	for _, line := range strings.Split(newText, "\n") {
		sb.WriteString("+" + line + "\n")
	}
	return sb.String()
}

var (
	ErrNoAction = errorf("no ACTION command found in response")
)

func errMissing(cmd, what string) error {
	return errorf("%s requires %s", cmd, what)
}

func errUnknown(cmd string) error {
	return errorf("unknown action: %s. Available: SCOPE, TREE, READ, SEARCH, EDIT, WRITE, BUILD, TEST, BASH, DONE, CLOSE_BEAD, GIT_COMMIT, GIT_PUSH, GIT_STATUS", cmd)
}

type simpleError struct{ msg string }

func (e *simpleError) Error() string { return e.msg }
func errorf(format string, args ...interface{}) error {
	return &simpleError{msg: strings.TrimSpace(strings.ReplaceAll(
		replaceArgs(format, args...), "\n", " ",
	))}
}

func replaceArgs(format string, args ...interface{}) string {
	if len(args) == 0 {
		return format
	}
	result := format
	for _, arg := range args {
		idx := strings.Index(result, "%s")
		if idx < 0 {
			break
		}
		s, ok := arg.(string)
		if !ok {
			s = "<?>"
		}
		result = result[:idx] + s + result[idx+2:]
	}
	return result
}
