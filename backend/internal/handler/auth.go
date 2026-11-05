package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/benzi/slink/backend/internal/dto"
	"github.com/benzi/slink/backend/internal/middleware"
	"github.com/benzi/slink/backend/internal/service"
	"github.com/benzi/slink/backend/pkg/errcode"
	"github.com/benzi/slink/backend/pkg/result"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Login POST /api/v1/auth/login
// 三层防爆破：每 IP 每分钟尝试上限 → 失败指数退避锁 → 凭证校验；
// 成功后 token 写 HttpOnly Cookie，响应体不携带 token
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, errcode.ParamError)
		return
	}
	ctx := c.Request.Context()
	ip := c.ClientIP()
	if h.auth.LoginAttempt(ctx, ip) > 10 {
		result.Fail(c, errcode.LoginRateLimited)
		return
	}
	if h.auth.LoginLocked(ctx, ip) > 0 {
		result.Fail(c, errcode.LoginRateLimited)
		return
	}
	token, err := h.auth.Login(ctx, req.Username, req.Password)
	h.auth.RecordLoginResult(ctx, ip, err == nil)
	if err != nil {
		result.Fail(c, errcode.LoginFailed)
		return
	}
	h.setAuthCookie(c, token)
	result.OK(c, dto.LoginResp{Username: req.Username, ShortDomain: h.auth.ShortDomain()})
}

// Me GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	username, isDefault, mustChange := h.auth.Me(c.Request.Context())
	result.OK(c, dto.MeResp{
		Username:           username,
		ShortDomain:        h.auth.ShortDomain(),
		IsDefaultPassword:  isDefault,
		MustChangePassword: mustChange,
	})
}

// Logout DELETE /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	_ = h.auth.Logout(c.Request.Context(), middleware.TokenFromRequest(c))
	h.clearAuthCookie(c)
	result.OK(c, nil)
}

// ChangeAccount PUT /api/v1/auth/account
// 修改用户名和/或密码；成功后旧会话全部吊销，新 token 写回 Cookie 保持当前会话
func (h *AuthHandler) ChangeAccount(c *gin.Context) {
	var req dto.AccountUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, errcode.ParamError)
		return
	}
	token, username, err := h.auth.ChangeAccount(c.Request.Context(), req.Username, req.OldPassword, req.NewPassword)
	if err != nil {
		result.Fail(c, err)
		return
	}
	h.setAuthCookie(c, token)
	result.OK(c, dto.LoginResp{Username: username, ShortDomain: h.auth.ShortDomain()})
}

// setAuthCookie 登录凭证走 HttpOnly Cookie（消除 XSS 窃取面），Max-Age 与 token TTL 一致
func (h *AuthHandler) setAuthCookie(c *gin.Context, token string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     middleware.TokenCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(h.auth.TokenTTL().Seconds()),
		HttpOnly: true,
		Secure:   cookieSecure(c, h.auth.CookieSecureOverride()),
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *AuthHandler) clearAuthCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: middleware.TokenCookie, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: cookieSecure(c, h.auth.CookieSecureOverride()),
		SameSite: http.SameSiteStrictMode,
	})
}

// cookieSecure Cookie 的 Secure 属性：显式配置优先；
// 自动模式 = 请求本身是 TLS，或受信代理转发（c.ClientIP 与直连地址不一致说明代理头已生效）的 https
func cookieSecure(c *gin.Context, override *bool) bool {
	if override != nil {
		return *override
	}
	if c.Request.TLS != nil {
		return true
	}
	return strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") && c.ClientIP() != c.RemoteIP()
}
