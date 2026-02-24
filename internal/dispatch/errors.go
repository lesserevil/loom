package dispatch

import (
	"encoding/json"
	"strings"
)

// isProviderError checks if the given error message indicates a provider-related error.
// These are transient infrastructure issues that will resolve on their own and should
// not trigger remediation bead creation.
func isProviderError(errMsg string) bool {
	if errMsg == "" {
		return false
	}
	// Connection/network errors
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "context canceled") ||
		strings.Contains(errMsg, "context deadline exceeded") ||
		strings.Contains(errMsg, "dial tcp") ||
		strings.Contains(errMsg, "no such host") ||
		strings.Contains(errMsg, "i/o timeout") ||
		strings.Contains(errMsg, "connection reset") ||
		strings.Contains(errMsg, "broken pipe") ||
		strings.Contains(errMsg, "EOF") {
		return true
	}
	// HTTP status code errors (rate limits, auth, server errors)
	if strings.Contains(errMsg, "status code 401") ||
		strings.Contains(errMsg, "status code 403") ||
		strings.Contains(errMsg, "status code 429") ||
		strings.Contains(errMsg, "status code 500") ||
		strings.Contains(errMsg, "status code 502") ||
		strings.Contains(errMsg, "status code 503") ||
		strings.Contains(errMsg, "status code 504") {
		return true
	}
	// Provider-specific error patterns
	if strings.Contains(errMsg, "all providers failed") ||
		strings.Contains(errMsg, "no eligible models") ||
		strings.Contains(errMsg, "budget exceeded") ||
		strings.Contains(errMsg, "authorization required") ||
		strings.Contains(errMsg, "rate limit") ||
		strings.Contains(errMsg, "quota exceeded") ||
		strings.Contains(errMsg, "capacity") ||
		strings.Contains(errMsg, "overloaded") ||
		strings.Contains(errMsg, "temporarily unavailable") ||
		strings.Contains(errMsg, "service unavailable") {
		return true
	}
	// LLM call failures that indicate provider issues
	if strings.Contains(errMsg, "LLM call failed") &&
		(strings.Contains(errMsg, "502") ||
			strings.Contains(errMsg, "503") ||
			strings.Contains(errMsg, "429") ||
			strings.Contains(errMsg, "timeout") ||
			strings.Contains(errMsg, "connection")) {
		return true
	}
	return false
}

// beadHasProviderErrors checks if a bead's context indicates it has been
// experiencing provider/infrastructure errors. This is used to prevent
// creating remediation beads for beads that are stuck due to transient
// infrastructure issues rather than agent logic problems.
func beadHasProviderErrors(beadContext map[string]string) bool {
	if beadContext == nil {
		return false
	}

	// Check last_run_error for provider errors
	if lastErr := beadContext["last_run_error"]; lastErr != "" {
		if isProviderError(lastErr) {
			return true
		}
	}

	// Check error_history for recent provider errors
	if historyJSON := beadContext["error_history"]; historyJSON != "" {
		var history []struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal([]byte(historyJSON), &history); err == nil {
			// If any of the last 3 errors are provider errors, skip remediation
			start := 0
			if len(history) > 3 {
				start = len(history) - 3
			}
			for i := start; i < len(history); i++ {
				if isProviderError(history[i].Error) {
					return true
				}
			}
		}
	}

	// Check loop_detection_reason for provider-related patterns
	if loopReason := beadContext["loop_detection_reason"]; loopReason != "" {
		if isProviderError(loopReason) {
			return true
		}
	}

	return false
}
