import { QRCodeSVG } from "qrcode.react";
import { Copy } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { getShortDomain } from "@/lib/auth";

/** 短链基础地址：优先配置的短链域名（补全 scheme），单域模式回落当前访问域名 */
function baseOrigin(): string {
  const domain = getShortDomain();
  if (!domain) return window.location.origin;
  const trimmed = domain.replace(/\/+$/, "");
  return /^https?:\/\//.test(trimmed) ? trimmed : `https://${trimmed}`;
}

export function fullShortUrl(shortUri: string) {
  return `${baseOrigin()}/${shortUri}`;
}

export function copyText(text: string) {
  navigator.clipboard.writeText(text).then(
    () => toast.success("已复制"),
    () => toast.error("复制失败"),
  );
}

interface Props {
  shortUri: string | null;
  onOpenChange: (open: boolean) => void;
}

export function QrCodeDialog({ shortUri, onOpenChange }: Props) {
  const url = shortUri ? fullShortUrl(shortUri) : "";
  return (
    <Dialog open={!!shortUri} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xs">
        <DialogHeader>
          <DialogTitle>短链接二维码</DialogTitle>
          <DialogDescription className="font-mono break-all">
            {url}
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col items-center gap-4 py-2">
          <div className="rounded-lg border bg-white p-3">
            <QRCodeSVG value={url} size={192} />
          </div>
          <Button variant="outline" className="gap-2" onClick={() => copyText(url)}>
            <Copy className="h-4 w-4" />
            复制链接
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
