package service

import (
	"fmt"
	"slices"
	"strings"

	"github.com/QuantumNous/new-api/dto"
)

const ChatImageIntentTargetModel = "MAI-Image-2.5"

var (
	chatImageToolSignals = []string{
		"image_generation",
		"image_generation_call",
		"gpt-image",
		"dall-e",
		"mai-image",
	}
	chatImageActionKeywords = []string{
		"生成", "画", "绘制", "设计", "制作", "创建", "改图", "编辑图片", "修图", "以图生图", "做",
		"generate", "draw", "create", "design", "make", "edit image", "image edit",
	}
	chatImageObjectKeywords = []string{
		"图片", "图像", "照片", "海报", "插画", "logo", "封面图", "封面", "概念图", "表情包", "壁纸", "头像",
		"image", "picture", "photo", "poster", "illustration", "cover", "concept art", "meme", "wallpaper", "avatar",
	}
	chatImageEditKeywords = []string{
		"改", "改成", "改为", "编辑", "修", "修图", "重绘", "换成", "转成", "变成",
		"edit", "modify", "redraw", "restyle", "transform", "turn into",
	}
	chatImageFollowUpKeywords = []string{
		"再", "再来", "再给我", "换一个", "来一个", "类似", "同款", "同风格", "这个风格", "这种风格", "模板",
		"another", "one more", "similar", "same style", "template",
	}
)

func ShouldRouteChatImageIntent(chatRequest *dto.GeneralOpenAIRequest) bool {
	if chatRequest == nil || !IsChatImageIntentSourceModel(chatRequest.Model) {
		return false
	}
	if hasRequiredImageToolChoice(chatRequest) {
		return true
	}

	latestUserMessage, ok := latestUserMessage(chatRequest.Messages)
	if !ok {
		return false
	}

	prompt := strings.ToLower(messageContentText(latestUserMessage))
	if strings.TrimSpace(prompt) == "" {
		return false
	}
	if isImagePromptWritingRequest(prompt) {
		return false
	}
	if messageHasImageInput(latestUserMessage) && hasAnyKeyword(prompt, chatImageEditKeywords) {
		return true
	}
	if hasRecentImageGenerationContext(chatRequest.Messages) && isFollowUpImageIntent(prompt) {
		return true
	}
	if hasImageIntentToolDefinition(chatRequest) && isTextImageIntent(prompt) {
		return true
	}
	return isTextImageIntent(prompt)
}

func isTextImageIntent(prompt string) bool {
	return hasAnyKeyword(prompt, chatImageActionKeywords) && hasAnyKeyword(prompt, chatImageObjectKeywords)
}

func isFollowUpImageIntent(prompt string) bool {
	return hasAnyKeyword(prompt, chatImageActionKeywords) && hasAnyKeyword(prompt, chatImageFollowUpKeywords)
}

func IsChatImageIntentSourceModel(model string) bool {
	normalized := strings.ToLower(strings.TrimSpace(model))
	return normalized == "gpt-5.4" || normalized == "gpt-5.4-mini" ||
		normalized == "gpt5.4" || normalized == "gpt5.4-mini"
}

func ExtractLatestUserText(messages []dto.Message) string {
	if message, ok := latestUserMessage(messages); ok {
		if text := messageContentText(message); strings.TrimSpace(text) != "" {
			return text
		}
	}
	if len(messages) == 0 {
		return ""
	}
	return messageContentText(messages[len(messages)-1])
}

func latestUserMessage(messages []dto.Message) (dto.Message, bool) {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i], true
		}
	}
	return dto.Message{}, false
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

func hasRecentImageGenerationContext(messages []dto.Message) bool {
	for i := len(messages) - 1; i >= 0; i-- {
		text := strings.ToLower(messageContentText(messages[i]))
		if text == "" {
			continue
		}
		if strings.Contains(text, "![generated image](") || strings.Contains(text, "data:image/") {
			return true
		}
		if messages[i].Role == "user" && isTextImageIntent(text) {
			return true
		}
	}
	return false
}

func hasRequiredImageToolChoice(chatRequest *dto.GeneralOpenAIRequest) bool {
	switch choice := chatRequest.ToolChoice.(type) {
	case map[string]any:
		if isImageToolMap(choice) {
			return true
		}
		if functionMap, ok := choice["function"].(map[string]any); ok && isImageToolMap(functionMap) {
			return true
		}
	case string:
		if strings.EqualFold(choice, "required") && len(chatRequest.Tools) == 1 {
			return isImageTool(chatRequest.Tools[0])
		}
	}
	return false
}

func hasImageIntentToolDefinition(chatRequest *dto.GeneralOpenAIRequest) bool {
	for _, tool := range chatRequest.Tools {
		if isImageTool(tool) {
			return true
		}
	}
	return false
}

func isImageTool(tool dto.ToolCallRequest) bool {
	if containsImageToolSignal(tool.Type) || containsImageToolSignal(tool.Function.Name) || containsImageToolSignal(tool.Function.Description) {
		return true
	}
	if tool.Function.Parameters != nil && containsImageToolSignal(fmt.Sprintf("%v", tool.Function.Parameters)) {
		return true
	}
	return false
}

func isImageToolMap(tool map[string]any) bool {
	for _, key := range []string{"type", "name", "description"} {
		if containsImageToolSignal(fmt.Sprintf("%v", tool[key])) {
			return true
		}
	}
	return false
}

func containsImageToolSignal(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return false
	}
	return slices.ContainsFunc(chatImageToolSignals, func(signal string) bool {
		return strings.Contains(normalized, signal)
	})
}

func messageHasImageInput(message dto.Message) bool {
	for _, item := range message.ParseContent() {
		if item.Type == dto.ContentTypeImageURL {
			return true
		}
	}
	return false
}

func hasAnyKeyword(text string, keywords []string) bool {
	return slices.ContainsFunc(keywords, func(keyword string) bool {
		return strings.Contains(text, keyword)
	})
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
