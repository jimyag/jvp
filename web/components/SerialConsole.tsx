"use client";

import { useEffect, useRef, useState } from "react";
import "@xterm/xterm/css/xterm.css";

interface SerialConsoleProps {
  wsUrl: string;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: string) => void;
}

export default function SerialConsole({
  wsUrl,
  onConnect,
  onDisconnect,
  onError,
}: SerialConsoleProps) {
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<any>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let term: any = null;
    let ws: WebSocket | null = null;
    let fitAddon: any = null;

    const initTerminal = async () => {
      if (!terminalRef.current) return;

      try {
        // 动态导入 xterm 以避免 SSR 问题
        const { Terminal } = await import("@xterm/xterm");
        const { FitAddon } = await import("@xterm/addon-fit");
        const { WebLinksAddon } = await import("@xterm/addon-web-links");

        // 创建终端
        term = new Terminal({
          cursorBlink: true,
          fontSize: 14,
          fontFamily: '"Cascadia Code", Menlo, "DejaVu Sans Mono", monospace',
          theme: {
            background: "#000000",
            foreground: "#ffffff",
            cursor: "#ffffff",
            cursorAccent: "#000000",
            selectionBackground: "rgba(255, 255, 255, 0.3)",
          },
          scrollback: 10000,
        });

        fitAddon = new FitAddon();
        term.loadAddon(fitAddon);
        term.loadAddon(new WebLinksAddon());

        term.open(terminalRef.current);
        fitAddon.fit();

        xtermRef.current = term;

        // 连接 WebSocket
        ws = new WebSocket(wsUrl);
        wsRef.current = ws;

        ws.onopen = () => {
          console.log("Serial console WebSocket connected");
          setLoading(false);
          onConnect?.();
          term.write("Serial console connected.\r\n");
        };

        ws.onmessage = (event) => {
          if (term) {
            term.write(event.data);
          }
        };

        ws.onerror = (event) => {
          console.error("Serial console WebSocket error:", event);
          setLoading(false);
          onError?.("WebSocket connection error");
          if (term) {
            term.write("\r\n\x1b[31mWebSocket connection error\x1b[0m\r\n");
          }
        };

        ws.onclose = () => {
          console.log("Serial console WebSocket disconnected");
          setLoading(false);
          onDisconnect?.();
          if (term) {
            term.write("\r\n\x1b[33mConnection closed\x1b[0m\r\n");
          }
        };

        // 终端输入发送到 WebSocket
        term.onData((data: string) => {
          if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(data);
          }
        });

        // 窗口调整大小时重新 fit
        const handleResize = () => {
          if (fitAddon) {
            try {
              fitAddon.fit();
            } catch (error) {
              console.error("Error fitting terminal:", error);
            }
          }
        };

        window.addEventListener("resize", handleResize);

        // 初始欢迎信息
        term.write("Connecting to serial console...\r\n");

        return () => {
          window.removeEventListener("resize", handleResize);
        };
      } catch (error) {
        console.error("Failed to initialize terminal:", error);
        setLoading(false);
        onError?.("Failed to initialize terminal");
      }
    };

    initTerminal();

    return () => {
      if (ws) {
        ws.close();
      }
      if (term) {
        term.dispose();
      }
    };
  }, [wsUrl, onConnect, onDisconnect, onError]);

  return (
    <div className="relative w-full h-full bg-black">
      {loading && (
        <div className="absolute inset-0 flex items-center justify-center bg-black bg-opacity-75 z-10">
          <div className="text-white text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-white mx-auto mb-4"></div>
            <p>Connecting to serial console...</p>
          </div>
        </div>
      )}
      <div ref={terminalRef} className="w-full h-full p-2" />
    </div>
  );
}
