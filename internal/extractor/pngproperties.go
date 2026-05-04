package extractor

import (
	"strings"
)

// ExtractPositiveFromPNGProperties extracts a prompt-like text from common PNG text fields
func ExtractPositiveFromPNGProperties(metadata map[string]string) string {
	candidateKeys := []string{
		"parameters",
		"positive",
		"positive_prompt",
		"description",
		"comment",
		"Comment",
		"Description",
		"Prompt",
		"PositivePrompt",
	}

	for _, key := range candidateKeys {
		if value, ok := metadata[key]; ok {
			text := strings.TrimSpace(value)

			if len(text) > 2000 {
				continue
			}
			if (strings.HasPrefix(text, "{") && strings.HasSuffix(text, "}")) ||
				(strings.HasPrefix(text, "[") && strings.HasSuffix(text, "]")) {
				continue
			}

			lower := strings.ToLower(text)
			if strings.HasPrefix(lower, "negative prompt:") {
				continue
			}

			if strings.HasPrefix(lower, "positive prompt:") {
				text = strings.TrimSpace(text[len("positive prompt:"):])
			}

			if IsValidPromptText(text) {
				return text
			}
		}
	}

	return ""
}
