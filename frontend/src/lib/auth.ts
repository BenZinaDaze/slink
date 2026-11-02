const TOKEN_KEY = "slink_token";
const USERNAME_KEY = "slink_username";

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function setAuth(token: string, username: string) {
  localStorage.setItem(TOKEN_KEY, token);
  localStorage.setItem(USERNAME_KEY, username);
}

export function getUsername(): string {
  return localStorage.getItem(USERNAME_KEY) ?? "";
}

export function clearAuth() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(USERNAME_KEY);
}

export function isLoggedIn(): boolean {
  return !!getToken();
}

/** 账号信息变更事件：布局层监听以刷新侧边栏用户名 */
export const AUTH_CHANGED_EVENT = "slink:auth-changed";

export function emitAuthChanged() {
  window.dispatchEvent(new Event(AUTH_CHANGED_EVENT));
}
