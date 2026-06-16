package controller

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
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
	chatImageBridgeTargetModel      = "MAI-Image-2.5"
	defaultImageBridgeSize          = "1024x1024"
	landscapeImageBridgeSize        = "1365x768"
	portraitImageBridgeSize         = "768x1365"
)

var imageBridgeSizePattern = regexp.MustCompile(`(?i)(\d{3,4})\s*[x×]\s*(\d{3,4})`)

func shouldBridgeChatImageRequest(c *gin.Context, relayFormat types.RelayFormat, request dto.Request) bool {
	if relayFormat != types.RelayFormatOpenAI {
		return false
	}
	if relayconstant.Path2RelayMode(c.Request.URL.Path) != relayconstant.RelayModeChatCompletions {
		return false
	}
	chatRequest, ok := request.(*dto.GeneralOpenAIRequest)
	if !ok {
		return false
	}
	return common.IsImageGenerationModel(chatRequest.Model) || shouldRouteChatImageIntentToBridge(chatRequest)
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
		size = inferImageSizeFromPrompt(prompt)
	}
	var width, height *int
	if shouldSendImageBridgeDimensions(chatRequest.Model) {
		width, height = imageDimensionsFromSize(size)
	}

	model := chatRequest.Model
	if shouldRouteChatImageIntentToBridge(chatRequest) {
		model = chatImageBridgeTargetModel
		width, height = imageDimensionsFromSize(size)
	}

	imageRequest := &dto.ImageRequest{
		Model:  model,
		Prompt: strings.TrimSpace(prompt),
		N:      common.GetPointer(n),
		Size:   size,
		Width:  width,
		Height: height,
		User:   chatRequest.User,
	}
	return imageRequest, lo.FromPtrOr(chatRequest.Stream, false), nil
}

func shouldRouteChatImageIntentToBridge(chatRequest *dto.GeneralOpenAIRequest) bool {
	if chatRequest == nil || !isChatImageIntentSourceModel(chatRequest.Model) {
		return false
	}
	prompt := strings.ToLower(extractImagePromptFromChatMessages(chatRequest.Messages))
	if strings.TrimSpace(prompt) == "" {
		return false
	}
	if isImagePromptWritingRequest(prompt) {
		return false
	}

	imageActions := []string{
		"生成", "画", "绘制", "设计", "制作", "创建", "改图", "编辑图片", "修图", "以图生图",
		"generate", "draw", "create", "design", "make", "edit image", "image edit",
	}
	imageObjects := []string{
		"图片", "图像", "照片", "海报", "插画", "logo", "封面图", "封面", "概念图", "表情包", "壁纸", "头像",
		"image", "picture", "photo", "poster", "illustration", "cover", "concept art", "meme", "wallpaper", "avatar",
	}
	for _, action := range imageActions {
		if !strings.Contains(prompt, action) {
			continue
		}
		for _, object := range imageObjects {
			if strings.Contains(prompt, object) {
				return true
			}
		}
	}
	return false
}

func isImagePromptWritingRequest(prompt string) bool {
	writingTargets := []string{"提示词", "prompt", "描述", "文案"}
	writingActions := []string{"写", "生成一段", "优化", "润色", "改写", "翻译", "整理"}
	for _, target := range writingTargets {
		if !strings.Contains(prompt, target) {
			continue
		}
		for _, action := range writingActions {
			if strings.Contains(prompt, action) {
				return true
			}
		}
	}
	return false
}

func isChatImageIntentSourceModel(model string) bool {
	normalized := strings.ToLower(strings.TrimSpace(model))
	return normalized == "gpt-5.4" || normalized == "gpt-5.4-mini" ||
		normalized == "gpt5.4" || normalized == "gpt5.4-mini"
}

func shouldSendImageBridgeDimensions(model string) bool {
	return strings.HasPrefix(strings.ToLower(model), "mai-image-")
}

func inferImageSizeFromPrompt(prompt string) string {
	normalized := strings.ToLower(prompt)
	if match := imageBridgeSizePattern.FindStringSubmatch(normalized); len(match) == 3 {
		width, _ := strconv.Atoi(match[1])
		height, _ := strconv.Atoi(match[2])
		if width > height {
			return landscapeImageBridgeSize
		}
		if height > width {
			return portraitImageBridgeSize
		}
	}

	if strings.Contains(normalized, "16:9") ||
		strings.Contains(normalized, "1080p") ||
		strings.Contains(normalized, "横屏") ||
		strings.Contains(normalized, "宽屏") ||
		strings.Contains(normalized, "壁纸") ||
		strings.Contains(normalized, "wallpaper") {
		return landscapeImageBridgeSize
	}

	if strings.Contains(normalized, "9:16") ||
		strings.Contains(normalized, "竖屏") ||
		strings.Contains(normalized, "手机壁纸") ||
		strings.Contains(normalized, "mobile wallpaper") {
		return portraitImageBridgeSize
	}

	return defaultImageBridgeSize
}

func imageDimensionsFromSize(size string) (*int, *int) {
	match := imageBridgeSizePattern.FindStringSubmatch(size)
	if len(match) != 3 {
		return nil, nil
	}
	width, err := strconv.Atoi(match[1])
	if err != nil {
		return nil, nil
	}
	height, err := strconv.Atoi(match[2])
	if err != nil {
		return nil, nil
	}
	return common.GetPointer(width), common.GetPointer(height)
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
