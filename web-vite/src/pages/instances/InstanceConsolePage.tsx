import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { AlertCircle, ExternalLink, Maximize2, Monitor, RotateCw, Terminal as TerminalIcon } from "lucide-react";
import PageHeader from "@/components/PageHeader";
import SerialConsole from "@/components/SerialConsole";
import { Badge, EmptyState, SegmentedControl, Spinner, StatusBadge } from "@/components/ui";
import { api, errorMessage } from "@/lib/api";
import type { Instance } from "@/lib/types";

type ConsoleType = "vnc" | "serial";

interface ConsoleInfo {
  instance_id: string;
  vnc_socket?: string;
  serial_device?: string;
  type: string;
}

export default function InstanceConsolePage() {
  const { nodeName = "", id: instanceId = "" } = useParams();
  const [consoleType, setConsoleType] = useState<ConsoleType>("vnc");
  const [instance, setInstance] = useState<Instance | null>(null);
  const [info, setInfo] = useState<ConsoleInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [connected, setConnected] = useState(false);
  const [session, setSession] = useState(0);
  const frameRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    api<{ instances: Instance[] }>("/api/describe-instances", { node_name: nodeName, instance_ids: [instanceId] })
      .then((data) => setInstance(data.instances?.[0] || null))
      .catch(() => setInstance(null));
  }, [nodeName, instanceId]);

  const loadConsole = useCallback(async () => {
    setLoading(true);
    setError("");
    setConnected(false);
    try {
      const data = await api<ConsoleInfo>("/api/get-instance-console", {
        node_name: nodeName,
        instance_id: instanceId,
        type: consoleType,
      });
      setInfo(data);
      if (consoleType === "vnc" && !data.vnc_socket) setError("This instance has no VNC display configured.");
      if (consoleType === "serial" && !data.serial_device) setError("This instance has no serial port configured.");
    } catch (err) {
      setError(errorMessage(err, "Failed to connect to console"));
    } finally {
      setLoading(false);
      setSession((s) => s + 1);
    }
  }, [nodeName, instanceId, consoleType]);

  useEffect(() => {
    loadConsole();
  }, [loadConsole]);

  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const wsUrl = `${protocol}//${window.location.host}/api/get-${consoleType}-console/${encodeURIComponent(nodeName)}/${encodeURIComponent(instanceId)}`;
  const vncPage = `/vnc.html?ws=${encodeURIComponent(wsUrl)}`;
  const detailPath = `/instances/${encodeURIComponent(nodeName)}/${encodeURIComponent(instanceId)}`;
  const listPath = `/instances?node=${encodeURIComponent(nodeName)}`;
  const label = instance?.name || instanceId;
  const ready = !loading && !error && info;

  const goFullscreen = () => {
    frameRef.current?.requestFullscreen?.().catch(() => undefined);
  };

  return (
    <div className="flex h-[calc(100vh-7.5rem)] min-h-[520px] flex-col lg:h-[calc(100vh-4rem)]">
      <PageHeader
        breadcrumbs={[
          { label: "Instances", to: listPath },
          { label, to: detailPath },
          { label: "Console" },
        ]}
        title={`Console · ${label}`}
        meta={
          <>
            {instance && <StatusBadge status={instance.state} />}
            {consoleType === "serial" && connected && (
              <Badge tone="success" dot pulse>
                Connected
              </Badge>
            )}
          </>
        }
        actions={
          <>
            <SegmentedControl
              value={consoleType}
              onChange={setConsoleType}
              options={[
                { value: "vnc", label: "Graphical", icon: <Monitor size={14} /> },
                { value: "serial", label: "Serial", icon: <TerminalIcon size={14} /> },
              ]}
            />
            <button className="btn-secondary w-9 px-0" onClick={loadConsole} title="Reconnect" aria-label="Reconnect">
              <RotateCw size={14} />
            </button>
            <button className="btn-secondary w-9 px-0" onClick={goFullscreen} title="Fullscreen" aria-label="Fullscreen" disabled={!ready}>
              <Maximize2 size={14} />
            </button>
            {consoleType === "vnc" && (
              <a href={vncPage} target="_blank" rel="noopener noreferrer" className="btn-secondary" aria-disabled={!ready}>
                <ExternalLink size={14} />
                New window
              </a>
            )}
          </>
        }
      />

      <div ref={frameRef} className="relative min-h-0 flex-1 overflow-hidden rounded-lg border border-line bg-[#0b0c0f] shadow-card">
        {loading ? (
          <div className="flex h-full items-center justify-center text-sm text-white/60">
            <Spinner className="mr-2 text-white/60" />
            Preparing console…
          </div>
        ) : error ? (
          <div className="flex h-full items-center justify-center bg-surface">
            <EmptyState
              icon={<AlertCircle size={20} />}
              title="Console unavailable"
              description={
                <>
                  {error}
                  {instance && instance.state !== "running" && (
                    <>
                      <br />
                      The instance is {instance.state}. Start it from the <Link to={detailPath} className="link">instance page</Link>.
                    </>
                  )}
                </>
              }
              action={
                <button className="btn-primary" onClick={loadConsole}>
                  <RotateCw size={14} />
                  Try again
                </button>
              }
            />
          </div>
        ) : consoleType === "vnc" ? (
          <iframe
            key={session}
            src={vncPage}
            title={`VNC console for ${label}`}
            className="h-full w-full border-0"
            allow="clipboard-read; clipboard-write; fullscreen"
          />
        ) : (
          <SerialConsole
            key={session}
            wsUrl={wsUrl}
            onConnect={() => setConnected(true)}
            onDisconnect={() => setConnected(false)}
            onError={(err) => setError(err)}
          />
        )}
      </div>
      <p className="mt-2 text-xs text-fg-subtle">
        {consoleType === "vnc"
          ? "Click inside the console to capture the keyboard. Use the Paste / Copy buttons in the console to sync the clipboard."
          : "The serial console needs a getty on ttyS0 inside the guest. Press Enter if nothing is shown."}
      </p>
    </div>
  );
}
