package actions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	lessonsFileName = "LESSONS.md"
	maxLessonsSize  = 8000 // Max chars to inject into prompt (keeps context manageable)
)

// LessonsFile manages a per-project LESSONS.md file for agent memory.
// Lessons are human-readable markdown, auto-recorded on failures,
// and injected into the system prompt at conversation start.
type LessonsFile struct {
	projectDir string
}

// NewLessonsFile creates a lessons file manager for a project directory.
func NewLessonsFile(projectDir string) *LessonsFile {
	return &LessonsFile{projectDir: projectDir}
}

// GetLessonsForPrompt reads LESSONS.md and returns its content truncated
// to maxLessonsSize characters (keeping the most recent lessons).
func (l *LessonsFile) GetLessonsForPrompt() string {
	path := filepath.Join(l.projectDir, lessonsFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return "" // No lessons file yet — that's fine
	}

	content := string(data)
	if len(content) > maxLessonsSize {
		// Keep the tail (most recent lessons)
		content = "...[earlier lessons truncated]...\n\n" + content[len(content)-maxLessonsSize:]
	}

	return content
}

// RecordLesson appends a lesson to LESSONS.md.
func (l *LessonsFile) RecordLesson(category, title, detail, beadID, agentID string) error {
	path := filepath.Join(l.projectDir, lessonsFileName)

	entry := fmt.Sprintf("\n## %s\n### %s: %s\n- Bead: %s, Agent: %s\n- %s\n",
		time.Now().UTC().Format("2006-01-02 15:04"),
		strings.ToUpper(category),
		title,
		beadID,
		agentID,
		detail,
	)

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open lessons file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("failed to write lesson: %w", err)
	}

	return nil
}

// RecordBuildFailure records a build failure as a lesson.
func (l *LessonsFile) RecordBuildFailure(errorOutput, beadID, agentID string) error {
	// Extract the key error line
	lines := strings.Split(errorOutput, "\n")
	var errorLines []string
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") || strings.Contains(lower, "undefined") || strings.Contains(lower, "cannot") {
			errorLines = append(errorLines, strings.TrimSpace(line))
			if len(errorLines) >= 3 {
				break
			}
		}
	}
	detail := "Build failed"
	if len(errorLines) > 0 {
		detail = strings.Join(errorLines, "\n- ")
	}

	return l.RecordLesson("build_failure", "Build failed", detail, beadID, agentID)
}

// RecordEditFailure records a failed edit attempt as a lesson.
func (l *LessonsFile) RecordEditFailure(filePath, errorMsg, beadID, agentID string) error {
	detail := fmt.Sprintf("Edit failed on %s: %s. Always READ the file first and copy exact text.", filePath, errorMsg)
	return l.RecordLesson("edit_failure", "Edit match failed", detail, beadID, agentID)
}
