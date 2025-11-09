package apierror_test

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"testing"

	"github.com/jimyag/jvp/pkg/apierror"
	"github.com/stretchr/testify/assert"
)

func TestError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testFunc func(*testing.T)
	}{
		{
			name: "Error_Error",
			testFunc: func(t *testing.T) {
				t.Parallel()
				err := apierror.NewError("TestError", "test message")
				expected := "[TestError] test message"
				assert.Equal(t, expected, err.Error())
			},
		},
		{
			name: "Error_Error_WithRawError",
			testFunc: func(t *testing.T) {
				t.Parallel()
				rawErr := fmt.Errorf("raw error")
				err := apierror.NewErrorWithRaw("TestError", "test message", rawErr)
				expected := "[TestError] test message (RawError: raw error)"
				assert.Equal(t, expected, err.Error())
			},
		},
		{
			name: "Error_Is_SameCode",
			testFunc: func(t *testing.T) {
				t.Parallel()
				err1 := apierror.NewError("TestError", "message 1")
				err2 := apierror.NewError("TestError", "message 2")
				assert.True(t, errors.Is(err1, err2))
			},
		},
		{
			name: "Error_Is_DifferentCode",
			testFunc: func(t *testing.T) {
				t.Parallel()
				err1 := apierror.NewError("TestError", "message")
				err2 := apierror.NewError("DifferentError", "message")
				assert.False(t, errors.Is(err1, err2))
			},
		},
		{
			name: "Error_Is_WithPredefinedError",
			testFunc: func(t *testing.T) {
				t.Parallel()
				err := apierror.NewError("InternalError", "different message")
				assert.True(t, errors.Is(err, apierror.ErrInternalError))
			},
		},
		{
			name: "Error_Unwrap_NoRawError",
			testFunc: func(t *testing.T) {
				t.Parallel()
				err := apierror.NewError("TestError", "test message")
				assert.Nil(t, errors.Unwrap(err))
			},
		},
		{
			name: "Error_Unwrap_WithRawError",
			testFunc: func(t *testing.T) {
				t.Parallel()
				rawErr := fmt.Errorf("raw error")
				err := apierror.NewErrorWithRaw("TestError", "test message", rawErr)
				assert.Equal(t, rawErr, errors.Unwrap(err))
			},
		},
		{
			name: "Error_As",
			testFunc: func(t *testing.T) {
				t.Parallel()
				err := apierror.NewError("TestError", "test message")
				var apiErr *apierror.Error
				assert.True(t, errors.As(err, &apiErr))
				assert.Equal(t, "TestError", apiErr.Code)
				assert.Equal(t, "test message", apiErr.Message)
			},
		},
		{
			name: "Error_JSON_Marshal_ExcludesRawError",
			testFunc: func(t *testing.T) {
				t.Parallel()
				rawErr := fmt.Errorf("raw error")
				err := apierror.NewErrorWithRaw("TestError", "test message", rawErr)
				jsonData, marshalErr := json.Marshal(err)
				assert.NoError(t, marshalErr)
				assert.NotContains(t, string(jsonData), "rawError")
				assert.Contains(t, string(jsonData), `"code":"TestError"`)
				assert.Contains(t, string(jsonData), `"message":"test message"`)
			},
		},
		{
			name: "Error_XML_Marshal_ExcludesRawError",
			testFunc: func(t *testing.T) {
				t.Parallel()
				rawErr := fmt.Errorf("raw error")
				err := apierror.NewErrorWithRaw("TestError", "test message", rawErr)
				xmlData, marshalErr := xml.Marshal(err)
				assert.NoError(t, marshalErr)
				assert.NotContains(t, string(xmlData), "RawError")
				assert.Contains(t, string(xmlData), "<Code>TestError</Code>")
				assert.Contains(t, string(xmlData), "<Message>test message</Message>")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

func TestErrorResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testFunc func(*testing.T)
	}{
		{
			name: "ErrorResponse_Error_SingleError",
			testFunc: func(t *testing.T) {
				t.Parallel()
				err := apierror.NewError("TestError", "test message")
				resp := apierror.NewErrorResponse("request-id", err)
				expected := "RequestID: request-id; [TestError] test message"
				assert.Equal(t, expected, resp.Error())
			},
		},
		{
			name: "ErrorResponse_Error_MultipleErrors",
			testFunc: func(t *testing.T) {
				t.Parallel()
				err1 := apierror.NewError("Error1", "message 1")
				err2 := apierror.NewError("Error2", "message 2")
				resp := apierror.NewErrorResponse("request-id", err1, err2)
				errorStr := resp.Error()
				assert.Contains(t, errorStr, "RequestID: request-id")
				assert.Contains(t, errorStr, "[Error1] message 1")
				assert.Contains(t, errorStr, "[Error2] message 2")
			},
		},
		{
			name: "ErrorResponse_AddError",
			testFunc: func(t *testing.T) {
				t.Parallel()
				resp := apierror.NewErrorResponse("request-id")
				err := apierror.NewError("TestError", "test message")
				resp.AddError(err)
				assert.Len(t, resp.Errors, 1)
				assert.Equal(t, "TestError", resp.Errors[0].Code)
			},
		},
		{
			name: "ErrorResponse_JSON_Marshal",
			testFunc: func(t *testing.T) {
				t.Parallel()
				err := apierror.NewError("TestError", "test message")
				resp := apierror.NewErrorResponse("request-id", err)
				jsonData, marshalErr := json.Marshal(resp)
				assert.NoError(t, marshalErr)
				assert.Contains(t, string(jsonData), `"requestID":"request-id"`)
				assert.Contains(t, string(jsonData), `"code":"TestError"`)
			},
		},
		{
			name: "ErrorResponse_XML_Marshal",
			testFunc: func(t *testing.T) {
				t.Parallel()
				err := apierror.NewError("TestError", "test message")
				resp := apierror.NewErrorResponse("request-id", err)
				xmlData, marshalErr := xml.Marshal(resp)
				assert.NoError(t, marshalErr)
				assert.Contains(t, string(xmlData), "<RequestID>request-id</RequestID>")
				assert.Contains(t, string(xmlData), "<Code>TestError</Code>")
			},
		},
		{
			name: "ErrorResponse_ToXML",
			testFunc: func(t *testing.T) {
				t.Parallel()
				err := apierror.NewError("TestError", "test message")
				resp := apierror.NewErrorResponse("request-id", err)
				xmlData, marshalErr := resp.ToXML()
				assert.NoError(t, marshalErr)
				assert.Contains(t, string(xmlData), "<RequestID>request-id</RequestID>")
				assert.Contains(t, string(xmlData), "<Code>TestError</Code>")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}
