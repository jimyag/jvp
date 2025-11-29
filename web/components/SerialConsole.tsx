"use client";

import { useEffect, useRef, useState, useCallback } from "react";
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
  const fitAddonRef = useRef<any>(null);
  const [loading, setLoading] = useState(true);
  const [isInitialized, setIsInitialized] = useState(false);

  useEffect(() => {
    // 如果已经初始化过，不要重复初始化
    if (isInitialized || xtermRef.current) return;

    let term: any = null;
    let ws: WebSocket | null = null;
    let fitAddon: any = null;
    let isMounted = true;

    const initTerminal = async () => {
      // 确保 DOM 元素存在
      if (!terminalRef.current || !isMounted) return;

      try {
        // 动态导入 xterm 以避免 SSR 问题
        const { Terminal } = await import("@xterm/xterm");
        const { FitAddon } = await import("@xterm/addon-fit");
        const { WebLinksAddon } = await import("@xterm/addon-web-links");

        // 再次检查 DOM 元素和挂载状态
        if (!terminalRef.current || !isMounted) return;

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

        // 确保容器有尺寸
        const container = terminalRef.current;
        if (container.clientWidth === 0 || container.clientHeight === 0) {
          // 等待容器有尺寸
          await new Promise(resolve => setTimeout(resolve, 100));
        }

        if (!isMounted) return;

        term.open(container);

        // 延迟 fit 以确保 DOM 完全渲染
        setTimeout(() => {
          if (fitAddon && isMounted) {
            try {
              fitAddon.fit();
            } catch (e) {
              console.warn("Initial fit failed:", e);
            }
          }
        }, 50);

        xtermRef.current = term;
        fitAddonRef.current = fitAddon;
        setIsInitialized(true);

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
          if (fitAddonRef.current) {
            try {
              fitAddonRef.current.fit();
            } catch (error) {
              console.error("Error fitting terminal:", error);
            }
          }
        };

        window.addEventListener("resize", handleResize);

        // 初始欢迎信息
        term.write("Connecting to serial console...\r\n");

        // 清理函数
        return () => {
          window.removeEventListener("resize", handleResize);
        };
      } catch (error) {
        console.error("Failed to initialize terminal:", error);
        if (isMounted) {
          setLoading(false);
          onError?.("Failed to initialize terminal");
        }
      }
    };

    initTerminal();

    return () => {
      isMounted = false;
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
      if (xtermRef.current) {
        xtermRef.current.dispose();
        xtermRef.current = null;
      }
      setIsInitialized(false);
    };
  }, [wsUrl]); // 只依赖 wsUrl，避免不必要的重新初始化

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
