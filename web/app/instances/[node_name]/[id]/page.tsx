"use client";

import { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import DashboardLayout from "@/components/DashboardLayout";
import Header from "@/components/Header";
import StatusBadge from "@/components/StatusBadge";
import ConfirmDialog from "@/components/ConfirmDialog";
import { useToast } from "@/components/ToastContainer";
import { Play, Square, RefreshCw, Trash2, ArrowLeft, Key, Edit, Monitor } from "lucide-react";
import Modal from "@/components/Modal";

interface Instance {
  id: string;
  name: string;
  state: string;
  node_name: string;
  vcpus: number;
  memory_mb: number;
  autostart?: boolean;
  template_id?: string;
  volume_id?: string;
  ip_address?: string;
  keypair_name?: string;
  created_at: string;
  updated_at?: string;
  domain_uuid?: string;
  domain_name?: string;
}

export default function InstanceDetailPage() {
  const toast = useToast();
  const params = useParams();
  const router = useRouter();
  const nodeName = params.node_name as string;
  const instanceId = params.id as string;

  const [instance, setInstance] = useState<Instance | null>(null);
  const [loading, setLoading] = useState(true);
  const [isResetPasswordModalOpen, setIsResetPasswordModalOpen] = useState(false);
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [deleteVolumes, setDeleteVolumes] = useState(false);
  const [newPassword, setNewPassword] = useState("");
  const [username, setUsername] = useState("root");
  const [editFormData, setEditFormData] = useState({
    name: "",
    vcpus: 2,
    memory_mb: 2048,
    autostart: false,
  });

  const fetchInstance = async () => {
    setLoading(true);
    try {
      const response = await fetch("/api/instances/describe", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          node_name: nodeName,
          instance_ids: [instanceId],
        }),
      });
      if (response.ok) {
        const data = await response.json();
        if (data.instances && data.instances.length > 0) {
          setInstance(data.instances[0]);
        }
      } else {
        toast.error("Failed to load instance details");
      }
    } catch (error) {
      console.error("Failed to fetch instance:", error);
      toast.error("Failed to load instance details");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchInstance();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [nodeName, instanceId]);

  const handleAction = async (action: string) => {
    try {
      const response = await fetch(`/api/instances/${action}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          node_name: nodeName,
          instance_ids: [instanceId],
        }),
      });

      if (response.ok) {
        const actionName = action === "start" ? "started" : action === "stop" ? "stopped" : "rebooted";
        toast.success(`Instance ${actionName} successfully!`);
        setTimeout(() => {
          fetchInstance();
        }, 2000);
      } else {
        const error = await response.json();
        toast.error(`Failed to ${action} instance: ${error.message || "Unknown error"}`);
      }
    } catch (error) {
      console.error(`Failed to ${action} instance:`, error);
      toast.error(`Failed to ${action} instance`);
    }
  };

  const handleDeleteClick = () => {
    setDeleteVolumes(false);
    setIsDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = async () => {
    try {
      const response = await fetch("/api/instances/terminate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          node_name: nodeName,
          instance_ids: [instanceId],
          delete_volumes: deleteVolumes,
        }),
      });

      if (response.ok) {
        toast.success("Instance terminated successfully!");
        router.push("/instances");
      } else {
        const error = await response.json();
        toast.error(`Failed to terminate instance: ${error.message || "Unknown error"}`);
      }
    } catch (error) {
      console.error("Failed to delete instance:", error);
      toast.error("Failed to delete instance");
    }
  };

  const handleResetPassword = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const response = await fetch("/api/instances/reset-password", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          node_name: nodeName,
          instance_id: instanceId,
          users: [{ username, new_password: newPassword }],
          auto_start: true,
        }),
      });

      if (response.ok) {
        setIsResetPasswordModalOpen(false);
        setNewPassword("");
        setUsername("root");
        toast.success("Password reset successfully!");
      } else {
        const error = await response.json();
        toast.error(`Failed to reset password: ${error.message || "Unknown error"}`);
      }
    } catch (error) {
      console.error("Failed to reset password:", error);
      toast.error("Failed to reset password");
    }
  };

  const handleEditClick = () => {
    if (instance) {
      setEditFormData({
        name: instance.name || "",
        vcpus: instance.vcpus || 2,
        memory_mb: instance.memory_mb || 2048,
        autostart: instance.autostart ?? false,
      });
      setIsEditModalOpen(true);
    }
  };

  const handleEditInstance = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const response = await fetch("/api/instances/modify-attribute", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          node_name: nodeName,
          instance_id: instanceId,
          name: editFormData.name,
          vcpus: editFormData.vcpus,
          memory_mb: editFormData.memory_mb,
          autostart: editFormData.autostart,
          live: instance?.state === "running",
        }),
      });

      if (response.ok) {
        setIsEditModalOpen(false);
        fetchInstance();
        toast.success("Instance attributes updated successfully!");
      } else {
        const error = await response.json();
        toast.error(`Failed to update instance: ${error.message || "Unknown error"}`);
      }
    } catch (error) {
      console.error("Failed to update instance:", error);
      toast.error("Failed to update instance attributes");
    }
  };

  if (loading) {
    return (
      <DashboardLayout>
        <div className="card text-center py-12">
          <p className="text-gray-500">Loading instance details...</p>
        </div>
      </DashboardLayout>
    );
  }

  if (!instance) {
    return (
      <DashboardLayout>
        <div className="card text-center py-12">
          <p className="text-gray-500">Instance not found</p>
          <button onClick={() => router.push("/instances")} className="btn-primary mt-4">
            Back to Instances
          </button>
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout>
      <Header
        title={instance.name}
        description={`Instance ID: ${instance.id} | Node: ${nodeName}`}
        action={
          <button
            onClick={() => router.push("/instances")}
            className="btn-secondary flex items-center gap-2"
          >
            <ArrowLeft size={16} />
            Back to List
          </button>
        }
        onRefresh={fetchInstance}
      />

      {/* Status and Actions */}
      <div className="card mb-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-xl font-bold text-primary mb-2">Status</h2>
            <StatusBadge status={instance.state} />
          </div>
          <div className="flex gap-2">
            {instance.state === "running" ? (
              <button
                onClick={() => handleAction("stop")}
                className="btn-secondary flex items-center gap-2"
                title="Stop"
              >
                <Square size={16} />
                Stop
              </button>
            ) : (
              <button
                onClick={() => handleAction("start")}
                className="btn-primary flex items-center gap-2"
                title="Start"
              >
                <Play size={16} />
                Start
              </button>
            )}
            <button
              onClick={() => handleAction("reboot")}
              className="btn-secondary flex items-center gap-2"
              title="Reboot"
            >
              <RefreshCw size={16} />
              Reboot
            </button>
            <button
              onClick={handleEditClick}
              className="btn-secondary flex items-center gap-2"
              title="Edit Instance"
            >
              <Edit size={16} />
              Edit
            </button>
            <button
              onClick={() => router.push(`/instances/${nodeName}/${instanceId}/console`)}
              className="btn-secondary flex items-center gap-2"
              title="Console"
            >
              <Monitor size={16} />
              Console
            </button>
            <button
              onClick={() => setIsResetPasswordModalOpen(true)}
              className="btn-secondary flex items-center gap-2"
              title="Reset Password"
            >
              <Key size={16} />
              Reset Password
            </button>
            <button
              onClick={handleDeleteClick}
              className="btn-danger flex items-center gap-2"
              title="Delete"
            >
              <Trash2 size={16} />
              Delete
            </button>
          </div>
        </div>
      </div>

      {/* Details */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Basic Information */}
        <div className="card">
          <h2 className="text-xl font-bold text-primary mb-4">Basic Information</h2>
          <dl className="space-y-3">
            <div>
              <dt className="text-sm font-medium text-gray-500">Instance ID</dt>
              <dd className="mt-1 text-sm text-gray-900 font-mono">{instance.id}</dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-gray-500">Name</dt>
              <dd className="mt-1 text-sm text-gray-900">{instance.name}</dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-gray-500">Node</dt>
              <dd className="mt-1 text-sm text-gray-900">{instance.node_name || nodeName}</dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-gray-500">Status</dt>
              <dd className="mt-1">
                <StatusBadge status={instance.state} />
              </dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-gray-500">Auto Start</dt>
              <dd className="mt-1 text-sm text-gray-900">
                {instance.autostart ? "Enabled" : "Disabled"}
              </dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-gray-500">IP Address</dt>
              <dd className="mt-1 text-sm text-gray-900 font-mono">
                {instance.ip_address || "N/A"}
              </dd>
            </div>
          </dl>
        </div>

        {/* Resources */}
        <div className="card">
          <h2 className="text-xl font-bold text-primary mb-4">Resources</h2>
          <dl className="space-y-3">
            <div>
              <dt className="text-sm font-medium text-gray-500">vCPUs</dt>
              <dd className="mt-1 text-sm text-gray-900">{instance.vcpus} cores</dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-gray-500">Memory</dt>
              <dd className="mt-1 text-sm text-gray-900">
                {(instance.memory_mb / 1024).toFixed(2)} GB
              </dd>
            </div>
            {instance.volume_id && (
              <div>
                <dt className="text-sm font-medium text-gray-500">Volume ID</dt>
                <dd className="mt-1 text-sm text-gray-900 font-mono">{instance.volume_id}</dd>
              </div>
            )}
          </dl>
        </div>

        {/* Configuration */}
        <div className="card">
          <h2 className="text-xl font-bold text-primary mb-4">Configuration</h2>
          <dl className="space-y-3">
            <div>
              <dt className="text-sm font-medium text-gray-500">Template ID</dt>
              <dd className="mt-1 text-sm text-gray-900 font-mono">
                {instance.template_id || "N/A"}
              </dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-gray-500">Domain UUID</dt>
              <dd className="mt-1 text-sm text-gray-900 font-mono">
                {instance.domain_uuid || "N/A"}
              </dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-gray-500">Key Pair</dt>
              <dd className="mt-1 text-sm text-gray-900">
                {instance.keypair_name || "None"}
              </dd>
            </div>
          </dl>
        </div>

        {/* Timestamps */}
        <div className="card">
          <h2 className="text-xl font-bold text-primary mb-4">Timestamps</h2>
          <dl className="space-y-3">
            <div>
              <dt className="text-sm font-medium text-gray-500">Created At</dt>
              <dd className="mt-1 text-sm text-gray-900">
                {instance.created_at ? new Date(instance.created_at).toLocaleString() : "N/A"}
              </dd>
            </div>
            {instance.updated_at && (
              <div>
                <dt className="text-sm font-medium text-gray-500">Updated At</dt>
                <dd className="mt-1 text-sm text-gray-900">
                  {new Date(instance.updated_at).toLocaleString()}
                </dd>
              </div>
            )}
          </dl>
        </div>
      </div>

      {/* SSH Connection Info */}
      {instance.ip_address && instance.state === "running" && (
        <div className="card mt-6">
          <h2 className="text-xl font-bold text-primary mb-4">Connection</h2>
          <div className="bg-gray-50 p-4 rounded border border-gray-200">
            <p className="text-sm text-gray-600 mb-2">SSH Command:</p>
            <code className="text-sm bg-gray-900 text-green-400 p-3 rounded block font-mono">
              ssh -i ~/.ssh/{instance.keypair_name || "your-key"}.pem ubuntu@{instance.ip_address}
            </code>
          </div>
        </div>
      )}

      {/* Reset Password Modal */}
      <Modal
        isOpen={isResetPasswordModalOpen}
        onClose={() => setIsResetPasswordModalOpen(false)}
        title="Reset Instance Password"
      >
        <form onSubmit={handleResetPassword} className="space-y-6">
          <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
            <p className="text-sm text-yellow-800">
              This will reset the password for the specified user.
              The instance may need to restart depending on the reset method available.
            </p>
          </div>

          <div>
            <label className="label">Username</label>
            <input
              type="text"
              className="input"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
              placeholder="root"
            />
          </div>

          <div>
            <label className="label">New Password</label>
            <input
              type="password"
              className="input"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              required
              minLength={8}
              placeholder="Enter new password (min 8 characters)"
            />
          </div>

          <div className="flex justify-end gap-3 pt-4">
            <button
              type="button"
              onClick={() => setIsResetPasswordModalOpen(false)}
              className="btn-secondary"
            >
              Cancel
            </button>
            <button type="submit" className="btn-primary">
              Reset Password
            </button>
          </div>
        </form>
      </Modal>

      {/* Edit Instance Modal */}
      <Modal
        isOpen={isEditModalOpen}
        onClose={() => setIsEditModalOpen(false)}
        title="Edit Instance Attributes"
      >
        <form onSubmit={handleEditInstance} className="space-y-6">
          {instance?.state === "running" && (
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
              <p className="text-sm text-blue-800">
                This instance is currently running. Changes will be applied dynamically if supported.
              </p>
            </div>
          )}

          <div>
            <label className="label">Instance Name</label>
            <input
              type="text"
              className="input"
              value={editFormData.name}
              onChange={(e) =>
                setEditFormData({ ...editFormData, name: e.target.value })
              }
              placeholder="my-instance"
            />
          </div>

          <div>
            <label className="label">vCPUs</label>
            <input
              type="number"
              className="input"
              value={editFormData.vcpus}
              onChange={(e) =>
                setEditFormData({ ...editFormData, vcpus: parseInt(e.target.value) })
              }
              min={1}
              max={16}
              required
            />
            <p className="text-xs text-gray-500 mt-1">
              Number of virtual CPUs (1-16)
            </p>
          </div>

          <div>
            <label className="label">Memory (MB)</label>
            <input
              type="number"
              className="input"
              value={editFormData.memory_mb}
              onChange={(e) =>
                setEditFormData({ ...editFormData, memory_mb: parseInt(e.target.value) })
              }
              min={512}
              step={512}
              required
            />
            <p className="text-xs text-gray-500 mt-1">
              Memory size in MB (minimum 512 MB)
            </p>
          </div>

          <div className="flex items-center justify-between rounded border p-4 bg-gray-50">
            <div>
              <p className="label mb-1">Auto Start</p>
              <p className="text-xs text-gray-500">Automatically start this instance when the node boots</p>
            </div>
            <label className="inline-flex items-center">
              <input
                type="checkbox"
                className="mr-2 h-4 w-4"
                checked={editFormData.autostart}
                onChange={(e) =>
                  setEditFormData({ ...editFormData, autostart: e.target.checked })
                }
              />
              <span className="text-sm text-gray-700">
                {editFormData.autostart ? "Enabled" : "Disabled"}
              </span>
            </label>
          </div>

          <div className="flex justify-end gap-3 pt-4">
            <button
              type="button"
              onClick={() => setIsEditModalOpen(false)}
              className="btn-secondary"
            >
              Cancel
            </button>
            <button type="submit" className="btn-primary">
              Save Changes
            </button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        isOpen={isDeleteDialogOpen}
        onClose={() => setIsDeleteDialogOpen(false)}
        onConfirm={handleDeleteConfirm}
        title="Terminate Instance"
        message="Are you sure you want to terminate this instance? This action cannot be undone and all data will be permanently lost."
        confirmText="Terminate"
        cancelText="Cancel"
        variant="danger"
        extraContent={
          <label className="flex items-center gap-2 text-sm text-gray-700">
            <input
              type="checkbox"
              className="h-4 w-4"
              checked={deleteVolumes}
              onChange={(e) => setDeleteVolumes(e.target.checked)}
            />
            Also delete associated disks
          </label>
        }
      />
    </DashboardLayout>
  );
}
