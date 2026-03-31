package common

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"nacos-plugs/model"
)

func (c *NacosClient) GetConfig(dataId, group string) (string, error) {
	params := url.Values{}
	params.Set("dataId", dataId)
	params.Set("group", group)
	params.Set("tenant", c.Config.Namespace)

	reqURL := fmt.Sprintf("%s/v1/cs/configs?%s", c.BaseURL, params.Encode())

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	c.SetAuth(req)

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("获取配置失败, 状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

func (c *NacosClient) ListConfigs(pageNo, pageSize int) (*model.ConfigListResponse, error) {
	params := url.Values{}
	params.Set("dataId", "")
	params.Set("group", "")
	params.Set("tenant", c.Config.Namespace)
	params.Set("pageNo", fmt.Sprintf("%d", pageNo))
	params.Set("pageSize", fmt.Sprintf("%d", pageSize))

	reqURL := fmt.Sprintf("%s/v1/cs/configs?%s", c.BaseURL, params.Encode())

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	c.SetAuth(req)

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result model.ConfigListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &result, nil
}

func (c *NacosClient) SearchConfigs(dataId, group string, pageNo, pageSize int) (*model.ConfigListResponse, error) {
	params := url.Values{}
	params.Set("search", "blur")
	params.Set("dataId", dataId)
	params.Set("group", group)
	params.Set("tenant", c.Config.Namespace)
	params.Set("pageNo", fmt.Sprintf("%d", pageNo))
	params.Set("pageSize", fmt.Sprintf("%d", pageSize))

	reqURL := fmt.Sprintf("%s/v1/cs/configs?%s", c.BaseURL, params.Encode())

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	c.SetAuth(req)

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result model.ConfigListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &result, nil
}
