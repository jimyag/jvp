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
	RegisterTemplate(ctx context.Context, req *entity.RegisterTemplateRequest) (*entity.Template, error)
	ListTemplates(ctx context.Context, req *entity.ListTemplatesRequest) ([]entity.Template, error)
	DescribeTemplate(ctx context.Context, req *entity.DescribeTemplateRequest) (*entity.Template, error)
	UpdateTemplate(ctx context.Context, req *entity.UpdateTemplateRequest) (*entity.Template, error)
	DeleteTemplate(ctx context.Context, req *entity.DeleteTemplateRequest) error
}

type Template struct {
	templateService TemplateServiceInterface
}

func NewTemplate(templateService *service.TemplateService) *Template {
	return &Template{
		templateService: templateService,
	}
}

func (t *Template) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/register-template", ginx.Adapt5(t.RegisterTemplate))
	router.POST("/list-templates", ginx.Adapt5(t.ListTemplates))
	router.POST("/describe-template", ginx.Adapt5(t.DescribeTemplate))
	router.POST("/update-template", ginx.Adapt5(t.UpdateTemplate))
	router.POST("/delete-template", ginx.Adapt5(t.DeleteTemplate))
}

func (t *Template) RegisterTemplate(ctx *gin.Context, req *entity.RegisterTemplateRequest) (*entity.RegisterTemplateResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("name", req.Name).
		Str("node_name", req.NodeName).
		Str("pool_name", req.PoolName).
		Msg("API: RegisterTemplate called")

	template, err := t.templateService.RegisterTemplate(ctx, req)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to register template")
		return nil, err
	}

	return &entity.RegisterTemplateResponse{Template: template}, nil
}

func (t *Template) ListTemplates(ctx *gin.Context, req *entity.ListTemplatesRequest) (*entity.ListTemplatesResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("pool_name", req.PoolName).
		Msg("API: ListTemplates called")

	templates, err := t.templateService.ListTemplates(ctx, req)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list templates")
		return nil, err
	}

	return &entity.ListTemplatesResponse{Templates: templates}, nil
}

func (t *Template) DescribeTemplate(ctx *gin.Context, req *entity.DescribeTemplateRequest) (*entity.DescribeTemplateResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("template_id", req.TemplateID).
		Str("node_name", req.NodeName).
		Msg("API: DescribeTemplate called")

	template, err := t.templateService.DescribeTemplate(ctx, req)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to describe template")
		return nil, err
	}

	return &entity.DescribeTemplateResponse{Template: template}, nil
}

func (t *Template) UpdateTemplate(ctx *gin.Context, req *entity.UpdateTemplateRequest) (*entity.UpdateTemplateResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("template_id", req.TemplateID).
		Str("node_name", req.NodeName).
		Msg("API: UpdateTemplate called")

	template, err := t.templateService.UpdateTemplate(ctx, req)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to update template")
		return nil, err
	}

	return &entity.UpdateTemplateResponse{Template: template}, nil
}

func (t *Template) DeleteTemplate(ctx *gin.Context, req *entity.DeleteTemplateRequest) (*entity.DeleteTemplateResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("template_id", req.TemplateID).
		Str("node_name", req.NodeName).
		Bool("delete_volume", req.DeleteVolume).
		Msg("API: DeleteTemplate called")

	if err := t.templateService.DeleteTemplate(ctx, req); err != nil {
		logger.Error().Err(err).Msg("Failed to delete template")
		return nil, err
	}

	return &entity.DeleteTemplateResponse{Deleted: true}, nil
}
