import { useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ChevronLeft,
  ChevronRight,
  Copy,
  MoreHorizontal,
  Pencil,
  Plus,
  QrCode,
  Search,
  Trash2,
  BarChart3,
  FolderPlus,
  ExternalLink,
} from "lucide-react";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  deleteGroup,
  deleteLink,
  createGroup,
  listGroups,
  pageLinks,
} from "@/api";
import { LinkFormDialog } from "@/components/link-form-dialog";
import { QrCodeDialog, copyText, fullShortUrl } from "@/components/qrcode-dialog";

export default function LinksPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();

  const [gid, setGid] = useState<string>("");
  const [keyword, setKeyword] = useState("");
  const [page, setPage] = useState(1);
  const size = 10;

  const [formOpen, setFormOpen] = useState(searchParams.get("create") === "1");
  const [editingId, setEditingId] = useState<number | null>(null);
  const [qrUri, setQrUri] = useState<string | null>(null);
  const [newGroup, setNewGroup] = useState<string | null>(null);

  const groupsQ = useQuery({ queryKey: ["groups"], queryFn: listGroups });
  const linksQ = useQuery({
    queryKey: ["links", gid, keyword, page],
    queryFn: () => pageLinks({ page, size, gid: gid || undefined, key: keyword || undefined }),
  });

  const delLink = useMutation({
    mutationFn: (id: number) => deleteLink(id),
    onSuccess: () => {
      toast.success("已移入回收站");
      queryClient.invalidateQueries({ queryKey: ["links"] });
      queryClient.invalidateQueries({ queryKey: ["groups"] });
    },
    onError: (e) => toast.error(e.message),
  });

  const addGroup = useMutation({
    mutationFn: (name: string) => createGroup(name),
    onSuccess: () => {
      toast.success("分组已创建");
      setNewGroup(null);
      queryClient.invalidateQueries({ queryKey: ["groups"] });
    },
    onError: (e) => toast.error(e.message),
  });

  const removeGroup = useMutation({
    mutationFn: (g: string) => deleteGroup(g),
    onSuccess: () => {
      toast.success("分组已删除");
      if (gid) setGid("");
      queryClient.invalidateQueries({ queryKey: ["groups"] });
    },
    onError: (e) => toast.error(e.message),
  });

  const groups = groupsQ.data ?? [];
  const links = linksQ.data?.records ?? [];
  const total = linksQ.data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / size));

  return (
    <div className="flex gap-4">
      {/* 分组栏 */}
      <Card className="hidden w-52 shrink-0 self-start p-3 lg:block">
        <div className="flex items-center justify-between px-1 pb-2">
          <span className="text-sm font-medium">分组</span>
          <Button
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            title="新建分组"
            onClick={() => setNewGroup("")}
          >
            <FolderPlus className="h-4 w-4" />
          </Button>
        </div>
        {newGroup !== null && (
          <div className="px-1 pb-2">
            <Input
              autoFocus
              placeholder="分组名称，回车保存"
              value={newGroup}
              onChange={(e) => setNewGroup(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && newGroup.trim()) {
                  addGroup.mutate(newGroup.trim());
                }
                if (e.key === "Escape") setNewGroup(null);
              }}
              className="h-8 text-sm"
            />
          </div>
        )}
        <div className="space-y-0.5">
          {/* 统一行结构：名称 flex-1 + 固定宽徽标 + 固定宽操作槽，
              保证所有行（含无删除按钮的"全部/默认分组"）的徽标严格同列 */}
          <div
            className={`group flex w-full items-center gap-1 rounded-md px-2 py-1.5 text-sm hover:bg-accent ${!gid ? "bg-accent font-medium" : ""}`}
          >
            <button
              className="min-w-0 flex-1 truncate text-left"
              onClick={() => {
                setGid("");
                setPage(1);
              }}
            >
              全部
            </button>
            <Badge variant="secondary" className="min-w-6 justify-center font-mono text-xs tabular-nums">
              {groups.reduce((s, g) => s + g.linkCount, 0)}
            </Badge>
            <span className="w-4 shrink-0" />
          </div>
          {groups.map((g) => (
            <div
              key={g.gid}
              className={`group flex w-full items-center gap-1 rounded-md px-2 py-1.5 text-sm hover:bg-accent ${gid === g.gid ? "bg-accent font-medium" : ""}`}
            >
              <button
                className="min-w-0 flex-1 truncate text-left"
                onClick={() => {
                  setGid(g.gid);
                  setPage(1);
                }}
              >
                {g.name}
              </button>
              <Badge variant="secondary" className="min-w-6 justify-center font-mono text-xs tabular-nums">
                {g.linkCount}
              </Badge>
              <span className="flex w-4 shrink-0 justify-center">
                {g.gid !== "default" && (
                  <button
                    className="text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100 hover:text-destructive"
                    title="删除分组"
                    onClick={() => {
                      if (confirm(`删除分组「${g.name}」？`)) removeGroup.mutate(g.gid);
                    }}
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                )}
              </span>
            </div>
          ))}
        </div>
      </Card>

      {/* 列表 */}
      <Card className="min-w-0 flex-1 p-4">
        <div className="mb-4 flex flex-wrap items-center gap-2">
          <div className="relative">
            <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="搜索短码 / 链接 / 标题"
              className="w-64 pl-8"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value);
                setPage(1);
              }}
            />
          </div>
          <div className="ml-auto flex items-center gap-2">
            <Button
              onClick={() => {
                setEditingId(null);
                setFormOpen(true);
              }}
              className="gap-2"
            >
              <Plus className="h-4 w-4" />
              新建短链接
            </Button>
          </div>
        </div>

        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>短链接</TableHead>
              <TableHead className="hidden md:table-cell">原始链接</TableHead>
              <TableHead className="hidden sm:table-cell">分组</TableHead>
              <TableHead className="text-right">PV / UV</TableHead>
              <TableHead className="hidden lg:table-cell">有效期</TableHead>
              <TableHead className="hidden sm:table-cell">状态</TableHead>
              <TableHead className="w-12" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {linksQ.isLoading && (
              <TableRow>
                <TableCell colSpan={7} className="h-24 text-center text-muted-foreground">
                  加载中...
                </TableCell>
              </TableRow>
            )}
            {linksQ.data && links.length === 0 && (
              <TableRow>
                <TableCell colSpan={7} className="h-24 text-center text-muted-foreground">
                  暂无短链接，点击右上角创建
                </TableCell>
              </TableRow>
            )}
            {links.map((l) => {
              const group = groups.find((g) => g.gid === l.gid);
              const expired =
                l.validDateType === 1 &&
                l.validDate &&
                new Date(l.validDate).getTime() < Date.now();
              return (
                <TableRow key={l.id}>
                  <TableCell>
                    <div className="flex items-center gap-1.5">
                      <button
                        className="font-mono text-sm font-medium text-primary hover:underline"
                        title="复制短链接"
                        onClick={() => copyText(fullShortUrl(l.shortUri))}
                      >
                        /{l.shortUri}
                      </button>
                      <button
                        className="text-muted-foreground hover:text-foreground"
                        title="二维码"
                        onClick={() => setQrUri(l.shortUri)}
                      >
                        <QrCode className="h-3.5 w-3.5" />
                      </button>
                      <a
                        href={`/${l.shortUri}`}
                        target="_blank"
                        rel="noreferrer"
                        className="text-muted-foreground hover:text-foreground"
                        title="打开短链接"
                      >
                        <ExternalLink className="h-3.5 w-3.5" />
                      </a>
                    </div>
                    {l.title && (
                      <p className="max-w-56 truncate text-xs text-muted-foreground">
                        {l.title}
                      </p>
                    )}
                  </TableCell>
                  <TableCell className="hidden max-w-64 md:table-cell">
                    <a
                      href={l.originUrl}
                      target="_blank"
                      rel="noreferrer"
                      className="block truncate text-sm text-muted-foreground hover:text-foreground hover:underline"
                      title={l.originUrl}
                    >
                      {l.originUrl}
                    </a>
                  </TableCell>
                  <TableCell className="hidden sm:table-cell">
                    <Badge variant="outline">{group?.name ?? l.gid}</Badge>
                  </TableCell>
                  <TableCell className="text-right font-mono text-sm">
                    {l.totalPv} / {l.totalUv}
                  </TableCell>
                  <TableCell className="hidden text-sm lg:table-cell">
                    {expired ? (
                      <span className="text-destructive">已过期</span>
                    ) : l.validDateType === 0 ? (
                      "永久"
                    ) : (
                      new Date(l.validDate!).toLocaleDateString("zh-CN")
                    )}
                  </TableCell>
                  <TableCell className="hidden sm:table-cell">
                    {l.enableStatus === 0 ? (
                      <Badge>启用</Badge>
                    ) : (
                      <Badge variant="destructive">停用</Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon" className="h-8 w-8">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem onClick={() => copyText(fullShortUrl(l.shortUri))}>
                          <Copy className="h-4 w-4" /> 复制链接
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => setQrUri(l.shortUri)}>
                          <QrCode className="h-4 w-4" /> 二维码
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => navigate(`/links/${l.id}`)}>
                          <BarChart3 className="h-4 w-4" /> 访问监控
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          onClick={() => {
                            setEditingId(l.id);
                            setFormOpen(true);
                          }}
                        >
                          <Pencil className="h-4 w-4" /> 编辑
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          className="text-destructive focus:text-destructive"
                          onClick={() => delLink.mutate(l.id)}
                        >
                          <Trash2 className="h-4 w-4" /> 移入回收站
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>

        {/* 分页 */}
        <div className="mt-4 flex items-center justify-between text-sm text-muted-foreground">
          <span>
            共 {total} 条 · 第 {page}/{totalPages} 页
          </span>
          <div className="flex gap-1">
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              disabled={page <= 1}
              onClick={() => setPage((p) => p - 1)}
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              disabled={page >= totalPages}
              onClick={() => setPage((p) => p + 1)}
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </Card>

      <LinkFormDialog
        open={formOpen}
        onOpenChange={(v) => {
          setFormOpen(v);
          if (!v && searchParams.get("create")) {
            searchParams.delete("create");
            setSearchParams(searchParams, { replace: true });
          }
        }}
        groups={groups}
        editing={links.find((l) => l.id === editingId) ?? null}
      />
      <QrCodeDialog shortUri={qrUri} onOpenChange={() => setQrUri(null)} />
    </div>
  );
}
