package extractor

import (
	"strconv"
	"strings"
)

// IsValidPromptText checks if extracted text is a valid prompt
func IsValidPromptText(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}

	// Reject if it's just a pure number (node ID reference)
	if _, err := strconv.Atoi(text); err == nil {
		return false
	}

	// Reject if it's a very short number-like string (likely a node reference)
	if len(text) <= 5 {
		replaced := strings.ReplaceAll(text, ".", "")
		replaced = strings.ReplaceAll(replaced, "-", "")
		if _, err := strconv.Atoi(replaced); err == nil {
			return false
		}
	}

	return true
}
