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
