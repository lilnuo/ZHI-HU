package handler

import (
	"github.com/gin-gonic/gin"
	"go-zhihu/pkg/e"
)

// 统一响应
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(200, Response{Code: e.Success,
		Msg:  "success",
		Data: data,
	})
}
func ErrorResponse(c *gin.Context, err error) {
	if biz, ok := err.(*e.Error); ok {
		c.JSON(200, Response{
			Code: biz.Code,
			Msg:  biz.Msg,
			Data: nil,
		})
		return
	}
	c.JSON(500, Response{
		Code: e.Error1,
		Msg:  "服务器内部错误",
		Data: nil,
	})
}
