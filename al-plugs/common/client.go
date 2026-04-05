package common

import (
	"al-plugs/config"

	bssopenapi20171214 "github.com/alibabacloud-go/bssopenapi-20171214/v6/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	credential "github.com/aliyun/credentials-go/credentials"
)

// CreateBssClient 创建阿里云 BSS OpenAPI 客户端
// 使用凭据初始化账号Client
func CreateBssClient(cfg *config.Config) (*bssopenapi20171214.Client, error) {
	var cred credential.Credential
	var err error

	// 如果配置中有 AccessKey，则使用 AccessKey 方式
	if cfg.Aliyun.AccessKeyID != "" && cfg.Aliyun.AccessKeySecret != "" {
		credConfig := &credential.Config{
			Type:            tea.String("access_key"),
			AccessKeyId:     tea.String(cfg.Aliyun.AccessKeyID),
			AccessKeySecret: tea.String(cfg.Aliyun.AccessKeySecret),
		}
		cred, err = credential.NewCredential(credConfig)
		if err != nil {
			return nil, err
		}
	} else {
		// 否则使用默认凭据链（环境变量、凭据文件等）
		// 工程代码建议使用更安全的无AK方式，凭据配置方式请参见：https://help.aliyun.com/document_detail/378661.html
		cred, err = credential.NewCredential(nil)
		if err != nil {
			return nil, err
		}
	}

	config := &openapi.Config{
		Credential: cred,
		// Endpoint 请参考 https://api.aliyun.com/product/BssOpenApi
		Endpoint: tea.String("business.aliyuncs.com"),
	}

	client, err := bssopenapi20171214.NewClient(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}
