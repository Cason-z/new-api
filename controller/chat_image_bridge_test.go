package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestBuildImageRequestFromChatRequestUsesLastUserMessage(t *testing.T) {
	stream := true
	n := 2
	request := &dto.GeneralOpenAIRequest{
		Model:  "MAI-Image-2.5",
		Stream: &stream,
		N:      &n,
		Messages: []dto.Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "生成一个1920x1080p的猫猫壁纸 要求布偶猫"},
			{Role: "assistant", Content: "处理中"},
		},
	}

	imageRequest, originalStream, err := buildImageRequestFromChatRequest(request)

	require.NoError(t, err)
	require.True(t, originalStream)
	require.Equal(t, "MAI-Image-2.5", imageRequest.Model)
	require.Equal(t, "生成一个1920x1080p的猫猫壁纸 要求布偶猫", imageRequest.Prompt)
	require.Equal(t, "1024x1024", imageRequest.Size)
	require.NotNil(t, imageRequest.N)
	require.Equal(t, uint(2), *imageRequest.N)
}

func TestBuildImageRequestFromChatRequestReadsArrayTextContent(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model: "MAI-Image-2.5",
		Messages: []dto.Message{
			{
				Role: "user",
				Content: []any{
					map[string]any{"type": "text", "text": "画一只布偶猫"},
					map[string]any{"type": "text", "text": "16:9 壁纸"},
				},
			},
		},
	}

	imageRequest, originalStream, err := buildImageRequestFromChatRequest(request)

	require.NoError(t, err)
	require.False(t, originalStream)
	require.Equal(t, "画一只布偶猫\n16:9 壁纸", imageRequest.Prompt)
}

func TestBuildImageRequestFromChatRequestRejectsEmptyPrompt(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model:    "MAI-Image-2.5",
		Messages: []dto.Message{{Role: "user", Content: "   "}},
	}

	_, _, err := buildImageRequestFromChatRequest(request)

	require.ErrorContains(t, err, "prompt must be provided")
}
