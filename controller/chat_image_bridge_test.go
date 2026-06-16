package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
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
	require.Equal(t, "1365x768", imageRequest.Size)
	require.NotNil(t, imageRequest.Width)
	require.NotNil(t, imageRequest.Height)
	require.Equal(t, 1365, *imageRequest.Width)
	require.Equal(t, 768, *imageRequest.Height)
	require.NotNil(t, imageRequest.N)
	require.Equal(t, uint(2), *imageRequest.N)
}

func TestShouldBridgeGPT54ImageIntentToMAIImage(t *testing.T) {
	c := gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	request := &dto.GeneralOpenAIRequest{
		Model:    "gpt-5.4-mini",
		Messages: []dto.Message{{Role: "user", Content: "生成一张1920x1080p的猫猫海报"}},
	}

	require.True(t, shouldBridgeChatImageRequest(c, types.RelayFormatOpenAI, request))

	imageRequest, _, err := buildImageRequestFromChatRequest(request)
	require.NoError(t, err)
	require.Equal(t, "MAI-Image-2.5", imageRequest.Model)
	require.Equal(t, "1365x768", imageRequest.Size)
	require.NotNil(t, imageRequest.Width)
	require.NotNil(t, imageRequest.Height)
}

func TestShouldNotBridgeGPT54NormalTextChat(t *testing.T) {
	c := gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	request := &dto.GeneralOpenAIRequest{
		Model:    "gpt-5.4",
		Messages: []dto.Message{{Role: "user", Content: "解释一下量子计算是什么"}},
	}

	require.False(t, shouldBridgeChatImageRequest(c, types.RelayFormatOpenAI, request))
}

func TestShouldNotBridgeGPT54ImagePromptWriting(t *testing.T) {
	c := gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	request := &dto.GeneralOpenAIRequest{
		Model:    "gpt-5.4-mini",
		Messages: []dto.Message{{Role: "user", Content: "帮我生成一段图片描述提示词"}},
	}

	require.False(t, shouldBridgeChatImageRequest(c, types.RelayFormatOpenAI, request))
}

func TestShouldBridgeGPT54FollowUpImageIntentAfterGeneratedImage(t *testing.T) {
	c := gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	request := &dto.GeneralOpenAIRequest{
		Model: "gpt-5.4",
		Messages: []dto.Message{
			{Role: "user", Content: "给我生成一个关于早餐店的海报模板"},
			{Role: "assistant", Content: "![generated image](data:image/png;base64,abc)\n\nA vertical breakfast shop poster template"},
			{Role: "user", Content: "再给我生成一个修理店模板 修车的"},
		},
	}

	require.True(t, shouldBridgeChatImageRequest(c, types.RelayFormatOpenAI, request))
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
	require.Equal(t, "1365x768", imageRequest.Size)
	require.NotNil(t, imageRequest.Width)
	require.NotNil(t, imageRequest.Height)
	require.Equal(t, 1365, *imageRequest.Width)
	require.Equal(t, 768, *imageRequest.Height)
}

func TestBuildImageRequestFromChatRequestDefaultsToSquareSize(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model:    "MAI-Image-2.5",
		Messages: []dto.Message{{Role: "user", Content: "画一只布偶猫头像"}},
	}

	imageRequest, _, err := buildImageRequestFromChatRequest(request)

	require.NoError(t, err)
	require.Equal(t, "1024x1024", imageRequest.Size)
	require.NotNil(t, imageRequest.Width)
	require.NotNil(t, imageRequest.Height)
	require.Equal(t, 1024, *imageRequest.Width)
	require.Equal(t, 1024, *imageRequest.Height)
}

func TestBuildImageRequestFromChatRequestDoesNotSendMAIDimensionsToOtherModels(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model:    "gpt-image-1",
		Messages: []dto.Message{{Role: "user", Content: "生成一个1920x1080p的猫猫壁纸"}},
	}

	imageRequest, _, err := buildImageRequestFromChatRequest(request)

	require.NoError(t, err)
	require.Equal(t, "1365x768", imageRequest.Size)
	require.Nil(t, imageRequest.Width)
	require.Nil(t, imageRequest.Height)
}

func TestBuildImageRequestFromChatRequestRejectsEmptyPrompt(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{
		Model:    "MAI-Image-2.5",
		Messages: []dto.Message{{Role: "user", Content: "   "}},
	}

	_, _, err := buildImageRequestFromChatRequest(request)

	require.ErrorContains(t, err, "prompt must be provided")
}
