import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Link2, Search } from "lucide-react";

import {
  Command,
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { pageLinks, type LinkItem } from "@/api";

export function CommandMenu({ children }: { children: React.ReactNode }) {
  const [open, setOpen] = useState(false);
  const [keyword, setKeyword] = useState("");
  const navigate = useNavigate();

  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((v) => !v);
      }
    };
    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, []);

  const { data } = useQuery({
    queryKey: ["cmd-links", keyword],
    queryFn: () => pageLinks({ page: 1, size: 8, key: keyword }),
    enabled: open,
  });

  const links: LinkItem[] = data?.records ?? [];

  return (
    <>
      <span onClick={() => setOpen(true)}>{children}</span>
      <CommandDialog open={open} onOpenChange={setOpen}>
        {/* vega 版 CommandDialog 不内嵌 Command 根组件，CommandInput 必须包裹在 Command 内 */}
        <Command>
          <CommandInput
            placeholder="搜索短码 / 原始链接 / 标题..."
            value={keyword}
            onValueChange={setKeyword}
          />
          <CommandList>
            <CommandEmpty>未找到匹配的短链接</CommandEmpty>
            <CommandGroup heading="短链接">
              {links.map((l) => (
                <CommandItem
                  key={l.id}
                  value={`${l.shortUri} ${l.originUrl} ${l.title}`}
                  onSelect={() => {
                    setOpen(false);
                    navigate(`/links/${l.id}`);
                  }}
                >
                  <Link2 className="h-4 w-4" />
                  <span className="font-mono">/{l.shortUri}</span>
                  <span className="truncate text-muted-foreground">
                    {l.title || l.originUrl}
                  </span>
                </CommandItem>
              ))}
            </CommandGroup>
            <CommandGroup heading="操作">
              <CommandItem
                onSelect={() => {
                  setOpen(false);
                  navigate("/links?create=1");
                }}
              >
                <Search className="h-4 w-4" />
                新建短链接
              </CommandItem>
            </CommandGroup>
          </CommandList>
        </Command>
      </CommandDialog>
    </>
  );
}
