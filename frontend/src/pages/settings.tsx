import { useEffect, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Loader2, UserRound } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { getMe, updateAccount } from "@/api";
import { emitAuthChanged, setAuth } from "@/lib/auth";

export default function SettingsPage() {
  const meQ = useQuery({ queryKey: ["me"], queryFn: getMe });

  const [username, setUsername] = useState("");
  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  useEffect(() => {
    if (meQ.data?.username) setUsername(meQ.data.username);
  }, [meQ.data?.username]);

  const mutation = useMutation({
    mutationFn: () =>
      updateAccount({
        username: username.trim() || undefined,
        oldPassword,
        newPassword: newPassword || undefined,
      }),
    onSuccess: (data) => {
      // 接口已吊销全部旧会话并返回新 token，本地续期的同时刷新用户名
      setAuth(data.token, data.username);
      emitAuthChanged();
      toast.success("已保存，其他登录会话已全部下线");
      setOldPassword("");
      setNewPassword("");
      setConfirmPassword("");
      meQ.refetch();
    },
    onError: (e) => toast.error(e.message),
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (username.trim().length < 2) {
      toast.warning("用户名至少 2 个字符");
      return;
    }
    if (newPassword && newPassword.length < 6) {
      toast.warning("新密码至少 6 位");
      return;
    }
    if (newPassword !== confirmPassword) {
      toast.warning("两次输入的新密码不一致");
      return;
    }
    mutation.mutate();
  };

  return (
    <div className="mx-auto max-w-xl space-y-4">
      <h1 className="text-lg font-semibold">设置</h1>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <UserRound className="h-4 w-4" />
            账号
          </CardTitle>
          <CardDescription>
            修改用户名或密码需验证原密码；保存后其他设备的登录会话将全部下线
          </CardDescription>
        </CardHeader>
        <CardContent>
          {meQ.data?.isDefaultPassword && (
            <p className="mb-4 rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-sm text-amber-600 dark:text-amber-400">
              当前仍在使用配置文件中的初始密码，建议尽快修改。
            </p>
          )}
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="username">用户名</Label>
              <Input
                id="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="2-32 个字符"
              />
            </div>
            <Separator />
            <div className="space-y-2">
              <Label htmlFor="old">原密码 *</Label>
              <Input
                id="old"
                type="password"
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
                autoComplete="current-password"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="new">新密码</Label>
              <Input
                id="new"
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                placeholder="留空则仅修改用户名"
                autoComplete="new-password"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="confirm">确认新密码</Label>
              <Input
                id="confirm"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder="再次输入新密码"
                autoComplete="new-password"
              />
            </div>
            <Button type="submit" disabled={mutation.isPending || !oldPassword}>
              {mutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
              保存修改
            </Button>
          </form>
        </CardContent>
      </Card>

      <Separator />
      <p className="text-xs text-muted-foreground">slink · 简单高效的短链接系统</p>
    </div>
  );
}
