import { useState, useEffect, lazy, Suspense } from "react";
import { useParams, useNavigate } from "react-router-dom";
import Header from "@/components/Header";
import { useToast } from "@/components/ToastContainer";
import { Monitor, Terminal as TerminalIcon, ArrowLeft, AlertCircle } from "lucide-react";

// 动态导入避免 SSR 问题
const VNCConsole = lazy(() => import("@/components/VNCConsole"));
const SerialConsole = lazy(() => import("@/components/SerialConsole"));

type ConsoleType = "vnc" | "serial";

interface ConsoleInfo {
  instance_id: string;
  vnc_socket?: string;
  vnc_port?: number;
  vnc_token?: string;
  serial_device?: string;
  serial_port?: number;
  serial_token?: string;
  type: string;
}

export default function InstanceConsolePage() {
  const params = useParams();
  const navigate = useNavigate();
  const { nodeName, id: instanceId } = params;
  const toast = useToast();

  const [consoleType, setConsoleType] = useState<ConsoleType>("vnc");
  const [consoleInfo, setConsoleInfo] = useState<ConsoleInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string>("");

  useEffect(() => {
    if (nodeName && instanceId) {
      fetchConsoleInfo();
    }
  }, [nodeName, instanceId, consoleType]);

  const fetchConsoleInfo = async () => {
    setLoading(true);
    setError("");
    try {
      const response = await fetch("/api/instances/console", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          node_name: nodeName,
          instance_id: instanceId,
          type: consoleType,
        }),
      });

      if (response.ok) {
        const data = await response.json();
        setConsoleInfo(data);

        // 检查是否支持请求的控制台类型
        if (consoleType === "vnc" && !data.vnc_socket) {
          setError("VNC console is not available for this instance");
          toast.error("VNC console is not configured");
        } else if (consoleType === "serial" && !data.serial_device) {
          setError("Serial console is not available for this instance");
          toast.error("Serial console is not available");
        }
      } else {
        const errorData = await response.json();
        const errorMsg = errorData?.message || "Failed to connect to console";
        setError(errorMsg);
        toast.error(errorMsg);
      }
    } catch (error: any) {
      console.error("Failed to fetch console info:", error);
      const errorMsg = error?.message || "Failed to connect to console";
      setError(errorMsg);
      toast.error(errorMsg);
    } finally {
      setLoading(false);
    }
  };

  const getWebSocketURL = () => {
    if (!consoleInfo) return "";

    // 使用后端的 WebSocket 代理端点，包含 node_name
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const host = window.location.host; // 包含端口号
    const endpoint = consoleType === "vnc" ? "vnc" : "serial";

    return `${protocol}//${host}/api/console/${endpoint}/${nodeName}/${instanceId}`;
  };

  const handleConsoleTypeChange = (type: ConsoleType) => {
    setConsoleType(type);
    setConnected(false);
    setError("");
  };

  if (!nodeName || !instanceId) {
    return (
      <div className="card text-center py-12">
        <p className="text-gray-500">Invalid instance parameters</p>
        <button onClick={() => navigate("/instances")} className="btn-primary mt-4">
          Back to Instances
        </button>
      </div>
    );
  }

  return (
    <>
      <Header
        title={`Instance Console`}
        description={`Console access for ${instanceId.substring(0, 12)}... on node ${nodeName}`}
        action={
          <button
            onClick={() => navigate(`/instances/${nodeName}/${instanceId}`)}
            className="btn-secondary flex items-center gap-2"
          >
            <ArrowLeft size={16} />
            Back to Instance
          </button>
        }
      />

      {/* Console Type Selector */}
      <div className="card mb-4">
        <div className="flex items-center justify-between">
          <div className="flex gap-2">
            <button
              onClick={() => handleConsoleTypeChange("vnc")}
              className={`flex items-center gap-2 px-4 py-2 rounded transition-colors ${
                consoleType === "vnc"
                  ? "bg-blue-600 text-white"
                  : "bg-gray-200 text-gray-700 hover:bg-gray-300"
              }`}
            >
              <Monitor size={16} />
              VNC Console
            </button>
            <button
              onClick={() => handleConsoleTypeChange("serial")}
              className={`flex items-center gap-2 px-4 py-2 rounded transition-colors ${
                consoleType === "serial"
                  ? "bg-blue-600 text-white"
                  : "bg-gray-200 text-gray-700 hover:bg-gray-300"
              }`}
            >
              <TerminalIcon size={16} />
              Serial Console
            </button>
          </div>

          {connected && (
            <div className="flex items-center gap-2 text-sm text-green-600">
              <div className="w-2 h-2 bg-green-600 rounded-full animate-pulse"></div>
              Connected
            </div>
          )}
        </div>

        {error && (
          <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded-lg flex items-start gap-2">
            <AlertCircle size={18} className="text-red-600 mt-0.5 flex-shrink-0" />
            <div className="text-sm text-red-800">{error}</div>
          </div>
        )}

        {consoleInfo && !error && (
          <div className="mt-4 text-sm text-gray-600">
            <p className="font-medium mb-1">Console Information:</p>
            {consoleType === "vnc" && consoleInfo.vnc_socket && (
              <p className="font-mono text-xs">VNC Socket: {consoleInfo.vnc_socket}</p>
            )}
            {consoleType === "serial" && consoleInfo.serial_device && (
              <p className="font-mono text-xs">Serial Device: {consoleInfo.serial_device}</p>
            )}
          </div>
        )}
      </div>

      {/* Console Display */}
      <div className="card p-0 overflow-hidden flex-1" style={{ minHeight: "400px", height: "calc(100vh - 320px)" }}>
        {loading ? (
          <div className="flex items-center justify-center h-full bg-gray-50">
            <div className="text-center">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
              <p className="text-gray-500">Loading console...</p>
            </div>
          </div>
        ) : error ? (
          <div className="flex items-center justify-center h-full bg-gray-50">
            <div className="text-center max-w-md">
              <AlertCircle size={48} className="text-red-500 mx-auto mb-4" />
              <h3 className="text-lg font-semibold text-gray-900 mb-2">
                Console Not Available
              </h3>
              <p className="text-gray-600 mb-4">{error}</p>
              <button
                onClick={fetchConsoleInfo}
                className="btn-primary"
              >
                Retry Connection
              </button>
            </div>
          </div>
        ) : consoleInfo && getWebSocketURL() ? (
          consoleType === "vnc" ? (
            <Suspense fallback={<div className="flex items-center justify-center h-full bg-gray-50"><p className="text-gray-500">Loading VNC Console...</p></div>}>
              <VNCConsole
                wsUrl={getWebSocketURL()}
                onConnect={() => {
                  setConnected(true);
                  toast.success("VNC console connected");
                }}
                onDisconnect={() => {
                  setConnected(false);
                  toast.info("VNC console disconnected");
                }}
                onError={(err) => {
                  setError(err);
                  toast.error(err);
                }}
              />
            </Suspense>
          ) : (
            <Suspense fallback={<div className="flex items-center justify-center h-full bg-gray-50"><p className="text-gray-500">Loading Serial Console...</p></div>}>
              <SerialConsole
                wsUrl={getWebSocketURL()}
                onConnect={() => {
                  setConnected(true);
                  toast.success("Serial console connected");
                }}
                onDisconnect={() => {
                  setConnected(false);
                  toast.info("Serial console disconnected");
                }}
                onError={(err) => {
                  setError(err);
                  toast.error(err);
                }}
              />
            </Suspense>
          )
        ) : (
          <div className="flex items-center justify-center h-full bg-gray-50">
            <p className="text-gray-500">Console configuration incomplete</p>
          </div>
        )}
      </div>

      {/* Help Information */}
      <div className="card mt-4 bg-blue-50 border-blue-200">
        <h3 className="text-sm font-semibold text-blue-900 mb-2">Console Tips</h3>
        <ul className="text-sm text-blue-800 space-y-1">
          <li>* <strong>VNC Console:</strong> Provides full graphical access to the instance</li>
          <li>* <strong>Serial Console:</strong> Provides text-based terminal access</li>
          <li>* Use Ctrl+Alt+Del through the VNC menu if needed</li>
          <li>* Serial console requires the instance to be configured with a serial port</li>
        </ul>
      </div>
    </>
  );
}

