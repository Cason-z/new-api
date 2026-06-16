package service

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestShouldRouteChatImageIntentForLogoRequest(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model:    "gpt-5.4-mini",
		Messages: []dto.Message{{Role: "user", Content: "做一个极简科技公司 logo"}}}

	require.True(t, ShouldRouteChatImageIntent(request))
}

func TestShouldRouteChatImageIntentForImageEditWithAttachedImage(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model: "gpt-5.4",
		Messages: []dto.Message{
			{
				Role: "user",
				Content: []any{
					map[string]any{"type": dto.ContentTypeImageURL, "image_url": map[string]any{"url": "https://example.com/cat.png"}},
					map[string]any{"type": dto.ContentTypeText, "text": "把这张图改成插画风格"},
				},
			},
		},
	}

	require.True(t, ShouldRouteChatImageIntent(request))
}

func TestShouldRouteChatImageIntentForRequiredImageToolChoice(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model: "gpt-5.4-mini",
		Messages: []dto.Message{
			{Role: "user", Content: "请直接生成成品"},
		},
		ToolChoice: "required",
		Tools: []dto.ToolCallRequest{
			{
				Type: "image_generation",
				Function: dto.FunctionRequest{
					Name: "image_generation",
				},
			},
		},
	}

	require.True(t, ShouldRouteChatImageIntent(request))
}

func TestShouldNotRouteChatImageIntentForPosterCopywriting(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model:    "gpt-5.4-mini",
		Messages: []dto.Message{{Role: "user", Content: "帮我写一个海报文案"}}}

	require.False(t, ShouldRouteChatImageIntent(request))
}

func TestShouldNotRouteChatImageIntentForConceptArtExplanation(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model:    "gpt-5.4",
		Messages: []dto.Message{{Role: "user", Content: "解释一下什么是概念图"}}}

	require.False(t, ShouldRouteChatImageIntent(request))
}

func TestShouldRouteChatImageIntentForFollowUpTemplateAfterGeneratedImage(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model: "gpt-5.4",
		Messages: []dto.Message{
			{Role: "user", Content: "给我生成一个关于早餐店的海报模板"},
			{Role: "assistant", Content: "![generated image](data:image/png;base64,abc)\n\nA vertical breakfast shop poster template"},
			{Role: "user", Content: "再给我生成一个修理店模板 修车的"},
		},
	}

	require.True(t, ShouldRouteChatImageIntent(request))
}

func TestShouldNotRouteChatImageIntentForFollowUpNonVisualTextRequest(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model: "gpt-5.4",
		Messages: []dto.Message{
			{Role: "user", Content: "给我生成一个关于早餐店的海报模板"},
			{Role: "assistant", Content: "![generated image](data:image/png;base64,abc)\n\nA vertical breakfast shop poster template"},
			{Role: "user", Content: "再给我生成一个早餐店营销方案"},
		},
	}

	require.False(t, ShouldRouteChatImageIntent(request))
}
