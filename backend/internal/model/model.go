package model

import "time"

// Group 分组
type Group struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"-"`
	GID       string    `gorm:"column:gid;uniqueIndex;size:32" json:"gid"`
	Name      string    `gorm:"size:64" json:"name"`
	SortOrder int       `gorm:"column:sort_order" json:"sortOrder"`
	DelFlag   int       `gorm:"column:del_flag" json:"-"`
	CreatedAt time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (Group) TableName() string { return "t_group" }

// Link 短链接。DeletedAt=0 表示未删除；软删写入删除时间戳（秒）。
type Link struct {
	ID            int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ShortURI      string    `gorm:"column:short_uri;size:16" json:"shortUri"`
	OriginURL     string    `gorm:"column:origin_url;size:1024" json:"originUrl"`
	GID           string    `gorm:"column:gid;size:32;default:default" json:"gid"`
	Describe      string    `gorm:"column:describe;size:1024" json:"describe"`
	ValidDateType int       `gorm:"column:valid_date_type" json:"validDateType"` // 0永久 1自定义
	ValidDate     *time.Time `gorm:"column:valid_date" json:"validDate,omitempty"`
	EnableStatus  int       `gorm:"column:enable_status" json:"enableStatus"` // 0启用 1禁用
	TotalPV       int64     `gorm:"column:total_pv" json:"totalPv"`
	TotalUV       int64     `gorm:"column:total_uv" json:"totalUv"`
	TotalUIP      int64     `gorm:"column:total_uip" json:"totalUip"`
	Title         string    `gorm:"column:title" json:"title"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt     time.Time `gorm:"column:updated_at" json:"updatedAt"`
	DeletedAt     int64     `gorm:"column:deleted_at;default:0" json:"-"`
}

func (Link) TableName() string { return "t_link" }

// AccessStats 按天/小时聚合的访问统计
type AccessStats struct {
	ID       int64  `gorm:"primaryKey;autoIncrement"`
	ShortURI string `gorm:"column:short_uri;size:16"`
	Date     string `gorm:"column:date;size:10"` // YYYY-MM-DD
	Hour     int    `gorm:"column:hour"`         // 0-23，-1 表示全天聚合行
	Weekday  int    `gorm:"column:weekday"`      // 1-7 周一为1
	PV       int64  `gorm:"column:pv"`
	UV       int64  `gorm:"column:uv"`
	UIP      int64  `gorm:"column:uip"`
}

func (AccessStats) TableName() string { return "t_link_access_stats" }

// DimStats 维度统计（locale/browser/os/device）
type DimStats struct {
	ID       int64  `gorm:"primaryKey;autoIncrement"`
	ShortURI string `gorm:"column:short_uri;size:16"`
	Date     string `gorm:"column:date;size:10"`
	Dim      string `gorm:"column:dim;size:16"`
	Value    string `gorm:"column:value;size:64"`
	Cnt      int64  `gorm:"column:cnt"`
}

func (DimStats) TableName() string { return "t_link_dim_stats" }

// AccessLog 访问明细
type AccessLog struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ShortURI  string    `gorm:"column:short_uri;size:16" json:"shortUri"`
	IP        string    `gorm:"column:ip;size:64" json:"ip"`
	Browser   string    `gorm:"column:browser;size:64" json:"browser"`
	OS        string    `gorm:"column:os;size:64" json:"os"`
	Device    string    `gorm:"column:device;size:64" json:"device"`
	Locale    string    `gorm:"column:locale;size:128" json:"locale"`
	CreatedAt time.Time `gorm:"column:created_at" json:"createdAt"`
}

func (AccessLog) TableName() string { return "t_link_access_logs" }

// Setting 系统设置 key-value
type Setting struct {
	K         string    `gorm:"column:k;primaryKey;size:64"`
	V         string    `gorm:"column:v"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (Setting) TableName() string { return "t_settings" }
