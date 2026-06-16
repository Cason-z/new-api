package relay

import (
	"encoding/json"
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
