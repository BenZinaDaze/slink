import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowLeft,
  ChevronLeft,
  ChevronRight,
  MousePointerClick,
  Users,
  Globe,
} from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import { Area, AreaChart, Bar, BarChart, CartesianGrid, XAxis } from "recharts";
import {
  dimStats,
  getLink,
  hourlyStats,
  linkStats,
  recordStats,
  weekdayStats,
  today,
  daysAgo,
} from "@/api";

const RANGE_OPTIONS = [
  { label: "今天", value: 0 },
  { label: "近 7 天", value: 6 },
  { label: "近 30 天", value: 29 },
];

const DIMS = [
  { key: "locale", label: "地域" },
  { key: "browser", label: "浏览器" },
  { key: "os", label: "操作系统" },
  { key: "device", label: "设备" },
] as const;

const trendConfig = {
  pv: { label: "PV", color: "var(--chart-1)" },
  uv: { label: "UV", color: "var(--chart-2)" },
} satisfies ChartConfig;

export default function LinkStatsPage() {
  const { id } = useParams<{ id: string }>();
  const linkId = Number(id);
  const [range, setRange] = useState(6);
  const [dim, setDim] = useState<(typeof DIMS)[number]["key"]>("locale");
  const [page, setPage] = useState(1);
  const size = 10;

  const start = daysAgo(range);
  const end = today();

  const linkQ = useQuery({
    queryKey: ["link", linkId],
    queryFn: () => getLink(linkId),
  });
  const statsQ = useQuery({
    queryKey: ["link-stats", linkId, start, end],
    queryFn: () => linkStats(linkId, start, end),
  });
  const hourlyQ = useQuery({
    queryKey: ["hourly", linkId, start, end],
    queryFn: () => hourlyStats(linkId, start, end),
  });
  const weekdayQ = useQuery({
    queryKey: ["weekday", linkId, start, end],
    queryFn: () => weekdayStats(linkId, start, end),
  });
  const dimQ = useQuery({
    queryKey: ["dim", linkId, dim, start, end],
    queryFn: () => dimStats(linkId, dim, start, end),
  });
  const recordsQ = useQuery({
    queryKey: ["records", linkId, page],
    queryFn: () => recordStats(linkId, page, size),
  });

  const link = linkQ.data;
  const summary = statsQ.data?.summary;
  const daily = statsQ.data?.daily ?? [];
  const records = recordsQ.data?.records ?? [];
  const recordTotal = recordsQ.data?.total ?? 0;
  const recordPages = Math.max(1, Math.ceil(recordTotal / size));

  const kpis = [
    { label: "访问量 PV", value: summary?.pv ?? 0, icon: MousePointerClick },
    { label: "独立访客 UV", value: summary?.uv ?? 0, icon: Users },
    { label: "独立 IP", value: summary?.uip ?? 0, icon: Globe },
  ];

  return (
    <div className="space-y-4">
      {/* 头部 */}
      <div className="flex flex-wrap items-center gap-3">
        <Button variant="ghost" size="icon" asChild>
          <Link to="/links">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div className="min-w-0">
          <h1 className="truncate text-lg font-semibold">
            {link?.title || link?.originUrl || "…"}
          </h1>
          {link && (
            <p className="truncate font-mono text-sm text-muted-foreground">
              /{link.shortUri} → {link.originUrl}
            </p>
          )}
        </div>
        <div className="ml-auto">
          <Select
            value={String(range)}
            onValueChange={(v) => {
              setRange(Number(v));
              setPage(1);
            }}
          >
            <SelectTrigger className="w-28">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {RANGE_OPTIONS.map((o) => (
                <SelectItem key={o.value} value={String(o.value)}>
                  {o.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* KPI */}
      <div className="grid gap-4 sm:grid-cols-3">
        {kpis.map((k) => (
          <Card key={k.label}>
            <CardContent className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-muted">
                <k.icon className="h-5 w-5 text-muted-foreground" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">{k.label}</p>
                <p className="text-2xl font-semibold tabular-nums">{k.value}</p>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* 趋势 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">访问趋势</CardTitle>
          <CardDescription>按天统计 PV / UV</CardDescription>
        </CardHeader>
        <CardContent>
          <ChartContainer config={trendConfig} className="h-64 w-full">
            <AreaChart data={daily} margin={{ left: -16 }}>
              <CartesianGrid vertical={false} />
              <XAxis
                dataKey="date"
                tickLine={false}
                axisLine={false}
                tickFormatter={(v: string) => v.slice(5)}
                minTickGap={24}
              />
              <ChartTooltip content={<ChartTooltipContent />} />
              <Area
                dataKey="pv"
                type="monotone"
                fill="var(--color-pv)"
                fillOpacity={0.15}
                stroke="var(--color-pv)"
                strokeWidth={2}
              />
              <Area
                dataKey="uv"
                type="monotone"
                fill="var(--color-uv)"
                fillOpacity={0.15}
                stroke="var(--color-uv)"
                strokeWidth={2}
              />
            </AreaChart>
          </ChartContainer>
        </CardContent>
      </Card>

      {/* 小时 / 星期 / 维度分布 */}
      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">时段分布</CardTitle>
            <CardDescription>按小时 PV</CardDescription>
          </CardHeader>
          <CardContent>
            <ChartContainer
              config={{ pv: { label: "PV", color: "var(--chart-1)" } }}
              className="h-52 w-full"
            >
              <BarChart data={hourlyQ.data ?? []} margin={{ left: -16 }}>
                <CartesianGrid vertical={false} />
                <XAxis
                  dataKey="label"
                  tickLine={false}
                  axisLine={false}
                  interval={3}
                />
                <ChartTooltip content={<ChartTooltipContent />} />
                <Bar dataKey="value" fill="var(--color-pv)" radius={4} />
              </BarChart>
            </ChartContainer>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">星期分布</CardTitle>
            <CardDescription>按星期 PV</CardDescription>
          </CardHeader>
          <CardContent>
            <ChartContainer
              config={{ pv: { label: "PV", color: "var(--chart-3)" } }}
              className="h-52 w-full"
            >
              <BarChart data={weekdayQ.data ?? []} margin={{ left: -16 }}>
                <CartesianGrid vertical={false} />
                <XAxis dataKey="label" tickLine={false} axisLine={false} />
                <ChartTooltip content={<ChartTooltipContent />} />
                <Bar dataKey="value" fill="var(--color-pv)" radius={4} />
              </BarChart>
            </ChartContainer>
          </CardContent>
        </Card>

        <Card className="lg:col-span-2">
          <CardHeader className="flex-row items-center gap-4 space-y-0">
            <div>
              <CardTitle className="text-base">维度分布</CardTitle>
              <CardDescription>选择维度查看访问构成</CardDescription>
            </div>
            <Tabs value={dim} onValueChange={(v) => setDim(v as typeof dim)} className="ml-auto">
              <TabsList>
                {DIMS.map((d) => (
                  <TabsTrigger key={d.key} value={d.key}>
                    {d.label}
                  </TabsTrigger>
                ))}
              </TabsList>
            </Tabs>
          </CardHeader>
          <CardContent>
            {(dimQ.data ?? []).length === 0 ? (
              <p className="py-8 text-center text-sm text-muted-foreground">
                暂无数据
              </p>
            ) : (
              <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-5">
                {(dimQ.data ?? []).map((p, i) => {
                  const max = Math.max(...(dimQ.data ?? []).map((x) => x.value), 1);
                  return (
                    <div
                      key={p.label}
                      className="rounded-lg border p-3"
                      style={{
                        background: `linear-gradient(to top, var(--accent) ${Math.round((p.value / max) * 100) * 0.25}%, transparent 0%)`,
                      }}
                    >
                      <p className="truncate text-sm font-medium" title={p.label}>
                        {p.label}
                      </p>
                      <p className="text-lg font-semibold tabular-nums">{p.value}</p>
                      <p className="text-xs text-muted-foreground">#{i + 1}</p>
                    </div>
                  );
                })}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* 访问明细 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">访问明细</CardTitle>
          <CardDescription>最近 {recordTotal} 条访问记录</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>时间</TableHead>
                <TableHead>IP</TableHead>
                <TableHead className="hidden md:table-cell">地域</TableHead>
                <TableHead className="hidden sm:table-cell">浏览器</TableHead>
                <TableHead className="hidden sm:table-cell">操作系统</TableHead>
                <TableHead className="hidden lg:table-cell">设备</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {records.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} className="h-20 text-center text-muted-foreground">
                    暂无访问记录
                  </TableCell>
                </TableRow>
              )}
              {records.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className="whitespace-nowrap text-sm">
                    {new Date(r.createdAt).toLocaleString("zh-CN", { hour12: false })}
                  </TableCell>
                  <TableCell className="font-mono text-sm">{r.ip}</TableCell>
                  <TableCell className="hidden text-sm md:table-cell">{r.locale}</TableCell>
                  <TableCell className="hidden text-sm sm:table-cell">{r.browser}</TableCell>
                  <TableCell className="hidden text-sm sm:table-cell">{r.os}</TableCell>
                  <TableCell className="hidden lg:table-cell">
                    <Badge variant="outline">{r.device}</Badge>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <div className="mt-4 flex items-center justify-between text-sm text-muted-foreground">
            <span>
              共 {recordTotal} 条 · 第 {page}/{recordPages} 页
            </span>
            <div className="flex gap-1">
              <Button variant="outline" size="icon" className="h-8 w-8" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <Button variant="outline" size="icon" className="h-8 w-8" disabled={page >= recordPages} onClick={() => setPage((p) => p + 1)}>
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
