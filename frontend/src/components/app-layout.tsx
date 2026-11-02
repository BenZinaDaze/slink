import { useEffect, useState } from "react";
import { Outlet, useLocation, useNavigate } from "react-router-dom";
import {
  LayoutDashboard,
  Link2,
  Trash2,
  Settings,
  Command,
  LogOut,
} from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { TooltipProvider } from "@/components/ui/tooltip";
import { SidebarProvider, Sidebar, SidebarContent, SidebarGroup, SidebarGroupContent, SidebarMenu, SidebarMenuItem, SidebarMenuButton, SidebarHeader, SidebarFooter, SidebarInset, SidebarTrigger } from "@/components/ui/sidebar";
import { ThemeToggle } from "@/components/theme-toggle";
import { CommandMenu } from "@/components/command-menu";
import { AUTH_CHANGED_EVENT, clearAuth, getUsername } from "@/lib/auth";
import { logout } from "@/api";

const NAV = [
  { to: "/", label: "仪表盘", icon: LayoutDashboard },
  { to: "/links", label: "短链接", icon: Link2 },
  { to: "/recycle-bin", label: "回收站", icon: Trash2 },
  { to: "/settings", label: "设置", icon: Settings },
];

export function AppLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const [username, setUsername] = useState(getUsername());

  useEffect(() => {
    const refresh = () => setUsername(getUsername());
    window.addEventListener(AUTH_CHANGED_EVENT, refresh);
    return () => window.removeEventListener(AUTH_CHANGED_EVENT, refresh);
  }, []);

  const isActive = (to: string) =>
    to === "/" ? location.pathname === "/" : location.pathname.startsWith(to);

  const handleLogout = async () => {
    try {
      await logout();
    } catch {
      // token 失效也继续本地登出
    }
    clearAuth();
    toast.success("已退出登录");
    navigate("/login");
  };

  return (
    <TooltipProvider>
    <SidebarProvider>
      <Sidebar collapsible="icon">
        <SidebarHeader className="flex items-center gap-2 px-4 py-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground font-bold">
            S
          </div>
          <span className="text-base font-semibold tracking-tight group-data-[collapsible=icon]:hidden">
            slink 短链接
          </span>
        </SidebarHeader>
        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupContent>
              <SidebarMenu>
                {NAV.map((item) => (
                  <SidebarMenuItem key={item.to}>
                    <SidebarMenuButton
                      isActive={isActive(item.to)}
                      tooltip={item.label}
                      onClick={() => navigate(item.to)}
                    >
                      <item.icon />
                      <span>{item.label}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        </SidebarContent>
        <SidebarFooter>
          <div className="flex items-center gap-2 px-2 py-1 group-data-[collapsible=icon]:hidden">
            <span className="text-xs text-muted-foreground truncate">
              {username || "admin"}
            </span>
            <Button variant="ghost" size="icon" className="ml-auto h-7 w-7" aria-label="退出登录" onClick={handleLogout}>
              <LogOut className="h-4 w-4" />
            </Button>
          </div>
        </SidebarFooter>
      </Sidebar>

      <SidebarInset>
        <header className="flex h-14 items-center gap-2 border-b px-4 sticky top-0 z-20 bg-background/80 backdrop-blur">
          <SidebarTrigger />
          <Separator orientation="vertical" className="h-5" />
          <CommandMenu>
            <Button
              variant="outline"
              size="sm"
              className="ml-2 gap-2 text-muted-foreground w-56 justify-start"
            >
              <Command className="h-3.5 w-3.5" />
              搜索短链接
            </Button>
          </CommandMenu>
          <div className="ml-auto">
            <ThemeToggle />
          </div>
        </header>
        <main className="flex-1 p-4 md:p-6">
          <Outlet />
        </main>
      </SidebarInset>
    </SidebarProvider>
    </TooltipProvider>
  );
}
