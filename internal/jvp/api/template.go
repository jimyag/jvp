package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/internal/jvp/service"
	"github.com/jimyag/jvp/pkg/ginx"
	"github.com/rs/zerolog"
)

// TemplateServiceInterface 定义模板服务的接口
type TemplateServiceInterface interface {
	ListVMTemplates(ctx context.Context) ([]entity.VMTemplate, error)
}

type Template struct {
	instanceService TemplateServiceInterface
}

func NewTemplate(instanceService *service.InstanceService) *Template {
	return &Template{
		instanceService: instanceService,
	}
}

func (t *Template) RegisterRoutes(router *gin.RouterGroup) {
	// 注册 /api/vm-templates 路由
	router.GET("/vm-templates", ginx.Adapt3(t.ListVMTemplates))
}

// ListVMTemplates 列出所有 VM 模板
func (t *Template) ListVMTemplates(ctx *gin.Context) (*entity.ListVMTemplatesResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().Msg("ListVMTemplates called")

	templates, err := t.instanceService.ListVMTemplates(ctx)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to list VM templates")
		return nil, err
	}

	logger.Info().
		Int("count", len(templates)).
		Msg("VM templates listed successfully")

	return &entity.ListVMTemplatesResponse{
		Templates: templates,
	}, nil
}
