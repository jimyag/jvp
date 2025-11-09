package apierror

// AWS EC2 服务器错误
// https://docs.aws.amazon.com/zh_cn/AWSEC2/latest/APIReference/errors-overview.html#api-error-codes-table-server
var (
	// ErrBandwidthLimitExceeded 已达到 Amazon EC2 实例可用网络带宽的限制
	// 更多信息请参考 Amazon EC2 instance network bandwidth
	ErrBandwidthLimitExceeded = &Error{
		Code:    "BandwidthLimitExceeded",
		Message: "You've reached the limit on the network bandwidth that is available to an Amazon EC2 instance. For more information, see Amazon EC2 instance network bandwidth.",
	}

	// ErrInsufficientAddressCapacity 没有足够的可用地址来满足您的最小请求
	// 减少您请求的地址数量或等待额外容量可用
	ErrInsufficientAddressCapacity = &Error{
		Code:    "InsufficientAddressCapacity",
		Message: "Not enough available addresses to satisfy your minimum request. Reduce the number of addresses you are requesting or wait for additional capacity to become available.",
	}

	// ErrInsufficientCapacity 没有足够的容量来满足您的导入实例请求
	// 您可以等待额外容量可用
	ErrInsufficientCapacity = &Error{
		Code:    "InsufficientCapacity",
		Message: "There is not enough capacity to fulfill your import instance request. You can wait for additional capacity to become available.",
	}

	// ErrInsufficientInstanceCapacity 没有足够的容量来满足您的请求
	// 此错误可能发生在启动新实例、重启已停止的实例、创建新的容量预留或修改现有容量预留时
	// 减少请求中的实例数量，或等待额外容量可用。您也可以尝试通过选择不同的实例类型来启动实例（稍后可以调整大小）
	ErrInsufficientInstanceCapacity = &Error{
		Code:    "InsufficientInstanceCapacity",
		Message: "There is not enough capacity to fulfill your request. This error can occur if you launch a new instance, restart a stopped instance, create a new Capacity Reservation, or modify an existing Capacity Reservation. Reduce the number of instances in your request, or wait for additional capacity to become available. You can also try launching an instance by selecting different instance types (which you can resize at a later stage). The returned message might also give specific guidance about how to solve the problem.",
	}

	// ErrInsufficientHostCapacity 没有足够的容量来满足您的专用主机请求
	// 减少请求中的专用主机数量，或等待额外容量可用
	ErrInsufficientHostCapacity = &Error{
		Code:    "InsufficientHostCapacity",
		Message: "There is not enough capacity to fulfill your Dedicated Host request. Reduce the number of Dedicated Hosts in your request, or wait for additional capacity to become available.",
	}

	// ErrInsufficientReservedInstanceCapacity 没有足够的可用预留实例来满足您的最小请求
	// 减少请求中的预留实例数量或等待额外容量可用
	ErrInsufficientReservedInstanceCapacity = &Error{
		Code:    "InsufficientReservedInstanceCapacity",
		Message: "Not enough available Reserved Instances to satisfy your minimum request. Reduce the number of Reserved Instances in your request or wait for additional capacity to become available.",
	}

	// ErrInsufficientVolumeCapacity 没有足够的容量来满足您的 EBS 卷配置请求
	// 您可以尝试配置不同的卷类型、不同可用区的 EBS 卷，或等待额外容量可用
	ErrInsufficientVolumeCapacity = &Error{
		Code:    "InsufficientVolumeCapacity",
		Message: "There is not enough capacity to fulfill your EBS volume provision request. You can try to provision a different volume type, EBS volume in a different availability zone, or you can wait for additional capacity to become available.",
	}

	// ErrServerInternal 发生了内部错误
	// 重试您的请求，但如果问题仍然存在，请通过在 AWS re:Post 上发布消息与我们联系并提供详细信息
	ErrServerInternal = &Error{
		Code:    "ServerInternal",
		Message: "An internal error has occurred. Retry your request, but if the problem persists, contact us with details by posting a message on AWS re:Post.",
	}

	// ErrInternalFailure 由于未知错误、异常或故障，请求处理失败
	ErrInternalFailure = &Error{
		Code:    "InternalFailure",
		Message: "The request processing has failed because of an unknown error, exception, or failure.",
	}

	// ErrRequestLimitExceeded 您的账户超过了 Amazon EC2 API 允许的最大请求速率
	// 为了获得最佳结果，请在请求之间使用递增或可变的睡眠间隔
	// 更多信息请参考 Query API request rate
	ErrRequestLimitExceeded = &Error{
		Code:    "RequestLimitExceeded",
		Message: "The maximum request rate permitted by the Amazon EC2 APIs has been exceeded for your account. For best results, use an increasing or variable sleep interval between requests. For more information, see Query API request rate.",
	}

	// ErrServiceUnavailable 由于服务器临时故障，请求失败
	ErrServiceUnavailable = &Error{
		Code:    "ServiceUnavailable",
		Message: "The request has failed due to a temporary failure of the server.",
	}

	// ErrInternalError 发生了内部错误
	// 重试您的请求，但如果问题仍然存在，请通过在 AWS re:Post 上发布消息与我们联系并提供详细信息
	ErrInternalError = &Error{
		Code:    "InternalError",
		Message: "An internal error has occurred. Retry your request, but if the problem persists, contact us with details by posting a message on AWS re:Post.",
	}

	// ErrUnavailable 服务器过载，无法处理请求
	ErrUnavailable = &Error{
		Code:    "Unavailable",
		Message: "The server is overloaded and can't handle the request.",
	}
)
