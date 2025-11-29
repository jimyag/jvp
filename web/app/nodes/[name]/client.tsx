"use client";

import { useState, useEffect, useMemo } from "react";
import { useParams, useRouter, usePathname } from "next/navigation";
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
  Box,
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
  const pathname = usePathname();
  const router = useRouter();
  const rawNodeName = params.name as string;
  const nodeName = useMemo(() => {
    const segments = pathname.split("/").filter(Boolean);
    const urlNode = segments[1];
    const isPlaceholder = rawNodeName === "placeholder-node";
    return isPlaceholder && urlNode ? urlNode : rawNodeName;
  }, [pathname, rawNodeName]);
  const toast = useToast();

  const [node, setNode] = useState<Node | null>(null);
  const [summary, setSummary] = useState<NodeSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [showDevices, setShowDevices] = useState<string | null>(null);
  const [devicesData, setDevicesData] = useState<any>(null);
  const [devicesLoading, setDevicesLoading] = useState(false);

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

  const handleViewDevices = async (deviceType: string) => {
    setShowDevices(deviceType);
    setDevicesLoading(true);

    try {
      let endpoint = "";
      switch (deviceType) {
        case "pci":
          endpoint = "/api/describe-node-pci";
          break;
        case "usb":
          endpoint = "/api/describe-node-usb";
          break;
        case "net":
          endpoint = "/api/describe-node-net";
          break;
        case "disks":
          endpoint = "/api/describe-node-disks";
          break;
        case "gpu":
          endpoint = "/api/describe-node-gpu";
          break;
        case "vms":
          endpoint = "/api/describe-node-vms";
          break;
      }

      const data = await apiPost(endpoint, { name: nodeName });
      setDevicesData(data);
    } catch (error: any) {
      console.error(`Failed to fetch ${deviceType} devices:`, error);
      toast.error(error?.message || `Failed to fetch ${deviceType} devices`);
    } finally {
      setDevicesLoading(false);
    }
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
      <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-6 gap-4">
        <button
          onClick={() => handleViewDevices("pci")}
          className="card hover:shadow-md transition-shadow p-4 flex flex-col items-center text-center"
        >
          <Cpu size={24} className="text-blue-600 mb-2" />
          <span className="font-medium">PCI Devices</span>
          <span className="text-xs text-gray-500 mt-1">View PCI devices</span>
        </button>
        <button
          onClick={() => handleViewDevices("gpu")}
          className="card hover:shadow-md transition-shadow p-4 flex flex-col items-center text-center"
        >
          <MonitorCheck size={24} className="text-indigo-600 mb-2" />
          <span className="font-medium">GPU Devices</span>
          <span className="text-xs text-gray-500 mt-1">View GPU devices</span>
        </button>
        <button
          onClick={() => handleViewDevices("usb")}
          className="card hover:shadow-md transition-shadow p-4 flex flex-col items-center text-center"
        >
          <Usb size={24} className="text-purple-600 mb-2" />
          <span className="font-medium">USB Devices</span>
          <span className="text-xs text-gray-500 mt-1">View USB devices</span>
        </button>
        <button
          onClick={() => handleViewDevices("net")}
          className="card hover:shadow-md transition-shadow p-4 flex flex-col items-center text-center"
        >
          <Network size={24} className="text-green-600 mb-2" />
          <span className="font-medium">Network</span>
          <span className="text-xs text-gray-500 mt-1">View network interfaces</span>
        </button>
        <button
          onClick={() => handleViewDevices("disks")}
          className="card hover:shadow-md transition-shadow p-4 flex flex-col items-center text-center"
        >
          <HardDrive size={24} className="text-orange-600 mb-2" />
          <span className="font-medium">Disks</span>
          <span className="text-xs text-gray-500 mt-1">View physical disks</span>
        </button>
        <button
          onClick={() => handleViewDevices("vms")}
          className="card hover:shadow-md transition-shadow p-4 flex flex-col items-center text-center"
        >
          <Box size={24} className="text-pink-600 mb-2" />
          <span className="font-medium">Virtual Machines</span>
          <span className="text-xs text-gray-500 mt-1">View VMs on node</span>
        </button>
      </div>

      {/* Devices Modal */}
      {showDevices && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full mx-4 max-h-[80vh] overflow-hidden flex flex-col">
            <div className="p-6 border-b flex justify-between items-center">
              <h2 className="text-xl font-semibold">
                {showDevices === "pci" && "PCI Devices"}
                {showDevices === "gpu" && "GPU Devices"}
                {showDevices === "usb" && "USB Devices"}
                {showDevices === "net" && "Network Interfaces"}
                {showDevices === "disks" && "Physical Disks"}
                {showDevices === "vms" && "Virtual Machines"}
              </h2>
              <button
                onClick={() => {
                  setShowDevices(null);
                  setDevicesData(null);
                }}
                className="text-gray-400 hover:text-gray-600"
              >
                ✕
              </button>
            </div>

            <div className="p-6 overflow-y-auto">
              {devicesLoading ? (
                <div className="flex justify-center py-12">
                  <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
                </div>
              ) : (
                <div className="space-y-4">
                  {showDevices === "pci" && devicesData?.devices && (
                    <div>
                      {devicesData.devices.length === 0 ? (
                        <p className="text-gray-500 text-center py-8">No PCI devices found</p>
                      ) : (
                        <div className="space-y-2">
                          {devicesData.devices.map((device: any, idx: number) => (
                            <div key={idx} className="border rounded p-3">
                              <div className="flex justify-between items-start">
                                <div>
                                  <div className="font-mono text-sm font-medium">{device.address}</div>
                                  <div className="text-sm text-gray-700 mt-1">{device.device}</div>
                                  <div className="text-xs text-gray-500 mt-1">{device.vendor}</div>
                                </div>
                                {device.iommu_group >= 0 && (
                                  <span className="px-2 py-1 bg-blue-100 text-blue-800 text-xs rounded">
                                    IOMMU: {device.iommu_group}
                                  </span>
                                )}
                              </div>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  )}

                  {showDevices === "usb" && devicesData?.devices && (
                    <div>
                      {devicesData.devices.length === 0 ? (
                        <p className="text-gray-500 text-center py-8">No USB devices found</p>
                      ) : (
                        <div className="space-y-2">
                          {devicesData.devices.map((device: any, idx: number) => (
                            <div key={idx} className="border rounded p-3">
                              <div className="font-medium">{device.product || "Unknown Device"}</div>
                              <div className="text-sm text-gray-600 mt-1">{device.vendor}</div>
                              <div className="flex gap-4 mt-2 text-xs text-gray-500">
                                {device.vendor_id && <span>Vendor: {device.vendor_id}</span>}
                                {device.product_id && <span>Product: {device.product_id}</span>}
                              </div>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  )}

                  {showDevices === "net" && devicesData?.interfaces && (
                    <div>
                      {devicesData.interfaces.length === 0 ? (
                        <p className="text-gray-500 text-center py-8">No network interfaces found</p>
                      ) : (
                        <div className="space-y-2">
                          {devicesData.interfaces.map((iface: any, idx: number) => (
                            <div key={idx} className="border rounded p-3">
                              <div className="flex justify-between items-start">
                                <div>
                                  <div className="font-medium">{iface.name}</div>
                                  <div className="text-sm text-gray-600 mt-1 font-mono">{iface.mac}</div>
                                </div>
                                <div className="flex gap-2">
                                  {iface.state && (
                                    <span className={`px-2 py-1 text-xs rounded ${
                                      iface.state === 'up' ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-600'
                                    }`}>
                                      {iface.state}
                                    </span>
                                  )}
                                  {iface.speed && (
                                    <span className="px-2 py-1 bg-blue-100 text-blue-800 text-xs rounded">
                                      {iface.speed}
                                    </span>
                                  )}
                                </div>
                              </div>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  )}

                  {showDevices === "disks" && devicesData?.disks && (
                    <div>
                      {devicesData.disks.length === 0 ? (
                        <p className="text-gray-500 text-center py-8">No disks found</p>
                      ) : (
                        <div className="space-y-2">
                          {devicesData.disks.map((disk: any, idx: number) => (
                            <div key={idx} className="border rounded p-3">
                              <div className="flex justify-between items-start">
                                <div className="flex-1">
                                  <div className="font-medium font-mono">{disk.name}</div>
                                  {disk.model && (
                                    <div className="text-sm text-gray-700 mt-1">{disk.model}</div>
                                  )}
                                  <div className="flex gap-4 mt-2 text-xs text-gray-500">
                                    {disk.serial && <span>Serial: {disk.serial}</span>}
                                    {disk.size > 0 && (
                                      <span>Size: {(disk.size / (1024 * 1024 * 1024)).toFixed(2)} GB</span>
                                    )}
                                  </div>
                                </div>
                                <span className={`px-2 py-1 text-xs rounded ${
                                  disk.type === 'NVMe' ? 'bg-purple-100 text-purple-800' :
                                  disk.type === 'SSD' ? 'bg-blue-100 text-blue-800' :
                                  'bg-gray-100 text-gray-600'
                                }`}>
                                  {disk.type}
                                </span>
                              </div>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  )}

                  {showDevices === "gpu" && devicesData?.devices && (
                    <div>
                      {devicesData.devices.length === 0 ? (
                        <p className="text-gray-500 text-center py-8">No GPU devices found</p>
                      ) : (
                        <div className="space-y-2">
                          {devicesData.devices.map((device: any, idx: number) => (
                            <div key={idx} className="border rounded p-3">
                              <div className="flex justify-between items-start">
                                <div className="flex-1">
                                  <div className="font-mono text-sm font-medium">{device.address}</div>
                                  <div className="text-sm text-gray-700 mt-1">{device.device}</div>
                                  <div className="text-xs text-gray-500 mt-1">{device.vendor}</div>
                                  {device.memory > 0 && (
                                    <div className="text-xs text-gray-500 mt-1">
                                      Memory: {(device.memory / (1024 * 1024)).toFixed(0)} MB
                                    </div>
                                  )}
                                </div>
                                <div className="flex flex-col gap-1">
                                  <span className="px-2 py-1 bg-indigo-100 text-indigo-800 text-xs rounded">
                                    GPU
                                  </span>
                                  {device.iommu_group >= 0 && (
                                    <span className="px-2 py-1 bg-blue-100 text-blue-800 text-xs rounded">
                                      IOMMU: {device.iommu_group}
                                    </span>
                                  )}
                                </div>
                              </div>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  )}

                  {showDevices === "vms" && devicesData?.vms && (
                    <div>
                      {devicesData.vms.length === 0 ? (
                        <p className="text-gray-500 text-center py-8">No virtual machines found</p>
                      ) : (
                        <div className="space-y-2">
                          {devicesData.vms.map((vm: any, idx: number) => (
                            <div key={idx} className="border rounded p-3">
                              <div className="flex justify-between items-start">
                                <div className="flex-1">
                                  <div className="font-medium">{vm.name}</div>
                                  <div className="text-xs text-gray-500 mt-1 font-mono">
                                    UUID: {vm.uuid}
                                  </div>
                                  {vm.cpus > 0 && (
                                    <div className="text-xs text-gray-600 mt-2">
                                      vCPUs: {vm.cpus}
                                    </div>
                                  )}
                                  {vm.memory > 0 && (
                                    <div className="text-xs text-gray-600">
                                      Memory: {(vm.memory / 1024 / 1024).toFixed(2)} GB
                                    </div>
                                  )}
                                </div>
                                <span className={`px-2 py-1 text-xs rounded ${
                                  vm.state === 'running' ? 'bg-green-100 text-green-800' :
                                  vm.state === 'paused' ? 'bg-yellow-100 text-yellow-800' :
                                  vm.state === 'shutoff' ? 'bg-gray-100 text-gray-600' :
                                  'bg-red-100 text-red-800'
                                }`}>
                                  {vm.state}
                                </span>
                              </div>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </DashboardLayout>
  );
}

export async function generateStaticParams() {
  return [];
}
