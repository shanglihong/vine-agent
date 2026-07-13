package dto

import "net/http"

type BaseResp struct {
	Code    int
	Message string
	Data    any
}

func NewErrorResp(err error) BaseResp {
	return BaseResp{
		Code:    http.StatusInternalServerError,
		Message: err.Error(),
	}
}

func NewSuccessResp(data any) BaseResp {
	return BaseResp{
		Code:    0,
		Message: http.StatusText(http.StatusOK),
		Data:    data,
	}
}

type Null struct{}
