package extractor

import (
	"strings"
)

// ExtractPositiveFromParametersStrict extracts positive prompts from A1111 parameters metadata
func ExtractPositiveFromParametersStrict(metadata map[string]string) string {
	paramsText, ok := metadata["parameters"]
	if !ok {
		return ""
	}

	lines := strings.Split(paramsText, "\n")
	var positiveLines []string

	for _, line := range lines {
		stripped := strings.TrimSpace(line)
		lower := strings.ToLower(stripped)

		if strings.HasPrefix(lower, "negative prompt:") {
			break
		}
		if strings.HasPrefix(lower, "steps:") {
			break
		}

		if idx := strings.Index(lower, " steps:"); idx != -1 {
			stripped = strings.TrimSpace(stripped[:idx])
			lower = strings.ToLower(stripped)
			if stripped == "" {
				break
			}
		}

		if strings.HasPrefix(lower, "positive prompt:") {
			stripped = strings.TrimSpace(stripped[len("positive prompt:"):])
			lower = strings.ToLower(stripped)
		}

		if stripped != "" {
			positiveLines = append(positiveLines, stripped)
		}
	}

	promptText := strings.Join(positiveLines, "\n")
	if IsValidPromptText(promptText) {
		return promptText
	}
	return ""
}
