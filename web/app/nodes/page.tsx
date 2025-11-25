"use client";

import { useState, useEffect } from "react";
import DashboardLayout from "@/components/DashboardLayout";
import Header from "@/components/Header";
import Table from "@/components/Table";
import StatusBadge from "@/components/StatusBadge";
import { useToast } from "@/components/ToastContainer";
import { apiPost } from "@/lib/api";
import { RefreshCw, Server, Info } from "lucide-react";
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
        <button
          onClick={() => handleViewDetails(node.name)}
          className="btn-secondary flex items-center gap-2"
        >
          <Info size={16} />
          Details
        </button>
      ),
    },
  ];

  return (
    <DashboardLayout>
      <Header
        title="Nodes"
        description="Manage physical and virtual nodes in the cluster"
        action={
          <button
            onClick={fetchNodes}
            disabled={refreshing}
            className="btn-primary flex items-center gap-2"
          >
            <RefreshCw size={16} className={refreshing ? "animate-spin" : ""} />
            Refresh
          </button>
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
    </DashboardLayout>
  );
}
