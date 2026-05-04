package extractor

import (
	"fmt"
	"strings"
)

// ResolveNodeReference resolves a node reference in prompt data
func ResolveNodeReference(nodeRef interface{}, promptData map[string]interface{}, depth int) interface{} {
	if depth > 10 {
		return nodeRef
	}

	if ref, ok := nodeRef.([]interface{}); ok && len(ref) == 2 {
		nodeID := fmt.Sprintf("%v", ref[0])
		if node, ok := promptData[nodeID].(map[string]interface{}); ok {
			classType, _ := node["class_type"].(string)
			inputs, ok := node["inputs"].(map[string]interface{})
			if !ok {
				return nodeRef
			}

			if classType == "String" {
				for _, field := range []string{"String", "string", "text", "value"} {
					if val, ok := inputs[field]; ok && val != nil {
						return ResolveNodeReference(val, promptData, depth+1)
					}
				}
			} else if classType == "KepStringLiteral" {
				for _, field := range []string{"string", "String", "text", "value"} {
					if val, ok := inputs[field]; ok && val != nil {
						return ResolveNodeReference(val, promptData, depth+1)
					}
				}
			}

			// Try common field names
			for field, val := range inputs {
				fLower := strings.ToLower(field)
				if fLower == "text" || fLower == "string" || fLower == "value" || fLower == "prompt" || fLower == "content" {
					if val != nil {
						return ResolveNodeReference(val, promptData, depth+1)
					}
				}
			}
		}
	}

	return nodeRef
}

// ExtractPositiveFromPromptData extracts positive prompts from ComfyUI prompt JSON
func ExtractPositiveFromPromptData(promptData map[string]interface{}, processedNodes map[string]struct{}) []PromptEntry {
	var positivePrompts []PromptEntry

	// Step 1: Identify nodes that are explicitly linked to "negative" inputs of samplers or conditioning nodes
	negativeNodeIDs := make(map[string]struct{})
	for _, v := range promptData {
		node, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		inputs, ok := node["inputs"].(map[string]interface{})
		if !ok {
			continue
		}

		for inputName, inputVal := range inputs {
			inputNameLower := strings.ToLower(inputName)
			if strings.Contains(inputNameLower, "negative") {
				traceNegativeConditioning(inputVal, promptData, negativeNodeIDs, 0)
			}
		}
	}

	for nodeID, v := range promptData {
		node, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		classType, _ := node["class_type"].(string)

		if _, processed := processedNodes[nodeID]; processed {
			continue
		}

		if classType == "CLIPTextEncode" {
			inputs, ok := node["inputs"].(map[string]interface{})
			if !ok {
				continue
			}

			var textContent string
			for _, fieldName := range []string{"text", "prompt", "string", "conditioning"} {
				if textValue, ok := inputs[fieldName]; ok {
					resolvedValue := ResolveNodeReference(textValue, promptData, 0)
					if resolvedValue != nil {
						switch rv := resolvedValue.(type) {
						case string:
							textContent = rv
						case []interface{}:
							var parts []string
							for _, item := range rv {
								if item != nil {
									parts = append(parts, fmt.Sprint(item))
								}
							}
							textContent = strings.Join(parts, "\n")
						default:
							textContent = fmt.Sprint(rv)
						}

						if strings.TrimSpace(textContent) != "" {
							break
						}
					}
				}
			}

			if strings.TrimSpace(textContent) != "" && IsValidPromptText(textContent) {
				textContentLower := strings.ToLower(strings.TrimSpace(textContent))
				
				// Check for negative prompt indicators in the text itself
				isNegative := strings.HasPrefix(textContentLower, "negative") ||
					strings.Contains(textContentLower, "negative prompt")

				// Check _meta for title if available
				if meta, ok := node["_meta"].(map[string]interface{}); ok {
					if title, ok := meta["title"].(string); ok {
						titleLower := strings.ToLower(title)
						if strings.Contains(titleLower, "negative") || strings.Contains(titleLower, "neg") {
							isNegative = true
						}
					}
				}

				// Check if this node was traced back from a negative input
				if _, ok := negativeNodeIDs[nodeID]; ok {
					isNegative = true
				}

				if !isNegative {
					positivePrompts = append(positivePrompts, PromptEntry{
						Text:     textContent,
						NodeID:   nodeID,
						NodeType: classType,
						Source:   "prompt",
					})
					processedNodes[nodeID] = struct{}{}
				}
			}
		}
	}

	return positivePrompts
}

// traceNegativeConditioning traces back from a negative input to find all contributing CLIPTextEncode nodes
func traceNegativeConditioning(val interface{}, promptData map[string]interface{}, negativeNodeIDs map[string]struct{}, depth int) {
	if depth > 20 {
		return
	}

	ref, ok := val.([]interface{})
	if !ok || len(ref) != 2 {
		return
	}

	nodeID := fmt.Sprintf("%v", ref[0])
	negativeNodeIDs[nodeID] = struct{}{}

	node, ok := promptData[nodeID].(map[string]interface{})
	if !ok {
		return
	}

	classType, _ := node["class_type"].(string)
	inputs, ok := node["inputs"].(map[string]interface{})
	if !ok {
		return
	}

	// Trace through conditioning nodes
	if strings.Contains(classType, "Conditioning") {
		for _, inputVal := range inputs {
			traceNegativeConditioning(inputVal, promptData, negativeNodeIDs, depth+1)
		}
	}
}
