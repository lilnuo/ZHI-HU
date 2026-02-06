package e

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 通用 Response 结构
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// SuccessResponse 成功响应
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: Success,
		Msg:  "success",
		Data: data,
	})
}

// ErrorResponse 错误响应
func ErrorResponse(c *gin.Context, err error) {
	// 如果是我们自定义的业务错误
	var bizErr *Error
	if errors.As(err, &bizErr) {
		c.JSON(http.StatusOK, Response{
			Code: bizErr.Code,
			Msg:  bizErr.Msg,
			Data: nil,
		})
		return
	}
	log.Printf("[system error]URI :%s|Error :%v\n", c.Request.URL.Path, err)
	c.JSON(http.StatusInternalServerError, Response{
		Code: ErrorServer, // 500
		Msg:  "服务器内部错误",
		Data: nil,
	})
}
