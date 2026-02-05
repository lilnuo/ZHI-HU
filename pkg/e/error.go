package e

import "fmt"

const (
	Success         = 0
	Error1          = 500
	InvalidParams   = 400
	ErrUserExist    = 10001
	ErrUserNotFound = 10002
	ErrPassword     = 10003
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
	ErrUserNotFoundInstance = New(ErrUserNotFound, "用户不存在")
	ErrPasswordInstance     = New(ErrPassword, "密码错误")
)
