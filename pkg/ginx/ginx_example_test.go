package ginx_test

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/pkg/ginx"
)

// 示例：有参数，有返回值，有 error
type CreateArticleArgs struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type Article struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func ExampleAdapt5() {
	router := gin.Default()

	router.POST("/articles", ginx.Adapt5(func(c *gin.Context, args *CreateArticleArgs) (*Article, error) {
		article := &Article{
			ID:        1,
			Title:     args.Title,
			Content:   args.Content,
			CreatedAt: time.Now(),
		}
		return article, nil
	}))

	router.Run(":8080")
}

// 示例：有参数，只有 error
type DescribeArticleArgs struct {
	ID int64 `uri:"id"`
}

func ExampleAdapt4() {
	router := gin.Default()

	router.DELETE("/articles/:id", ginx.Adapt4(func(c *gin.Context, args *DescribeArticleArgs) error {
		// 执行删除操作
		return nil
	}))

	router.Run(":8080")
}

// 示例：无参数，有返回值
func ExampleAdapt2() {
	router := gin.Default()

	router.GET("/health", ginx.Adapt2(func(c *gin.Context) string {
		return "ok"
	}))

	router.Run(":8080")
}

// 示例：无参数，只有 error
func ExampleAdapt1() {
	router := gin.Default()

	router.GET("/check", ginx.Adapt1(func(c *gin.Context) error {
		// 执行检查
		return nil
	}))

	router.Run(":8080")
}

// 示例：无参数，无返回值
func ExampleAdapt0() {
	router := gin.Default()

	router.GET("/ping", ginx.Adapt0(func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	}))

	router.Run(":8080")
}

// 示例：参数验证
type CreateUserArgs struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (args *CreateUserArgs) IsValid() error {
	if args.Username == "" {
		return &ValidationError{Field: "username", Message: "username is required"}
	}
	if len(args.Password) < 6 {
		return &ValidationError{Field: "password", Message: "password must be at least 6 characters"}
	}
	return nil
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func ExampleAdapt5_validation() {
	router := gin.Default()

	router.POST("/users", ginx.Adapt5(func(c *gin.Context, args *CreateUserArgs) (map[string]string, error) {
		return map[string]string{"username": args.Username}, nil
	}))

	router.Run(":8080")
}
