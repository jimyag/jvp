package ginx

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// isXMLResponse 检查是否应该使用 XML 格式响应
func isXMLResponse(ctx *gin.Context) bool {
	format := getResponseFormat(ctx)
	if format == "xml" {
		return true
	}
	// 如果没有设置，检查 Accept header
	accept := ctx.GetHeader("Accept")
	return strings.Contains(accept, "application/xml") ||
		strings.Contains(accept, "text/xml")
}

// renderResponse 渲染响应
// 根据请求的 Content-Type 或 Accept header 决定响应格式
// 如果请求是 XML，响应也是 XML；否则默认使用 JSON
func renderResponse(ctx *gin.Context, response interface{}) {
	if response == nil {
		ctx.Status(http.StatusNoContent)
		return
	}

	useXML := isXMLResponse(ctx)

	// 基本类型特殊处理
	switch v := response.(type) {
	case string:
		ctx.String(http.StatusOK, v)
		return
	case int, int8, int16, int32, int64:
		if useXML {
			ctx.XML(http.StatusOK, gin.H{"value": v})
		} else {
			ctx.JSON(http.StatusOK, gin.H{"value": v})
		}
		return
	case uint, uint8, uint16, uint32, uint64:
		if useXML {
			ctx.XML(http.StatusOK, gin.H{"value": v})
		} else {
			ctx.JSON(http.StatusOK, gin.H{"value": v})
		}
		return
	case float32, float64:
		if useXML {
			ctx.XML(http.StatusOK, gin.H{"value": v})
		} else {
			ctx.JSON(http.StatusOK, gin.H{"value": v})
		}
		return
	case bool:
		if useXML {
			ctx.XML(http.StatusOK, gin.H{"value": v})
		} else {
			ctx.JSON(http.StatusOK, gin.H{"value": v})
		}
		return
	}

	// 其他类型根据格式序列化
	if useXML {
		ctx.XML(http.StatusOK, response)
	} else {
		ctx.JSON(http.StatusOK, response)
	}
}

// renderError 渲染错误响应
func renderError(ctx *gin.Context, statusCode int, err error) {
	useXML := isXMLResponse(ctx)
	errorMsg := gin.H{"error": err.Error()}
	if useXML {
		ctx.XML(statusCode, errorMsg)
	} else {
		ctx.JSON(statusCode, errorMsg)
	}
}
