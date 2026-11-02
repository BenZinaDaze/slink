import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ChevronLeft, ChevronRight, RotateCcw, Search } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { pageRecycle, restoreLink } from "@/api";

export default function RecycleBinPage() {
  const queryClient = useQueryClient();
  const [keyword, setKeyword] = useState("");
  const [page, setPage] = useState(1);
  const size = 10;

  const { data, isLoading } = useQuery({
    queryKey: ["recycle", keyword, page],
    queryFn: () => pageRecycle(page, size, keyword),
  });

  const restore = useMutation({
    mutationFn: (id: number) => restoreLink(id),
    onSuccess: () => {
      toast.success("已恢复");
      queryClient.invalidateQueries({ queryKey: ["recycle"] });
      queryClient.invalidateQueries({ queryKey: ["links"] });
      queryClient.invalidateQueries({ queryKey: ["groups"] });
    },
    onError: (e) => toast.error(e.message),
  });

  const records = data?.records ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / size));

  return (
    <Card className="p-4">
      <div className="mb-4 flex items-center gap-2">
        <h1 className="text-lg font-semibold">回收站</h1>
        <div className="relative ml-auto">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="搜索短码 / 链接"
            className="w-64 pl-8"
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value);
              setPage(1);
            }}
          />
        </div>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>短链接</TableHead>
            <TableHead className="hidden md:table-cell">原始链接</TableHead>
            <TableHead>历史点击</TableHead>
            <TableHead className="hidden sm:table-cell">创建时间</TableHead>
            <TableHead className="w-24" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading && (
            <TableRow>
              <TableCell colSpan={5} className="h-24 text-center text-muted-foreground">
                加载中...
              </TableCell>
            </TableRow>
          )}
          {!isLoading && records.length === 0 && (
            <TableRow>
              <TableCell colSpan={5} className="h-24 text-center text-muted-foreground">
                回收站是空的
              </TableCell>
            </TableRow>
          )}
          {records.map((l) => (
            <TableRow key={l.id}>
              <TableCell className="font-mono text-sm">/{l.shortUri}</TableCell>
              <TableCell className="hidden max-w-72 md:table-cell">
                <span className="block truncate text-sm text-muted-foreground" title={l.originUrl}>
                  {l.originUrl}
                </span>
              </TableCell>
              <TableCell className="font-mono text-sm">{l.totalPv}</TableCell>
              <TableCell className="hidden text-sm sm:table-cell">
                {new Date(l.createdAt).toLocaleDateString("zh-CN")}
              </TableCell>
              <TableCell>
                <Button
                  variant="outline"
                  size="sm"
                  className="gap-1.5"
                  onClick={() => restore.mutate(l.id)}
                  disabled={restore.isPending}
                >
                  <RotateCcw className="h-3.5 w-3.5" />
                  恢复
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <div className="mt-4 flex items-center justify-between text-sm text-muted-foreground">
        <span>
          共 {total} 条 · 第 {page}/{totalPages} 页
        </span>
        <div className="flex gap-1">
          <Button variant="outline" size="icon" className="h-8 w-8" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <Button variant="outline" size="icon" className="h-8 w-8" disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </Card>
  );
}
