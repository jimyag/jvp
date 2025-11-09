package ginx

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// isXMLRequest 检查请求是否为 XML 格式
func isXMLRequest(ctx *gin.Context) bool {
	contentType := ctx.GetHeader("Content-Type")
	return strings.Contains(contentType, "application/xml") ||
		strings.Contains(contentType, "text/xml")
}

// bindArgs 绑定请求参数到 args 结构体
// 优先级：XML/JSON Body（根据 Content-Type）> URI 参数 > Query 参数 > Form 参数
// 默认使用 JSON，如果 Content-Type 包含 xml，则使用 XML
func bindArgs(ctx *gin.Context, args interface{}) error {
	// 1. 尝试从 XML/JSON body 绑定
	// 直接尝试绑定，不依赖 ContentLength（因为 ContentLength 可能不准确）
	// 根据 Content-Type 决定使用 XML 还是 JSON
	if isXMLRequest(ctx) {
		if err := ctx.ShouldBindXML(args); err == nil {
			// XML 绑定成功，同时尝试绑定 URI 和 Query 参数
			_ = ctx.ShouldBindUri(args)
			_ = ctx.ShouldBindQuery(args)
			// 标记使用 XML 格式
			setResponseFormat(ctx, "xml")
			return nil
		}
	} else {
		// 默认使用 JSON
		if err := ctx.ShouldBindJSON(args); err == nil {
			// JSON 绑定成功，同时尝试绑定 URI 和 Query 参数
			_ = ctx.ShouldBindUri(args)
			_ = ctx.ShouldBindQuery(args)
			// 标记使用 JSON 格式
			setResponseFormat(ctx, "json")
			return nil
		}
	}

	// 2. 尝试从 URI 参数绑定
	if err := ctx.ShouldBindUri(args); err == nil {
		// URI 绑定成功，同时绑定 Query 参数
		_ = ctx.ShouldBindQuery(args)
		// 默认使用 JSON
		setResponseFormat(ctx, "json")
		return nil
	}

	// 3. 尝试从 Query 参数绑定
	if err := ctx.ShouldBindQuery(args); err == nil {
		// 默认使用 JSON
		setResponseFormat(ctx, "json")
		return nil
	}

	// 4. 尝试从 Form 绑定
	// 默认使用 JSON
	setResponseFormat(ctx, "json")
	return ctx.ShouldBind(args)
}
