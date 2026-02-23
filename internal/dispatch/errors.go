package dispatch

import "strings"

// isProviderError checks if the given error is a provider-related error.
func isProviderError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "502 all providers failed") ||
		strings.Contains(errMsg, "429 budget exceeded") ||
		strings.Contains(errMsg, "5xx")
}