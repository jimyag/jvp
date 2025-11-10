// Package apierror 提供 AWS 风格的错误类型，用于所有服务的统一错误处理
package apierror

import (
	"encoding/xml"
	"fmt"
)

// ErrorResponse AWS 风格的错误响应结构
type ErrorResponse struct {
	XMLName   xml.Name `xml:"Response"     json:"-"`
	Errors    []Error  `xml:"Errors>Error" json:"errors"`
	RequestID string   `xml:"RequestID"    json:"requestID"`
}

func (er *ErrorResponse) Error() string {
	str := fmt.Sprintf("RequestID: %s", er.RequestID)
	for _, e := range er.Errors {
		str += fmt.Sprintf("; %s", e.Error())
	}
	return str
}

// Error 单个错误信息
type Error struct {
	Code       string `xml:"Code"    json:"code"`
	Message    string `xml:"Message" json:"message"`
	HTTPStatus int    `xml:"-"       json:"-"` // HTTP 状态码，不会序列化到响应中
	RawError   error  `xml:"-"       json:"-"` // 内部错误，用于服务端调试，不会序列化到响应中
}

// Error 实现 error 接口
func (e *Error) Error() string {
	str := fmt.Sprintf("[%s] %s", e.Code, e.Message)
	if e.RawError != nil {
		str += fmt.Sprintf(" (RawError: %v)", e.RawError)
	}
	return str
}

// Is 实现 errors.Is 接口，用于错误类型判断
// 如果 target 是 *Error 类型且 Code 相同，则返回 true
func (e *Error) Is(target error) bool {
	if target == nil {
		return false
	}

	t, ok := target.(*Error)
	if !ok {
		return false
	}

	// 如果 e 或 t 为 nil，返回 false
	if e == nil || t == nil {
		return false
	}

	// 比较错误代码
	return e.Code == t.Code
}

// Unwrap 实现 errors.Unwrap 接口，返回底层错误
// 如果设置了 RawError，则返回 RawError；否则返回 nil
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.RawError
}

// 编译时检查 Error 是否实现了所有必需的接口
var _ interface {
	// Error 必须实现的接口
	Error() string
	// 实现了该接口后，可以使用 errors.Is() 函数来判断错误类型
	Is(target error) bool
	// 实现了该接口后，可以使用 errors.As() 和 errors.Unwrap() 函数来获取到原始的错误类型
	Unwrap() error
} = (*Error)(nil)

// NewError 创建新的错误
// 默认 HTTP 状态码为 500
func NewError(code, message string) *Error {
	return &Error{
		Code:       code,
		Message:    message,
		HTTPStatus: 500,
	}
}

// NewErrorWithStatus 创建新的错误，指定 HTTP 状态码
func NewErrorWithStatus(code, message string, httpStatus int) *Error {
	return &Error{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// NewErrorWithRaw 创建新的错误，包含原始错误信息
// rawError 用于服务端调试，不会序列化到响应中
// 默认 HTTP 状态码为 500
func NewErrorWithRaw(code, message string, rawError error) *Error {
	return &Error{
		Code:       code,
		Message:    message,
		HTTPStatus: 500,
		RawError:   rawError,
	}
}

// NewErrorWithRawAndStatus 创建新的错误，包含原始错误信息和 HTTP 状态码
func NewErrorWithRawAndStatus(code, message string, httpStatus int, rawError error) *Error {
	return &Error{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		RawError:   rawError,
	}
}

// NewErrorResponse 创建新的错误响应
func NewErrorResponse(requestID string, errors ...*Error) *ErrorResponse {
	errs := make([]Error, len(errors))
	for i, e := range errors {
		errs[i] = *e
	}
	return &ErrorResponse{
		Errors:    errs,
		RequestID: requestID,
	}
}

// AddError 添加错误到响应
func (er *ErrorResponse) AddError(err *Error) {
	er.Errors = append(er.Errors, *err)
}

// ToXML 转换为 XML 格式
func (er *ErrorResponse) ToXML() ([]byte, error) {
	return xml.MarshalIndent(er, "", "    ")
}

// WrapError 包装预定义的错误，添加原始错误信息
// 保留预定义错误的 Code 和 HTTPStatus，但使用自定义消息和原始错误
func WrapError(baseErr *Error, message string, rawError error) *Error {
	return &Error{
		Code:       baseErr.Code,
		Message:    message,
		HTTPStatus: baseErr.HTTPStatus,
		RawError:   rawError,
	}
}
