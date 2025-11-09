package ginx

import (
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
)

// Adapt0 适配无参数、无返回值的 handler
func Adapt0(fn func(*gin.Context)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		fn(ctx)
	}
}

// Adapt1 适配无参数、只有 error 的 handler
func Adapt1(fn func(*gin.Context) error) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		_ = fn(ctx)
	}
}

// Adapt2 适配无参数、只有返回值的 handler
func Adapt2[T any](fn func(*gin.Context) T) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		result := fn(ctx)
		renderResponse(ctx, result)
	}
}

// Adapt3 适配无参数、有返回值和 error 的 handler
func Adapt3[T any](fn func(*gin.Context) (T, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		result, err := fn(ctx)
		if err != nil {
			// 对于无参数的 handler，默认使用 JSON
			setResponseFormat(ctx, "json")
			renderError(ctx, http.StatusInternalServerError, err)
			return
		}
		renderResponse(ctx, result)
	}
}

// Adapt4 适配有参数、只有 error 的 handler
func Adapt4[T any](fn func(*gin.Context, *T) error) gin.HandlerFunc {
	var argsType T
	argsTypeValue := reflect.TypeOf(argsType)

	return func(ctx *gin.Context) {
		// 绑定参数
		argsValue := reflect.New(argsTypeValue)
		args := argsValue.Interface()

		if err := bindArgs(ctx, args); err != nil {
			renderError(ctx, http.StatusBadRequest, err)
			return
		}

		// 验证参数（如果实现了 IsValid 方法）
		if validator, ok := args.(interface{ IsValid() error }); ok {
			if err := validator.IsValid(); err != nil {
				renderError(ctx, http.StatusBadRequest, err)
				return
			}
		}

		// 调用 handler
		if err := fn(ctx, args.(*T)); err != nil {
			renderError(ctx, http.StatusInternalServerError, err)
			return
		}

		ctx.Status(http.StatusNoContent)
	}
}

// Adapt5 适配有参数、有返回值和 error 的 handler
func Adapt5[TArgs any, TResp any](fn func(*gin.Context, *TArgs) (TResp, error)) gin.HandlerFunc {
	var argsType TArgs
	argsTypeValue := reflect.TypeOf(argsType)

	return func(ctx *gin.Context) {
		// 绑定参数
		argsValue := reflect.New(argsTypeValue)
		args := argsValue.Interface()

		if err := bindArgs(ctx, args); err != nil {
			renderError(ctx, http.StatusBadRequest, err)
			return
		}

		// 验证参数（如果实现了 IsValid 方法）
		if validator, ok := args.(interface{ IsValid() error }); ok {
			if err := validator.IsValid(); err != nil {
				renderError(ctx, http.StatusBadRequest, err)
				return
			}
		}

		// 调用 handler
		result, err := fn(ctx, args.(*TArgs))
		if err != nil {
			renderError(ctx, http.StatusInternalServerError, err)
			return
		}

		// 处理响应
		renderResponse(ctx, result)
	}
}

// Adapt6 适配有参数、只有返回值的 handler
func Adapt6[TArgs any, TResp any](fn func(*gin.Context, *TArgs) TResp) gin.HandlerFunc {
	var argsType TArgs
	argsTypeValue := reflect.TypeOf(argsType)

	return func(ctx *gin.Context) {
		// 绑定参数
		argsValue := reflect.New(argsTypeValue)
		args := argsValue.Interface()

		if err := bindArgs(ctx, args); err != nil {
			renderError(ctx, http.StatusBadRequest, err)
			return
		}

		// 验证参数（如果实现了 IsValid 方法）
		if validator, ok := args.(interface{ IsValid() error }); ok {
			if err := validator.IsValid(); err != nil {
				renderError(ctx, http.StatusBadRequest, err)
				return
			}
		}

		// 调用 handler
		result := fn(ctx, args.(*TArgs))
		renderResponse(ctx, result)
	}
}
