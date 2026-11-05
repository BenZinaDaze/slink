const USERNAME_KEY = "slink_username";
const SHORT_DOMAIN_KEY = "slink_short_domain";
const LEGACY_TOKEN_KEY = "slink_token";

/**
 * 管理后台的路由前缀（隐藏入口模式由后端注入 <meta name="slink-admin-base">）。
 * 根路径模式下为空串，路由挂 /
 */
export function adminBase(): string {
	return document.querySelector('meta[name="slink-admin-base"]')?.getAttribute("content") ?? "";
}

// 旧版本把登录 token 存 localStorage（XSS 可窃取），已迁移为 HttpOnly Cookie；
// 启动时清理遗留值，存量会话需重新登录一次
localStorage.removeItem(LEGACY_TOKEN_KEY);

export function getUsername(): string {
  return localStorage.getItem(USERNAME_KEY) ?? "";
}

/** 配置的短链域名（含 scheme）；空 = 单域模式，短链与当前访问域名一致 */
export function getShortDomain(): string {
  return localStorage.getItem(SHORT_DOMAIN_KEY) ?? "";
}

export function setAuth(username: string, shortDomain: string) {
  localStorage.setItem(USERNAME_KEY, username);
  if (shortDomain) localStorage.setItem(SHORT_DOMAIN_KEY, shortDomain);
  else localStorage.removeItem(SHORT_DOMAIN_KEY);
  localStorage.removeItem(LEGACY_TOKEN_KEY);
}

export function clearAuth() {
  localStorage.removeItem(USERNAME_KEY);
  localStorage.removeItem(SHORT_DOMAIN_KEY);
}

/**
 * 仅作 UI 路由的本地判定：有用户名标志即认为已登录。
 * 真实凭证是 HttpOnly Cookie（JS 不可读），失效后由 API 层收到 401 统一清标志跳登录页。
 */
export function isLoggedIn(): boolean {
  return !!getUsername();
}

/** 账号信息变更事件：布局层监听以刷新侧边栏用户名 */
export const AUTH_CHANGED_EVENT = "slink:auth-changed";

export function emitAuthChanged() {
  window.dispatchEvent(new Event(AUTH_CHANGED_EVENT));
}
