package ginx

import (
	"github.com/gin-gonic/gin"
)

// contextKey 用于在 gin.Context 中存储值的类型安全 key
type contextKey struct{}

// responseFormatKey 用于存储响应格式（"json" 或 "xml"）
var responseFormatKey = contextKey{}

// setResponseFormat 设置响应格式
func setResponseFormat(ctx *gin.Context, format string) {
	ctx.Set(responseFormatKey, format)
}

// getResponseFormat 获取响应格式，如果不存在则返回默认值
func getResponseFormat(ctx *gin.Context) string {
	format, exists := ctx.Get(responseFormatKey)
	if !exists {
		return "json" // 默认使用 JSON
	}
	if str, ok := format.(string); ok {
		return str
	}
	return "json"
}
