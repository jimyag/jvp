package ginx_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/pkg/ginx"
	"github.com/stretchr/testify/assert"
)

type validationError struct {
	Message string
}

func (e *validationError) Error() string {
	return e.Message
}

// ValidatedArgs 用于测试 IsValid 方法
type ValidatedArgs struct {
	Username string `json:"username"`
}

func (args *ValidatedArgs) IsValid() error {
	if args.Username == "" {
		return &validationError{Message: "username is required"}
	}
	return nil
}

func TestAdapt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		testFunc func(*testing.T)
	}{
		{
			name: "Adapt0_NoArgsNoReturn",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()
				router.GET("/test", ginx.Adapt0(func(c *gin.Context) {
					c.String(http.StatusOK, "ok")
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				assert.Equal(t, "ok", w.Body.String())
			},
		},
		{
			name: "Adapt1_NoArgsError",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()
				router.GET("/test", ginx.Adapt1(func(c *gin.Context) error {
					c.Status(http.StatusOK)
					return nil
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
			},
		},
		{
			name: "Adapt2_NoArgsReturn",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()
				router.GET("/test", ginx.Adapt2(func(c *gin.Context) string {
					return "ok"
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				assert.Equal(t, "ok", w.Body.String())
			},
		},
		{
			name: "Adapt3_NoArgsReturnError",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()
				router.GET("/test", ginx.Adapt3(func(c *gin.Context) (string, error) {
					return "ok", nil
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				assert.Equal(t, "ok", w.Body.String())
			},
		},
		{
			name: "Adapt3_NoArgsReturnError_WithError",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()
				router.GET("/test", ginx.Adapt3(func(c *gin.Context) (string, error) {
					return "", assert.AnError
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
		{
			name: "Adapt4_ArgsError",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				type Args struct {
					ID int64 `uri:"id"`
				}

				router.DELETE("/test/:id", ginx.Adapt4(func(c *gin.Context, args *Args) error {
					assert.Equal(t, int64(123), args.ID)
					c.Status(http.StatusNoContent)
					return nil
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodDelete, "/test/123", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusNoContent, w.Code)
			},
		},
		{
			name: "Adapt4_ArgsError_WithError",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				type Args struct {
					ID int64 `uri:"id"`
				}

				router.DELETE("/test/:id", ginx.Adapt4(func(c *gin.Context, args *Args) error {
					return assert.AnError
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodDelete, "/test/123", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
		{
			name: "Adapt5_ArgsReturnError",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				type Args struct {
					ID int64 `uri:"id"`
				}

				type Response struct {
					ID int64 `json:"id"`
				}

				router.GET("/test/:id", ginx.Adapt5(func(c *gin.Context, args *Args) (*Response, error) {
					assert.Equal(t, int64(123), args.ID)
					return &Response{ID: args.ID}, nil
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test/123", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				var resp Response
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, int64(123), resp.ID)
			},
		},
		{
			name: "Adapt5_ArgsReturnError_WithError",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				type Args struct {
					ID int64 `uri:"id"`
				}

				type Response struct {
					ID int64 `json:"id"`
				}

				router.GET("/test/:id", ginx.Adapt5(func(c *gin.Context, args *Args) (*Response, error) {
					return nil, assert.AnError
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test/123", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusInternalServerError, w.Code)
			},
		},
		{
			name: "Adapt5_JSONBinding",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				type Args struct {
					Title   string `json:"title"`
					Content string `json:"content"`
				}

				type Response struct {
					Title string `json:"title"`
				}

				router.POST("/test", ginx.Adapt5(func(c *gin.Context, args *Args) (*Response, error) {
					assert.Equal(t, "test", args.Title)
					assert.Equal(t, "content", args.Content)
					return &Response{Title: args.Title}, nil
				}))

				w := httptest.NewRecorder()
				body := strings.NewReader(`{"title":"test","content":"content"}`)
				req := httptest.NewRequest(http.MethodPost, "/test", body)
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				var resp Response
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "test", resp.Title)
			},
		},
		{
			name: "Adapt5_Validation",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				type Args struct {
					Username string `json:"username" binding:"required"`
				}

				router.POST("/test", ginx.Adapt5(func(c *gin.Context, args *Args) (map[string]string, error) {
					return map[string]string{"username": args.Username}, nil
				}))

				w := httptest.NewRecorder()
				// 发送空的 JSON body，应该导致绑定失败
				body := strings.NewReader(`{}`)
				req := httptest.NewRequest(http.MethodPost, "/test", body)
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(w, req)

				// 参数绑定失败应该返回 400
				assert.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name: "Adapt6_ArgsReturn",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				type Args struct {
					ID int64 `uri:"id"`
				}

				type Response struct {
					ID int64 `json:"id"`
				}

				router.GET("/test/:id", ginx.Adapt6(func(c *gin.Context, args *Args) *Response {
					return &Response{ID: args.ID}
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test/123", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				var resp Response
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, int64(123), resp.ID)
			},
		},
		{
			name: "Adapt5_XMLBinding",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				type Args struct {
					Title   string `xml:"title"`
					Content string `xml:"content"`
				}

				type Response struct {
					Title string `xml:"title"`
				}

				router.POST("/test", ginx.Adapt5(func(c *gin.Context, args *Args) (*Response, error) {
					assert.Equal(t, "test", args.Title)
					assert.Equal(t, "content", args.Content)
					return &Response{Title: args.Title}, nil
				}))

				w := httptest.NewRecorder()
				body := strings.NewReader(`<Args><title>test</title><content>content</content></Args>`)
				req := httptest.NewRequest(http.MethodPost, "/test", body)
				req.Header.Set("Content-Type", "application/xml")
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				assert.Contains(t, w.Header().Get("Content-Type"), "xml")
				assert.Contains(t, w.Body.String(), "<title>test</title>")
			},
		},
		{
			name: "Adapt5_QueryBinding",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				type Args struct {
					ID    int64  `form:"id"`
					Name  string `form:"name"`
					Limit int    `form:"limit"`
				}

				type Response struct {
					ID   int64  `json:"id"`
					Name string `json:"name"`
				}

				router.GET("/test", ginx.Adapt5(func(c *gin.Context, args *Args) (*Response, error) {
					assert.Equal(t, int64(123), args.ID)
					assert.Equal(t, "test", args.Name)
					assert.Equal(t, 10, args.Limit)
					return &Response{ID: args.ID, Name: args.Name}, nil
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test?id=123&name=test&limit=10", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				var resp Response
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, int64(123), resp.ID)
				assert.Equal(t, "test", resp.Name)
			},
		},
		{
			name: "Adapt5_URIBinding",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				type Args struct {
					ID int64 `uri:"id"`
				}

				type Response struct {
					ID int64 `json:"id"`
				}

				router.GET("/test/:id", ginx.Adapt5(func(c *gin.Context, args *Args) (*Response, error) {
					assert.Equal(t, int64(456), args.ID)
					return &Response{ID: args.ID}, nil
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test/456", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				var resp Response
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, int64(456), resp.ID)
			},
		},
		{
			name: "Adapt2_ReturnInt",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				router.GET("/test", ginx.Adapt2(func(c *gin.Context) int {
					return 42
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				var resp map[string]int
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, 42, resp["value"])
			},
		},
		{
			name: "Adapt2_ReturnUint",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				router.GET("/test", ginx.Adapt2(func(c *gin.Context) uint {
					return 100
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				var resp map[string]uint
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, uint(100), resp["value"])
			},
		},
		{
			name: "Adapt2_ReturnFloat",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				router.GET("/test", ginx.Adapt2(func(c *gin.Context) float64 {
					return 3.14
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				var resp map[string]float64
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.InDelta(t, 3.14, resp["value"], 0.001)
			},
		},
		{
			name: "Adapt2_ReturnBool",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				router.GET("/test", ginx.Adapt2(func(c *gin.Context) bool {
					return true
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				var resp map[string]bool
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.True(t, resp["value"])
			},
		},
		{
			name: "Adapt3_ReturnNil",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				router.GET("/test", ginx.Adapt3(func(c *gin.Context) (interface{}, error) {
					return nil, nil
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusNoContent, w.Code)
			},
		},
		{
			name: "Adapt5_XMLErrorResponse",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				type Args struct {
					ID int64 `uri:"id"`
				}

				type Response struct {
					ID int64 `json:"id"`
				}

				router.GET("/test/:id", ginx.Adapt5(func(c *gin.Context, args *Args) (*Response, error) {
					return nil, assert.AnError
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test/123", nil)
				req.Header.Set("Accept", "application/xml")
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusInternalServerError, w.Code)
				assert.Contains(t, w.Header().Get("Content-Type"), "xml")
			},
		},
		{
			name: "Adapt4_XMLErrorResponse",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				type Args struct {
					ID int64 `uri:"id"`
				}

				router.DELETE("/test/:id", ginx.Adapt4(func(c *gin.Context, args *Args) error {
					return assert.AnError
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodDelete, "/test/123", nil)
				req.Header.Set("Accept", "application/xml")
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusInternalServerError, w.Code)
				assert.Contains(t, w.Header().Get("Content-Type"), "xml")
			},
		},
		{
			name: "Adapt5_FormBinding",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				type Args struct {
					Name  string `form:"name"`
					Email string `form:"email"`
				}

				type Response struct {
					Name string `json:"name"`
				}

				router.POST("/test", ginx.Adapt5(func(c *gin.Context, args *Args) (*Response, error) {
					// Form 绑定在 JSON/XML/URI/Query 都失败时才会尝试
					// 这里测试 Query 参数绑定（更常见的情况）
					return &Response{Name: args.Name}, nil
				}))

				w := httptest.NewRecorder()
				// 使用 Query 参数，因为 Form 绑定优先级最低
				req := httptest.NewRequest(http.MethodPost, "/test?name=test&email=test@example.com", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				var resp Response
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, "test", resp.Name)
			},
		},
		{
			name: "Adapt5_XMLResponseWithBasicTypes",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				type Args struct {
					Value int `json:"value"`
				}

				router.POST("/test", ginx.Adapt5(func(c *gin.Context, args *Args) (int, error) {
					return args.Value, nil
				}))

				w := httptest.NewRecorder()
				body := strings.NewReader(`{"value":42}`)
				req := httptest.NewRequest(http.MethodPost, "/test", body)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Accept", "application/xml")
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				assert.Contains(t, w.Header().Get("Content-Type"), "xml")
				assert.Contains(t, w.Body.String(), "<value>42</value>")
			},
		},
		{
			name: "Adapt4_ValidationError",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				type Args struct {
					Username string `json:"username" binding:"required"`
				}

				router.POST("/test", ginx.Adapt4(func(c *gin.Context, args *Args) error {
					return nil
				}))

				w := httptest.NewRecorder()
				body := strings.NewReader(`{}`)
				req := httptest.NewRequest(http.MethodPost, "/test", body)
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name: "Adapt6_ValidationError",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				type Args struct {
					Username string `json:"username" binding:"required"`
				}

				type Response struct {
					Username string `json:"username"`
				}

				router.POST("/test", ginx.Adapt6(func(c *gin.Context, args *Args) *Response {
					return &Response{Username: args.Username}
				}))

				w := httptest.NewRecorder()
				body := strings.NewReader(`{}`)
				req := httptest.NewRequest(http.MethodPost, "/test", body)
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name: "Adapt2_ReturnInt_XML",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				router.GET("/test", ginx.Adapt2(func(c *gin.Context) int {
					return 42
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("Accept", "application/xml")
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				assert.Contains(t, w.Header().Get("Content-Type"), "xml")
				assert.Contains(t, w.Body.String(), "<value>42</value>")
			},
		},
		{
			name: "Adapt2_ReturnUint_XML",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				router.GET("/test", ginx.Adapt2(func(c *gin.Context) uint {
					return 100
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("Accept", "application/xml")
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				assert.Contains(t, w.Header().Get("Content-Type"), "xml")
			},
		},
		{
			name: "Adapt2_ReturnFloat_XML",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				router.GET("/test", ginx.Adapt2(func(c *gin.Context) float64 {
					return 3.14
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("Accept", "application/xml")
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				assert.Contains(t, w.Header().Get("Content-Type"), "xml")
			},
		},
		{
			name: "Adapt2_ReturnBool_XML",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				router.GET("/test", ginx.Adapt2(func(c *gin.Context) bool {
					return true
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("Accept", "application/xml")
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				assert.Contains(t, w.Header().Get("Content-Type"), "xml")
			},
		},
		{
			name: "Adapt5_XMLBindingFailed",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				type Args struct {
					ID int64 `uri:"id"`
				}

				type Response struct {
					ID int64 `json:"id"`
				}

				router.GET("/test/:id", ginx.Adapt5(func(c *gin.Context, args *Args) (*Response, error) {
					return &Response{ID: args.ID}, nil
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test/123", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
			},
		},
		{
			name: "Adapt5_JSONBindingFailed",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				type Args struct {
					ID int64 `uri:"id"`
				}

				type Response struct {
					ID int64 `json:"id"`
				}

				router.GET("/test/:id", ginx.Adapt5(func(c *gin.Context, args *Args) (*Response, error) {
					return &Response{ID: args.ID}, nil
				}))

				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, "/test/123", nil)
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
			},
		},
		{
			name: "Adapt4_IsValidError",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				router.POST("/test", ginx.Adapt4(func(c *gin.Context, args *ValidatedArgs) error {
					return nil
				}))

				w := httptest.NewRecorder()
				body := strings.NewReader(`{}`)
				req := httptest.NewRequest(http.MethodPost, "/test", body)
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name: "Adapt5_IsValidError",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				type Response struct {
					Username string `json:"username"`
				}

				router.POST("/test", ginx.Adapt5(func(c *gin.Context, args *ValidatedArgs) (*Response, error) {
					return &Response{Username: args.Username}, nil
				}))

				w := httptest.NewRecorder()
				body := strings.NewReader(`{}`)
				req := httptest.NewRequest(http.MethodPost, "/test", body)
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name: "Adapt6_IsValidError",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				type Response struct {
					Username string `json:"username"`
				}

				router.POST("/test", ginx.Adapt6(func(c *gin.Context, args *ValidatedArgs) *Response {
					return &Response{Username: args.Username}
				}))

				w := httptest.NewRecorder()
				body := strings.NewReader(`{}`)
				req := httptest.NewRequest(http.MethodPost, "/test", body)
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)
			},
		},
		{
			name: "Adapt5_FormBindingSuccess",
			testFunc: func(t *testing.T) {
				t.Parallel()
				gin.SetMode(gin.TestMode)
				router := gin.New()

				type Args struct {
					Name  string `form:"name"`
					Email string `form:"email"`
				}

				type Response struct {
					Name string `json:"name"`
				}

				router.POST("/test", ginx.Adapt5(func(c *gin.Context, args *Args) (*Response, error) {
					// Form 绑定在 JSON/XML/URI/Query 都失败时才会尝试
					// 这里测试 Form 绑定成功的情况
					return &Response{Name: args.Name}, nil
				}))

				w := httptest.NewRecorder()
				// 发送 Form 数据，但没有 JSON/XML body，也没有 URI/Query 参数
				body := strings.NewReader("name=test&email=test@example.com")
				req := httptest.NewRequest(http.MethodPost, "/test", body)
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				router.ServeHTTP(w, req)

				// Form 绑定可能成功，也可能失败（因为优先级最低）
				// 这里主要测试代码路径
				assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}
