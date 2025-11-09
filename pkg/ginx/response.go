package ginx

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/pkg/apierror"
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
func renderResponse(ctx *gin.Context, response any) {
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
// 如果 err 是 *apierror.Error 或 *apierror.ErrorResponse，直接序列化错误对象
// 否则使用默认的错误格式
func renderError(ctx *gin.Context, statusCode int, err error) {
	useXML := isXMLResponse(ctx)

	// 检查是否是 apierror.Error
	if apiErr, ok := err.(*apierror.Error); ok {
		// 使用错误对象中定义的 HTTP 状态码
		if apiErr.HTTPStatus > 0 {
			statusCode = apiErr.HTTPStatus
		}
		// 创建 ErrorResponse 用于序列化
		errorResp := apierror.NewErrorResponse("", apiErr)
		if useXML {
			ctx.XML(statusCode, errorResp)
		} else {
			ctx.JSON(statusCode, errorResp)
		}
		return
	}

	// 检查是否是 apierror.ErrorResponse
	if errorResp, ok := err.(*apierror.ErrorResponse); ok {
		// 从第一个错误中获取 HTTP 状态码（如果有）
		if len(errorResp.Errors) > 0 && errorResp.Errors[0].HTTPStatus > 0 {
			statusCode = errorResp.Errors[0].HTTPStatus
		}
		if useXML {
			ctx.XML(statusCode, errorResp)
		} else {
			ctx.JSON(statusCode, errorResp)
		}
		return
	}

	// 默认错误格式
	errorMsg := gin.H{"error": err.Error()}
	if useXML {
		ctx.XML(statusCode, errorMsg)
	} else {
		ctx.JSON(statusCode, errorMsg)
	}
}
