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
	return false
}
