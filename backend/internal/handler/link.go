package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/benzi/slink/backend/internal/dto"
	"github.com/benzi/slink/backend/internal/repo"
	"github.com/benzi/slink/backend/internal/service"
	"github.com/benzi/slink/backend/pkg/errcode"
	"github.com/benzi/slink/backend/pkg/result"
)

type LinkHandler struct {
	link *service.LinkService
}

func NewLinkHandler(link *service.LinkService) *LinkHandler {
	return &LinkHandler{link: link}
}

// Create POST /api/v1/links
func (h *LinkHandler) Create(c *gin.Context) {
	var req dto.LinkCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, errcode.ParamError)
		return
	}
	resp, err := h.link.Create(c.Request.Context(), req)
	if err != nil {
		result.Fail(c, err)
		return
	}
	result.OK(c, resp)
}

// Update PUT /api/v1/links/:id
func (h *LinkHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.LinkUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, errcode.ParamError)
		return
	}
	resp, err := h.link.Update(c.Request.Context(), id, req)
	if err != nil {
		result.Fail(c, err)
		return
	}
	result.OK(c, resp)
}

// Page GET /api/v1/links
func (h *LinkHandler) Page(c *gin.Context) {
	var req dto.PageReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.Fail(c, errcode.ParamError)
		return
	}
	if req.Size == 0 {
		req.Size = 10
	}
	links, total, err := h.link.Page(c.Request.Context(), repo.LinkQuery{
		GID: req.GID, Keyword: req.Key, Order: req.Order, Page: req.Page, Size: req.Size,
	})
	if err != nil {
		result.Fail(c, err)
		return
	}
	result.OK(c, result.Page(req.Page, req.Size, total, links))
}

// GetByID GET /api/v1/links/:id
func (h *LinkHandler) GetByID(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	resp, err := h.link.GetByID(c.Request.Context(), id)
	if err != nil {
		result.Fail(c, err)
		return
	}
	result.OK(c, resp)
}

// Delete DELETE /api/v1/links/:id
func (h *LinkHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.link.SoftDelete(c.Request.Context(), id); err != nil {
		result.Fail(c, err)
		return
	}
	result.OK(c, nil)
}

// FetchTitle GET /api/v1/links/title?url=
func (h *LinkHandler) FetchTitle(c *gin.Context) {
	rawURL := c.Query("url")
	if rawURL == "" {
		result.Fail(c, errcode.ParamError)
		return
	}
	title, err := h.link.FetchTitle(c.Request.Context(), rawURL)
	if err != nil {
		result.Fail(c, err)
		return
	}
	result.OK(c, map[string]string{"title": title})
}

// PageRecycle GET /api/v1/recycle-bin
func (h *LinkHandler) PageRecycle(c *gin.Context) {
	var req dto.PageReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.Fail(c, errcode.ParamError)
		return
	}
	if req.Size == 0 {
		req.Size = 10
	}
	links, total, err := h.link.PageRecycle(c.Request.Context(), req.Page, req.Size, req.Key)
	if err != nil {
		result.Fail(c, err)
		return
	}
	result.OK(c, result.Page(req.Page, req.Size, total, links))
}

// Restore POST /api/v1/recycle-bin/:id/restore
func (h *LinkHandler) Restore(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	resp, err := h.link.Restore(c.Request.Context(), id)
	if err != nil {
		result.Fail(c, err)
		return
	}
	result.OK(c, resp)
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		result.Fail(c, errcode.ParamError)
		return 0, false
	}
	return id, true
}
