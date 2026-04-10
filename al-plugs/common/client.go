package common

import (
	"al-plugs/config"

	bssopenapi20171214 "github.com/alibabacloud-go/bssopenapi-20171214/v6/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
	credential "github.com/aliyun/credentials-go/credentials"
)

// CreateBssClient 创建阿里云 BSS OpenAPI 客户端
func CreateBssClient(cfg *config.Config) (*bssopenapi20171214.Client, error) {
	cred, err := newCredential(cfg)
	if err != nil {
		return nil, err
	}

	openapiCfg := &openapi.Config{
		Credential: cred,
		Endpoint:   tea.String("business.aliyuncs.com"),
	}

	return bssopenapi20171214.NewClient(openapiCfg)
}

// CreateEcsClient 创建阿里云 ECS 客户端
func CreateEcsClient(cfg *config.Config) (*ecs20140526.Client, error) {
	cred, err := newCredential(cfg)
	if err != nil {
		return nil, err
	}

	endpoint := "ecs." + cfg.Aliyun.RegionID + ".aliyuncs.com"
	openapiCfg := &openapi.Config{
		Credential: cred,
		Endpoint:   tea.String(endpoint),
	}

	return ecs20140526.NewClient(openapiCfg)
}

// newCredential 根据配置创建凭据
func newCredential(cfg *config.Config) (credential.Credential, error) {
	if cfg.Aliyun.AccessKeyID != "" && cfg.Aliyun.AccessKeySecret != "" {
		credConfig := &credential.Config{
			Type:            tea.String("access_key"),
			AccessKeyId:     tea.String(cfg.Aliyun.AccessKeyID),
			AccessKeySecret: tea.String(cfg.Aliyun.AccessKeySecret),
		}
		return credential.NewCredential(credConfig)
	}
	return credential.NewCredential(nil)
}
