package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"github.com/benzi/slink/backend/internal/config"
	"github.com/benzi/slink/backend/internal/middleware"
	"github.com/benzi/slink/backend/internal/repo"
	"github.com/benzi/slink/backend/pkg/errcode"
)

const (
	settingKeyPassword = "password_hash"
	settingKeyUsername = "username"
	keyTokenVersion    = "auth:ver"
	keyLoginFails      = "rl:login:fail:" // 连续失败计数，值 = 次数
	keyLoginLock       = "rl:login:lock:" // 指数退避锁，值无关紧要，TTL 即锁时长
)

// loginLockDuration 连续失败 n 次后的锁定时长：满 5 次起锁，30s 起步每次翻倍，封顶 1h
func loginLockDuration(fails int) time.Duration {
	if fails < 5 {
		return 0
	}
	shift := fails - 5
	if shift > 16 { // 防移位溢出，30s<<16 已远超 1h
		shift = 16
	}
	if d := 30 * time.Second << uint(shift); d > 0 && d <= time.Hour {
		return d
	}
	return time.Hour
}

// AuthService 单用户认证。
//
// 令牌机制：不透明随机 token（UUID，122 位熵）存 Redis，值格式 "username#version"。
// version 为凭证版本号——修改用户名/密码时自增，全部旧 token 立即失效（全端下线），
// 当前会话由接口返回新 token 无缝续期。相比 JWT：可随时吊销、无需管理签名密钥、
// 登出即删；代价是每次校验多一次 Redis 读，对单机部署完全可接受。
type AuthService struct {
	cfg       *config.Config
	rdb       *redis.Client
	setting   *repo.SettingRepo
	dummyHash []byte // 用户名不匹配时也执行一次 bcrypt 比较，消除时序侧信道
}

func NewAuthService(cfg *config.Config, rdb *redis.Client, setting *repo.SettingRepo) *AuthService {
	dummy, _ := bcrypt.GenerateFromPassword([]byte(uuid.NewString()), bcrypt.DefaultCost)
	return &AuthService{cfg: cfg, rdb: rdb, setting: setting, dummyHash: dummy}
}

// effectiveUsername 生效用户名：设置页修改后存 SQLite，优先于配置文件
func (s *AuthService) effectiveUsername(ctx context.Context) string {
	if u, ok, err := s.setting.Get(settingKeyUsername); err == nil && ok && strings.TrimSpace(u) != "" {
		return strings.TrimSpace(u)
	}
	return s.cfg.Auth.Username
}

// verifyPassword 校验密码：settings 哈希优先，否则用配置初始密码
func (s *AuthService) verifyPassword(password string) bool {
	hash, ok, err := s.setting.Get(settingKeyPassword)
	if err == nil && ok {
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
	}
	return password == s.cfg.Auth.Password
}

// version 当前凭证版本号
func (s *AuthService) version(ctx context.Context) int {
	v, _ := s.rdb.Get(ctx, keyTokenVersion).Int()
	return v
}

// Login 校验并签发 token。
// 用户名不匹配时同样执行一次 bcrypt 比较，保证两种失败路径耗时一致，
// 防止通过响应时间探测用户名是否存在。
func (s *AuthService) Login(ctx context.Context, username, password string) (string, error) {
	if username != s.effectiveUsername(ctx) {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		return "", errcode.LoginFailed
	}
	if !s.verifyPassword(password) {
		return "", errcode.LoginFailed
	}
	token, err := s.issueToken(ctx, username)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *AuthService) issueToken(ctx context.Context, username string) (string, error) {
	token := uuid.NewString()
	val := fmt.Sprintf("%s#%d", username, s.version(ctx))
	if err := s.rdb.Set(ctx, middleware.TokenKey+token, val, s.cfg.Auth.TokenTTL).Err(); err != nil {
		return "", err
	}
	return token, nil
}

// Logout 删除 token（单设备登出）
func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.rdb.Del(ctx, middleware.TokenKey+token).Err()
}

// Verify 实现 middleware.TokenSaver：校验 token 存在且凭证版本未过期
func (s *AuthService) Verify(ctx context.Context, token string) (string, bool) {
	val, err := s.rdb.Get(ctx, middleware.TokenKey+token).Result()
	if err != nil {
		return "", false
	}
	username, ver, ok := splitTokenValue(val)
	if !ok || ver != s.version(ctx) {
		return "", false
	}
	return username, true
}

func splitTokenValue(val string) (string, int, bool) {
	i := strings.LastIndexByte(val, '#')
	if i <= 0 {
		return "", 0, false
	}
	ver, err := strconv.Atoi(val[i+1:])
	if err != nil {
		return "", 0, false
	}
	return val[:i], ver, true
}

// Me 当前账号信息：isDefault = 是否仍为配置初始密码（供横幅提示），
// mustChange = 是否强制要求修改（未被 auth.allow_default_password 豁免）
func (s *AuthService) Me(ctx context.Context) (username string, isDefault, mustChange bool) {
	username = s.effectiveUsername(ctx)
	isDefault = s.verifyPassword(s.cfg.Auth.Password)
	return username, isDefault, isDefault && !s.cfg.Auth.AllowDefaultPassword
}

// ShortDomain 配置的短链域名原值（含 scheme，可能为空），供前端拼完整短链
func (s *AuthService) ShortDomain() string { return s.cfg.ShortLink.Domain }

// ChangeAccount 修改用户名和/或密码。需验证原密码；成功后自增凭证版本号
// 吊销全部已签发 token（全端下线），并返回新 token 保持当前会话。
func (s *AuthService) ChangeAccount(ctx context.Context, newUsername, oldPassword, newPassword string) (string, string, error) {
	if !s.verifyPassword(oldPassword) {
		return "", "", errcode.OldPasswordWrong
	}
	newUsername = strings.TrimSpace(newUsername)
	if newUsername != "" {
		if l := len([]rune(newUsername)); l < 2 || l > 32 {
			return "", "", errcode.New("A0001", "用户名长度需在 2-32 个字符之间")
		}
	}
	if newPassword != "" && len(newPassword) < 8 {
		return "", "", errcode.New("A0001", "新密码至少 8 位")
	}

	if newUsername != "" && newUsername != s.cfg.Auth.Username {
		if err := s.setting.Set(settingKeyUsername, newUsername); err != nil {
			return "", "", err
		}
	} else if newUsername != "" && newUsername == s.cfg.Auth.Username {
		// 恢复为配置初始用户名：清除覆盖值
		_ = s.setting.Set(settingKeyUsername, "")
	}
	if newPassword != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return "", "", err
		}
		if err := s.setting.Set(settingKeyPassword, string(hash)); err != nil {
			return "", "", err
		}
	}

	// 自增版本号：全部旧 token 失效
	if err := s.rdb.Incr(ctx, keyTokenVersion).Err(); err != nil {
		return "", "", err
	}

	username := newUsername
	if username == "" {
		username = s.effectiveUsername(ctx)
	}
	token, err := s.issueToken(ctx, username)
	if err != nil {
		return "", "", err
	}
	return token, username, nil
}

// LoginAttempt 记录并返回该 IP 一分钟内的登录尝试次数（第一层频控）
func (s *AuthService) LoginAttempt(ctx context.Context, ip string) int {
	key := "rl:login:" + ip
	n, _ := s.rdb.Incr(ctx, key).Result()
	if n == 1 {
		s.rdb.Expire(ctx, key, time.Minute)
	}
	return int(n)
}

// LoginLocked 该 IP 是否处于指数退避锁定中，返回剩余锁定秒数（0 = 未锁）
func (s *AuthService) LoginLocked(ctx context.Context, ip string) int {
	ttl, _ := s.rdb.TTL(ctx, keyLoginLock+ip).Result()
	if ttl <= 0 {
		return 0
	}
	return int(ttl.Seconds()) + 1 // 向上取整，避免边界秒数归零
}

// RecordLoginResult 记录登录结果：成功清空失败计数；失败自增计数并按需写入退避锁
func (s *AuthService) RecordLoginResult(ctx context.Context, ip string, ok bool) {
	if ok {
		s.rdb.Del(ctx, keyLoginFails+ip, keyLoginLock+ip)
		return
	}
	key := keyLoginFails + ip
	n, _ := s.rdb.Incr(ctx, key).Result()
	if n == 1 {
		s.rdb.Expire(ctx, key, 15*time.Minute) // 15 分钟无失败则重新计数
	}
	if d := loginLockDuration(int(n)); d > 0 {
		s.rdb.Set(ctx, keyLoginLock+ip, 1, d)
	}
}

// TokenTTL 登录态有效期（供 Cookie Max-Age 使用）
func (s *AuthService) TokenTTL() time.Duration { return s.cfg.Auth.TokenTTL }

// CookieSecureOverride 配置显式指定 Cookie Secure 时的值；nil = 自动探测
func (s *AuthService) CookieSecureOverride() *bool { return s.cfg.Auth.CookieSecure }
