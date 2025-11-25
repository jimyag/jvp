"use client";

import { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import DashboardLayout from "@/components/DashboardLayout";
import Header from "@/components/Header";
import StatusBadge from "@/components/StatusBadge";
import { useToast } from "@/components/ToastContainer";
import { apiPost } from "@/lib/api";
import {
  ArrowLeft,
  Server,
  Cpu,
  MemoryStick,
  HardDrive,
  Network,
  Usb,
  MonitorCheck,
  Power,
  PowerOff,
  RefreshCw,
} from "lucide-react";

interface Node {
  name: string;
  uuid: string;
  uri: string;
  type: string;
  state: string;
}

interface CPUInfo {
  cores: number;
  threads: number;
  model: string;
  vendor: string;
  frequency: number;
  arch: string;
  cache_size: number;
  flags: string[];
}

interface MemoryInfo {
  total: number;
  available: number;
  used: number;
  usage_percent: number;
  swap_total: number;
  swap_used: number;
}

interface NodeSummary {
  cpu: CPUInfo;
  memory: MemoryInfo;
  numa: any;
  hugepages: any;
  virtualization: any;
}

export default function NodeDetailPage() {
  const params = useParams();
  const router = useRouter();
  const nodeName = params.name as string;
  const toast = useToast();

  const [node, setNode] = useState<Node | null>(null);
  const [summary, setSummary] = useState<NodeSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);

  useEffect(() => {
    fetchNodeDetails();
    fetchNodeSummary();
  }, [nodeName]);

  const fetchNodeDetails = async () => {
    try {
      const data = await apiPost<Node>("/api/describe-node", {
        name: nodeName,
      });
      setNode(data);
    } catch (error: any) {
      console.error("Failed to fetch node:", error);
      toast.error(error?.message || "Failed to fetch node details");
    }
  };

  const fetchNodeSummary = async () => {
    try {
      const data = await apiPost<NodeSummary>("/api/describe-node-summary", {
        name: nodeName,
      });
      setSummary(data);
    } catch (error: any) {
      console.error("Failed to fetch node summary:", error);
      toast.error(error?.message || "Failed to fetch node summary");
    } finally {
      setLoading(false);
    }
  };

  const handleEnableNode = async () => {
    setActionLoading(true);
    try {
      await apiPost("/api/enable-node", { name: nodeName });
      toast.success("Node enabled successfully");
      await fetchNodeDetails();
    } catch (error: any) {
      console.error("Failed to enable node:", error);
      toast.error(error?.message || "Failed to enable node");
    } finally {
      setActionLoading(false);
    }
  };

  const handleDisableNode = async () => {
    setActionLoading(true);
    try {
      await apiPost("/api/disable-node", { name: nodeName });
      toast.success("Node disabled successfully");
      await fetchNodeDetails();
    } catch (error: any) {
      console.error("Failed to disable node:", error);
      toast.error(error?.message || "Failed to disable node");
    } finally {
      setActionLoading(false);
    }
  };

  const handleRefresh = async () => {
    setLoading(true);
    await Promise.all([fetchNodeDetails(), fetchNodeSummary()]);
  };

  const formatBytes = (bytes: number): string => {
    const gb = bytes / (1024 * 1024 * 1024);
    return `${gb.toFixed(2)} GB`;
  };

  const getStateColor = (state: string) => {
    switch (state) {
      case "online":
        return "green";
      case "offline":
        return "red";
      case "maintenance":
        return "yellow";
      default:
        return "gray";
    }
  };

  if (loading) {
    return (
      <DashboardLayout>
        <div className="flex justify-center items-center h-64">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        </div>
      </DashboardLayout>
    );
  }

  if (!node) {
    return (
      <DashboardLayout>
        <div className="text-center py-12">
          <Server size={48} className="mx-auto text-gray-400 mb-4" />
          <p className="text-gray-500">Node not found</p>
          <button onClick={() => router.push("/nodes")} className="btn-primary mt-4">
            Back to Nodes
          </button>
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout>
      <Header
        title={`Node: ${node.name}`}
        description={`Details and hardware information for node ${node.name}`}
        action={
          <div className="flex gap-2">
            <button
              onClick={handleRefresh}
              disabled={loading}
              className="btn-secondary flex items-center gap-2"
            >
              <RefreshCw size={16} className={loading ? "animate-spin" : ""} />
              Refresh
            </button>
            <button
              onClick={() => router.push("/nodes")}
              className="btn-secondary flex items-center gap-2"
            >
              <ArrowLeft size={16} />
              Back
            </button>
          </div>
        }
      />

      {/* Node Basic Info */}
      <div className="card mb-4">
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <Server size={20} />
          Node Information
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="text-sm font-medium text-gray-600">Name</label>
            <p className="text-gray-900 font-medium">{node.name}</p>
          </div>
          <div>
            <label className="text-sm font-medium text-gray-600">UUID</label>
            <p className="text-gray-900 font-mono text-sm">{node.uuid}</p>
          </div>
          <div>
            <label className="text-sm font-medium text-gray-600">Type</label>
            <div className="mt-1">
              <StatusBadge status={node.type} color="blue" text={node.type} />
            </div>
          </div>
          <div>
            <label className="text-sm font-medium text-gray-600">State</label>
            <div className="mt-1">
              <StatusBadge
                status={node.state}
                color={getStateColor(node.state)}
                text={node.state}
              />
            </div>
          </div>
          <div className="md:col-span-2">
            <label className="text-sm font-medium text-gray-600">URI</label>
            <p className="text-gray-900 font-mono text-sm">{node.uri}</p>
          </div>
        </div>

        {/* Node Actions */}
        <div className="mt-6 pt-6 border-t flex gap-2">
          <button
            onClick={handleEnableNode}
            disabled={actionLoading || node.state === "online"}
            className="btn-primary flex items-center gap-2"
          >
            <Power size={16} />
            Enable Node
          </button>
          <button
            onClick={handleDisableNode}
            disabled={actionLoading || node.state === "maintenance"}
            className="btn-secondary flex items-center gap-2"
          >
            <PowerOff size={16} />
            Disable Node
          </button>
        </div>
      </div>

      {/* Hardware Summary */}
      {summary && (
        <>
          {/* CPU Info */}
          <div className="card mb-4">
            <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
              <Cpu size={20} />
              CPU Information
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label className="text-sm font-medium text-gray-600">Model</label>
                <p className="text-gray-900">{summary.cpu.model}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-600">Vendor</label>
                <p className="text-gray-900">{summary.cpu.vendor}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-600">Architecture</label>
                <p className="text-gray-900">{summary.cpu.arch}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-600">Cores</label>
                <p className="text-gray-900">{summary.cpu.cores}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-600">Threads</label>
                <p className="text-gray-900">{summary.cpu.threads}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-600">Frequency</label>
                <p className="text-gray-900">{summary.cpu.frequency} MHz</p>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-600">Cache Size</label>
                <p className="text-gray-900">{summary.cpu.cache_size} KB</p>
              </div>
              <div className="md:col-span-2">
                <label className="text-sm font-medium text-gray-600">Flags</label>
                <div className="flex flex-wrap gap-1 mt-1">
                  {summary.cpu.flags.slice(0, 10).map((flag) => (
                    <span
                      key={flag}
                      className="px-2 py-0.5 bg-gray-100 text-gray-700 text-xs rounded"
                    >
                      {flag}
                    </span>
                  ))}
                  {summary.cpu.flags.length > 10 && (
                    <span className="px-2 py-0.5 bg-gray-100 text-gray-500 text-xs rounded">
                      +{summary.cpu.flags.length - 10} more
                    </span>
                  )}
                </div>
              </div>
            </div>
          </div>

          {/* Memory Info */}
          <div className="card mb-4">
            <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
              <MemoryStick size={20} />
              Memory Information
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label className="text-sm font-medium text-gray-600">Total Memory</label>
                <p className="text-gray-900">{formatBytes(summary.memory.total)}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-600">Available</label>
                <p className="text-gray-900">{formatBytes(summary.memory.available)}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-600">Used</label>
                <p className="text-gray-900">{formatBytes(summary.memory.used)}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-600">Usage</label>
                <div className="mt-1">
                  <div className="flex items-center gap-2">
                    <div className="flex-1 bg-gray-200 rounded-full h-2">
                      <div
                        className="bg-blue-600 h-2 rounded-full"
                        style={{ width: `${summary.memory.usage_percent}%` }}
                      ></div>
                    </div>
                    <span className="text-sm text-gray-700">
                      {summary.memory.usage_percent.toFixed(1)}%
                    </span>
                  </div>
                </div>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-600">Swap Total</label>
                <p className="text-gray-900">{formatBytes(summary.memory.swap_total)}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-600">Swap Used</label>
                <p className="text-gray-900">{formatBytes(summary.memory.swap_used)}</p>
              </div>
            </div>
          </div>

          {/* Virtualization Info */}
          {summary.virtualization && (
            <div className="card mb-4">
              <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
                <MonitorCheck size={20} />
                Virtualization Features
              </h2>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div className="flex items-center gap-2">
                  <div
                    className={`w-3 h-3 rounded-full ${
                      summary.virtualization.vtx ? "bg-green-500" : "bg-gray-300"
                    }`}
                  ></div>
                  <span className="text-sm">VT-x / AMD-V</span>
                </div>
                <div className="flex items-center gap-2">
                  <div
                    className={`w-3 h-3 rounded-full ${
                      summary.virtualization.ept ? "bg-green-500" : "bg-gray-300"
                    }`}
                  ></div>
                  <span className="text-sm">EPT / NPT</span>
                </div>
                <div className="flex items-center gap-2">
                  <div
                    className={`w-3 h-3 rounded-full ${
                      summary.virtualization.iommu ? "bg-green-500" : "bg-gray-300"
                    }`}
                  ></div>
                  <span className="text-sm">IOMMU</span>
                </div>
                <div className="flex items-center gap-2">
                  <div
                    className={`w-3 h-3 rounded-full ${
                      summary.virtualization.nested_virt ? "bg-green-500" : "bg-gray-300"
                    }`}
                  ></div>
                  <span className="text-sm">Nested Virt</span>
                </div>
              </div>
            </div>
          )}
        </>
      )}

      {/* Quick Links */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <button className="card hover:shadow-md transition-shadow p-4 flex flex-col items-center text-center">
          <Cpu size={24} className="text-blue-600 mb-2" />
          <span className="font-medium">PCI Devices</span>
          <span className="text-xs text-gray-500 mt-1">View PCI devices</span>
        </button>
        <button className="card hover:shadow-md transition-shadow p-4 flex flex-col items-center text-center">
          <Usb size={24} className="text-purple-600 mb-2" />
          <span className="font-medium">USB Devices</span>
          <span className="text-xs text-gray-500 mt-1">View USB devices</span>
        </button>
        <button className="card hover:shadow-md transition-shadow p-4 flex flex-col items-center text-center">
          <Network size={24} className="text-green-600 mb-2" />
          <span className="font-medium">Network</span>
          <span className="text-xs text-gray-500 mt-1">View network interfaces</span>
        </button>
        <button className="card hover:shadow-md transition-shadow p-4 flex flex-col items-center text-center">
          <HardDrive size={24} className="text-orange-600 mb-2" />
          <span className="font-medium">Disks</span>
          <span className="text-xs text-gray-500 mt-1">View physical disks</span>
        </button>
      </div>
    </DashboardLayout>
  );
}
