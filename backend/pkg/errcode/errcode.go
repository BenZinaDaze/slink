package errcode

import "fmt"

// Error 统一业务错误
// A* 用户端错误，B* 系统错误，C* 短链业务错误
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func New(code, message string) *Error {
	return &Error{Code: code, Message: message}
}

var (
	Success           = New("0", "success")
	ServerError       = New("B0001", "系统访问错误，请稍候重试")
	ParamError        = New("A0001", "参数校验失败")
	Unauthorized      = New("A0231", "登录已过期或未登录")
	LoginFailed       = New("A0210", "用户名或密码错误")
	LoginRateLimited  = New("A0301", "登录尝试过于频繁，请稍后再试")
	OldPasswordWrong  = New("A0220", "原密码错误")

	GroupNotFound     = New("C0001", "分组不存在")
	GroupHasLinks     = New("C0002", "分组下存在短链接，无法删除")
	GroupNameExists   = New("C0003", "分组名称已存在")

	LinkNotFound      = New("C0101", "短链接不存在")
	ShortURIExists    = New("C0102", "短链接后缀已存在")
	ShortURIInvalid   = New("C0103", "自定义后缀仅支持 2-16 位字母、数字、- 或 _")
	OriginURLInvalid  = New("C0104", "原始链接格式错误，仅支持 http/https 且长度不超过 1024")
	URLNotWhitelisted = New("C0105", "目标域名不在白名单内")
	LinkExpired       = New("C0106", "短链接已过期")
	LinkDisabled      = New("C0107", "短链接已被禁用")
	InternalURLDenied = New("C0108", "不允许访问内网地址")
)
