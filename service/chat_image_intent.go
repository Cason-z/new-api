package service

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/dto"
)

const ChatImageIntentTargetModel = "MAI-Image-2.5"

func ShouldRouteChatImageIntent(chatRequest *dto.GeneralOpenAIRequest) bool {
	if chatRequest == nil || !IsChatImageIntentSourceModel(chatRequest.Model) {
		return false
	}
	prompt := strings.ToLower(ExtractLatestUserText(chatRequest.Messages))
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

func IsChatImageIntentSourceModel(model string) bool {
	normalized := strings.ToLower(strings.TrimSpace(model))
	return normalized == "gpt-5.4" || normalized == "gpt-5.4-mini" ||
		normalized == "gpt5.4" || normalized == "gpt5.4-mini"
}

func ExtractLatestUserText(messages []dto.Message) string {
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
