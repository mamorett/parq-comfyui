package extractor

import (
	"encoding/json"
	"fmt"
	"github.com/trithemius/parq-comfyui/internal/pngreader"
)

// PromptEntry represents an extracted prompt
type PromptEntry struct {
	Text     string `json:"text"`
	NodeID   string `json:"node_id,omitempty"`
	NodeType string `json:"node_type,omitempty"`
	Title    string `json:"title,omitempty"`
	Source   string `json:"source,omitempty"`
}

// ExtractionResult contains all extracted prompts from a file
type ExtractionResult struct {
	PositivePrompts []PromptEntry
}

// Extract extracts positive prompts from a PNG file
func Extract(path string, useParameters bool) (*ExtractionResult, error) {
	metadata, _, err := pngreader.ReadPNGMetadata(path)
	if err != nil {
		return nil, fmt.Errorf("error reading PNG metadata: %v", err)
	}

	result := &ExtractionResult{}

	if useParameters {
		promptText := ExtractPositiveFromParametersStrict(metadata)
		if promptText != "" {
			result.PositivePrompts = append(result.PositivePrompts, PromptEntry{
				Text:     promptText,
				NodeID:   "parameters",
				NodeType: "parameters",
				Title:    "Parameters",
				Source:   "parameters",
			})
		} else {
			promptText = ExtractPositiveFromPNGProperties(metadata)
			if promptText != "" {
				result.PositivePrompts = append(result.PositivePrompts, PromptEntry{
					Text:     promptText,
					NodeID:   "png_properties",
					NodeType: "png_properties",
					Title:    "PNG Properties",
					Source:   "png_properties",
				})
			}
		}
		return result, nil
	}

	processedNodes := make(map[string]struct{})

	// Try workflow first
	if workflowJSON, ok := metadata["workflow"]; ok {
		var workflowData map[string]interface{}
		if err := json.Unmarshal([]byte(workflowJSON), &workflowData); err == nil {
			prompts := ExtractPositiveFromWorkflow(workflowData, processedNodes)
			result.PositivePrompts = append(result.PositivePrompts, prompts...)
		}
	}

	// Try prompt data if nothing found or to complement
	if promptJSON, ok := metadata["prompt"]; ok {
		var promptData map[string]interface{}
		if err := json.Unmarshal([]byte(promptJSON), &promptData); err == nil {
			prompts := ExtractPositiveFromPromptData(promptData, processedNodes)
			result.PositivePrompts = append(result.PositivePrompts, prompts...)
		}
	}

	return result, nil
}
