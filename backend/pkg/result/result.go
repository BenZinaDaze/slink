package result

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/benzi/slink/backend/pkg/errcode"
)

// R 统一响应结构
type R struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Success   bool   `json:"success"`
	RequestID string `json:"requestId"`
}

// PageResp 分页数据
type PageResp struct {
	Current  int64 `json:"current"`
	Size     int64 `json:"size"`
	Total    int64 `json:"total"`
	Records  any   `json:"records"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, R{
		Code:      errcode.Success.Code,
		Message:   errcode.Success.Message,
		Data:      data,
		Success:   true,
		RequestID: c.GetString("requestId"),
	})
}

func Fail(c *gin.Context, err error) {
	var e *errcode.Error
	if !errors.As(err, &e) {
		e = errcode.ServerError
	}
	c.JSON(http.StatusOK, R{
		Code:      e.Code,
		Message:   err.Error(),
		Success:   false,
		RequestID: c.GetString("requestId"),
	})
}

// Page 构造分页响应
func Page(current, size, total int64, records any) *PageResp {
	return &PageResp{Current: current, Size: size, Total: total, Records: records}
}
