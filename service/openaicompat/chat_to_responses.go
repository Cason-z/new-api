package openaicompat

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/samber/lo"
)

func normalizeChatToolType(toolType string) string {
	switch strings.TrimSpace(toolType) {
	case dto.BuildInToolWebSearchPreview:
		// Azure/OpenAI responses on Foundry accept `web_search`, while many
		// OpenAI-compatible clients still send `web_search_preview`.
		return "web_search"
	default:
		return strings.TrimSpace(toolType)
	}
}

func isResponsesBuiltInToolType(toolType string) bool {
	switch normalizeChatToolType(toolType) {
	case "web_search":
		return true
	default:
		return false
	}
}

func hasResponsesBuiltInTool(tools []dto.ToolCallRequest) bool {
	for _, tool := range tools {
		if isResponsesBuiltInToolType(tool.Type) {
			return true
		}
	}
	return false
}

func normalizeChatToolChoiceForResponses(toolChoice any) any {
	if toolChoice == nil {
		return nil
	}

	switch v := toolChoice.(type) {
	case string:
		return v
	default:
		var m map[string]any
		if b, err := common.Marshal(v); err == nil {
			_ = common.Unmarshal(b, &m)
		}
		if m == nil {
			return v
		}
		if t, _ := m["type"].(string); t != "" {
			normalizedType := normalizeChatToolType(t)
			m["type"] = normalizedType
			if isResponsesBuiltInToolType(normalizedType) {
				delete(m, "function")
				delete(m, "name")
			}
		}
		return m
	}
}

func normalizeChatImageURLToString(v any) any {
	switch vv := v.(type) {
	case string:
		return vv
	case map[string]any:
		if url := common.Interface2String(vv["url"]); url != "" {
			return url
		}
		return v
	case dto.MessageImageUrl:
		if vv.Url != "" {
			return vv.Url
		}
		return v
	case *dto.MessageImageUrl:
		if vv != nil && vv.Url != "" {
			return vv.Url
		}
		return v
	default:
		return v
	}
}

func convertChatResponseFormatToResponsesText(reqFormat *dto.ResponseFormat) json.RawMessage {
	if reqFormat == nil || strings.TrimSpace(reqFormat.Type) == "" {
		return nil
	}

	format := map[string]any{
		"type": reqFormat.Type,
	}

	if reqFormat.Type == "json_schema" && len(reqFormat.JsonSchema) > 0 {
		var chatSchema map[string]any
		if err := common.Unmarshal(reqFormat.JsonSchema, &chatSchema); err == nil {
			for key, value := range chatSchema {
				if key == "type" {
					continue
				}
				format[key] = value
			}

			if nested, ok := format["json_schema"].(map[string]any); ok {
				for key, value := range nested {
					if _, exists := format[key]; !exists {
						format[key] = value
					}
				}
				delete(format, "json_schema")
			}
		} else {
			format["json_schema"] = reqFormat.JsonSchema
		}
	}

	textRaw, _ := common.Marshal(map[string]any{
		"format": format,
	})
	return textRaw
}

func ChatCompletionsRequestToResponsesRequest(req *dto.GeneralOpenAIRequest) (*dto.OpenAIResponsesRequest, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if req.Model == "" {
		return nil, errors.New("model is required")
	}
	if lo.FromPtrOr(req.N, 1) > 1 {
		return nil, fmt.Errorf("n>1 is not supported in responses compatibility mode")
	}

	var instructionsParts []string
	inputItems := make([]map[string]any, 0, len(req.Messages))

	for _, msg := range req.Messages {
		role := strings.TrimSpace(msg.Role)
		if role == "" {
			continue
		}

		if role == "tool" || role == "function" {
			callID := strings.TrimSpace(msg.ToolCallId)

			var output any
			if msg.Content == nil {
				output = ""
			} else if msg.IsStringContent() {
				output = msg.StringContent()
			} else {
				if b, err := common.Marshal(msg.Content); err == nil {
					output = string(b)
				} else {
					output = fmt.Sprintf("%v", msg.Content)
				}
			}

			if callID == "" {
				inputItems = append(inputItems, map[string]any{
					"role":    "user",
					"content": fmt.Sprintf("[tool_output_missing_call_id] %v", output),
				})
				continue
			}

			inputItems = append(inputItems, map[string]any{
				"type":    "function_call_output",
				"call_id": callID,
				"output":  output,
			})
			continue
		}

		// Prefer mapping system/developer messages into `instructions`.
		if role == "system" || role == "developer" {
			if msg.Content == nil {
				continue
			}
			if msg.IsStringContent() {
				if s := strings.TrimSpace(msg.StringContent()); s != "" {
					instructionsParts = append(instructionsParts, s)
				}
				continue
			}
			parts := msg.ParseContent()
			var sb strings.Builder
			for _, part := range parts {
				if part.Type == dto.ContentTypeText && strings.TrimSpace(part.Text) != "" {
					if sb.Len() > 0 {
						sb.WriteString("\n")
					}
					sb.WriteString(part.Text)
				}
			}
			if s := strings.TrimSpace(sb.String()); s != "" {
				instructionsParts = append(instructionsParts, s)
			}
			continue
		}

		item := map[string]any{
			"role": role,
		}

		if msg.Content == nil {
			item["content"] = ""
			inputItems = append(inputItems, item)

			if role == "assistant" {
				for _, tc := range msg.ParseToolCalls() {
					if strings.TrimSpace(tc.ID) == "" {
						continue
					}
					if tc.Type != "" && tc.Type != "function" {
						continue
					}
					name := strings.TrimSpace(tc.Function.Name)
					if name == "" {
						continue
					}
					inputItems = append(inputItems, map[string]any{
						"type":      "function_call",
						"call_id":   tc.ID,
						"name":      name,
						"arguments": tc.Function.Arguments,
					})
				}
			}
			continue
		}

		if msg.IsStringContent() {
			item["content"] = msg.StringContent()
			inputItems = append(inputItems, item)

			if role == "assistant" {
				for _, tc := range msg.ParseToolCalls() {
					if strings.TrimSpace(tc.ID) == "" {
						continue
					}
					if tc.Type != "" && tc.Type != "function" {
						continue
					}
					name := strings.TrimSpace(tc.Function.Name)
					if name == "" {
						continue
					}
					inputItems = append(inputItems, map[string]any{
						"type":      "function_call",
						"call_id":   tc.ID,
						"name":      name,
						"arguments": tc.Function.Arguments,
					})
				}
			}
			continue
		}

		parts := msg.ParseContent()
		contentParts := make([]map[string]any, 0, len(parts))
		for _, part := range parts {
			switch part.Type {
			case dto.ContentTypeText:
				textType := "input_text"
				if role == "assistant" {
					textType = "output_text"
				}
				contentParts = append(contentParts, map[string]any{
					"type": textType,
					"text": part.Text,
				})
			case dto.ContentTypeImageURL:
				contentParts = append(contentParts, map[string]any{
					"type":      "input_image",
					"image_url": normalizeChatImageURLToString(part.ImageUrl),
				})
			case dto.ContentTypeInputAudio:
				contentParts = append(contentParts, map[string]any{
					"type":        "input_audio",
					"input_audio": part.InputAudio,
				})
			case dto.ContentTypeFile:
				contentParts = append(contentParts, map[string]any{
					"type": "input_file",
					"file": part.File,
				})
			case dto.ContentTypeVideoUrl:
				contentParts = append(contentParts, map[string]any{
					"type":      "input_video",
					"video_url": part.VideoUrl,
				})
			default:
				contentParts = append(contentParts, map[string]any{
					"type": part.Type,
				})
			}
		}
		item["content"] = contentParts
		inputItems = append(inputItems, item)

		if role == "assistant" {
			for _, tc := range msg.ParseToolCalls() {
				if strings.TrimSpace(tc.ID) == "" {
					continue
				}
				if tc.Type != "" && tc.Type != "function" {
					continue
				}
				name := strings.TrimSpace(tc.Function.Name)
				if name == "" {
					continue
				}
				inputItems = append(inputItems, map[string]any{
					"type":      "function_call",
					"call_id":   tc.ID,
					"name":      name,
					"arguments": tc.Function.Arguments,
				})
			}
		}
	}

	inputRaw, err := common.Marshal(inputItems)
	if err != nil {
		return nil, err
	}

	var instructionsRaw json.RawMessage
	if len(instructionsParts) > 0 {
		instructions := strings.Join(instructionsParts, "\n\n")
		instructionsRaw, _ = common.Marshal(instructions)
	}

	var toolsRaw json.RawMessage
	if req.Tools != nil || shouldAutoEnableWebSearch(req) {
		tools := make([]map[string]any, 0, len(req.Tools)+1)
		if shouldAutoEnableWebSearch(req) && !hasResponsesBuiltInTool(req.Tools) {
			tools = append(tools, map[string]any{
				"type": "web_search",
			})
		}
		for _, tool := range req.Tools {
			normalizedType := normalizeChatToolType(tool.Type)
			switch {
			case isResponsesBuiltInToolType(normalizedType):
				tools = append(tools, map[string]any{
					"type": normalizedType,
				})
			case normalizedType == "function":
				tools = append(tools, map[string]any{
					"type":        "function",
					"name":        tool.Function.Name,
					"description": tool.Function.Description,
					"parameters":  tool.Function.Parameters,
				})
			default:
				// Best-effort: keep original tool shape for unknown types.
				var m map[string]any
				if b, err := common.Marshal(tool); err == nil {
					_ = common.Unmarshal(b, &m)
				}
				if len(m) == 0 {
					m = map[string]any{"type": normalizedType}
				} else if rawType, ok := m["type"].(string); ok {
					m["type"] = normalizeChatToolType(rawType)
				}
				if t, _ := m["type"].(string); isResponsesBuiltInToolType(t) {
					delete(m, "function")
					delete(m, "name")
				}
				tools = append(tools, m)
			}
		}
		toolsRaw, _ = common.Marshal(tools)
	}

	var toolChoiceRaw json.RawMessage
	if req.ToolChoice != nil {
		switch v := normalizeChatToolChoiceForResponses(req.ToolChoice).(type) {
		case string:
			toolChoiceRaw, _ = common.Marshal(v)
		default:
			var m map[string]any
			if b, err := common.Marshal(v); err == nil {
				_ = common.Unmarshal(b, &m)
			}
			if m == nil {
				toolChoiceRaw, _ = common.Marshal(v)
			} else if t, _ := m["type"].(string); t == "function" {
				// Chat: {"type":"function","function":{"name":"..."}}
				// Responses: {"type":"function","name":"..."}
				if name, ok := m["name"].(string); ok && name != "" {
					toolChoiceRaw, _ = common.Marshal(map[string]any{
						"type": "function",
						"name": name,
					})
				} else if fn, ok := m["function"].(map[string]any); ok {
					if name, ok := fn["name"].(string); ok && name != "" {
						toolChoiceRaw, _ = common.Marshal(map[string]any{
							"type": "function",
							"name": name,
						})
					} else {
						toolChoiceRaw, _ = common.Marshal(v)
					}
				} else {
					toolChoiceRaw, _ = common.Marshal(v)
				}
			} else {
				toolChoiceRaw, _ = common.Marshal(v)
			}
		}
	}

	var parallelToolCallsRaw json.RawMessage
	if req.ParallelTooCalls != nil {
		parallelToolCallsRaw, _ = common.Marshal(*req.ParallelTooCalls)
	}

	textRaw := convertChatResponseFormatToResponsesText(req.ResponseFormat)

	maxOutputTokens := lo.FromPtrOr(req.MaxTokens, uint(0))
	maxCompletionTokens := lo.FromPtrOr(req.MaxCompletionTokens, uint(0))
	if maxCompletionTokens > maxOutputTokens {
		maxOutputTokens = maxCompletionTokens
	}
	// OpenAI Responses API rejects max_output_tokens < 16 when explicitly provided.
	//if maxOutputTokens > 0 && maxOutputTokens < 16 {
	//	maxOutputTokens = 16
	//}

	var topP *float64
	if req.TopP != nil {
		topP = common.GetPointer(lo.FromPtr(req.TopP))
	}

	out := &dto.OpenAIResponsesRequest{
		Model:             req.Model,
		Input:             inputRaw,
		Instructions:      instructionsRaw,
		Stream:            req.Stream,
		Temperature:       req.Temperature,
		Text:              textRaw,
		ToolChoice:        toolChoiceRaw,
		Tools:             toolsRaw,
		TopP:              topP,
		User:              req.User,
		ParallelToolCalls: parallelToolCallsRaw,
		Store:             req.Store,
		Metadata:          req.Metadata,
	}
	if req.MaxTokens != nil || req.MaxCompletionTokens != nil {
		out.MaxOutputTokens = lo.ToPtr(maxOutputTokens)
	}

	if req.ReasoningEffort != "" {
		out.Reasoning = &dto.Reasoning{
			Effort:  req.ReasoningEffort,
			Summary: "detailed",
		}
	}

	return out, nil
}
