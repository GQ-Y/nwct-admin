package models

import "time"

// Response 统一响应格式
type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// NewResponse 创建响应
func NewResponse(code int, message string, data interface{}) *Response {
	return &Response{
		Code:      code,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// SuccessResponse 成功响应
func SuccessResponse(data interface{}) *Response {
	return NewResponse(200, "success", data)
}

// ErrorResponse 错误响应
func ErrorResponse(code int, message string) *Response {
	return NewResponse(code, message, nil)
}

