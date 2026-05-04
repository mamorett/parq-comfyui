package extractor

import (
	"fmt"
	"strings"
)

// ExtractPositiveFromWorkflow extracts positive prompts from ComfyUI workflow JSON
func ExtractPositiveFromWorkflow(workflowData map[string]interface{}, processedNodes map[string]struct{}) []PromptEntry {
	var positivePrompts []PromptEntry

	nodes, ok := workflowData["nodes"].([]interface{})
	if !ok {
		return nil
	}

	for _, n := range nodes {
		node, ok := n.(map[string]interface{})
		if !ok {
			continue
		}

		nodeID := fmt.Sprintf("%v", node["id"])
		nodeType, _ := node["type"].(string)
		title, _ := node["title"].(string)
		titleLower := strings.ToLower(title)

		if _, processed := processedNodes[nodeID]; processed {
			continue
		}

		// Look for CLIPTextEncode nodes
		isCLIPTextEncode := nodeType == "CLIPTextEncode" ||
			strings.Contains(strings.ToLower(nodeType), "cliptext")
		
		if !isCLIPTextEncode {
			if props, ok := node["properties"].(map[string]interface{}); ok {
				if srName, ok := props["Node name for S&R"].(string); ok && srName == "CLIPTextEncode" {
					isCLIPTextEncode = true
				}
			}
		}

		if isCLIPTextEncode {
			var promptText string

			// Try widgets_values
			if widgetsValues, ok := node["widgets_values"].([]interface{}); ok && len(widgetsValues) > 0 {
				promptValue := widgetsValues[0]
				switch v := promptValue.(type) {
				case string:
					promptText = v
				case []interface{}:
					var parts []string
					for _, item := range v {
						if item != nil {
							parts = append(parts, fmt.Sprint(item))
						}
					}
					promptText = strings.Join(parts, "\n")
				default:
					if v != nil {
						promptText = fmt.Sprint(v)
					}
				}
			}

			// If empty, try inputs
			if strings.TrimSpace(promptText) == "" {
				if inputs, ok := node["inputs"].([]interface{}); ok {
					for _, inputItem := range inputs {
						item, ok := inputItem.(map[string]interface{})
						if !ok {
							continue
						}
						name, _ := item["name"].(string)
						if name == "text" || name == "prompt" || name == "string" {
							if widget, ok := item["widget"].(map[string]interface{}); ok {
								val, ok := widget["value"].(string)
								if !ok {
									val, _ = widget["default"].(string)
								}
								promptText = val
							}
							break
						}
					}
				}
			}

			if strings.TrimSpace(promptText) == "" || !IsValidPromptText(promptText) {
				continue
			}

			promptTextLower := strings.ToLower(promptText)

			isPositive := strings.Contains(titleLower, "positive") ||
				strings.Contains(titleLower, "pos") ||
				(title == "" && !strings.HasPrefix(promptTextLower, "negative")) ||
				(title == "untitled" && !strings.HasPrefix(promptTextLower, "negative"))

			isNegative := strings.Contains(titleLower, "negative") ||
				strings.Contains(titleLower, "neg") ||
				strings.HasPrefix(strings.TrimSpace(promptTextLower), "negative")

			if isPositive && !isNegative {
				positivePrompts = append(positivePrompts, PromptEntry{
					Text:     promptText,
					NodeID:   nodeID,
					NodeType: nodeType,
					Title:    title,
					Source:   "workflow",
				})
				processedNodes[nodeID] = struct{}{}
			}
		}
	}

	return positivePrompts
}
