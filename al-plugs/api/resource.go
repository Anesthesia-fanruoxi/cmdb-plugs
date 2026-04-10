package api

import (
	"encoding/json"
	"strings"
	"time"

	"al-plugs/common"
	"al-plugs/config"
	"al-plugs/logger"
	"al-plugs/model"

	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
)

// ResourceAPI 资源相关接口
type ResourceAPI struct {
	cfg *config.Config
}

// NewResourceAPI 创建资源API实例
func NewResourceAPI(cfg *config.Config) *ResourceAPI {
	return &ResourceAPI{cfg: cfg}
}

// QueryEcsExpiry 查询 ECS 实例过期时间（支持多区域）
// GET /api/resource/ecs/expiry
func (r *ResourceAPI) QueryEcsExpiry(c *gin.Context) {
	startTime := time.Now()
	regionIDs := r.cfg.Aliyun.RegionIDs

	var allInstances []model.EcsInstance
	for _, regionID := range regionIDs {
		client, err := common.CreateEcsClient(r.cfg, regionID)
		if err != nil {
			logger.Error("创建 ECS 客户端失败 [%s]: %v", regionID, err)
			c.JSON(500, model.NewErrorResponse(500, "创建 ECS 客户端失败: "+err.Error()))
			return
		}

		instances, err := fetchAllEcsInstances(client, regionID)
		if err != nil {
			if sdkErr, ok := err.(*tea.SDKError); ok {
				msg := buildSdkErrorMsg(sdkErr)
				logger.Error("查询 ECS 实例失败 [%s]: %s", regionID, msg)
				c.JSON(500, model.NewErrorResponse(500, msg))
				return
			}
			logger.Error("查询 ECS 实例失败 [%s]: %v", regionID, err)
			c.JSON(500, model.NewErrorResponse(500, err.Error()))
			return
		}
		allInstances = append(allInstances, instances...)
	}

	result := &model.EcsExpiryResult{
		RegionIDs: regionIDs,
		Total:     len(allInstances),
		Instances: allInstances,
	}

	logger.Info("查询 ECS 实例成功，区域 %v，共 %d 个，耗时 %dms", regionIDs, len(allInstances), time.Since(startTime).Milliseconds())
	c.JSON(200, model.NewSuccessResponse(result))
}

// fetchAllEcsInstances 分页拉取所有 ECS 实例
func fetchAllEcsInstances(client *ecs20140526.Client, regionID string) ([]model.EcsInstance, error) {
	var all []model.EcsInstance
	pageNumber := int32(1)
	pageSize := int32(100)
	runtime := &util.RuntimeOptions{}

	for {
		req := &ecs20140526.DescribeInstancesRequest{
			RegionId:   tea.String(regionID),
			PageNumber: tea.Int32(pageNumber),
			PageSize:   tea.Int32(pageSize),
		}

		resp, err := client.DescribeInstancesWithOptions(req, runtime)
		if err != nil {
			return nil, err
		}

		if resp.Body == nil || resp.Body.Instances == nil {
			break
		}

		for _, inst := range resp.Body.Instances.Instance {
			all = append(all, model.EcsInstance{
				InstanceID:   tea.StringValue(inst.InstanceId),
				InstanceName: tea.StringValue(inst.InstanceName),
				Status:       tea.StringValue(inst.Status),
				ExpiredTime:  tea.StringValue(inst.ExpiredTime),
				RegionID:     tea.StringValue(inst.RegionId),
				InstanceType: tea.StringValue(inst.InstanceType),
				ChargeType:   tea.StringValue(inst.InstanceChargeType),
			})
		}

		total := tea.Int32Value(resp.Body.TotalCount)
		if int32(len(all)) >= total {
			break
		}
		pageNumber++
	}

	return all, nil
}

// buildSdkErrorMsg 构造 SDK 错误信息（含诊断建议）
func buildSdkErrorMsg(sdkErr *tea.SDKError) string {
	msg := tea.StringValue(sdkErr.Message)
	var data interface{}
	d := json.NewDecoder(strings.NewReader(tea.StringValue(sdkErr.Data)))
	err := d.Decode(&data)
	if err != nil {
		return ""
	}
	if m, ok := data.(map[string]interface{}); ok {
		if recommend, exists := m["Recommend"]; exists {
			msg += " | 建议: " + recommend.(string)
		}
	}
	return msg
}
