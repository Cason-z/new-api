package controller

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

const (
	chatImageBridgeContextKey       = "chat_image_bridge"
	chatImageBridgeStreamContextKey = "chat_image_bridge_stream"
)

func shouldBridgeChatImageRequest(c *gin.Context, relayFormat types.RelayFormat, request dto.Request) bool {
	if relayFormat != types.RelayFormatOpenAI {
		return false
	}
	if relayconstant.Path2RelayMode(c.Request.URL.Path) != relayconstant.RelayModeChatCompletions {
		return false
	}
	chatRequest, ok := request.(*dto.GeneralOpenAIRequest)
	return ok && common.IsImageGenerationModel(chatRequest.Model)
}

func buildImageRequestFromChatRequest(chatRequest *dto.GeneralOpenAIRequest) (*dto.ImageRequest, bool, error) {
	if chatRequest == nil {
		return nil, false, errors.New("request is nil")
	}

	prompt := extractImagePromptFromChatMessages(chatRequest.Messages)
	if strings.TrimSpace(prompt) == "" {
		return nil, false, errors.New("prompt must be provided, and must be a non empty string")
	}

	n := uint(1)
	if chatRequest.N != nil && *chatRequest.N > 0 {
		n = uint(*chatRequest.N)
	}

	size := chatRequest.Size
	if size == "" {
		size = "1024x1024"
	}

	imageRequest := &dto.ImageRequest{
		Model:  chatRequest.Model,
		Prompt: strings.TrimSpace(prompt),
		N:      common.GetPointer(n),
		Size:   size,
		User:   chatRequest.User,
	}
	return imageRequest, lo.FromPtrOr(chatRequest.Stream, false), nil
}

func extractImagePromptFromChatMessages(messages []dto.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "user" {
			continue
		}
		if text := messageContentText(messages[i]); strings.TrimSpace(text) != "" {
			return text
		}
	}
	if len(messages) == 0 {
		return ""
	}
	return messageContentText(messages[len(messages)-1])
}

func messageContentText(message dto.Message) string {
	switch content := message.Content.(type) {
	case string:
		return content
	case []any:
		var parts []string
		for _, item := range content {
			if itemMap, ok := item.(map[string]any); ok && itemMap["type"] == dto.ContentTypeText {
				if text, ok := itemMap["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	case []dto.MediaContent:
		var parts []string
		for _, item := range content {
			if item.Type == dto.ContentTypeText {
				parts = append(parts, item.Text)
			}
		}
		return strings.Join(parts, "\n")
	default:
		if content == nil {
			return ""
		}
		return fmt.Sprintf("%v", content)
	}
}
