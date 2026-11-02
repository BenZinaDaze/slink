package handler

import (
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

// Login POST /api/v1/auth/login（每 IP 每分钟限 10 次）
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, errcode.ParamError)
		return
	}
	ip := middleware.ClientIP(c)
	if h.auth.LoginAttempt(c.Request.Context(), ip) > 10 {
		result.Fail(c, errcode.LoginRateLimited)
		return
	}
	token, err := h.auth.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		result.Fail(c, errcode.LoginFailed)
		return
	}
	result.OK(c, dto.LoginResp{Token: token, Username: req.Username})
}

// Me GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	username, isDefault := h.auth.Me(c.Request.Context())
	result.OK(c, dto.MeResp{Username: username, IsDefaultPassword: isDefault})
}

// Logout DELETE /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	token := bearerToken(c)
	_ = h.auth.Logout(c.Request.Context(), token)
	result.OK(c, nil)
}

// ChangeAccount PUT /api/v1/auth/account
// 修改用户名和/或密码；成功后旧会话全部吊销，返回新 token 续期当前会话
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
	result.OK(c, dto.LoginResp{Token: token, Username: username})
}

func bearerToken(c *gin.Context) string {
	if h := c.GetHeader("Authorization"); len(h) > 7 && h[:7] == "Bearer " {
		return h[7:]
	}
	return c.GetHeader("token")
}
