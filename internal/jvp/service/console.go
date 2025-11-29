package service

import (
	"context"
	"fmt"

	"github.com/jimyag/jvp/internal/jvp/entity"
	"github.com/jimyag/jvp/pkg/apierror"
	"github.com/rs/zerolog"
)

// GetConsoleInfo 获取实例控制台连接信息
func (s *InstanceService) GetConsoleInfo(ctx context.Context, req *entity.GetConsoleRequest) (*entity.GetConsoleResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("node_name", req.NodeName).
		Str("instance_id", req.InstanceID).
		Str("type", req.Type).
		Msg("Getting console info for instance")

	// 1. 获取节点的 libvirt 客户端
	client, err := s.nodeService.GetNodeStorage(ctx, req.NodeName)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get node connection", err)
	}

	// 2. 验证实例存在
	instance, err := s.GetInstance(ctx, req.NodeName, req.InstanceID)
	if err != nil {
		return nil, apierror.NewErrorWithStatus(
			"ResourceNotFound",
			fmt.Sprintf("Instance %s not found on node %s", req.InstanceID, req.NodeName),
			404,
		)
	}

	// 3. 验证实例状态（建议运行状态才能连接控制台）
	if instance.State != "running" {
		logger.Warn().
			Str("instance_id", req.InstanceID).
			Str("state", instance.State).
			Msg("Instance is not running, console may not be available")
	}

	// 4. 获取 domain
	domain, err := client.GetDomainByName(req.InstanceID)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get domain from libvirt", err)
	}

	// 5. 获取控制台信息
	consoleInfo, err := client.GetDomainConsoleInfo(domain)
	if err != nil {
		return nil, apierror.WrapError(apierror.ErrInternalError, "Failed to get console info", err)
	}

	response := &entity.GetConsoleResponse{
		InstanceID: req.InstanceID,
	}

	// 6. 根据请求类型返回相应的控制台信息
	switch req.Type {
	case "vnc":
		if consoleInfo.VNCSocket == "" {
			return nil, apierror.NewErrorWithStatus(
				"ConsoleNotAvailable",
				"VNC console is not configured for this instance",
				400,
			)
		}
		response.VNCSocket = consoleInfo.VNCSocket
		response.Type = "vnc"
		logger.Info().
			Str("instance_id", req.InstanceID).
			Str("vnc_socket", consoleInfo.VNCSocket).
			Msg("VNC console info retrieved")

	case "serial":
		if consoleInfo.SerialDevice == "" {
			return nil, apierror.NewErrorWithStatus(
				"ConsoleNotAvailable",
				"Serial console is not available for this instance",
				400,
			)
		}
		response.SerialDevice = consoleInfo.SerialDevice
		response.Type = "serial"
		logger.Info().
			Str("instance_id", req.InstanceID).
			Str("serial_device", consoleInfo.SerialDevice).
			Msg("Serial console info retrieved")

	default:
		// 返回所有可用的控制台类型
		response.VNCSocket = consoleInfo.VNCSocket
		response.SerialDevice = consoleInfo.SerialDevice
		if response.VNCSocket != "" && response.SerialDevice != "" {
			response.Type = "both"
		} else if response.VNCSocket != "" {
			response.Type = "vnc"
		} else if response.SerialDevice != "" {
			response.Type = "serial"
		} else {
			return nil, apierror.NewErrorWithStatus(
				"ConsoleNotAvailable",
				"No console is available for this instance",
				400,
			)
		}
		logger.Info().
			Str("instance_id", req.InstanceID).
			Str("vnc_socket", consoleInfo.VNCSocket).
			Str("serial_device", consoleInfo.SerialDevice).
			Str("type", response.Type).
			Msg("Console info retrieved")
	}

	return response, nil
}
