import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { Toaster } from "@/components/ui/sonner";

import { AppLayout } from "@/components/app-layout";
import { initTheme } from "@/components/theme-toggle";
import { isLoggedIn } from "@/lib/auth";
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

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route
            element={
              <RequireAuth>
                <AppLayout />
              </RequireAuth>
            }
          >
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
