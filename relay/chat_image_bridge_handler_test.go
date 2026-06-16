package relay

import (
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestWriteChatImageJSONResponseUsesClientRequestedModel(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	common.SetContextKey(c, constant.ContextKeyClientRequestedModel, "gpt-5.4-mini")

	info := &relaycommon.RelayInfo{OriginModelName: "MAI-Image-2.5"}
	writeChatImageJSONResponse(c, info, "![generated image](https://example.com/cat.png)", dto.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2})

	var response dto.OpenAITextResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "gpt-5.4-mini", response.Model)
	require.Equal(t, "chat.completion", response.Object)
}

func TestWriteChatImageStreamResponseUsesClientRequestedModel(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	common.SetContextKey(c, constant.ContextKeyClientRequestedModel, "gpt-5.4")

	info := &relaycommon.RelayInfo{OriginModelName: "MAI-Image-2.5"}
	writeChatImageStreamResponse(c, info, "![generated image](https://example.com/cat.png)", dto.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2})

	body := recorder.Body.String()
	require.Contains(t, body, `"model":"gpt-5.4"`)
	require.NotContains(t, body, `"model":"MAI-Image-2.5"`)
	require.True(t, strings.Contains(recorder.Header().Get("Content-Type"), "text/event-stream"))
}

func TestImageResponseToMarkdownUsesDetectedMimeType(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	var buffer strings.Builder
	encoder := base64.NewEncoder(base64.StdEncoding, &buffer)
	require.NoError(t, jpeg.Encode(encoder, img, nil))
	require.NoError(t, encoder.Close())

	markdown := imageResponseToMarkdown(dto.ImageResponse{
		Data: []dto.ImageData{
			{B64Json: buffer.String()},
		},
	})

	require.Contains(t, markdown, "data:image/jpeg;base64,")
	require.NotContains(t, markdown, "data:image/png;base64,")
}
