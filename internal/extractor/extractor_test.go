package extractor

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractPositiveFromWorkflow(t *testing.T) {
	data, err := os.ReadFile("testdata/workflow.json")
	assert.NoError(t, err)

	var workflow map[string]interface{}
	err = json.Unmarshal(data, &workflow)
	assert.NoError(t, err)

	processed := make(map[string]struct{})
	prompts := ExtractPositiveFromWorkflow(workflow, processed)

	assert.Len(t, prompts, 1)
	assert.Equal(t, "a beautiful landscape", prompts[0].Text)
	assert.Equal(t, "1", prompts[0].NodeID)
}

func TestExtractPositiveFromPromptData(t *testing.T) {
	data, err := os.ReadFile("testdata/prompt.json")
	assert.NoError(t, err)

	var promptData map[string]interface{}
	err = json.Unmarshal(data, &promptData)
	assert.NoError(t, err)

	processed := make(map[string]struct{})
	prompts := ExtractPositiveFromPromptData(promptData, processed)

	// In prompt data, without titles, we check if it starts with "negative"
	// "a cute cat" is positive. "ugly" doesn't start with "negative" but we might need more heuristic.
	// The Python code: is_negative = ('negative' in str(text_content).lower()[:50])
	// "ugly" does not have "negative" in it. So it might be included if not careful.
	// Actually, "ugly" is just a short text.
	
	assert.Len(t, prompts, 2) // Both might be included if they don't have "negative"
}

func TestIsValidPromptText(t *testing.T) {
	assert.True(t, IsValidPromptText("a beautiful landscape"))
	assert.False(t, IsValidPromptText(""))
	assert.False(t, IsValidPromptText("123"))
	assert.False(t, IsValidPromptText("1.0"))
}

func TestExtractPositiveFromParametersStrict(t *testing.T) {
	metadata := map[string]string{
		"parameters": "a cat in a hat\nNegative prompt: lowres, bad anatomy\nSteps: 20, Sampler: Euler",
	}
	prompt := ExtractPositiveFromParametersStrict(metadata)
	assert.Equal(t, "a cat in a hat", prompt)

	metadata = map[string]string{
		"parameters": "Positive prompt: a dog\nSteps: 10",
	}
	prompt = ExtractPositiveFromParametersStrict(metadata)
	assert.Equal(t, "a dog", prompt)
}
