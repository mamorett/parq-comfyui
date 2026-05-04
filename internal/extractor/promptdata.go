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
				isNegative := strings.HasPrefix(strings.ToLower(strings.TrimSpace(textContent)), "negative")
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
