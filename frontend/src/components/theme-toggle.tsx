import { Moon, Sun } from "lucide-react";
import { Button } from "@/components/ui/button";

export function ThemeToggle() {
  const isDark = document.documentElement.classList.contains("dark");
  return (
    <Button
      variant="ghost"
      size="icon"
      aria-label="切换主题"
      onClick={() => {
        document.documentElement.classList.toggle("dark");
        localStorage.setItem(
          "slink_theme",
          isDark ? "light" : "dark",
        );
      }}
    >
      <Sun className="h-4 w-4 dark:hidden" />
      <Moon className="hidden h-4 w-4 dark:block" />
    </Button>
  );
}

/** 初始化主题：优先本地记忆，其次跟随系统 */
export function initTheme() {
  const saved = localStorage.getItem("slink_theme");
  if (saved === "dark" || (!saved && matchMedia("(prefers-color-scheme: dark)").matches)) {
    document.documentElement.classList.add("dark");
  }
}
