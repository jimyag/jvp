"use client";

import { useState, useEffect } from "react";
import DashboardLayout from "@/components/DashboardLayout";
import Header from "@/components/Header";
import Table from "@/components/Table";
import StatusBadge from "@/components/StatusBadge";
import Modal from "@/components/Modal";
import ConfirmDialog from "@/components/ConfirmDialog";
import { useToast } from "@/components/ToastContainer";
import { Plus, Trash2, Link as LinkIcon, Unlink } from "lucide-react";
import { apiPost } from "@/lib/api";

interface Volume {
  volumeID: string;
  sizeGB: number;
  snapshotID?: string;
  availabilityZone?: string;
  state: string;
  volumeType?: string;
  iops?: number;
  encrypted?: boolean;
  attachments?: any[];
  createTime: string;
  tags?: any[];
}

export default function VolumesPage() {
  const toast = useToast();
  const [volumes, setVolumes] = useState<Volume[]>([]);
  const [loading, setLoading] = useState(true);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isAttachModalOpen, setIsAttachModalOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [selectedVolume, setSelectedVolume] = useState<string>("");
  const [volumeToDelete, setVolumeToDelete] = useState<string>("");
  const [formData, setFormData] = useState({
    sizeGB: 10,
    snapshotID: "",
    volumeType: "gp2",
  });
  const [attachData, setAttachData] = useState({
    instance_id: "",
  });

  const fetchVolumes = async () => {
    setLoading(true);
    try {
      const data = await apiPost<{ volumes: Volume[] }>("/api/volumes/describe", {});
      setVolumes(data.volumes || []);
    } catch (error) {
      console.error("Failed to fetch volumes:", error);
      toast.error("Failed to load volumes. Please check if backend is running.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchVolumes();
  }, []);

  const handleCreateVolume = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const response = await fetch("/api/volumes/create", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(formData),
      });

      if (response.ok) {
        setIsCreateModalOpen(false);
        fetchVolumes();
        setFormData({ sizeGB: 10, snapshotID: "", volumeType: "gp2" });
        toast.success("Volume created successfully!");
      } else {
        const error = await response.json();
        toast.error(`Failed to create volume: ${error.message || "Unknown error"}`);
      }
    } catch (error) {
      console.error("Failed to create volume:", error);
      toast.error("Failed to create volume. Please try again.");
    }
  };

  const handleAttachVolume = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const response = await fetch("/api/volumes/attach", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          volumeID: selectedVolume,
          instanceID: attachData.instance_id,
        }),
      });

      if (response.ok) {
        setIsAttachModalOpen(false);
        fetchVolumes();
        setAttachData({ instance_id: "" });
        toast.success("Volume attached successfully!");
      } else {
        const error = await response.json();
        toast.error(`Failed to attach volume: ${error.message || "Unknown error"}`);
      }
    } catch (error) {
      console.error("Failed to attach volume:", error);
      toast.error("Failed to attach volume. Please try again.");
    }
  };

  const handleDetachVolume = async (volumeId: string) => {
    try {
      const response = await fetch("/api/volumes/detach", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ volumeID: volumeId }),
      });

      if (response.ok) {
        fetchVolumes();
        toast.success("Volume detached successfully!");
      } else {
        const error = await response.json();
        toast.error(`Failed to detach volume: ${error.message || "Unknown error"}`);
      }
    } catch (error) {
      console.error("Failed to detach volume:", error);
      toast.error("Failed to detach volume. Please try again.");
    }
  };

  const handleDeleteClick = (volumeId: string) => {
    setVolumeToDelete(volumeId);
    setIsDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = async () => {
    try {
      const response = await fetch("/api/volumes/delete", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ volumeID: volumeToDelete }),
      });

      if (response.ok) {
        fetchVolumes();
        toast.success("Volume deleted successfully!");
      } else {
        const error = await response.json();
        toast.error(`Failed to delete volume: ${error.message || "Unknown error"}`);
      }
    } catch (error) {
      console.error("Failed to delete volume:", error);
      toast.error("Failed to delete volume. Please try again.");
    }
  };

  const columns = [
    {
      key: "volumeID",
      label: "Volume ID",
      render: (value: unknown) => (
        <span className="font-mono text-xs">
          {String(value).substring(0, 12)}...
        </span>
      ),
    },
    {
      key: "state",
      label: "Status",
      render: (value: unknown) => <StatusBadge status={String(value)} />,
    },
    {
      key: "sizeGB",
      label: "Size",
      render: (value: unknown) => <span>{String(value)} GB</span>,
    },
    {
      key: "volumeType",
      label: "Type",
      render: (value: unknown) => <span>{String(value || "gp2")}</span>,
    },
    {
      key: "attachments",
      label: "Attached To",
      render: (value: unknown) => {
        const attachments = value as any[];
        if (!attachments || attachments.length === 0) return <span>-</span>;
        const instanceId = attachments[0]?.instanceID;
        return instanceId ? <span className="font-mono text-xs">{instanceId.substring(0, 12)}...</span> : <span>-</span>;
      },
    },
    {
      key: "actions",
      label: "Actions",
      render: (_: unknown, row: any) => {
        const volume = row as Volume;
        const isAttached = volume.attachments && volume.attachments.length > 0;
        return (
          <div className="flex gap-2">
            {isAttached ? (
              <button
                onClick={() => handleDetachVolume(volume.volumeID)}
                className="p-2 text-gray-600 hover:text-orange-600 transition-colors"
                title="Detach"
              >
                <Unlink size={16} />
              </button>
            ) : (
              <button
                onClick={() => {
                  setSelectedVolume(volume.volumeID);
                  setIsAttachModalOpen(true);
                }}
                className="p-2 text-gray-600 hover:text-blue-600 transition-colors"
                title="Attach"
              >
                <LinkIcon size={16} />
              </button>
            )}
            <button
              onClick={() => handleDeleteClick(volume.volumeID)}
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
        title="Volumes"
        description="Manage your block storage volumes"
        action={
          <button
            onClick={() => setIsCreateModalOpen(true)}
            className="btn-primary flex items-center gap-2"
          >
            <Plus size={16} />
            Create Volume
          </button>
        }
        onRefresh={fetchVolumes}
      />

      {loading ? (
        <div className="card text-center py-12">
          <p className="text-gray-500">Loading volumes...</p>
        </div>
      ) : (
        <Table
          columns={columns}
          data={volumes}
          emptyMessage="No volumes found. Create your first volume to get started."
        />
      )}

      <Modal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        title="Create New Volume"
      >
        <form onSubmit={handleCreateVolume} className="space-y-6">
          <div>
            <label className="label">Size (GB)</label>
            <input
              type="number"
              className="input"
              value={formData.sizeGB}
              onChange={(e) =>
                setFormData({ ...formData, sizeGB: Number(e.target.value) })
              }
              min="1"
              required
            />
          </div>

          <div>
            <label className="label">Volume Type</label>
            <select
              className="input"
              value={formData.volumeType}
              onChange={(e) =>
                setFormData({ ...formData, volumeType: e.target.value })
              }
            >
              <option value="gp2">GP2 (General Purpose SSD)</option>
              <option value="gp3">GP3 (General Purpose SSD)</option>
              <option value="io1">IO1 (Provisioned IOPS SSD)</option>
              <option value="standard">Standard (Magnetic)</option>
            </select>
          </div>

          <div>
            <label className="label">Snapshot ID (Optional)</label>
            <input
              type="text"
              className="input"
              value={formData.snapshotID}
              onChange={(e) =>
                setFormData({ ...formData, snapshotID: e.target.value })
              }
              placeholder="snap-xxx (留空创建空卷)"
            />
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
              Create Volume
            </button>
          </div>
        </form>
      </Modal>

      <Modal
        isOpen={isAttachModalOpen}
        onClose={() => setIsAttachModalOpen(false)}
        title="Attach Volume"
      >
        <form onSubmit={handleAttachVolume} className="space-y-6">
          <div>
            <label className="label">Instance ID</label>
            <input
              type="text"
              className="input"
              value={attachData.instance_id}
              onChange={(e) =>
                setAttachData({ instance_id: e.target.value })
              }
              required
              placeholder="Enter instance ID"
            />
          </div>

          <div className="flex justify-end gap-3 pt-4">
            <button
              type="button"
              onClick={() => setIsAttachModalOpen(false)}
              className="btn-secondary"
            >
              Cancel
            </button>
            <button type="submit" className="btn-primary">
              Attach Volume
            </button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        isOpen={isDeleteDialogOpen}
        onClose={() => setIsDeleteDialogOpen(false)}
        onConfirm={handleDeleteConfirm}
        title="Delete Volume"
        message="Are you sure you want to delete this volume? This action cannot be undone and all data will be permanently lost."
        confirmText="Delete"
        cancelText="Cancel"
        variant="danger"
      />
    </DashboardLayout>
  );
}
