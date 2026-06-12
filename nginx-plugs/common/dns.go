package common

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"nginx-plugs/config"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	dnsEndpoint   = "https://alidns.aliyuncs.com"
	dnsAPIVersion = "2015-01-09"
)

// DNSRecord DNS记录信息
type DNSRecord struct {
	RecordId   string `json:"RecordId"`
	RR         string `json:"RR"`
	Type       string `json:"Type"`
	Value      string `json:"Value"`
	DomainName string `json:"DomainName"`
}

// AddDomainRecordResponse 添加DNS记录响应
type AddDomainRecordResponse struct {
	RecordId  string `json:"RecordId"`
	RequestId string `json:"RequestId"`
}

// ErrorResponse 阿里云错误响应
type aliyunErrorResponse struct {
	Code      string `json:"Code"`
	Message   string `json:"Message"`
	RequestId string `json:"RequestId"`
}

// AddDomainRecord 添加DNS记录
// domain: 主域名（如 hzlsg.com）
// rr: 二级域名（如 api）
// recordType: 记录类型（CNAME 或 A）
// value: 记录值（CNAME为域名，A为IP）
func AddDomainRecord(domain, rr, recordType, value string) (*AddDomainRecordResponse, error) {
	params := map[string]string{
		"Action":     "AddDomainRecord",
		"DomainName": domain,
		"RR":         rr,
		"Type":       recordType,
		"Value":      value,
	}

	body, err := doRequest(params)
	if err != nil {
		return nil, err
	}

	var resp AddDomainRecordResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	Logger.Infof("DNS %s记录已添加: %s.%s -> %s (RecordId: %s)", recordType, rr, domain, value, resp.RecordId)
	return &resp, nil
}

// DeleteDomainRecordById 按RecordId直接删除DNS记录
func DeleteDomainRecordById(recordId, serverName string) error {
	params := map[string]string{
		"Action":   "DeleteDomainRecord",
		"RecordId": recordId,
	}

	_, err := doRequest(params)
	if err != nil {
		return fmt.Errorf("删除DNS记录失败: %w", err)
	}

	Logger.Infof("DNS记录已删除(ById): %s (RecordId: %s)", serverName, recordId)
	return nil
}

// doRequest 执行阿里云API请求
func doRequest(bizParams map[string]string) ([]byte, error) {
	aliyunConf := config.GetAliyunConfig()
	if aliyunConf.AccessKeyID == "" || aliyunConf.AccessKeySecret == "" {
		return nil, fmt.Errorf("阿里云AccessKey未配置")
	}

	// 构建公共参数
	params := map[string]string{
		"Format":           "JSON",
		"Version":          dnsAPIVersion,
		"AccessKeyId":      aliyunConf.AccessKeyID,
		"SignatureMethod":  "HMAC-SHA1",
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"SignatureVersion": "1.0",
		"SignatureNonce":   strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	// 合并业务参数
	for k, v := range bizParams {
		params[k] = v
	}

	// 构造签名字符串
	signature := signRequest(params, aliyunConf.AccessKeySecret)
	params["Signature"] = signature

	// 构造请求
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}

	resp, err := http.PostForm(dnsEndpoint, values)
	if err != nil {
		return nil, fmt.Errorf("请求阿里云API失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查错误响应
	if resp.StatusCode != http.StatusOK {
		var errResp aliyunErrorResponse
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Code != "" {
			return nil, fmt.Errorf("阿里云API错误 [%s]: %s", errResp.Code, errResp.Message)
		}
		return nil, fmt.Errorf("阿里云API返回HTTP %d: %s", resp.StatusCode, string(body))
	}

	// 检查业务错误（HTTP 200但返回了错误码）
	var errCheck aliyunErrorResponse
	if json.Unmarshal(body, &errCheck) == nil && errCheck.Code != "" {
		return nil, fmt.Errorf("阿里云API错误 [%s]: %s", errCheck.Code, errCheck.Message)
	}

	return body, nil
}

// signRequest 生成阿里云API签名
func signRequest(params map[string]string, accessKeySecret string) string {
	// 1. 按参数名排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 2. 构造规范化查询字符串
	var canonicalized []string
	for _, k := range keys {
		canonicalized = append(canonicalized, percentEncode(k)+"="+percentEncode(params[k]))
	}
	canonicalizedQuery := strings.Join(canonicalized, "&")

	// 3. 构造签名字符串: POST&%2F&<canonicalized_query>
	stringToSign := "POST&%2F&" + percentEncode(canonicalizedQuery)

	// 4. HMAC-SHA1签名
	mac := hmac.New(sha1.New, []byte(accessKeySecret+"&"))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return signature
}

// percentEncode URL编码（符合阿里云规范）
func percentEncode(s string) string {
	s = url.QueryEscape(s)
	s = strings.ReplaceAll(s, "+", "%20")
	s = strings.ReplaceAll(s, "*", "%2A")
	s = strings.ReplaceAll(s, "%7E", "~")
	return s
}
