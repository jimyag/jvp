"use client";

import { useEffect, useState, useRef } from "react";

interface VNCConsoleProps {
  wsUrl: string;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: string) => void;
}

export default function VNCConsole({
  wsUrl,
  onConnect,
  onDisconnect,
}: VNCConsoleProps) {
  const [iframeUrl, setIframeUrl] = useState<string>("");
  const hasConnected = useRef(false);

  useEffect(() => {
    if (wsUrl) {
      // 构建 iframe URL,传递 WebSocket 地址
      const url = `/vnc.html?ws=${encodeURIComponent(wsUrl)}`;
      setIframeUrl(url);

      // 只在首次连接时触发 onConnect
      if (!hasConnected.current) {
        const timer = setTimeout(() => {
          if (!hasConnected.current) {
            hasConnected.current = true;
            onConnect?.();
          }
        }, 2000);

        return () => clearTimeout(timer);
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [wsUrl]);

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
