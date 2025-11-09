// Package ginx 提供 gin 框架的 handler 适配器，支持自动参数绑定和响应处理
//
// 支持 JSON 和 XML 格式：
//   - 默认使用 JSON 格式
//   - 如果请求的 Content-Type 包含 "application/xml" 或 "text/xml"，则使用 XML 解析请求
//   - 如果使用 XML 解析请求，响应也会使用 XML 格式
//   - 错误响应也会根据请求格式自动选择 JSON 或 XML
//
// 支持多种 handler 函数签名：
//
//	// 1. 有参数，有返回值，有 error
//	func(c *gin.Context, args *Args) (resp, error)
//
//	// 2. 有参数，只有 error
//	func(c *gin.Context, args *Args) error
//
//	// 3. 有参数，只有返回值
//	func(c *gin.Context, args *Args) resp
//
//	// 4. 无参数，有返回值，有 error
//	func(c *gin.Context) (resp, error)
//
//	// 5. 无参数，只有 error
//	func(c *gin.Context) error
//
//	// 6. 无参数，只有返回值
//	func(c *gin.Context) resp
//
//	// 7. 无参数，无返回值
//	func(c *gin.Context)
//
// 使用示例：
//
//	router := gin.Default()
//
//	// 有参数，有返回值，有 error
//	router.POST("/articles", ginx.Adapt5(func(c *gin.Context, args *CreateArticleArgs) (*Article, error) {
//	    return &Article{...}, nil
//	}))
//
//	// 有参数，只有 error
//	router.DELETE("/articles/:id", ginx.Adapt4(func(c *gin.Context, args *DescribeArticleArgs) error {
//	    return nil
//	}))
//
//	// 无参数，有返回值
//	router.GET("/health", ginx.Adapt2(func(c *gin.Context) string {
//	    return "ok"
//	}))
package ginx
