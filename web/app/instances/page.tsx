"use client";

import { useState, useEffect } from "react";
import DashboardLayout from "@/components/DashboardLayout";
import Header from "@/components/Header";
import Table from "@/components/Table";
import StatusBadge from "@/components/StatusBadge";
import Modal from "@/components/Modal";
import { Play, Square, RefreshCw, Trash2, Plus } from "lucide-react";

interface Instance {
  id: string;
  name: string;
  state: string;
  vcpus: number;
  memory_mb: number;
  image_id?: string;
  volume_id?: string;
  created_at: string;
  domain_uuid?: string;
  domain_name?: string;
}

export default function InstancesPage() {
  const [instances, setInstances] = useState<Instance[]>([]);
  const [loading, setLoading] = useState(true);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [formData, setFormData] = useState({
    image_id: "",
    size_gb: 20,
    memory_mb: 2048,
    vcpus: 2,
    keypair_ids: [] as string[],
  });

  const fetchInstances = async () => {
    setLoading(true);
    try {
      const response = await fetch("/api/instances/describe", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });
      if (response.ok) {
        const data = await response.json();
        setInstances(data.instances || []);
      }
    } catch (error) {
      console.error("Failed to fetch instances:", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchInstances();
  }, []);

  const handleCreateInstance = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const response = await fetch("/api/instances/run", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(formData),
      });

      if (response.ok) {
        setIsCreateModalOpen(false);
        fetchInstances();
        setFormData({
          image_id: "",
          size_gb: 20,
          memory_mb: 2048,
          vcpus: 2,
          keypair_ids: [],
        });
      }
    } catch (error) {
      console.error("Failed to create instance:", error);
    }
  };

  const handleAction = async (instanceId: string, action: string) => {
    try {
      await fetch(`/api/instances/${action}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ instance_ids: [instanceId] }),
      });
      fetchInstances();
    } catch (error) {
      console.error(`Failed to ${action} instance:`, error);
    }
  };

  const handleDelete = async (instanceId: string) => {
    if (!confirm("Are you sure you want to delete this instance?")) return;

    try {
      await fetch("/api/instances/terminate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ instance_ids: [instanceId] }),
      });
      fetchInstances();
    } catch (error) {
      console.error("Failed to delete instance:", error);
    }
  };

  const columns = [
    {
      key: "id",
      label: "ID",
      render: (value: unknown) => (
        <a href={`/instances/${value}`} className="text-accent hover:underline font-mono text-xs">
          {String(value).substring(0, 12)}...
        </a>
      ),
    },
    {
      key: "name",
      label: "Name",
      render: (_: unknown, row: any) => (
        <a href={`/instances/${row.id}`} className="text-primary hover:text-accent font-medium">
          {row.name || row.domain_name || "N/A"}
        </a>
      ),
    },
    {
      key: "state",
      label: "Status",
      render: (value: unknown) => <StatusBadge status={String(value)} />,
    },
    {
      key: "vcpus",
      label: "vCPUs",
      render: (value: unknown) => <span>{String(value)} cores</span>,
    },
    {
      key: "memory_mb",
      label: "Memory",
      render: (value: unknown) => <span>{(Number(value) / 1024).toFixed(1)} GB</span>,
    },
    {
      key: "image_id",
      label: "Image",
      render: (value: unknown) => <span>{value ? String(value).substring(0, 12) + "..." : "N/A"}</span>
    },
    {
      key: "actions",
      label: "Actions",
      render: (_: unknown, row: Record<string, unknown>) => {
        const instance = row as unknown as Instance;
        return (
          <div className="flex gap-2">
            {instance.state === "running" ? (
              <button
                onClick={() => handleAction(instance.id, "stop")}
                className="p-2 text-gray-600 hover:text-red-600 transition-colors"
                title="Stop"
              >
                <Square size={16} />
              </button>
            ) : (
              <button
                onClick={() => handleAction(instance.id, "start")}
                className="p-2 text-gray-600 hover:text-green-600 transition-colors"
                title="Start"
              >
                <Play size={16} />
              </button>
            )}
            <button
              onClick={() => handleAction(instance.id, "restart")}
              className="p-2 text-gray-600 hover:text-blue-600 transition-colors"
              title="Restart"
            >
              <RefreshCw size={16} />
            </button>
            <button
              onClick={() => handleDelete(instance.id)}
              className="p-2 text-gray-600 hover:text-red-600 transition-colors"
              title="Delete"
            >
              <Trash2 size={16} />
            </button>
          </div>
        );
      },
    },
  ];

  return (
    <DashboardLayout>
      <Header
        title="Instances"
        description="Manage your virtual machine instances"
        action={
          <button
            onClick={() => setIsCreateModalOpen(true)}
            className="btn-primary flex items-center gap-2"
          >
            <Plus size={16} />
            Create Instance
          </button>
        }
        onRefresh={fetchInstances}
      />

      {loading ? (
        <div className="card text-center py-12">
          <p className="text-gray-500">Loading instances...</p>
        </div>
      ) : (
        <Table
          columns={columns}
          data={instances}
          emptyMessage="No instances found. Create your first instance to get started."
        />
      )}

      <Modal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        title="Create New Instance"
        maxWidth="lg"
      >
        <form onSubmit={handleCreateInstance} className="space-y-6">
          <div>
            <label className="label">Image ID</label>
            <input
              type="text"
              className="input"
              value={formData.image_id}
              onChange={(e) =>
                setFormData({ ...formData, image_id: e.target.value })
              }
              placeholder="ubuntu-jammy (留空使用默认)"
            />
            <p className="text-xs text-gray-500 mt-1">留空将使用默认 ubuntu-jammy 镜像</p>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="label">vCPUs</label>
              <input
                type="number"
                className="input"
                value={formData.vcpus}
                onChange={(e) =>
                  setFormData({ ...formData, vcpus: Number(e.target.value) })
                }
                min="1"
                max="32"
              />
            </div>

            <div>
              <label className="label">Memory (MB)</label>
              <input
                type="number"
                className="input"
                value={formData.memory_mb}
                onChange={(e) =>
                  setFormData({ ...formData, memory_mb: Number(e.target.value) })
                }
                min="512"
                step="512"
              />
              <p className="text-xs text-gray-500 mt-1">
                {(formData.memory_mb / 1024).toFixed(1)} GB
              </p>
            </div>
          </div>

          <div>
            <label className="label">Disk Size (GB)</label>
            <input
              type="number"
              className="input"
              value={formData.size_gb}
              onChange={(e) =>
                setFormData({ ...formData, size_gb: Number(e.target.value) })
              }
              min="10"
            />
          </div>

          <div>
            <label className="label">Key Pair IDs (Optional)</label>
            <input
              type="text"
              className="input"
              placeholder="kp-xxx,kp-yyy (逗号分隔多个)"
              onChange={(e) => {
                const ids = e.target.value.split(',').map(s => s.trim()).filter(Boolean);
                setFormData({ ...formData, keypair_ids: ids });
              }}
            />
            <p className="text-xs text-gray-500 mt-1">输入密钥对 ID,多个用逗号分隔</p>
          </div>

          <div className="flex justify-end gap-3 pt-4">
            <button
              type="button"
              onClick={() => setIsCreateModalOpen(false)}
              className="btn-secondary"
            >
              Cancel
            </button>
            <button type="submit" className="btn-primary">
              Create Instance
            </button>
          </div>
        </form>
      </Modal>
    </DashboardLayout>
  );
}
