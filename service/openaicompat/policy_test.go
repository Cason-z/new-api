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

func TestShouldChatCompletionsUseResponsesForRequest_AutoWebSearchIntent(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Model: "gpt-5.4",
		Messages: []dto.Message{
			{Role: "user", Content: "GitHub skills 今天有哪些更新"},
		},
	}

	if !ShouldChatCompletionsUseResponsesForRequest(req) {
		t.Fatal("expected current-update request to use responses path")
	}
}

func TestShouldChatCompletionsUseResponsesForRequest_NoAutoWebSearchForNormalChat(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Model: "gpt-5.4",
		Messages: []dto.Message{
			{Role: "user", Content: "帮我解释一下 GitHub skills 是什么"},
		},
	}

	if ShouldChatCompletionsUseResponsesForRequest(req) {
		t.Fatal("did not expect normal explanation request to force responses path")
	}
}
