import { useEffect, useRef, useState } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import "@xterm/xterm/css/xterm.css";
import { Loader2 } from "lucide-react";

interface SerialConsoleProps {
  wsUrl: string;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: string) => void;
}

export default function SerialConsole({ wsUrl, onConnect, onDisconnect, onError }: SerialConsoleProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [connecting, setConnecting] = useState(true);
  const callbacks = useRef({ onConnect, onDisconnect, onError });

  useEffect(() => {
    callbacks.current = { onConnect, onDisconnect, onError };
  });

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 13,
      lineHeight: 1.2,
      fontFamily: '"JetBrains Mono", "Cascadia Code", Menlo, "DejaVu Sans Mono", monospace',
      theme: {
        background: "#0b0c0f",
        foreground: "#e6e8ec",
        cursor: "#e6e8ec",
        cursorAccent: "#0b0c0f",
        selectionBackground: "rgba(110, 140, 245, 0.35)",
      },
      scrollback: 10000,
    });
    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.loadAddon(new WebLinksAddon());
    term.open(container);

    const fit = () => {
      try {
        fitAddon.fit();
      } catch {
        // 容器尚未完成布局时 fit 可能失败，下一次 resize 会重试
      }
    };
    const resizeObserver = new ResizeObserver(fit);
    resizeObserver.observe(container);
    requestAnimationFrame(fit);

    term.write("\x1b[90mConnecting to serial console…\x1b[0m\r\n");

    const ws = new WebSocket(wsUrl);
    ws.binaryType = "arraybuffer";
    const decoder = new TextDecoder();
    let disposed = false;

    ws.onopen = () => {
      if (disposed) return;
      setConnecting(false);
      term.write("\x1b[32mConnected.\x1b[0m Press Enter if the prompt does not appear.\r\n");
      term.focus();
      callbacks.current.onConnect?.();
    };
    ws.onmessage = (event) => {
      if (event.data instanceof ArrayBuffer) {
        term.write(decoder.decode(event.data, { stream: true }));
      } else if (event.data instanceof Blob) {
        event.data.arrayBuffer().then((buf) => term.write(decoder.decode(buf, { stream: true })));
      } else {
        term.write(event.data);
      }
    };
    ws.onerror = () => {
      if (disposed) return;
      setConnecting(false);
      term.write("\r\n\x1b[31mWebSocket connection error\x1b[0m\r\n");
      callbacks.current.onError?.("WebSocket connection error");
    };
    ws.onclose = () => {
      if (disposed) return;
      setConnecting(false);
      term.write("\r\n\x1b[33mConnection closed\x1b[0m\r\n");
      callbacks.current.onDisconnect?.();
    };

    const dataListener = term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) ws.send(data);
    });

    return () => {
      disposed = true;
      resizeObserver.disconnect();
      dataListener.dispose();
      ws.close();
      term.dispose();
    };
  }, [wsUrl]);

  return (
    <div className="relative h-full w-full bg-[#0b0c0f]">
      {connecting && (
        <div className="absolute inset-0 z-10 flex items-center justify-center bg-[#0b0c0f]/80 text-sm text-white/70">
          <Loader2 size={18} className="mr-2 animate-spin" />
          Connecting…
        </div>
      )}
      <div ref={containerRef} className="h-full w-full p-3" />
    </div>
  );
}
