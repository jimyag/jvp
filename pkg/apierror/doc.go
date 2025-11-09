// Package apierror 提供 AWS 风格的错误类型，用于所有服务的统一错误处理
//
// 错误响应格式支持 XML 和 JSON 两种格式：
//
//	XML 格式：
//	<Response>
//	    <Errors>
//	        <Error>
//	            <Code>InvalidInstanceID.NotFound</Code>
//	            <Message>The instance ID 'i-1a2b3c4d' does not exist</Message>
//	        </Error>
//	    </Errors>
//	    <RequestID>ea966190-f9aa-478e-9ede-example</RequestID>
//	</Response>
//
//	JSON 格式：
//	{
//	    "errors": [
//	        {
//	            "code": "InvalidInstanceID.NotFound",
//	            "message": "The instance ID 'i-1a2b3c4d' does not exist"
//	        }
//	    ],
//	    "requestId": "ea966190-f9aa-478e-9ede-example"
//	}
//
// 使用示例：
//
//	// 创建错误
//	err := apierror.NewError("InvalidInstanceID.NotFound", "The instance ID 'i-1a2b3c4d' does not exist")
//
//	// 创建错误响应
//	errorResp := apierror.NewErrorResponse("request-id", err)
//
//	// 在 gin 中使用
//	c.XML(http.StatusNotFound, errorResp)
//	// 或
//	c.JSON(http.StatusNotFound, errorResp)
//
// AWS EC2 服务器错误变量（可在代码中直接使用）：
//
//   - ErrBandwidthLimitExceeded: 已达到网络带宽限制
//   - ErrInsufficientAddressCapacity: 地址容量不足
//   - ErrInsufficientCapacity: 容量不足（导入实例）
//   - ErrInsufficientInstanceCapacity: 实例容量不足
//   - ErrInsufficientHostCapacity: 专用主机容量不足
//   - ErrInsufficientReservedInstanceCapacity: 预留实例容量不足
//   - ErrInsufficientVolumeCapacity: 存储卷容量不足
//   - ErrServerInternal: 服务器内部错误
//   - ErrInternalFailure: 内部故障
//   - ErrRequestLimitExceeded: 请求速率超限
//   - ErrServiceUnavailable: 服务不可用
//   - ErrInternalError: 内部错误
//   - ErrUnavailable: 服务器过载
//
// 使用示例：
//
//	// 直接使用预定义的错误
//	errorResp := apierror.NewErrorResponse("request-id", apierror.ErrInsufficientInstanceCapacity)
//
//	// 或创建自定义错误
//	err := apierror.NewError("CustomError", "Custom error message")
package apierror
