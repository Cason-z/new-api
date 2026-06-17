package openaicompat

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
)

func TestShouldChatCompletionsUseResponsesForRequest_WebSearchPreview(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Tools: []dto.ToolCallRequest{
			{Type: dto.BuildInToolWebSearchPreview},
		},
	}

	if !ShouldChatCompletionsUseResponsesForRequest(req) {
		t.Fatal("expected web_search_preview request to use responses path")
	}
}

func TestShouldChatCompletionsUseResponsesForRequest_FunctionOnly(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Tools: []dto.ToolCallRequest{
			{Type: "function", Function: dto.FunctionRequest{Name: "lookup_weather"}},
		},
	}

	if ShouldChatCompletionsUseResponsesForRequest(req) {
		t.Fatal("did not expect plain function tools to force responses path")
	}
}
