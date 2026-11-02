package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/benzi/slink/backend/internal/dto"
	"github.com/benzi/slink/backend/internal/service"
	"github.com/benzi/slink/backend/pkg/errcode"
	"github.com/benzi/slink/backend/pkg/result"
)

type GroupHandler struct {
	group *service.GroupService
}

func NewGroupHandler(group *service.GroupService) *GroupHandler {
	return &GroupHandler{group: group}
}

// List GET /api/v1/groups
func (h *GroupHandler) List(c *gin.Context) {
	groups, err := h.group.List(c.Request.Context())
	if err != nil {
		result.Fail(c, err)
		return
	}
	result.OK(c, groups)
}

// Create POST /api/v1/groups
func (h *GroupHandler) Create(c *gin.Context) {
	var req dto.GroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, errcode.ParamError)
		return
	}
	g, err := h.group.Create(c.Request.Context(), req)
	if err != nil {
		result.Fail(c, err)
		return
	}
	result.OK(c, g)
}

// Update PUT /api/v1/groups/:gid
func (h *GroupHandler) Update(c *gin.Context) {
	var req dto.GroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, errcode.ParamError)
		return
	}
	if err := h.group.Update(c.Request.Context(), c.Param("gid"), req); err != nil {
		result.Fail(c, err)
		return
	}
	result.OK(c, nil)
}

// Delete DELETE /api/v1/groups/:gid
func (h *GroupHandler) Delete(c *gin.Context) {
	if err := h.group.Delete(c.Request.Context(), c.Param("gid")); err != nil {
		result.Fail(c, err)
		return
	}
	result.OK(c, nil)
}

// Sort PUT /api/v1/groups/sort
func (h *GroupHandler) Sort(c *gin.Context) {
	var req dto.GroupSortReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Fail(c, errcode.ParamError)
		return
	}
	if err := h.group.Sort(c.Request.Context(), req); err != nil {
		result.Fail(c, err)
		return
	}
	result.OK(c, nil)
}
