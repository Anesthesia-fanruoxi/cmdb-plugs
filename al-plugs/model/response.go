package model

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// AccountBalance 账户余额信息
type AccountBalance struct {
	AvailableAmount     string `json:"available_amount"`      // 可用余额
	AvailableCashAmount string `json:"available_cash_amount"` // 现金余额
	CreditAmount        string `json:"credit_amount"`         // 信用额度
	Currency            string `json:"currency"`              // 币种
}

// EcsInstance ECS 实例信息
type EcsInstance struct {
	InstanceID   string `json:"instance_id"`   // 实例ID
	InstanceName string `json:"instance_name"` // 实例名称
	Status       string `json:"status"`        // 实例状态
	ExpiredTime  string `json:"expired_time"`  // 到期时间
	RegionID     string `json:"region_id"`     // 区域ID
	InstanceType string `json:"instance_type"` // 实例规格
	ChargeType   string `json:"charge_type"`   // 付费类型
}

// EcsExpiryResult ECS 过期查询结果
type EcsExpiryResult struct {
	RegionID  string        `json:"region_id"` // 查询区域
	Total     int           `json:"total"`     // 实例总数
	Instances []EcsInstance `json:"instances"` // 实例列表
}

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(data interface{}) *Response {
	return &Response{
		Code:    200,
		Message: "success",
		Data:    data,
	}
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(code int, message string) *Response {
	return &Response{
		Code:    code,
		Message: message,
	}
}
