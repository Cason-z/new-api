package openaicompat

import (
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

func ShouldChatCompletionsUseResponsesPolicy(policy model_setting.ChatCompletionsToResponsesPolicy, channelID int, channelType int, model string) bool {
	if !policy.IsChannelEnabled(channelID, channelType) {
		return false
	}
	return matchAnyRegex(policy.ModelPatterns, model)
}

func ShouldChatCompletionsUseResponsesGlobal(channelID int, channelType int, model string) bool {
	return ShouldChatCompletionsUseResponsesPolicy(
		model_setting.GetGlobalSettings().ChatCompletionsToResponsesPolicy,
		channelID,
		channelType,
		model,
	)
}

func ShouldChatCompletionsUseResponsesForRequest(req *dto.GeneralOpenAIRequest) bool {
	if req == nil {
		return false
	}
	for _, tool := range req.Tools {
		switch strings.TrimSpace(tool.Type) {
		case dto.BuildInToolWebSearchPreview, "web_search":
			return true
		}
	}
	return shouldAutoEnableWebSearch(req)
}

func shouldAutoEnableWebSearch(req *dto.GeneralOpenAIRequest) bool {
	if req == nil {
		return false
	}
	if req.WebSearchOptions != nil {
		return true
	}
	model := strings.ToLower(strings.TrimSpace(req.Model))
	if !strings.HasPrefix(model, "gpt-5.4") {
		return false
	}
	latestUserText := strings.ToLower(strings.TrimSpace(latestUserText(req.Messages)))
	if latestUserText == "" {
		return false
	}
	return hasSearchIntent(latestUserText)
}

func latestUserText(messages []dto.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if strings.TrimSpace(msg.Role) != "user" || msg.Content == nil {
			continue
		}
		if msg.IsStringContent() {
			return msg.StringContent()
		}
		var parts []string
		for _, part := range msg.ParseContent() {
			if part.Type == dto.ContentTypeText && strings.TrimSpace(part.Text) != "" {
				parts = append(parts, part.Text)
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

func hasSearchIntent(text string) bool {
	alwaysSearchTerms := []string{
		"联网", "实时", "搜索", "搜一下", "查一下", "检索", "新闻", "热搜",
		"搜下", "查询", "查找", "天气", "气温", "温度", "降雨", "下雨", "空气质量", "aqi",
		"航班", "汇率", "股价", "价格", "赛程", "比分",
		"web search", "search the web", "browse", "internet", "online",
		"weather", "temperature", "forecast", "stock price", "exchange rate",
	}
	for _, term := range alwaysSearchTerms {
		if strings.Contains(text, term) {
			return true
		}
	}

	timeTerms := []string{
		"今天", "今日", "现在", "当前", "最新", "最近", "刚刚", "本周", "这周", "今年",
		"today", "latest", "recent", "current", "now", "this week", "this year",
	}
	updateTerms := []string{
		"更新", "进展", "动态", "发生", "发布", "release", "releases", "commit", "commits", "pull request", "pr",
		"update", "updates", "changed", "changes", "what's new", "whats new",
	}
	hasTime := false
	for _, term := range timeTerms {
		if strings.Contains(text, term) {
			hasTime = true
			break
		}
	}
	if !hasTime {
		return false
	}
	for _, term := range updateTerms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}
