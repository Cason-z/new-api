package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func TestRewriteChatImageIntentModelRoutesGPT54MiniToMAI(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	body := `{"model":"gpt-5.4-mini","messages":[{"role":"user","content":"生成一张1920x1080p的猫猫海报"}]}`
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	got := rewriteChatImageIntentModel(c, "gpt-5.4-mini")

	if got != service.ChatImageIntentTargetModel {
		t.Fatalf("rewriteChatImageIntentModel() = %q, want %q", got, service.ChatImageIntentTargetModel)
	}
}

func TestRewriteChatImageIntentModelKeepsNormalGPT54MiniChat(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	body := `{"model":"gpt-5.4-mini","messages":[{"role":"user","content":"解释一下量子计算是什么"}]}`
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	got := rewriteChatImageIntentModel(c, "gpt-5.4-mini")

	if got != "gpt-5.4-mini" {
		t.Fatalf("rewriteChatImageIntentModel() = %q, want %q", got, "gpt-5.4-mini")
	}
}
