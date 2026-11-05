import { QueryClient, QueryClientProvider, useQuery } from "@tanstack/react-query";
import { BrowserRouter, Navigate, Route, Routes, useLocation } from "react-router-dom";
import { Toaster } from "@/components/ui/sonner";

import { AppLayout } from "@/components/app-layout";
import { initTheme } from "@/components/theme-toggle";
import { isLoggedIn } from "@/lib/auth";
import { getMe } from "@/api";
import LoginPage from "@/pages/login";
import DashboardPage from "@/pages/dashboard";
import LinksPage from "@/pages/links";
import LinkStatsPage from "@/pages/link-stats";
import RecycleBinPage from "@/pages/recycle-bin";
import SettingsPage from "@/pages/settings";

initTheme();

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, refetchOnWindowFocus: false },
  },
});

function RequireAuth({ children }: { children: React.ReactNode }) {
  if (!isLoggedIn()) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

/** 初始密码未修改且未被豁免时，强制停留在设置页直至完成修改 */
function ForceChangePasswordGate() {
  const location = useLocation();
  const meQ = useQuery({ queryKey: ["me"], queryFn: getMe });
  if (meQ.data?.mustChangePassword && location.pathname !== "/settings") {
    return <Navigate to="/settings" replace />;
  }
  return null;
}

/** 登录态布局：本地登录标志 + 强改密守卫 + 布局骨架 */
function AuthedLayout() {
  return (
    <RequireAuth>
      <ForceChangePasswordGate />
      <AppLayout />
    </RequireAuth>
  );
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route element={<AuthedLayout />}>
            <Route path="/" element={<DashboardPage />} />
            <Route path="/links" element={<LinksPage />} />
            <Route path="/links/:id" element={<LinkStatsPage />} />
            <Route path="/recycle-bin" element={<RecycleBinPage />} />
            <Route path="/settings" element={<SettingsPage />} />
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
      <Toaster position="top-center" richColors />
    </QueryClientProvider>
  );
}
