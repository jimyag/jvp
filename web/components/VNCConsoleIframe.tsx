"use client";

import { useEffect, useState } from "react";

interface VNCConsoleIframeProps {
  wsUrl: string;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: string) => void;
}

export default function VNCConsoleIframe({
  wsUrl,
  onConnect,
  onDisconnect,
}: VNCConsoleIframeProps) {
  const [iframeUrl, setIframeUrl] = useState<string>("");

  useEffect(() => {
    if (wsUrl) {
      // 构建 iframe URL,传递 WebSocket 地址
      const url = `/vnc.html?ws=${encodeURIComponent(wsUrl)}`;
      setIframeUrl(url);

      // 简单的连接状态模拟 (实际状态由iframe内部管理)
      const timer = setTimeout(() => {
        onConnect?.();
      }, 2000);

      return () => clearTimeout(timer);
    }
  }, [wsUrl, onConnect]);

  if (!iframeUrl) {
    return (
      <div className="flex items-center justify-center h-full bg-black">
        <p className="text-white">Initializing VNC console...</p>
      </div>
    );
  }

  return (
    <iframe
      src={iframeUrl}
      className="w-full h-full border-0"
      title="VNC Console"
      allow="fullscreen"
    />
  );
}
