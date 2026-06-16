package relay

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func ChatImageBridgeHelper(c *gin.Context, info *relaycommon.RelayInfo, streamChatResponse bool) (newAPIError *types.NewAPIError) {
	info.InitChannelMeta(c)

	imageReq, ok := info.Request.(*dto.ImageRequest)
	if !ok {
		return types.NewErrorWithStatusCode(fmt.Errorf("invalid request type, expected dto.ImageRequest, got %T", info.Request), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	request, err := common.DeepCopy(imageReq)
	if err != nil {
		return types.NewError(fmt.Errorf("failed to copy request to ImageRequest: %w", err), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}
	request.Stream = nil

	if err = helper.ModelMappedHelper(c, info, request); err != nil {
		return types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
	}

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	convertedRequest, err := adaptor.ConvertImageRequest(c, info, *request)
	if err != nil {
		return types.NewError(err, types.ErrorCodeConvertRequestFailed)
	}
	relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)

	jsonData, err := common.Marshal(convertedRequest)
	if err != nil {
		return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}

	if len(info.ParamOverride) > 0 {
		jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
		if err != nil {
			return newAPIErrorFromParamOverride(err)
		}
	}

	body, size, closer, err := relaycommon.NewOutboundJSONBody(jsonData)
	if err != nil {
		return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}
	defer closer.Close()
	info.UpstreamRequestBodySize = size

	resp, err := adaptor.DoRequest(c, info, body)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}

	httpResp, ok := resp.(*http.Response)
	if !ok || httpResp == nil {
		return types.NewOpenAIError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}
	defer service.CloseResponseBodyGracefully(httpResp)

	statusCodeMappingStr := c.GetString("status_code_mapping")
	if httpResp.StatusCode != http.StatusOK {
		newAPIError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
		service.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return newAPIError
	}

	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	var imageResp dto.ImageResponse
	if err = common.Unmarshal(responseBody, &imageResp); err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	var usageResp dto.SimpleResponse
	_ = common.Unmarshal(responseBody, &usageResp)
	if oaiError := usageResp.GetOpenAIError(); oaiError != nil && oaiError.Type != "" {
		return types.WithOpenAIError(*oaiError, httpResp.StatusCode)
	}

	markdown := imageResponseToMarkdown(imageResp)
	if markdown == "" {
		return types.NewOpenAIError(fmt.Errorf("image response does not contain url or b64_json"), types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	usage := usageResp.Usage
	if usage.TotalTokens == 0 {
		usage.TotalTokens = 1
	}
	if usage.PromptTokens == 0 {
		usage.PromptTokens = 1
	}

	if streamChatResponse {
		writeChatImageStreamResponse(c, info, markdown, usage)
	} else {
		writeChatImageJSONResponse(c, info, markdown, usage)
	}

	imageN := uint(1)
	if request.N != nil {
		imageN = *request.N
	}
	if info.PriceData.UsePrice {
		if _, hasN := info.PriceData.OtherRatios["n"]; !hasN {
			info.PriceData.AddOtherRatio("n", float64(imageN))
		}
	}

	var logContent []string
	if request.Size != "" {
		logContent = append(logContent, fmt.Sprintf("大小 %s", request.Size))
	}
	if request.Quality != "" {
		logContent = append(logContent, fmt.Sprintf("品质 %s", request.Quality))
	}
	if imageN > 0 {
		logContent = append(logContent, fmt.Sprintf("生成数量 %d", imageN))
	}
	service.PostTextConsumeQuota(c, info, &usage, logContent)
	return nil
}

func imageResponseToMarkdown(response dto.ImageResponse) string {
	var parts []string
	for _, image := range response.Data {
		imageURL := image.Url
		if imageURL == "" && image.B64Json != "" {
			imageURL = "data:image/png;base64," + image.B64Json
		}
		if imageURL == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("![generated image](%s)", imageURL))
		if image.RevisedPrompt != "" {
			parts = append(parts, image.RevisedPrompt)
		}
	}
	return strings.Join(parts, "\n\n")
}

func writeChatImageJSONResponse(c *gin.Context, info *relaycommon.RelayInfo, content string, usage dto.Usage) {
	created := time.Now().Unix()
	if info != nil {
		info.SetFirstResponseTime()
	}
	model := getChatImageResponseModel(c, info)
	c.JSON(http.StatusOK, dto.OpenAITextResponse{
		Id:      helper.GetResponseID(c),
		Object:  "chat.completion",
		Created: created,
		Model:   model,
		Choices: []dto.OpenAITextResponseChoice{
			{
				Index: 0,
				Message: dto.Message{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: constant.FinishReasonStop,
			},
		},
		Usage: usage,
	})
}

func writeChatImageStreamResponse(c *gin.Context, info *relaycommon.RelayInfo, content string, usage dto.Usage) {
	id := helper.GetResponseID(c)
	created := time.Now().Unix()
	if info != nil {
		info.SetFirstResponseTime()
	}
	model := getChatImageResponseModel(c, info)

	helper.SetEventStreamHeaders(c)
	c.Status(http.StatusOK)
	_ = helper.ObjectData(c, helper.GenerateStartEmptyResponse(id, created, model, nil))
	_ = helper.ObjectData(c, &dto.ChatCompletionsStreamResponse{
		Id:      id,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   model,
		Choices: []dto.ChatCompletionsStreamResponseChoice{
			{
				Index: 0,
				Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
					Content: common.GetPointer(content),
				},
			},
		},
	})
	_ = helper.ObjectData(c, helper.GenerateStopResponse(id, created, model, constant.FinishReasonStop))
	_ = helper.ObjectData(c, helper.GenerateFinalUsageResponse(id, created, model, usage))
	helper.Done(c)
}

func getChatImageResponseModel(c *gin.Context, info *relaycommon.RelayInfo) string {
	if model := common.GetContextKeyString(c, constant.ContextKeyClientRequestedModel); model != "" {
		return model
	}
	if info != nil {
		return info.OriginModelName
	}
	return ""
}
