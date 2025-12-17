import { useEffect, useState } from "react";
import {
  Network,
  RefreshCw,
  Plus,
  Play,
  Square,
  Trash2,
} from "lucide-react";
import { apiPost } from "@/lib/api";
import { useToast } from "@/components/ToastContainer";
import Header from "@/components/Header";

// Network types
interface LibvirtNetwork {
  name: string;
  uuid: string;
  node_name: string;
  type: string;
  mode: string;
  bridge: string;
  state: string;
  autostart: boolean;
  persistent: boolean;
  ip_address: string;
  netmask: string;
  dhcp_start: string;
  dhcp_end: string;
}

interface HostBridge {
  name: string;
  state: string;
  mac: string;
  ips: string[];
  interfaces: string[];
  stp: boolean;
  mtu: number;
}

interface Node {
  name: string;
  uuid: string;
  uri: string;
  type: string;
  state: string;
}

interface AvailableInterface {
  name: string;
  mac: string;
  state: string;
  bound_to?: string; // 绑定到的网桥（空表示未绑定）
}

type TabType = "networks" | "bridges";

export default function NetworksPage() {
  // Networks state
  const [networks, setNetworks] = useState<LibvirtNetwork[]>([]);
  const [bridges, setBridges] = useState<HostBridge[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [activeTab, setActiveTab] = useState<TabType>("networks");

  // Node selection
  const [nodes, setNodes] = useState<Node[]>([]);
  const [selectedNode, setSelectedNode] = useState<string>("");
  const [loadingNodes, setLoadingNodes] = useState(true);

  // Create network modal
  const [showCreateNetworkModal, setShowCreateNetworkModal] = useState(false);
  const [createNetworkForm, setCreateNetworkForm] = useState({
    name: "",
    mode: "nat",
    ip_address: "192.168.100.1",
    netmask: "255.255.255.0",
    dhcp_start: "192.168.100.100",
    dhcp_end: "192.168.100.200",
    autostart: true,
  });
  const [creatingNetwork, setCreatingNetwork] = useState(false);

  // Create bridge modal
  const [showCreateBridgeModal, setShowCreateBridgeModal] = useState(false);
  const [createBridgeForm, setCreateBridgeForm] = useState({
    bridge_name: "",
    stp: false,
    interfaces: [] as string[],
  });
  const [creatingBridge, setCreatingBridge] = useState(false);
  const [availableInterfaces, setAvailableInterfaces] = useState<AvailableInterface[]>([]);
  const [loadingInterfaces, setLoadingInterfaces] = useState(false);

  const toast = useToast();

  useEffect(() => {
    fetchNodes();
  }, []);

  useEffect(() => {
    if (selectedNode && !loadingNodes) {
      fetchData();
    }
  }, [selectedNode, activeTab]);

  const fetchNodes = async () => {
    setLoadingNodes(true);
    try {
      const response = await apiPost<{ nodes: Node[] }>("/api/list-nodes", {});
      const nodeList = response.nodes || [];
      setNodes(nodeList);

      if (nodeList.length > 0 && (!selectedNode || !nodeList.some((n) => n.name === selectedNode))) {
        setSelectedNode(nodeList[0].name);
      }
    } catch (error: any) {
      console.error("Failed to fetch nodes:", error);
      toast.error(error?.message || "Failed to fetch nodes");
    } finally {
      setLoadingNodes(false);
    }
  };

  const fetchData = async () => {
    if (!selectedNode) return;

    setRefreshing(true);
    try {
      if (activeTab === "networks") {
        const response = await apiPost<{ networks: LibvirtNetwork[] }>(
          "/api/list-networks",
          { node_name: selectedNode }
        );
        setNetworks(response.networks || []);
      } else {
        const response = await apiPost<{ bridges: HostBridge[] }>(
          "/api/list-bridges",
          { node_name: selectedNode }
        );
        setBridges(response.bridges || []);
      }
    } catch (error: any) {
      console.error("Failed to fetch data:", error);
      toast.error(error?.message || "Failed to fetch data");
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  // Network actions
  const handleCreateNetwork = async () => {
    if (!createNetworkForm.name) {
      toast.error("Please enter a network name");
      return;
    }

    setCreatingNetwork(true);
    try {
      await apiPost("/api/create-network", {
        node_name: selectedNode,
        ...createNetworkForm,
      });
      toast.success(`Network ${createNetworkForm.name} created successfully`);
      setShowCreateNetworkModal(false);
      setCreateNetworkForm({
        name: "",
        mode: "nat",
        ip_address: "192.168.100.1",
        netmask: "255.255.255.0",
        dhcp_start: "192.168.100.100",
        dhcp_end: "192.168.100.200",
        autostart: true,
      });
      await fetchData();
    } catch (error: any) {
      console.error("Failed to create network:", error);
      toast.error(error?.message || "Failed to create network");
    } finally {
      setCreatingNetwork(false);
    }
  };

  const handleStartNetwork = async (networkName: string) => {
    try {
      await apiPost("/api/start-network", {
        node_name: selectedNode,
        network_name: networkName,
      });
      toast.success(`Network ${networkName} started successfully`);
      await fetchData();
    } catch (error: any) {
      console.error("Failed to start network:", error);
      toast.error(error?.message || "Failed to start network");
    }
  };

  const handleStopNetwork = async (networkName: string) => {
    if (!confirm(`Are you sure you want to stop network "${networkName}"?`)) {
      return;
    }

    try {
      await apiPost("/api/stop-network", {
        node_name: selectedNode,
        network_name: networkName,
      });
      toast.success(`Network ${networkName} stopped successfully`);
      await fetchData();
    } catch (error: any) {
      console.error("Failed to stop network:", error);
      toast.error(error?.message || "Failed to stop network");
    }
  };

  const handleDeleteNetwork = async (networkName: string) => {
    if (!confirm(`Are you sure you want to delete network "${networkName}"? This action cannot be undone.`)) {
      return;
    }

    try {
      await apiPost("/api/delete-network", {
        node_name: selectedNode,
        network_name: networkName,
      });
      toast.success(`Network ${networkName} deleted successfully`);
      await fetchData();
    } catch (error: any) {
      console.error("Failed to delete network:", error);
      toast.error(error?.message || "Failed to delete network");
    }
  };

  // Bridge actions
  const fetchAvailableInterfaces = async () => {
    if (!selectedNode) return;
    setLoadingInterfaces(true);
    try {
      const response = await apiPost<{ interfaces: AvailableInterface[] }>(
        "/api/list-available-interfaces",
        { node_name: selectedNode }
      );
      setAvailableInterfaces(response.interfaces || []);
    } catch (error: any) {
      console.error("Failed to fetch available interfaces:", error);
    } finally {
      setLoadingInterfaces(false);
    }
  };

  const handleOpenCreateBridgeModal = () => {
    setShowCreateBridgeModal(true);
    fetchAvailableInterfaces();
  };

  const handleCreateBridge = async () => {
    if (!createBridgeForm.bridge_name) {
      toast.error("Please enter a bridge name");
      return;
    }

    setCreatingBridge(true);
    try {
      await apiPost("/api/create-bridge", {
        node_name: selectedNode,
        ...createBridgeForm,
      });
      toast.success(`Bridge ${createBridgeForm.bridge_name} created successfully`);
      setShowCreateBridgeModal(false);
      setCreateBridgeForm({ bridge_name: "", stp: false, interfaces: [] });
      await fetchData();
    } catch (error: any) {
      console.error("Failed to create bridge:", error);
      toast.error(error?.message || "Failed to create bridge");
    } finally {
      setCreatingBridge(false);
    }
  };

  const handleDeleteBridge = async (bridgeName: string) => {
    if (!confirm(`Are you sure you want to delete bridge "${bridgeName}"? This action cannot be undone.`)) {
      return;
    }

    try {
      await apiPost("/api/delete-bridge", {
        node_name: selectedNode,
        bridge_name: bridgeName,
      });
      toast.success(`Bridge ${bridgeName} deleted successfully`);
      await fetchData();
    } catch (error: any) {
      console.error("Failed to delete bridge:", error);
      toast.error(error?.message || "Failed to delete bridge");
    }
  };

  const getStateColor = (state: string): string => {
    switch (state.toLowerCase()) {
      case "active":
      case "up":
        return "text-green-600 bg-green-100";
      case "inactive":
      case "down":
        return "text-gray-600 bg-gray-100";
      default:
        return "text-gray-600 bg-gray-100";
    }
  };

  const getModeColor = (mode: string): string => {
    switch (mode.toLowerCase()) {
      case "nat":
        return "text-blue-600 bg-blue-100";
      case "bridge":
        return "text-purple-600 bg-purple-100";
      case "isolated":
        return "text-orange-600 bg-orange-100";
      default:
        return "text-gray-600 bg-gray-100";
    }
  };

  if (loadingNodes || loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <RefreshCw className="w-8 h-8 animate-spin text-primary mx-auto mb-2" />
          <p className="text-gray-600">
            {loadingNodes ? "Loading nodes..." : "Loading networks..."}
          </p>
        </div>
      </div>
    );
  }

  if (nodes.length === 0) {
    return (
      <div className="space-y-6">
        <Header
          title="Networks"
          description="Manage libvirt networks and host bridges"
        />
        <div className="text-center py-12 bg-gray-50 rounded-lg border border-gray-200">
          <Network className="w-12 h-12 text-gray-400 mx-auto mb-4" />
          <p className="text-gray-600 mb-2">No nodes available</p>
          <p className="text-sm text-gray-500">
            Please create a node first to manage networks
          </p>
        </div>
      </div>
    );
  }

  return (
    <>
      <div className="space-y-6">
        {/* Header */}
        <Header
          title="Networks"
          description="Manage libvirt networks and host bridges"
          action={
            <div className="flex gap-2">
              {/* Node Selector */}
              <select
                value={selectedNode}
                onChange={(e) => setSelectedNode(e.target.value)}
                className="px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary bg-white"
              >
                {nodes.map((node) => (
                  <option key={node.name} value={node.name}>
                    {node.name} ({node.type})
                  </option>
                ))}
              </select>

              <button
                onClick={fetchData}
                disabled={refreshing}
                className="flex items-center gap-2 px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50"
              >
                <RefreshCw className={`w-4 h-4 ${refreshing ? "animate-spin" : ""}`} />
                Refresh
              </button>
              <button
                onClick={() => activeTab === "networks" ? setShowCreateNetworkModal(true) : handleOpenCreateBridgeModal()}
                className="flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90"
              >
                <Plus className="w-4 h-4" />
                {activeTab === "networks" ? "Create Network" : "Create Bridge"}
              </button>
            </div>
          }
        />

        {/* Tabs */}
        <div className="border-b border-gray-200">
          <nav className="-mb-px flex space-x-8">
            <button
              onClick={() => setActiveTab("networks")}
              className={`py-2 px-1 border-b-2 font-medium text-sm ${
                activeTab === "networks"
                  ? "border-primary text-primary"
                  : "border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300"
              }`}
            >
              Libvirt Networks
            </button>
            <button
              onClick={() => setActiveTab("bridges")}
              className={`py-2 px-1 border-b-2 font-medium text-sm ${
                activeTab === "bridges"
                  ? "border-primary text-primary"
                  : "border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300"
              }`}
            >
              Host Bridges
            </button>
          </nav>
        </div>

        {/* Content */}
        {activeTab === "networks" ? (
          // Libvirt Networks List
          networks.length === 0 ? (
            <div className="text-center py-12 bg-gray-50 rounded-lg border border-gray-200">
              <Network className="w-12 h-12 text-gray-400 mx-auto mb-4" />
              <p className="text-gray-600">No libvirt networks found</p>
              <button
                onClick={() => setShowCreateNetworkModal(true)}
                className="mt-4 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90"
              >
                Create Your First Network
              </button>
            </div>
          ) : (
            <div className="space-y-4">
              {networks.map((network) => (
                <div
                  key={network.name}
                  className="bg-white border border-gray-200 rounded-lg overflow-hidden"
                >
                  <div className="p-4">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3 flex-1">
                        <Network className="w-8 h-8 text-primary" />
                        <div className="flex-1">
                          <div className="flex items-center gap-3">
                            <span className="text-lg font-semibold">
                              {network.name}
                            </span>
                            <span
                              className={`px-2 py-1 text-xs font-medium rounded-full ${getStateColor(
                                network.state
                              )}`}
                            >
                              {network.state}
                            </span>
                            <span
                              className={`px-2 py-1 text-xs font-medium rounded-full ${getModeColor(
                                network.mode
                              )}`}
                            >
                              {network.mode}
                            </span>
                            {network.autostart && (
                              <span className="px-2 py-1 text-xs font-medium text-green-600 bg-green-100 rounded-full">
                                Autostart
                              </span>
                            )}
                          </div>
                          <div className="text-sm text-gray-600 mt-1 space-x-4">
                            {network.bridge && (
                              <span>Bridge: {network.bridge}</span>
                            )}
                            {network.ip_address && (
                              <span>
                                IP: {network.ip_address}/{network.netmask ?
                                  (network.netmask === "255.255.255.0" ? "24" :
                                   network.netmask === "255.255.0.0" ? "16" : network.netmask)
                                  : ""}
                              </span>
                            )}
                            {network.dhcp_start && network.dhcp_end && (
                              <span>
                                DHCP: {network.dhcp_start} - {network.dhcp_end}
                              </span>
                            )}
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        {network.state.toLowerCase() === "inactive" ? (
                          <button
                            onClick={() => handleStartNetwork(network.name)}
                            className="flex items-center gap-1 px-3 py-1.5 text-sm text-green-700 bg-green-50 hover:bg-green-100 rounded-lg"
                            title="Start"
                          >
                            <Play className="w-4 h-4" />
                            Start
                          </button>
                        ) : (
                          <button
                            onClick={() => handleStopNetwork(network.name)}
                            className="flex items-center gap-1 px-3 py-1.5 text-sm text-gray-700 bg-gray-50 hover:bg-gray-100 rounded-lg"
                            title="Stop"
                          >
                            <Square className="w-4 h-4" />
                            Stop
                          </button>
                        )}
                        <button
                          onClick={() => handleDeleteNetwork(network.name)}
                          className="flex items-center gap-1 px-3 py-1.5 text-sm text-red-700 bg-red-50 hover:bg-red-100 rounded-lg"
                          title="Delete"
                        >
                          <Trash2 className="w-4 h-4" />
                          Delete
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )
        ) : (
          // Host Bridges List
          bridges.length === 0 ? (
            <div className="text-center py-12 bg-gray-50 rounded-lg border border-gray-200">
              <Network className="w-12 h-12 text-gray-400 mx-auto mb-4" />
              <p className="text-gray-600">No host bridges found</p>
              <button
                onClick={handleOpenCreateBridgeModal}
                className="mt-4 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90"
              >
                Create Your First Bridge
              </button>
            </div>
          ) : (
            <div className="space-y-4">
              {bridges.map((bridge) => (
                <div
                  key={bridge.name}
                  className="bg-white border border-gray-200 rounded-lg overflow-hidden"
                >
                  <div className="p-4">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3 flex-1">
                        <Network className="w-8 h-8 text-purple-600" />
                        <div className="flex-1">
                          <div className="flex items-center gap-3">
                            <span className="text-lg font-semibold">
                              {bridge.name}
                            </span>
                            <span
                              className={`px-2 py-1 text-xs font-medium rounded-full ${getStateColor(
                                bridge.state
                              )}`}
                            >
                              {bridge.state}
                            </span>
                            {bridge.stp && (
                              <span className="px-2 py-1 text-xs font-medium text-blue-600 bg-blue-100 rounded-full">
                                STP
                              </span>
                            )}
                          </div>
                          <div className="text-sm text-gray-600 mt-1">
                            <div className="flex flex-wrap gap-x-4 gap-y-1">
                              {bridge.mac && (
                                <span>MAC: {bridge.mac}</span>
                              )}
                              {bridge.ips && bridge.ips.length > 0 && (
                                <span>IPs: {bridge.ips.join(", ")}</span>
                              )}
                              {bridge.mtu > 0 && (
                                <span>MTU: {bridge.mtu}</span>
                              )}
                            </div>
                            {bridge.interfaces && bridge.interfaces.length > 0 && (
                              <div className="mt-2">
                                <span className="text-gray-500">Interfaces: </span>
                                <div className="flex flex-wrap gap-1 mt-1">
                                  {bridge.interfaces.map((iface) => {
                                    const isVeth = iface.startsWith("veth") || iface.startsWith("vnet");
                                    const isPhysical = iface.startsWith("eth") || iface.startsWith("en") || iface.startsWith("eno");
                                    return (
                                      <span
                                        key={iface}
                                        className={`px-2 py-0.5 text-xs rounded ${
                                          isPhysical
                                            ? "bg-blue-100 text-blue-700"
                                            : isVeth
                                            ? "bg-gray-100 text-gray-600"
                                            : "bg-purple-100 text-purple-700"
                                        }`}
                                      >
                                        {iface}
                                      </span>
                                    );
                                  })}
                                </div>
                              </div>
                            )}
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => handleDeleteBridge(bridge.name)}
                          className="flex items-center gap-1 px-3 py-1.5 text-sm text-red-700 bg-red-50 hover:bg-red-100 rounded-lg"
                          title="Delete"
                        >
                          <Trash2 className="w-4 h-4" />
                          Delete
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )
        )}
      </div>

      {/* Create Network Modal */}
      {showCreateNetworkModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md max-h-[90vh] overflow-y-auto">
            <h2 className="text-xl font-semibold mb-4">Create Network</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Network Name *
                </label>
                <input
                  type="text"
                  value={createNetworkForm.name}
                  onChange={(e) =>
                    setCreateNetworkForm({ ...createNetworkForm, name: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
                  placeholder="e.g., mynetwork"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Mode
                </label>
                <select
                  value={createNetworkForm.mode}
                  onChange={(e) =>
                    setCreateNetworkForm({ ...createNetworkForm, mode: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
                >
                  <option value="nat">NAT (Network Address Translation)</option>
                  <option value="isolated">Isolated (No external access)</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  IP Address (Gateway)
                </label>
                <input
                  type="text"
                  value={createNetworkForm.ip_address}
                  onChange={(e) =>
                    setCreateNetworkForm({ ...createNetworkForm, ip_address: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
                  placeholder="192.168.100.1"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Netmask
                </label>
                <select
                  value={createNetworkForm.netmask}
                  onChange={(e) =>
                    setCreateNetworkForm({ ...createNetworkForm, netmask: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
                >
                  <option value="255.255.255.0">255.255.255.0 (/24 - 254 hosts)</option>
                  <option value="255.255.0.0">255.255.0.0 (/16 - 65534 hosts)</option>
                  <option value="255.255.255.128">255.255.255.128 (/25 - 126 hosts)</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  DHCP Start
                </label>
                <input
                  type="text"
                  value={createNetworkForm.dhcp_start}
                  onChange={(e) =>
                    setCreateNetworkForm({ ...createNetworkForm, dhcp_start: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
                  placeholder="192.168.100.100"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  DHCP End
                </label>
                <input
                  type="text"
                  value={createNetworkForm.dhcp_end}
                  onChange={(e) =>
                    setCreateNetworkForm({ ...createNetworkForm, dhcp_end: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
                  placeholder="192.168.100.200"
                />
              </div>
              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="autostart"
                  checked={createNetworkForm.autostart}
                  onChange={(e) =>
                    setCreateNetworkForm({ ...createNetworkForm, autostart: e.target.checked })
                  }
                  className="h-4 w-4 text-primary focus:ring-primary border-gray-300 rounded"
                />
                <label htmlFor="autostart" className="ml-2 block text-sm text-gray-700">
                  Start network automatically on boot
                </label>
              </div>
            </div>
            <div className="flex gap-2 mt-6">
              <button
                onClick={() => {
                  setShowCreateNetworkModal(false);
                  setCreateNetworkForm({
                    name: "",
                    mode: "nat",
                    ip_address: "192.168.100.1",
                    netmask: "255.255.255.0",
                    dhcp_start: "192.168.100.100",
                    dhcp_end: "192.168.100.200",
                    autostart: true,
                  });
                }}
                className="flex-1 px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={handleCreateNetwork}
                disabled={creatingNetwork}
                className="flex-1 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50"
              >
                {creatingNetwork ? "Creating..." : "Create"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Create Bridge Modal */}
      {showCreateBridgeModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md max-h-[90vh] overflow-y-auto">
            <h2 className="text-xl font-semibold mb-4">Create Bridge</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Bridge Name *
                </label>
                <input
                  type="text"
                  value={createBridgeForm.bridge_name}
                  onChange={(e) =>
                    setCreateBridgeForm({ ...createBridgeForm, bridge_name: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
                  placeholder="e.g., br0"
                />
              </div>

              {/* Network Interfaces Selection */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Bind Network Interfaces (Optional)
                </label>
                {loadingInterfaces ? (
                  <div className="text-sm text-gray-500 py-2">Loading interfaces...</div>
                ) : availableInterfaces.length === 0 ? (
                  <div className="text-sm text-gray-500 py-2">No available interfaces found</div>
                ) : (
                  <div className="border border-gray-300 rounded-lg p-2 max-h-40 overflow-y-auto">
                    {availableInterfaces.map((iface) => {
                      const isBound = !!iface.bound_to;
                      return (
                        <label
                          key={iface.name}
                          className={`flex items-center gap-2 p-2 rounded ${
                            isBound ? "opacity-50 cursor-not-allowed" : "hover:bg-gray-50 cursor-pointer"
                          }`}
                        >
                          <input
                            type="checkbox"
                            checked={createBridgeForm.interfaces.includes(iface.name)}
                            disabled={isBound}
                            onChange={(e) => {
                              if (e.target.checked) {
                                setCreateBridgeForm({
                                  ...createBridgeForm,
                                  interfaces: [...createBridgeForm.interfaces, iface.name],
                                });
                              } else {
                                setCreateBridgeForm({
                                  ...createBridgeForm,
                                  interfaces: createBridgeForm.interfaces.filter((i) => i !== iface.name),
                                });
                              }
                            }}
                            className="h-4 w-4 text-primary focus:ring-primary border-gray-300 rounded"
                          />
                          <span className="flex-1 text-sm">
                            <span className="font-medium">{iface.name}</span>
                            {iface.mac && (
                              <span className="text-gray-500 ml-2 text-xs">{iface.mac}</span>
                            )}
                            {isBound && (
                              <span className="text-orange-600 ml-2 text-xs">(bound to {iface.bound_to})</span>
                            )}
                          </span>
                          <span
                            className={`px-2 py-0.5 text-xs rounded ${
                              iface.state === "up"
                                ? "bg-green-100 text-green-700"
                                : "bg-gray-100 text-gray-600"
                            }`}
                          >
                            {iface.state}
                          </span>
                        </label>
                      );
                    })}
                  </div>
                )}
                <p className="text-xs text-gray-500 mt-1">
                  Select physical network interfaces to bind to this bridge
                </p>
              </div>

              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="stp"
                  checked={createBridgeForm.stp}
                  onChange={(e) =>
                    setCreateBridgeForm({ ...createBridgeForm, stp: e.target.checked })
                  }
                  className="h-4 w-4 text-primary focus:ring-primary border-gray-300 rounded"
                />
                <label htmlFor="stp" className="ml-2 block text-sm text-gray-700">
                  Enable Spanning Tree Protocol (STP)
                </label>
              </div>
            </div>
            <div className="flex gap-2 mt-6">
              <button
                onClick={() => {
                  setShowCreateBridgeModal(false);
                  setCreateBridgeForm({ bridge_name: "", stp: false, interfaces: [] });
                }}
                className="flex-1 px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={handleCreateBridge}
                disabled={creatingBridge}
                className="flex-1 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50"
              >
                {creatingBridge ? "Creating..." : "Create"}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
