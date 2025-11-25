"use client";

import { useState, useEffect } from "react";
import DashboardLayout from "@/components/DashboardLayout";
import Header from "@/components/Header";
import Table from "@/components/Table";
import StatusBadge from "@/components/StatusBadge";
import { useToast } from "@/components/ToastContainer";
import { apiPost } from "@/lib/api";
import { RefreshCw, Server, Info, Plus, Trash2 } from "lucide-react";
import { useRouter } from "next/navigation";

interface Node {
  name: string;
  uuid: string;
  uri: string;
  type: string;
  state: string;
  created_at?: string;
  updated_at?: string;
}

export default function NodesPage() {
  const [nodes, setNodes] = useState<Node[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [createForm, setCreateForm] = useState({
    name: "",
    uri: "",
    type: "remote" as string,
  });
  const [creating, setCreating] = useState(false);
  const toast = useToast();
  const router = useRouter();

  useEffect(() => {
    fetchNodes();
  }, []);

  const fetchNodes = async () => {
    setRefreshing(true);
    try {
      const response = await apiPost<{ nodes: Node[] }>("/api/list-nodes", {});
      setNodes(response.nodes || []);
    } catch (error: any) {
      console.error("Failed to fetch nodes:", error);
      toast.error(error?.message || "Failed to fetch nodes");
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  const handleViewDetails = (nodeName: string) => {
    router.push(`/nodes/${nodeName}`);
  };

  const handleCreateNode = async () => {
    if (!createForm.name || !createForm.uri) {
      toast.error("Please fill in all required fields");
      return;
    }

    setCreating(true);
    try {
      await apiPost("/api/create-node", {
        name: createForm.name,
        uri: createForm.uri,
        type: createForm.type,
      });
      toast.success(`Node ${createForm.name} created successfully`);
      setShowCreateModal(false);
      setCreateForm({ name: "", uri: "", type: "remote" });
      await fetchNodes();
    } catch (error: any) {
      console.error("Failed to create node:", error);
      toast.error(error?.message || "Failed to create node");
    } finally {
      setCreating(false);
    }
  };

  const handleDeleteNode = async (nodeName: string) => {
    if (!confirm(`Are you sure you want to delete node "${nodeName}"?`)) {
      return;
    }

    try {
      await apiPost("/api/delete-node", { name: nodeName });
      toast.success(`Node ${nodeName} deleted successfully`);
      await fetchNodes();
    } catch (error: any) {
      console.error("Failed to delete node:", error);
      toast.error(error?.message || "Failed to delete node");
    }
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

  const getTypeColor = (type: string) => {
    switch (type) {
      case "local":
        return "blue";
      case "remote":
        return "purple";
      case "compute":
        return "green";
      case "storage":
        return "orange";
      case "hybrid":
        return "indigo";
      default:
        return "gray";
    }
  };

  const columns = [
    {
      key: "name",
      label: "Name",
      render: (_: unknown, node: Node) => (
        <div className="flex items-center gap-2">
          <Server size={16} className="text-gray-400" />
          <span className="font-medium">{node.name}</span>
        </div>
      ),
    },
    {
      key: "type",
      label: "Type",
      render: (_: unknown, node: Node) => (
        <StatusBadge
          status={node.type}
          color={getTypeColor(node.type)}
          text={node.type}
        />
      ),
    },
    {
      key: "state",
      label: "State",
      render: (_: unknown, node: Node) => (
        <StatusBadge
          status={node.state}
          color={getStateColor(node.state)}
          text={node.state}
        />
      ),
    },
    {
      key: "uri",
      label: "URI",
      render: (_: unknown, node: Node) => (
        <span className="font-mono text-sm text-gray-600">{node.uri}</span>
      ),
    },
    {
      key: "actions",
      label: "Actions",
      render: (_: unknown, node: Node) => (
        <div className="flex gap-2">
          <button
            onClick={() => handleViewDetails(node.name)}
            className="btn-secondary flex items-center gap-2"
          >
            <Info size={16} />
            Details
          </button>
          <button
            onClick={() => handleDeleteNode(node.name)}
            className="btn-danger flex items-center gap-2"
          >
            <Trash2 size={16} />
            Delete
          </button>
        </div>
      ),
    },
  ];

  return (
    <DashboardLayout>
      <Header
        title="Nodes"
        description="Manage physical and virtual nodes in the cluster"
        action={
          <div className="flex gap-2">
            <button
              onClick={() => setShowCreateModal(true)}
              className="btn-primary flex items-center gap-2"
            >
              <Plus size={16} />
              Add Node
            </button>
            <button
              onClick={fetchNodes}
              disabled={refreshing}
              className="btn-secondary flex items-center gap-2"
            >
              <RefreshCw size={16} className={refreshing ? "animate-spin" : ""} />
              Refresh
            </button>
          </div>
        }
      />

      <div className="card">
        {loading ? (
          <div className="flex justify-center py-12">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
          </div>
        ) : nodes.length === 0 ? (
          <div className="text-center py-12">
            <Server size={48} className="mx-auto text-gray-400 mb-4" />
            <p className="text-gray-500 mb-4">No nodes found</p>
          </div>
        ) : (
          <Table
            data={nodes}
            columns={columns}
            keyField="uuid"
          />
        )}
      </div>

      {/* Info Card */}
      <div className="card mt-4 bg-blue-50 border-blue-200">
        <h3 className="text-sm font-semibold text-blue-900 mb-2">Node Information</h3>
        <ul className="text-sm text-blue-800 space-y-1">
          <li>• <strong>Local:</strong> The node running this JVP instance</li>
          <li>• <strong>Remote:</strong> A node accessible via network (SSH/libvirt)</li>
          <li>• <strong>Compute:</strong> Specialized for running VMs</li>
          <li>• <strong>Storage:</strong> Specialized for storage operations</li>
          <li>• <strong>Hybrid:</strong> Can perform both compute and storage tasks</li>
        </ul>
      </div>

      {/* Create Node Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4">
            <div className="p-6">
              <h2 className="text-xl font-semibold mb-4">Add New Node</h2>

              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Node Name <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="text"
                    value={createForm.name}
                    onChange={(e) =>
                      setCreateForm({ ...createForm, name: e.target.value })
                    }
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="e.g., node1, server1"
                  />
                  <p className="text-xs text-gray-500 mt-1">
                    Unique identifier for this node
                  </p>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Libvirt URI <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="text"
                    value={createForm.uri}
                    onChange={(e) =>
                      setCreateForm({ ...createForm, uri: e.target.value })
                    }
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono text-sm"
                    placeholder="qemu+ssh://root@192.168.1.100/system"
                  />
                  <p className="text-xs text-gray-500 mt-1">
                    Connection URI for libvirt
                  </p>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Node Type
                  </label>
                  <select
                    value={createForm.type}
                    onChange={(e) =>
                      setCreateForm({ ...createForm, type: e.target.value })
                    }
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="local">Local</option>
                    <option value="remote">Remote</option>
                    <option value="compute">Compute</option>
                    <option value="storage">Storage</option>
                    <option value="hybrid">Hybrid</option>
                  </select>
                </div>
              </div>

              <div className="flex gap-2 mt-6">
                <button
                  onClick={handleCreateNode}
                  disabled={creating}
                  className="flex-1 btn-primary"
                >
                  {creating ? "Creating..." : "Create Node"}
                </button>
                <button
                  onClick={() => {
                    setShowCreateModal(false);
                    setCreateForm({ name: "", uri: "", type: "remote" });
                  }}
                  disabled={creating}
                  className="flex-1 btn-secondary"
                >
                  Cancel
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </DashboardLayout>
  );
}
