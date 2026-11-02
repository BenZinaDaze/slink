package dto

import "time"

// ---------- auth ----------

type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResp struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}

type MeResp struct {
	Username           string `json:"username"`
	IsDefaultPassword  bool   `json:"isDefaultPassword"` // 是否仍在使用配置初始密码
}

type AccountUpdateReq struct {
	Username    string `json:"username"`                         // 新用户名，空 = 不修改
	OldPassword string `json:"oldPassword" binding:"required"`   // 验证原密码
	NewPassword string `json:"newPassword"`                      // 新密码，空 = 不修改
}

// ---------- group ----------

type GroupReq struct {
	Name      string `json:"name" binding:"required,max=64"`
	SortOrder *int   `json:"sortOrder"`
}

type GroupSortItem struct {
	GID       string `json:"gid" binding:"required"`
	SortOrder int    `json:"sortOrder"`
}

type GroupSortReq struct {
	Items []GroupSortItem `json:"items" binding:"required,min=1"`
}

type GroupResp struct {
	GID       string    `json:"gid"`
	Name      string    `json:"name"`
	SortOrder int       `json:"sortOrder"`
	LinkCount int64     `json:"linkCount"`
	CreatedAt time.Time `json:"createdAt"`
}

// ---------- link ----------

type LinkCreateReq struct {
	OriginURL     string     `json:"originUrl" binding:"required,url,max=1024"`
	GID           string     `json:"gid" binding:"required"`
	CustomURI     string     `json:"customUri"`
	Describe      string     `json:"describe,max=1024"`
	ValidDateType int        `json:"validDateType"` // 0永久 1自定义
	ValidDate     *time.Time `json:"validDate"`
	Title         string     `json:"title"`
}

type LinkUpdateReq struct {
	OriginURL     string     `json:"originUrl" binding:"required,url,max=1024"`
	GID           string     `json:"gid" binding:"required"`
	Describe      string     `json:"describe,max=1024"`
	ValidDateType int        `json:"validDateType"`
	ValidDate     *time.Time `json:"validDate"`
	EnableStatus  *int       `json:"enableStatus"` // 0启用 1禁用
	Title         string     `json:"title"`
}

type LinkResp struct {
	ID            int64      `json:"id"`
	ShortURI      string     `json:"shortUri"`
	FullShortURL  string     `json:"fullShortUrl"`
	OriginURL     string     `json:"originUrl"`
	GID           string     `json:"gid"`
	Describe      string     `json:"describe"`
	ValidDateType int        `json:"validDateType"`
	ValidDate     *time.Time `json:"validDate,omitempty"`
	EnableStatus  int        `json:"enableStatus"`
	TotalPV       int64      `json:"totalPv"`
	TotalUV       int64      `json:"totalUv"`
	TotalUIP      int64      `json:"totalUip"`
	Title         string     `json:"title"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// ---------- stats ----------

// Point2 通用二维数据点（分布图/柱状图）
type Point2 struct {
	Label string `json:"label"`
	Value int64  `json:"value"`
}

type StatsRangeReq struct {
	Start string `form:"start" binding:"required,datetime=2006-01-02"`
	End   string `form:"end" binding:"required,datetime=2006-01-02"`
}

type DimReq struct {
	StatsRangeReq
	Dim string `form:"dim" binding:"required,oneof=locale browser os device"`
}

type PageReq struct {
	Page int64  `form:"page,default=1" binding:"min=1"`
	Size int64  `form:"size,default=10" binding:"min=1,max=100"`
	GID  string `form:"gid"`
	Key  string `form:"key"`
	Order string `form:"order"`
}
