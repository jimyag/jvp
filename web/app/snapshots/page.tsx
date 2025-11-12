"use client";

import { useState, useEffect } from "react";
import DashboardLayout from "@/components/DashboardLayout";
import Header from "@/components/Header";
import Table from "@/components/Table";
import StatusBadge from "@/components/StatusBadge";
import Modal from "@/components/Modal";
import ConfirmDialog from "@/components/ConfirmDialog";
import { useToast } from "@/components/ToastContainer";
import { Plus, Trash2, Copy } from "lucide-react";
import { apiPost } from "@/lib/api";

interface Snapshot {
  snapshotID: string;
  volumeID: string;
  state: string;
  startTime: string;
  progress: string;
  ownerID: string;
  description: string;
  encrypted: boolean;
  volumeSizeGB: number;
  tags?: any[];
}

export default function SnapshotsPage() {
  const toast = useToast();
  const [snapshots, setSnapshots] = useState<Snapshot[]>([]);
  const [loading, setLoading] = useState(true);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isCopyModalOpen, setIsCopyModalOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [selectedSnapshot, setSelectedSnapshot] = useState<string>("");
  const [snapshotToDelete, setSnapshotToDelete] = useState<string>("");
  const [formData, setFormData] = useState({
    volumeID: "",
    description: "",
  });
  const [copyData, setCopyData] = useState({
    description: "",
  });

  const fetchSnapshots = async () => {
    setLoading(true);
    try {
      const data = await apiPost<{ snapshots: Snapshot[] }>("/api/snapshots/describe", {});
      setSnapshots(data.snapshots || []);
    } catch (error) {
      console.error("Failed to fetch snapshots:", error);
      toast.error("Failed to load snapshots. Please check if backend is running.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchSnapshots();
  }, []);

  const handleCreateSnapshot = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const response = await fetch("/api/snapshots/create", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(formData),
      });

      if (response.ok) {
        setIsCreateModalOpen(false);
        fetchSnapshots();
        setFormData({ volumeID: "", description: "" });
        toast.success("Snapshot created successfully!");
      } else {
        const error = await response.json();
        toast.error(`Failed to create snapshot: ${error.message || "Unknown error"}`);
      }
    } catch (error) {
      console.error("Failed to create snapshot:", error);
      toast.error("Failed to create snapshot. Please try again.");
    }
  };

  const handleCopySnapshot = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const response = await fetch("/api/snapshots/copy", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          sourceSnapshotID: selectedSnapshot,
          description: copyData.description,
        }),
      });

      if (response.ok) {
        setIsCopyModalOpen(false);
        fetchSnapshots();
        setCopyData({ description: "" });
        toast.success("Snapshot copied successfully!");
      } else {
        const error = await response.json();
        toast.error(`Failed to copy snapshot: ${error.message || "Unknown error"}`);
      }
    } catch (error) {
      console.error("Failed to copy snapshot:", error);
      toast.error("Failed to copy snapshot. Please try again.");
    }
  };

  const handleDeleteClick = (snapshotId: string) => {
    setSnapshotToDelete(snapshotId);
    setIsDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = async () => {
    try {
      const response = await fetch("/api/snapshots/delete", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ snapshotID: snapshotToDelete }),
      });

      if (response.ok) {
        fetchSnapshots();
        toast.success("Snapshot deleted successfully!");
      } else {
        const error = await response.json();
        toast.error(`Failed to delete snapshot: ${error.message || "Unknown error"}`);
      }
    } catch (error) {
      console.error("Failed to delete snapshot:", error);
      toast.error("Failed to delete snapshot. Please try again.");
    }
  };

  const columns = [
    {
      key: "snapshotID",
      label: "Snapshot ID",
      render: (value: unknown) => (
        <span className="font-mono text-xs">
          {String(value).substring(0, 16)}...
        </span>
      ),
    },
    {
      key: "volumeID",
      label: "Volume ID",
      render: (value: unknown) => (
        <span className="font-mono text-xs">
          {String(value).substring(0, 16)}...
        </span>
      ),
    },
    {
      key: "state",
      label: "Status",
      render: (value: unknown) => <StatusBadge status={String(value)} />,
    },
    {
      key: "progress",
      label: "Progress",
      render: (value: unknown) => <span>{String(value)}</span>,
    },
    {
      key: "volumeSizeGB",
      label: "Size",
      render: (value: unknown) => <span>{String(value)} GB</span>,
    },
    {
      key: "description",
      label: "Description",
      render: (value: unknown) => (
        <span className="text-sm text-gray-600">
          {String(value) || "-"}
        </span>
      ),
    },
    {
      key: "startTime",
      label: "Created",
      render: (value: unknown) => {
        const date = new Date(String(value));
        return (
          <span className="text-sm text-gray-600">
            {date.toLocaleDateString()} {date.toLocaleTimeString()}
          </span>
        );
      },
    },
    {
      key: "actions",
      label: "Actions",
      render: (_: unknown, row: any) => {
        const snapshot = row as Snapshot;
        return (
          <div className="flex gap-2">
            <button
              onClick={() => {
                setSelectedSnapshot(snapshot.snapshotID);
                setIsCopyModalOpen(true);
              }}
              className="p-2 text-gray-600 hover:text-blue-600 transition-colors"
              title="Copy Snapshot"
            >
              <Copy size={16} />
            </button>
            <button
              onClick={() => handleDeleteClick(snapshot.snapshotID)}
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
        title="Snapshots"
        description="Manage your EBS snapshots"
        action={
          <button
            onClick={() => setIsCreateModalOpen(true)}
            className="btn-primary flex items-center gap-2"
          >
            <Plus size={16} />
            Create Snapshot
          </button>
        }
        onRefresh={fetchSnapshots}
      />

      {loading ? (
        <div className="card text-center py-12">
          <p className="text-gray-500">Loading snapshots...</p>
        </div>
      ) : (
        <Table
          columns={columns}
          data={snapshots}
          emptyMessage="No snapshots found. Create your first snapshot to backup your volumes."
        />
      )}

      <Modal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        title="Create New Snapshot"
      >
        <form onSubmit={handleCreateSnapshot} className="space-y-6">
          <div>
            <label className="label">Volume ID *</label>
            <input
              type="text"
              className="input"
              value={formData.volumeID}
              onChange={(e) =>
                setFormData({ ...formData, volumeID: e.target.value })
              }
              required
              placeholder="vol-xxx"
            />
            <p className="text-xs text-gray-500 mt-1">
              The ID of the volume to create a snapshot from
            </p>
          </div>

          <div>
            <label className="label">Description (Optional)</label>
            <textarea
              className="input"
              value={formData.description}
              onChange={(e) =>
                setFormData({ ...formData, description: e.target.value })
              }
              rows={3}
              placeholder="Enter a description for this snapshot"
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
              Create Snapshot
            </button>
          </div>
        </form>
      </Modal>

      <Modal
        isOpen={isCopyModalOpen}
        onClose={() => setIsCopyModalOpen(false)}
        title="Copy Snapshot"
      >
        <form onSubmit={handleCopySnapshot} className="space-y-6">
          <div>
            <label className="label">Source Snapshot ID</label>
            <input
              type="text"
              className="input"
              value={selectedSnapshot}
              disabled
            />
          </div>

          <div>
            <label className="label">Description (Optional)</label>
            <textarea
              className="input"
              value={copyData.description}
              onChange={(e) =>
                setCopyData({ description: e.target.value })
              }
              rows={3}
              placeholder="Enter a description for the copied snapshot"
            />
          </div>

          <div className="flex justify-end gap-3 pt-4">
            <button
              type="button"
              onClick={() => setIsCopyModalOpen(false)}
              className="btn-secondary"
            >
              Cancel
            </button>
            <button type="submit" className="btn-primary">
              Copy Snapshot
            </button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        isOpen={isDeleteDialogOpen}
        onClose={() => setIsDeleteDialogOpen(false)}
        onConfirm={handleDeleteConfirm}
        title="Delete Snapshot"
        message="Are you sure you want to delete this snapshot? This action cannot be undone and all data will be permanently lost."
        confirmText="Delete"
        cancelText="Cancel"
        variant="danger"
      />
    </DashboardLayout>
  );
}
