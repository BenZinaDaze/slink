package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/benzi/slink/backend/internal/dto"
	"github.com/benzi/slink/backend/internal/service"
	"github.com/benzi/slink/backend/pkg/errcode"
	"github.com/benzi/slink/backend/pkg/result"
)

type StatsHandler struct {
	stats     *service.StatsService
	dashboard *service.DashboardService
}

func NewStatsHandler(stats *service.StatsService, dashboard *service.DashboardService) *StatsHandler {
	return &StatsHandler{stats: stats, dashboard: dashboard}
}

// Overview GET /api/v1/links/:id/stats?start=&end=
func (h *StatsHandler) Overview(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.StatsRangeReq
	if err := c.ShouldBindQuery(&req); err != nil {
		// 不传日期时默认今天
		req = dto.StatsRangeReq{}
	}
	data, err := h.stats.Overview(c.Request.Context(), id, req.Start, req.End)
	if err != nil {
		result.Fail(c, err)
		return
	}
	result.OK(c, data)
}

// Hourly GET /api/v1/links/:id/stats/hourly?start=&end=
func (h *StatsHandler) Hourly(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.StatsRangeReq
	_ = c.ShouldBindQuery(&req)
	data, err := h.stats.Hourly(c.Request.Context(), id, req.Start, req.End)
	if err != nil {
		result.Fail(c, err)
		return
	}
	result.OK(c, data)
}

// Weekday GET /api/v1/links/:id/stats/weekday?start=&end=
func (h *StatsHandler) Weekday(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.StatsRangeReq
	_ = c.ShouldBindQuery(&req)
	data, err := h.stats.Weekday(c.Request.Context(), id, req.Start, req.End)
	if err != nil {
		result.Fail(c, err)
		return
	}
	result.OK(c, data)
}

// Dim GET /api/v1/links/:id/stats/dim?dim=locale|browser|os|device
func (h *StatsHandler) Dim(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.DimReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.Fail(c, errcode.ParamError)
		return
	}
	data, err := h.stats.Dim(c.Request.Context(), id, req.Dim, req.Start, req.End)
	if err != nil {
		result.Fail(c, err)
		return
	}
	result.OK(c, data)
}

// Records GET /api/v1/links/:id/stats/records?page=&size=
func (h *StatsHandler) Records(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.PageReq
	_ = c.ShouldBindQuery(&req)
	if req.Size == 0 {
		req.Size = 10
	}
	logs, total, err := h.stats.Records(c.Request.Context(), id, req.Page, req.Size)
	if err != nil {
		result.Fail(c, err)
		return
	}
	result.OK(c, result.Page(req.Page, req.Size, total, logs))
}

// Dashboard GET /api/v1/dashboard
func (h *StatsHandler) Dashboard(c *gin.Context) {
	data, err := h.dashboard.Overview(c.Request.Context())
	if err != nil {
		result.Fail(c, err)
		return
	}
	result.OK(c, data)
}
