package e

import "fmt"

const (
	Success            = 0
	ErrorServer        = 500
	ErrorInvalidParams = 400
	//用户错误代码
	ErrUserExist      = 10001
	ErrUserNotFound   = 10002
	ErrPassword       = 10003
	ErrorUserBanned   = 10004
	ErrorToken        = 10005
	ErrPermisson      = 10006
	ErrActionFailed   = 10007
	ErrorPostNotFound = 20001
	ErrUnAuthorized   = 40101
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
	ErrToken                = New(ErrorToken, "Token 生成失败")
	ErrPermission           = New(ErrPermisson, "无权修改")
	ErrSelfAction           = New(ErrActionFailed, "不能对自己执行此操作")
	ErrAlreadyFollowing     = New(ErrActionFailed, "已经关注了")
	ErrUserNormal           = New(ErrActionFailed, "用户状态正常，无需操作")
	ErrUnAuthorizedInstance = New(ErrUnAuthorized, "未登录或token无效")
)
