"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import {
  Database,
  RefreshCw,
  Plus,
  Play,
  Square,
  ExternalLink,
} from "lucide-react";
import { apiPost } from "@/lib/api";
import { useToast } from "@/components/ToastContainer";
import DashboardLayout from "@/components/DashboardLayout";
import Header from "@/components/Header";

interface StoragePool {
  name: string;
  uuid: string;
  state: string;
  type: string;
  capacity: number;
  allocation: number;
  available: number;
  path: string;
  volume_count: number;
}

interface ListStoragePoolsResponse {
  pools: StoragePool[];
}

interface Node {
  name: string;
  uuid: string;
  uri: string;
  type: string;
  state: string;
}

export default function StoragePoolsPage() {
  const router = useRouter();
  const [pools, setPools] = useState<StoragePool[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  // Node selection
  const [nodes, setNodes] = useState<Node[]>([]);
  const [selectedNode, setSelectedNode] = useState<string>("");
  const [loadingNodes, setLoadingNodes] = useState(true);

  // Create pool modal
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [createForm, setCreateForm] = useState({
    name: "",
    type: "dir",
    path: "",
  });
  const [creating, setCreating] = useState(false);

  const toast = useToast();

  useEffect(() => {
    fetchNodes();
  }, []);

  useEffect(() => {
    if (selectedNode !== null && !loadingNodes) {
      fetchPools();
    }
  }, [selectedNode]);

  const fetchNodes = async () => {
    setLoadingNodes(true);
    try {
      const response = await apiPost<{ nodes: Node[] }>("/api/list-nodes", {});
      const nodeList = response.nodes || [];
      setNodes(nodeList);
      // Auto-select first node if available
      if (nodeList.length > 0 && !selectedNode) {
        setSelectedNode(nodeList[0].name);
      }
    } catch (error: any) {
      console.error("Failed to fetch nodes:", error);
      toast.error(error?.message || "Failed to fetch nodes");
    } finally {
      setLoadingNodes(false);
    }
  };

  const fetchPools = async () => {
    if (!selectedNode) {
      return;
    }

    setRefreshing(true);
    try {
      const response = await apiPost<ListStoragePoolsResponse>(
        "/api/list-storage-pools",
        { node_name: selectedNode }
      );
      setPools(response.pools || []);
    } catch (error: any) {
      console.error("Failed to fetch storage pools:", error);
      toast.error(error?.message || "Failed to fetch storage pools");
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  const handleCreatePool = async () => {
    if (!createForm.name || !createForm.path) {
      toast.error("Please fill in all required fields");
      return;
    }

    setCreating(true);
    try {
      await apiPost("/api/create-storage-pool", {
        node_name: selectedNode,
        name: createForm.name,
        type: createForm.type,
        path: createForm.path,
      });
      toast.success(`Storage pool ${createForm.name} created successfully`);
      setShowCreateModal(false);
      setCreateForm({ name: "", type: "dir", path: "" });
      await fetchPools();
    } catch (error: any) {
      console.error("Failed to create storage pool:", error);
      toast.error(error?.message || "Failed to create storage pool");
    } finally {
      setCreating(false);
    }
  };

  const handleStartPool = async (poolName: string) => {
    try {
      await apiPost("/api/start-storage-pool", {
        node_name: selectedNode,
        pool_name: poolName,
      });
      toast.success(`Storage pool ${poolName} started successfully`);
      await fetchPools();
    } catch (error: any) {
      console.error("Failed to start storage pool:", error);
      toast.error(error?.message || "Failed to start storage pool");
    }
  };

  const handleStopPool = async (poolName: string) => {
    if (!confirm(`Are you sure you want to stop storage pool "${poolName}"?`)) {
      return;
    }

    try {
      await apiPost("/api/stop-storage-pool", {
        node_name: selectedNode,
        pool_name: poolName,
      });
      toast.success(`Storage pool ${poolName} stopped successfully`);
      await fetchPools();
    } catch (error: any) {
      console.error("Failed to stop storage pool:", error);
      toast.error(error?.message || "Failed to stop storage pool");
    }
  };

  const handleRefreshPool = async (poolName: string) => {
    try {
      await apiPost("/api/refresh-storage-pool", {
        node_name: selectedNode,
        pool_name: poolName,
      });
      toast.success(`Storage pool ${poolName} refreshed successfully`);
      await fetchPools();
    } catch (error: any) {
      console.error("Failed to refresh storage pool:", error);
      toast.error(error?.message || "Failed to refresh storage pool");
    }
  };

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
  };

  const getStateColor = (state: string): string => {
    switch (state.toLowerCase()) {
      case "active":
      case "running":
        return "text-green-600 bg-green-100";
      case "inactive":
        return "text-gray-600 bg-gray-100";
      case "building":
        return "text-yellow-600 bg-yellow-100";
      case "degraded":
        return "text-orange-600 bg-orange-100";
      case "inaccessible":
        return "text-red-600 bg-red-100";
      default:
        return "text-gray-600 bg-gray-100";
    }
  };

  if (loadingNodes || loading) {
    return (
      <DashboardLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <RefreshCw className="w-8 h-8 animate-spin text-primary mx-auto mb-2" />
            <p className="text-gray-600">
              {loadingNodes ? "Loading nodes..." : "Loading storage pools..."}
            </p>
          </div>
        </div>
      </DashboardLayout>
    );
  }

  // Show message if no nodes available
  if (nodes.length === 0) {
    return (
      <DashboardLayout>
        <div className="space-y-6">
          <Header
            title="Storage Pools"
            description="Manage libvirt storage pools and volumes"
          />
          <div className="text-center py-12 bg-gray-50 rounded-lg border border-gray-200">
            <Database className="w-12 h-12 text-gray-400 mx-auto mb-4" />
            <p className="text-gray-600 mb-2">No nodes available</p>
            <p className="text-sm text-gray-500">
              Please create a node first to manage storage pools
            </p>
          </div>
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout>
      <div className="space-y-6">
        {/* Header */}
        <Header
          title="Storage Pools"
          description="Manage libvirt storage pools and volumes"
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
                onClick={fetchPools}
                disabled={refreshing}
                className="flex items-center gap-2 px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50"
              >
                <RefreshCw className={`w-4 h-4 ${refreshing ? "animate-spin" : ""}`} />
                Refresh
              </button>
              <button
                onClick={() => setShowCreateModal(true)}
                className="flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90"
              >
                <Plus className="w-4 h-4" />
                Create Pool
              </button>
            </div>
          }
        />

        {/* Storage Pools List */}
        {pools.length === 0 ? (
          <div className="text-center py-12 bg-gray-50 rounded-lg border border-gray-200">
            <Database className="w-12 h-12 text-gray-400 mx-auto mb-4" />
            <p className="text-gray-600">No storage pools found</p>
            <button
              onClick={() => setShowCreateModal(true)}
              className="mt-4 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90"
            >
              Create Your First Pool
            </button>
          </div>
        ) : (
          <div className="space-y-4">
            {pools.map((pool) => {
              const usagePercent =
                pool.capacity > 0
                  ? ((pool.allocation / pool.capacity) * 100).toFixed(1)
                  : "0";

              return (
                <div
                  key={pool.name}
                  className="bg-white border border-gray-200 rounded-lg overflow-hidden"
                >
                  {/* Pool Header */}
                  <div className="p-4">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3 flex-1">
                        <Database className="w-8 h-8 text-primary" />
                        <div className="flex-1">
                          <div className="flex items-center gap-3">
                            <button
                              onClick={() =>
                                router.push(
                                  `/storage-pools/${encodeURIComponent(
                                    pool.name
                                  )}?node=${selectedNode}`
                                )
                              }
                              className="text-lg font-semibold text-primary hover:underline"
                            >
                              {pool.name}
                            </button>
                            <span
                              className={`px-2 py-1 text-xs font-medium rounded-full ${getStateColor(
                                pool.state
                              )}`}
                            >
                              {pool.state}
                            </span>
                            {pool.type && (
                              <span className="px-2 py-1 text-xs font-medium text-blue-600 bg-blue-100 rounded-full">
                                {pool.type}
                              </span>
                            )}
                          </div>
                          <p className="text-sm text-gray-600 mt-1">{pool.path}</p>
                        </div>
                      </div>
                      <div className="flex items-center gap-4">
                        <div className="text-right">
                          <div className="text-sm font-medium text-gray-900">
                            {formatBytes(pool.allocation)} / {formatBytes(pool.capacity)}
                          </div>
                          <div className="text-xs text-gray-600">
                            {usagePercent}% used · {formatBytes(pool.available)} free
                          </div>
                          <div className="text-xs text-gray-500 mt-0.5">
                            {pool.volume_count} volumes
                          </div>
                        </div>

                        {/* Action Buttons */}
                        <div className="flex items-center gap-2">
                          {pool.state.toLowerCase() === "inactive" ? (
                            <button
                              onClick={() => handleStartPool(pool.name)}
                              className="flex items-center gap-1 px-3 py-1.5 text-sm text-green-700 bg-green-50 hover:bg-green-100 rounded-lg"
                              title="Start"
                            >
                              <Play className="w-4 h-4" />
                              Start
                            </button>
                          ) : (
                            <button
                              onClick={() => handleStopPool(pool.name)}
                              className="flex items-center gap-1 px-3 py-1.5 text-sm text-gray-700 bg-gray-50 hover:bg-gray-100 rounded-lg"
                              title="Stop"
                            >
                              <Square className="w-4 h-4" />
                              Stop
                            </button>
                          )}
                          <button
                            onClick={() => handleRefreshPool(pool.name)}
                            className="flex items-center gap-1 px-3 py-1.5 text-sm text-gray-700 bg-gray-50 hover:bg-gray-100 rounded-lg"
                            title="Refresh"
                          >
                            <RefreshCw className="w-4 h-4" />
                          </button>
                          <button
                            onClick={() =>
                              router.push(
                                `/storage-pools/${encodeURIComponent(
                                  pool.name
                                )}?node=${selectedNode}`
                              )
                            }
                            className="flex items-center gap-1 px-3 py-1.5 text-sm text-primary bg-blue-50 hover:bg-blue-100 rounded-lg"
                            title="View Details"
                          >
                            <ExternalLink className="w-4 h-4" />
                            Details
                          </button>
                        </div>
                      </div>
                    </div>

                    {/* Progress Bar */}
                    <div className="mt-3 w-full bg-gray-200 rounded-full h-2">
                      <div
                        className="bg-primary h-2 rounded-full transition-all"
                        style={{ width: `${usagePercent}%` }}
                      />
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Create Pool Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h2 className="text-xl font-semibold mb-4">Create Storage Pool</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Pool Name *
                </label>
                <input
                  type="text"
                  value={createForm.name}
                  onChange={(e) =>
                    setCreateForm({ ...createForm, name: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
                  placeholder="e.g., default, images"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Type
                </label>
                <select
                  value={createForm.type}
                  onChange={(e) =>
                    setCreateForm({ ...createForm, type: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
                >
                  <option value="dir">Directory (dir)</option>
                  <option value="fs">Filesystem (fs)</option>
                  <option value="netfs">Network Filesystem (netfs)</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Path *
                </label>
                <input
                  type="text"
                  value={createForm.path}
                  onChange={(e) =>
                    setCreateForm({ ...createForm, path: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary"
                  placeholder="/var/lib/libvirt/images"
                />
              </div>
            </div>
            <div className="flex gap-2 mt-6">
              <button
                onClick={() => {
                  setShowCreateModal(false);
                  setCreateForm({ name: "", type: "dir", path: "" });
                }}
                className="flex-1 px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={handleCreatePool}
                disabled={creating}
                className="flex-1 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50"
              >
                {creating ? "Creating..." : "Create"}
              </button>
            </div>
          </div>
        </div>
      )}
    </DashboardLayout>
  );
}
