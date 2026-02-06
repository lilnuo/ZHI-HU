package e

import "fmt"

const (
	Success            = 0
	ErrorServer        = 500
	ErrorInvalidParams = 400
	//用户错误代码
	ErrUserExist    = 10001
	ErrUserNotFound = 10002
	ErrPassword     = 10003
	ErrorUserBanned = 10004
	//业务
	ErrorPostNotFound = 20001
)

type Error struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("code:%d,msg:%s", e.Code, e.Msg)
}
func New(code int, msg string) *Error {
	return &Error{
		Code: code,
		Msg:  msg,
	}
}

var (
	ErrSuccess              = New(Success, "success")
	ErrServer               = New(ErrorServer, "服务器内部错误")
	ErrInvalidArgs          = New(ErrorInvalidParams, "参数错误")
	ErrorUserExist          = New(ErrUserExist, "用户已存在")
	ErrUserNotFoundInstance = New(ErrUserNotFound, "用户不存在")
	ErrPasswordInstance     = New(ErrPassword, "密码错误")
	ErrUserBanned           = New(ErrorUserBanned, "用户被禁言")
	ErrPostNotFound         = New(ErrorPostNotFound, "文章不存在")
)
