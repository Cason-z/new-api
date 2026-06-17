package openaicompat

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

func TestChatCompletionsRequestToResponsesRequest_NormalizesWebSearchPreview(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Model: "gpt-5.4",
		Messages: []dto.Message{
			{
				Role:    "user",
				Content: "上海今天新闻",
			},
		},
		Tools: []dto.ToolCallRequest{
			{Type: dto.BuildInToolWebSearchPreview},
		},
		ToolChoice: map[string]any{
			"type": dto.BuildInToolWebSearchPreview,
		},
	}

	respReq, err := ChatCompletionsRequestToResponsesRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var tools []map[string]any
	if err := common.Unmarshal(respReq.Tools, &tools); err != nil {
		t.Fatalf("failed to decode tools: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if got := tools[0]["type"]; got != "web_search" {
		t.Fatalf("expected tool type web_search, got %#v", got)
	}

	var toolChoice map[string]any
	if err := common.Unmarshal(respReq.ToolChoice, &toolChoice); err != nil {
		t.Fatalf("failed to decode tool_choice: %v", err)
	}
	if got := toolChoice["type"]; got != "web_search" {
		t.Fatalf("expected tool_choice type web_search, got %#v", got)
	}
}
