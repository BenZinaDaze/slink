import { http } from "./client";

// ---------- 类型 ----------

export interface Group {
  gid: string;
  name: string;
  sortOrder: number;
  linkCount: number;
  createdAt: string;
}

export interface LinkItem {
  id: number;
  shortUri: string;
  originUrl: string;
  gid: string;
  describe: string;
  validDateType: number; // 0永久 1自定义
  validDate?: string;
  enableStatus: number; // 0启用 1禁用
  totalPv: number;
  totalUv: number;
  totalUip: number;
  title: string;
  createdAt: string;
  updatedAt: string;
}

export interface PageData<T> {
  current: number;
  size: number;
  total: number;
  records: T[];
}

export interface DailyPoint {
  date: string;
  pv: number;
  uv: number;
  uip: number;
}

export interface StatsSummary {
  pv: number;
  uv: number;
  uip: number;
}

export interface Point2 {
  label: string;
  value: number;
}

export interface DashboardData {
  linkTotal: number;
  today: StatsSummary;
  totalClick: number;
  daily: DailyPoint[];
  topLinks: LinkItem[];
}

export interface AccessRecord {
  id: number;
  ip: string;
  browser: string;
  os: string;
  device: string;
  locale: string;
  createdAt: string;
}

// ---------- auth ----------

export const login = (username: string, password: string) =>
  http.post<{ username: string; shortDomain: string }>("/api/v1/auth/login", {
    username,
    password,
  });

export const logout = () => http.delete<void>("/api/v1/auth/logout");

export interface MeInfo {
  username: string;
  shortDomain: string;
  isDefaultPassword: boolean;
  mustChangePassword: boolean;
}

export const getMe = () => http.get<MeInfo>("/api/v1/auth/me");

/** 修改用户名和/或密码；成功后其他会话全部下线，新凭证由 HttpOnly Cookie 续期 */
export const updateAccount = (payload: {
  username?: string;
  oldPassword: string;
  newPassword?: string;
}) =>
  http.put<{ username: string; shortDomain: string }>(
    "/api/v1/auth/account",
    payload,
  );

// ---------- groups ----------

export const listGroups = () => http.get<Group[]>("/api/v1/groups");

export const createGroup = (name: string) =>
  http.post<Group>("/api/v1/groups", { name });

export const updateGroup = (gid: string, name: string) =>
  http.put<void>(`/api/v1/groups/${gid}`, { name });

export const deleteGroup = (gid: string) =>
  http.delete<void>(`/api/v1/groups/${gid}`);

export const sortGroups = (items: { gid: string; sortOrder: number }[]) =>
  http.put<void>("/api/v1/groups/sort", { items });

// ---------- links ----------

export interface LinkPageQuery {
  page?: number;
  size?: number;
  gid?: string;
  key?: string;
  order?: string;
}

export const pageLinks = (q: LinkPageQuery = {}) => {
  const params = new URLSearchParams();
  if (q.page) params.set("page", String(q.page));
  if (q.size) params.set("size", String(q.size));
  if (q.gid) params.set("gid", q.gid);
  if (q.key) params.set("key", q.key);
  if (q.order) params.set("order", q.order);
  return http.get<PageData<LinkItem>>(`/api/v1/links?${params}`);
};

export interface LinkPayload {
  originUrl: string;
  gid: string;
  describe?: string;
  validDateType: number;
  validDate?: string;
  customUri?: string;
  title?: string;
  enableStatus?: number;
}

export const createLink = (payload: LinkPayload) =>
  http.post<LinkItem>("/api/v1/links", payload);

export const updateLink = (id: number, payload: LinkPayload) =>
  http.put<LinkItem>(`/api/v1/links/${id}`, payload);

export const getLink = (id: number) => http.get<LinkItem>(`/api/v1/links/${id}`);

export const deleteLink = (id: number) =>
  http.delete<void>(`/api/v1/links/${id}`);

export const fetchTitle = (url: string) =>
  http.get<{ title: string }>(
    `/api/v1/links/title?url=${encodeURIComponent(url)}`,
  );

// ---------- recycle bin ----------

export const pageRecycle = (page = 1, size = 10, key = "") => {
  const params = new URLSearchParams({
    page: String(page),
    size: String(size),
  });
  if (key) params.set("key", key);
  return http.get<PageData<LinkItem>>(`/api/v1/recycle-bin?${params}`);
};

export const restoreLink = (id: number) =>
  http.post<LinkItem>(`/api/v1/recycle-bin/${id}/restore`);

// ---------- stats ----------

const dateStr = (d: Date) => d.toISOString().slice(0, 10);
export const daysAgo = (n: number) => {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return dateStr(d);
};
export const today = () => daysAgo(0);

export const linkStats = (id: number, start: string, end: string) =>
  http.get<{ summary: StatsSummary; daily: DailyPoint[] }>(
    `/api/v1/links/${id}/stats?start=${start}&end=${end}`,
  );

export const hourlyStats = (id: number, start: string, end: string) =>
  http.get<Point2[]>(`/api/v1/links/${id}/stats/hourly?start=${start}&end=${end}`);

export const weekdayStats = (id: number, start: string, end: string) =>
  http.get<Point2[]>(`/api/v1/links/${id}/stats/weekday?start=${start}&end=${end}`);

export const dimStats = (
  id: number,
  dim: "locale" | "browser" | "os" | "device",
  start: string,
  end: string,
) =>
  http.get<Point2[]>(
    `/api/v1/links/${id}/stats/dim?dim=${dim}&start=${start}&end=${end}`,
  );

export const recordStats = (id: number, page = 1, size = 10) =>
  http.get<PageData<AccessRecord>>(
    `/api/v1/links/${id}/stats/records?page=${page}&size=${size}`,
  );

// ---------- dashboard ----------

export const getDashboard = () =>
  http.get<DashboardData>("/api/v1/dashboard");
