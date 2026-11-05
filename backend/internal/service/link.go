package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
	"gorm.io/gorm"

	"github.com/benzi/slink/backend/internal/config"
	"github.com/benzi/slink/backend/internal/dto"
	"github.com/benzi/slink/backend/internal/model"
	"github.com/benzi/slink/backend/internal/repo"
	"github.com/benzi/slink/backend/pkg/bloom"
	"github.com/benzi/slink/backend/pkg/errcode"
	"github.com/benzi/slink/backend/pkg/shortgen"
)

// 自定义后缀：2-16 位（短如 "go" 便于记忆，长如 "install-script" 语义明确），
// 仅字母/数字/-/_（不含 "."，避免与静态资源路径冲突）
var customURIRe = regexp.MustCompile(`^[0-9a-zA-Z_-]{2,16}$`)

// rootModeReserved 旧版单域根路径模式的保留字：SPA 挂在根路径，与前端路由冲突的
// 短码不允许创建，NoRoute 解析时也优先按 SPA 处理
var rootModeReserved = map[string]struct{}{
	"login": {}, "links": {}, "recycle-bin": {}, "settings": {},
	"assets": {}, "api": {}, "dashboard": {}, "link-error": {},
}

// ReservedPathsFor 计算当前模式下的保留短码集合：
// - 根路径模式（adminPath 为空）→ 全量前端路由保留字
// - 隐藏入口模式（配置 admin.path）→ 管理面整体挪到该前缀下，根路径只剩真实路由
//   （api/health）与管理入口本身；login/links 等都可以作为短码使用
func ReservedPathsFor(adminPath string) map[string]struct{} {
	if adminPath == "" {
		return rootModeReserved
	}
	return map[string]struct{}{
		adminPath: {}, "api": {}, "health": {},
	}
}

type LinkService struct {
	cfg   *config.Config
	link  *repo.LinkRepo
	group *repo.GroupRepo
	gen   *shortgen.Generator
	bloom *bloom.Filter
	cache *LinkCache
}

func NewLinkService(cfg *config.Config, link *repo.LinkRepo, group *repo.GroupRepo, gen *shortgen.Generator, bf *bloom.Filter, cache *LinkCache) *LinkService {
	return &LinkService{cfg: cfg, link: link, group: group, gen: gen, bloom: bf, cache: cache}
}

// isReserved 当前模式下该短码是否与真实路由冲突（保留字随 admin.path 动态变化）
func (s *LinkService) isReserved(uri string) bool {
	_, ok := ReservedPathsFor(s.cfg.Admin.Path)[uri]
	return ok
}

// validateURL 校验原始链接：协议/长度/白名单/防 SSRF（内网地址）
func (s *LinkService) validateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errcode.OriginURLInvalid
	}
	host := u.Hostname()
	if len(s.cfg.ShortLink.Whitelist) > 0 {
		ok := false
		for _, w := range s.cfg.ShortLink.Whitelist {
			if host == w || strings.HasSuffix(host, "."+w) {
				ok = true
				break
			}
		}
		if !ok {
			return errcode.URLNotWhitelisted
		}
	}
	if ip := net.ParseIP(host); ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()) {
		return errcode.InternalURLDenied
	}
	return nil
}

// Create 创建短链接：校验 → 生成短码（或自定义码）→ 写库 → 缓存预热 + 布隆过滤器
func (s *LinkService) Create(ctx context.Context, req dto.LinkCreateReq) (*dto.LinkResp, error) {
	if err := s.validateURL(req.OriginURL); err != nil {
		return nil, err
	}
	if _, err := s.group.GetByGID(req.GID); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.GroupNotFound
	} else if err != nil {
		return nil, err
	}
	if req.ValidDateType == 1 && (req.ValidDate == nil || req.ValidDate.Before(time.Now())) {
		return nil, errcode.New("A0001", "自定义有效期必须晚于当前时间")
	}

	shortURI := req.CustomURI
	if shortURI != "" {
		if !customURIRe.MatchString(shortURI) {
			return nil, errcode.ShortURIInvalid
		}
		if s.isReserved(shortURI) {
			return nil, errcode.ShortURIInvalid
		}
		if exists, err := s.shortURIExists(ctx, shortURI); err != nil {
			return nil, err
		} else if exists {
			return nil, errcode.ShortURIExists
		}
	} else {
		// 雪花短码理论上无冲突，布隆/唯一约束兜底重试
		for i := 0; i < 3; i++ {
			candidate := s.gen.NewCode()
			if s.isReserved(candidate) {
				continue
			}
			exists, err := s.shortURIExists(ctx, candidate)
			if err != nil {
				return nil, err
			}
			if !exists {
				shortURI = candidate
				break
			}
		}
		if shortURI == "" {
			return nil, errcode.ServerError
		}
	}

	link := &model.Link{
		ShortURI:      shortURI,
		OriginURL:     req.OriginURL,
		GID:           req.GID,
		Describe:      req.Describe,
		ValidDateType: req.ValidDateType,
		ValidDate:     req.ValidDate,
		Title:         req.Title,
	}
	if err := s.link.Create(link); err != nil {
		return nil, err
	}
	// 缓存预热 + 布隆过滤器登记
	_ = s.bloom.Add(ctx, shortURI)
	s.cache.Warm(ctx, link)
	return s.toResp(link), nil
}

func (s *LinkService) shortURIExists(ctx context.Context, uri string) (bool, error) {
	if _, err := s.link.GetByShortURI(uri); err == nil {
		return true, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	// 回收站中的同码不算冲突（唯一键含 deleted_at）
	return false, nil
}

func (s *LinkService) Page(ctx context.Context, q repo.LinkQuery) ([]dto.LinkResp, int64, error) {
	links, total, err := s.link.Page(q)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.LinkResp, 0, len(links))
	for i := range links {
		out = append(out, *s.toResp(&links[i]))
	}
	return out, total, nil
}

func (s *LinkService) GetByID(ctx context.Context, id int64) (*dto.LinkResp, error) {
	l, err := s.link.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.LinkNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.toResp(l), nil
}

// GetByShortURI 供统计等模块使用
func (s *LinkService) GetByShortURI(ctx context.Context, uri string) (*model.Link, error) {
	return s.link.GetByShortURI(uri)
}

func (s *LinkService) Update(ctx context.Context, id int64, req dto.LinkUpdateReq) (*dto.LinkResp, error) {
	if err := s.validateURL(req.OriginURL); err != nil {
		return nil, err
	}
	if _, err := s.group.GetByGID(req.GID); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.GroupNotFound
	} else if err != nil {
		return nil, err
	}
	l, err := s.link.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.LinkNotFound
	}
	if err != nil {
		return nil, err
	}
	l.OriginURL = req.OriginURL
	l.GID = req.GID
	l.Describe = req.Describe
	l.ValidDateType = req.ValidDateType
	l.ValidDate = req.ValidDate
	if req.EnableStatus != nil {
		l.EnableStatus = *req.EnableStatus
	}
	if req.Title != "" {
		l.Title = req.Title
	}
	if err := s.link.Update(l); err != nil {
		return nil, err
	}
	// 结构性变更：删除缓存（下次访问回源新值）
	s.cache.Evict(ctx, l.ShortURI)
	return s.toResp(l), nil
}

func (s *LinkService) SoftDelete(ctx context.Context, id int64) error {
	l, err := s.link.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errcode.LinkNotFound
	}
	if err != nil {
		return err
	}
	if err := s.link.SoftDelete(id); err != nil {
		return err
	}
	s.cache.Evict(ctx, l.ShortURI)
	return nil
}

func (s *LinkService) Restore(ctx context.Context, id int64) (*dto.LinkResp, error) {
	l, err := s.link.Restore(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.LinkNotFound
	}
	if err != nil {
		// 唯一约束冲突：短码已被新链接占用
		return nil, errcode.ShortURIExists
	}
	_ = s.bloom.Add(ctx, l.ShortURI)
	s.cache.Warm(ctx, l)
	return s.toResp(l), nil
}

func (s *LinkService) PageRecycle(ctx context.Context, page, size int64, keyword string) ([]dto.LinkResp, int64, error) {
	links, total, err := s.link.PageRecycle(page, size, keyword)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.LinkResp, 0, len(links))
	for i := range links {
		out = append(out, *s.toResp(&links[i]))
	}
	return out, total, nil
}

// FetchTitle 抓取原始链接的 <title>（3s 超时，禁内网防 SSRF）
func (s *LinkService) FetchTitle(ctx context.Context, rawURL string) (title string, err error) {
	if err := s.validateURL(rawURL); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", errcode.OriginURLInvalid
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; slink-bot/1.0)")
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("抓取失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("目标返回 %d", resp.StatusCode)
	}
	// 只读前 64KB，足够拿到 <head> 内的标题
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", err
	}
	return extractTitle(body), nil
}

func extractTitle(body []byte) string {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return ""
	}
	var find func(*html.Node) string
	find = func(n *html.Node) string {
		if n.Type == html.ElementNode && n.Data == "title" {
			var sb strings.Builder
			var text func(*html.Node)
			text = func(m *html.Node) {
				if m.Type == html.TextNode {
					sb.WriteString(m.Data)
				}
				for c := m.FirstChild; c != nil; c = c.NextSibling {
					text(c)
				}
			}
			text(n)
			return strings.TrimSpace(sb.String())
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if t := find(c); t != "" {
				return t
			}
		}
		return ""
	}
	title := find(doc)
	if len(title) > 256 {
		title = title[:256]
	}
	return title
}

func (s *LinkService) toResp(l *model.Link) *dto.LinkResp {
	return &dto.LinkResp{
		ID: l.ID, ShortURI: l.ShortURI, OriginURL: l.OriginURL, GID: l.GID,
		Describe: l.Describe, ValidDateType: l.ValidDateType, ValidDate: l.ValidDate,
		EnableStatus: l.EnableStatus, TotalPV: l.TotalPV, TotalUV: l.TotalUV, TotalUIP: l.TotalUIP,
		Title: l.Title, CreatedAt: l.CreatedAt, UpdatedAt: l.UpdatedAt,
	}
}
