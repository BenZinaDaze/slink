import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Link2, MousePointerClick, Plus, Users, Zap } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import { Area, AreaChart, CartesianGrid, XAxis } from "recharts";
import { getDashboard } from "@/api";

const config = {
  pv: { label: "PV", color: "var(--chart-1)" },
  uv: { label: "UV", color: "var(--chart-2)" },
} satisfies ChartConfig;

export default function DashboardPage() {
  const { data } = useQuery({
    queryKey: ["dashboard"],
    queryFn: getDashboard,
    refetchInterval: 30_000,
  });

  const kpis = [
    { label: "短链接总数", value: data?.linkTotal ?? 0, icon: Link2 },
    { label: "今日访问 PV", value: data?.today?.pv ?? 0, icon: MousePointerClick },
    { label: "今日访客 UV", value: data?.today?.uv ?? 0, icon: Users },
    { label: "累计点击", value: data?.totalClick ?? 0, icon: Zap },
  ];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-lg font-semibold">仪表盘</h1>
          <p className="text-sm text-muted-foreground">短链接系统运行概览</p>
        </div>
        <Button asChild className="gap-2">
          <Link to="/links?create=1">
            <Plus className="h-4 w-4" /> 新建短链接
          </Link>
        </Button>
      </div>

      {/* KPI 卡片行 */}
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
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

      {/* 趋势 + Top */}
      <div className="grid gap-4 lg:grid-cols-5">
        <Card className="lg:col-span-3">
          <CardHeader>
            <CardTitle className="text-base">近 30 天访问趋势</CardTitle>
            <CardDescription>全站 PV / UV</CardDescription>
          </CardHeader>
          <CardContent>
            <ChartContainer config={config} className="h-64 w-full">
              <AreaChart data={data?.daily ?? []} margin={{ left: -16 }}>
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

        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle className="text-base">热门短链接 Top 10</CardTitle>
            <CardDescription>近 30 天点击量排名</CardDescription>
          </CardHeader>
          <CardContent className="space-y-1">
            {(data?.topLinks ?? []).length === 0 && (
              <p className="py-8 text-center text-sm text-muted-foreground">
                暂无数据
              </p>
            )}
            {(data?.topLinks ?? []).map((l, i) => {
              const max = Math.max(...(data?.topLinks ?? []).map((x) => x.totalPv), 1);
              return (
                <Link
                  key={l.id}
                  to={`/links/${l.id}`}
                  className="flex items-center gap-3 rounded-md px-2 py-1.5 hover:bg-accent"
                >
                  <span className="w-5 text-right font-mono text-xs text-muted-foreground">
                    {i + 1}
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="truncate font-mono text-sm font-medium">
                      /{l.shortUri}
                    </p>
                    <div className="mt-1 h-1.5 overflow-hidden rounded-full bg-muted">
                      <div
                        className="h-full rounded-full bg-primary/60"
                        style={{ width: `${Math.max(2, (l.totalPv / max) * 100)}%` }}
                      />
                    </div>
                  </div>
                  <span className="font-mono text-sm tabular-nums">{l.totalPv}</span>
                </Link>
              );
            })}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
