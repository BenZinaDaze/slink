import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2, Wand2 } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import {
  createLink,
  fetchTitle,
  updateLink,
  type Group,
  type LinkItem,
} from "@/api";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  groups: Group[];
  editing?: LinkItem | null;
}

const toLocalInput = (iso?: string) => {
  if (!iso) return "";
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
};

export function LinkFormDialog({ open, onOpenChange, groups, editing }: Props) {
  const queryClient = useQueryClient();
  const [originUrl, setOriginUrl] = useState("");
  const [gid, setGid] = useState("");
  const [customUri, setCustomUri] = useState("");
  const [describe, setDescribe] = useState("");
  const [title, setTitle] = useState("");
  const [validDateType, setValidDateType] = useState(0);
  const [validDate, setValidDate] = useState("");
  const [enabled, setEnabled] = useState(true);
  const [fetchingTitle, setFetchingTitle] = useState(false);

  useEffect(() => {
    if (open) {
      setOriginUrl(editing?.originUrl ?? "");
      setGid(editing?.gid ?? groups[0]?.gid ?? "");
      setCustomUri("");
      setDescribe(editing?.describe ?? "");
      setTitle(editing?.title ?? "");
      setValidDateType(editing?.validDateType ?? 0);
      setValidDate(toLocalInput(editing?.validDate));
      setEnabled((editing?.enableStatus ?? 0) === 0);
    }
  }, [open, editing, groups]);

  const mutation = useMutation({
    mutationFn: async () => {
      const payload = {
        originUrl,
        gid,
        describe,
        title,
        validDateType,
        validDate:
          validDateType === 1 && validDate
            ? new Date(validDate).toISOString()
            : undefined,
        customUri: customUri || undefined,
        enableStatus: enabled ? 0 : 1,
      };
      if (editing) return updateLink(editing.id, payload);
      return createLink(payload);
    },
    onSuccess: () => {
      toast.success(editing ? "短链接已更新" : "创建成功");
      queryClient.invalidateQueries({ queryKey: ["links"] });
      queryClient.invalidateQueries({ queryKey: ["groups"] });
      onOpenChange(false);
    },
    onError: (err) => toast.error(err.message),
  });

  const handleFetchTitle = async () => {
    if (!originUrl) return;
    setFetchingTitle(true);
    try {
      const { title: t } = await fetchTitle(originUrl);
      if (t) {
        setTitle(t);
        toast.success("已抓取标题");
      } else {
        toast.info("目标页面未提供标题");
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "抓取失败");
    } finally {
      setFetchingTitle(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{editing ? "编辑短链接" : "新建短链接"}</DialogTitle>
          <DialogDescription>
            填写原始链接与配置；自定义后缀为 4-8 位字母、数字、- 或 _
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-2">
          <div className="space-y-2">
            <Label htmlFor="originUrl">原始链接 *</Label>
            <div className="flex gap-2">
              <Input
                id="originUrl"
                placeholder="https://example.com/very/long/url"
                value={originUrl}
                onChange={(e) => setOriginUrl(e.target.value)}
              />
              <Button
                type="button"
                variant="outline"
                size="icon"
                title="抓取标题"
                onClick={handleFetchTitle}
                disabled={fetchingTitle || !originUrl}
              >
                {fetchingTitle ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Wand2 className="h-4 w-4" />
                )}
              </Button>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="title">标题</Label>
            <Input
              id="title"
              placeholder="原始页面标题（可自动抓取）"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>分组 *</Label>
              <Select value={gid} onValueChange={setGid}>
                <SelectTrigger>
                  <SelectValue placeholder="选择分组" />
                </SelectTrigger>
                <SelectContent>
                  {groups.map((g) => (
                    <SelectItem key={g.gid} value={g.gid}>
                      {g.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            {!editing && (
              <div className="space-y-2">
                <Label htmlFor="customUri">自定义后缀</Label>
                <Input
                  id="customUri"
                  placeholder="留空自动生成"
                  value={customUri}
                  onChange={(e) => setCustomUri(e.target.value)}
                />
              </div>
            )}
            {editing && (
              <div className="space-y-2">
                <Label>启用状态</Label>
                <div className="flex h-9 items-center gap-2">
                  <Switch
                    checked={enabled}
                    onCheckedChange={setEnabled}
                    id="enabled"
                  />
                  <span className="text-sm text-muted-foreground">
                    {enabled ? "启用" : "停用"}
                  </span>
                </div>
              </div>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>有效期</Label>
              <Select
                value={String(validDateType)}
                onValueChange={(v) => setValidDateType(Number(v))}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="0">永久有效</SelectItem>
                  <SelectItem value="1">自定义</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {validDateType === 1 && (
              <div className="space-y-2">
                <Label htmlFor="validDate">失效时间</Label>
                <Input
                  id="validDate"
                  type="datetime-local"
                  value={validDate}
                  onChange={(e) => setValidDate(e.target.value)}
                />
              </div>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="describe">描述</Label>
            <Textarea
              id="describe"
              placeholder="备注信息（可选）"
              value={describe}
              onChange={(e) => setDescribe(e.target.value)}
              rows={2}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button
            onClick={() => mutation.mutate()}
            disabled={mutation.isPending || !originUrl || !gid}
          >
            {mutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
            {editing ? "保存" : "创建"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
