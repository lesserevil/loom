package actions

import (
	"strings"
	"unicode"
)

// MatchAndReplace tries multiple strategies to find oldText in content and replace it.
// Returns (newContent, matched, strategyUsed). Inspired by opencode's 9-tier replacer chain.
func MatchAndReplace(content, oldText, newText string) (string, bool, string) {
	if oldText == "" {
		return content, false, ""
	}

	// Strategy 1: Exact match
	if strings.Contains(content, oldText) {
		return strings.Replace(content, oldText, newText, 1), true, "exact"
	}

	// Strategy 2: Line-trimmed match — trim trailing whitespace per line
	if result, ok := lineTrimmedMatch(content, oldText, newText); ok {
		return result, true, "line_trimmed"
	}

	// Strategy 3: Whitespace-normalized match — collapse all whitespace
	if result, ok := whitespaceNormalizedMatch(content, oldText, newText); ok {
		return result, true, "whitespace_normalized"
	}

	// Strategy 4: Indentation-flexible match — strip leading indentation
	if result, ok := indentFlexibleMatch(content, oldText, newText); ok {
		return result, true, "indentation_flexible"
	}

	// Strategy 5: Block anchor match — match by first and last lines
	if result, ok := blockAnchorMatch(content, oldText, newText); ok {
		return result, true, "block_anchor"
	}

	return content, false, ""
}

// lineTrimmedMatch trims trailing whitespace from each line before comparing.
func lineTrimmedMatch(content, oldText, newText string) (string, bool) {
	trimLines := func(s string) string {
		lines := strings.Split(s, "\n")
		for i, line := range lines {
			lines[i] = strings.TrimRightFunc(line, unicode.IsSpace)
		}
		return strings.Join(lines, "\n")
	}

	trimmedContent := trimLines(content)
	trimmedOld := trimLines(oldText)

	if idx := strings.Index(trimmedContent, trimmedOld); idx >= 0 {
		// Find corresponding position in original content
		origIdx := mapTrimmedIndex(content, trimmedContent, idx)
		origEnd := mapTrimmedIndex(content, trimmedContent, idx+len(trimmedOld))
		if origIdx >= 0 && origEnd >= origIdx {
			return content[:origIdx] + newText + content[origEnd:], true
		}
	}
	return "", false
}

// whitespaceNormalizedMatch collapses all whitespace to single spaces.
func whitespaceNormalizedMatch(content, oldText, newText string) (string, bool) {
	normalize := func(s string) string {
		var sb strings.Builder
		lastWasSpace := false
		for _, r := range s {
			if unicode.IsSpace(r) {
				if !lastWasSpace {
					sb.WriteRune(' ')
					lastWasSpace = true
				}
			} else {
				sb.WriteRune(r)
				lastWasSpace = false
			}
		}
		return sb.String()
	}

	normContent := normalize(content)
	normOld := normalize(oldText)

	if idx := strings.Index(normContent, normOld); idx >= 0 {
		// Map back to original content using line-by-line approach
		lines := strings.Split(content, "\n")
		oldLines := strings.Split(oldText, "\n")
		if len(oldLines) > 0 {
			// Find the first line match
			firstOldLine := strings.TrimSpace(oldLines[0])
			for i, line := range lines {
				if strings.Contains(strings.TrimSpace(line), firstOldLine) {
					// Found start — extract the block
					endIdx := i + len(oldLines)
					if endIdx > len(lines) {
						endIdx = len(lines)
					}
					origBlock := strings.Join(lines[i:endIdx], "\n")
					return strings.Replace(content, origBlock, newText, 1), true
				}
			}
		}
	}
	return "", false
}

// indentFlexibleMatch strips all leading indentation before comparing.
func indentFlexibleMatch(content, oldText, newText string) (string, bool) {
	stripIndent := func(s string) string {
		lines := strings.Split(s, "\n")
		for i, line := range lines {
			lines[i] = strings.TrimLeftFunc(line, unicode.IsSpace)
		}
		return strings.Join(lines, "\n")
	}

	strippedContent := stripIndent(content)
	strippedOld := stripIndent(oldText)

	if !strings.Contains(strippedContent, strippedOld) {
		return "", false
	}

	// Find original block by matching stripped lines
	contentLines := strings.Split(content, "\n")
	oldLines := strings.Split(oldText, "\n")
	firstStripped := strings.TrimSpace(oldLines[0])

	for i := range contentLines {
		if strings.TrimSpace(contentLines[i]) == firstStripped {
			// Check if all lines match when stripped
			if i+len(oldLines) > len(contentLines) {
				continue
			}
			allMatch := true
			for j, oldLine := range oldLines {
				if strings.TrimSpace(contentLines[i+j]) != strings.TrimSpace(oldLine) {
					allMatch = false
					break
				}
			}
			if allMatch {
				origBlock := strings.Join(contentLines[i:i+len(oldLines)], "\n")
				return strings.Replace(content, origBlock, newText, 1), true
			}
		}
	}
	return "", false
}

// blockAnchorMatch matches by first and last non-empty lines of the old text.
func blockAnchorMatch(content, oldText, newText string) (string, bool) {
	oldLines := nonEmptyLines(oldText)
	if len(oldLines) < 2 {
		return "", false
	}

	firstAnchor := strings.TrimSpace(oldLines[0])
	lastAnchor := strings.TrimSpace(oldLines[len(oldLines)-1])

	contentLines := strings.Split(content, "\n")

	for i := range contentLines {
		if strings.TrimSpace(contentLines[i]) != firstAnchor {
			continue
		}
		// Found first anchor — search forward for last anchor
		for j := i + 1; j < len(contentLines) && j < i+len(oldLines)+5; j++ {
			if strings.TrimSpace(contentLines[j]) == lastAnchor {
				// Check that the block size is similar (within 50%)
				blockSize := j - i + 1
				expectedSize := len(oldLines)
				if blockSize >= expectedSize/2 && blockSize <= expectedSize*2 {
					origBlock := strings.Join(contentLines[i:j+1], "\n")
					return strings.Replace(content, origBlock, newText, 1), true
				}
			}
		}
	}
	return "", false
}

func nonEmptyLines(s string) []string {
	var result []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return result
}

// mapTrimmedIndex maps an index in trimmed content back to the original.
// This is approximate — returns the byte offset in original that corresponds
// to the same line position.
func mapTrimmedIndex(original, trimmed string, trimmedIdx int) int {
	if trimmedIdx <= 0 {
		return 0
	}
	if trimmedIdx >= len(trimmed) {
		return len(original)
	}
	// Count newlines before trimmedIdx to find line number
	lineNum := strings.Count(trimmed[:trimmedIdx], "\n")
	colInTrimmed := trimmedIdx - strings.LastIndex(trimmed[:trimmedIdx], "\n") - 1

	// Find same line in original
	origLines := strings.Split(original, "\n")
	if lineNum >= len(origLines) {
		return len(original)
	}

	offset := 0
	for i := 0; i < lineNum; i++ {
		offset += len(origLines[i]) + 1 // +1 for \n
	}
	// Add column offset (clamped to line length)
	if colInTrimmed > len(origLines[lineNum]) {
		colInTrimmed = len(origLines[lineNum])
	}
	return offset + colInTrimmed
}
