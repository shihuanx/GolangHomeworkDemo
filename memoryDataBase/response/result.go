package response

type Result struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func NewResult(code int, message string, data interface{}) *Result {
	return &Result{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

func Success(data interface{}) *Result {
	return NewResult(1, "success", data)
}

func SuccessWithoutData() *Result {
	return NewResult(1, "success", nil)
}

func Error(message string) *Result {
	return NewResult(0, message, nil)
}
