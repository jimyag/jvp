package apierror_test

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/pkg/apierror"
)

// 示例：创建和使用错误响应
func ExampleNewErrorResponse() {
	// 创建错误
	err := apierror.NewError(
		"InvalidInstanceID.NotFound",
		"The instance ID 'i-1a2b3c4d' does not exist",
	)

	// 创建错误响应
	errorResp := apierror.NewErrorResponse("ea966190-f9aa-478e-9ede-example", err)

	// JSON 格式
	jsonData, _ := json.Marshal(errorResp)
	fmt.Println(string(jsonData))
	// 输出：{"errors":[{"code":"InvalidInstanceID.NotFound","message":"The instance ID 'i-1a2b3c4d' does not exist"}],"requestID":"ea966190-f9aa-478e-9ede-example"}

	// XML 格式
	xmlData, _ := xml.MarshalIndent(errorResp, "", "    ")
	fmt.Println(string(xmlData))
	// 输出：
	// <Response>
	//     <Errors>
	//         <Error>
	//             <Code>InvalidInstanceID.NotFound</Code>
	//             <Message>The instance ID 'i-1a2b3c4d' does not exist</Message>
	//         </Error>
	//     </Errors>
	//     <RequestID>ea966190-f9aa-478e-9ede-example</RequestID>
	// </Response>
}

// 示例：在 gin 中使用错误响应
func ExampleErrorResponse_gin() {
	router := gin.Default()

	router.GET("/instances/:id", func(c *gin.Context) {
		instanceID := c.Param("id")

		if instanceID == "i-1a2b3c4d" {
			err := apierror.NewError(
				"InvalidInstanceID.NotFound",
				fmt.Sprintf("The instance ID '%s' does not exist", instanceID),
			)
			errorResp := apierror.NewErrorResponse("request-id", err)
			c.XML(http.StatusNotFound, errorResp)
			return
		}

		c.JSON(http.StatusOK, gin.H{"id": instanceID})
	})

	router.Run(":8080")
}

// 示例：使用预定义的 AWS EC2 错误
func ExampleErrorResponse_awsEC2() {
	errorResp := apierror.NewErrorResponse(
		"request-id",
		apierror.ErrInternalError,
		apierror.ErrInsufficientInstanceCapacity,
	)

	jsonData, _ := json.Marshal(errorResp)
	fmt.Println(string(jsonData))
}

// 示例：使用 RawError 进行服务端调试
func ExampleNewErrorWithRaw() {
	// 创建带原始错误的 API 错误
	internalErr := fmt.Errorf("database connection failed")
	err := apierror.NewErrorWithRaw(
		"InternalError",
		"An internal error has occurred",
		internalErr,
	)

	// 服务端可以访问 RawError 进行调试
	if err.RawError != nil {
		fmt.Printf("Debug: %v\n", err.RawError)
	}

	// 使用 errors.Unwrap 获取原始错误
	unwrapped := errors.Unwrap(err)
	if unwrapped != nil {
		fmt.Printf("Unwrapped: %v\n", unwrapped)
	}

	// 序列化时 RawError 不会被包含
	jsonData, _ := json.Marshal(err)
	fmt.Println(string(jsonData))
	// 输出：{"code":"InternalError","message":"An internal error has occurred"}
}
