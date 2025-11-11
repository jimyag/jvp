"use client";

import { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import DashboardLayout from "@/components/DashboardLayout";
import Header from "@/components/Header";
import StatusBadge from "@/components/StatusBadge";
import { Play, Square, RefreshCw, Trash2, ArrowLeft, Key } from "lucide-react";
import { apiPost } from "@/lib/api";
import Modal from "@/components/Modal";

interface Instance {
  id: string;
  name: string;
  state: string;
  vcpus: number;
  memory_mb: number;
  image_id?: string;
  volume_id?: string;
  ip_address?: string;
  keypair_name?: string;
  created_at: string;
  updated_at?: string;
  domain_uuid?: string;
  domain_name?: string;
}

export default function InstanceDetailPage() {
  const params = useParams();
  const router = useRouter();
  const instanceId = params.id as string;

  const [instance, setInstance] = useState<Instance | null>(null);
  const [loading, setLoading] = useState(true);
  const [isResetPasswordModalOpen, setIsResetPasswordModalOpen] = useState(false);
  const [newPassword, setNewPassword] = useState("");

  const fetchInstance = async () => {
    setLoading(true);
    try {
      const data = await apiPost<{ instances: Instance[] }>("/api/instances/describe", {
        instanceIDs: [instanceId],
      });
      if (data.instances && data.instances.length > 0) {
        setInstance(data.instances[0]);
      }
    } catch (error) {
      console.error("Failed to fetch instance:", error);
      alert("Failed to load instance details");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchInstance();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [instanceId]);

  const handleAction = async (action: string) => {
    try {
      await apiPost(`/api/instances/${action}`, {
        instanceIDs: [instanceId],
      });
      fetchInstance();
    } catch (error) {
      console.error(`Failed to ${action} instance:`, error);
      alert(`Failed to ${action} instance`);
    }
  };

  const handleDelete = async () => {
    if (!confirm("Are you sure you want to delete this instance?")) return;

    try {
      await apiPost("/api/instances/terminate", {
        instanceIDs: [instanceId],
      });
      router.push("/instances");
    } catch (error) {
      console.error("Failed to delete instance:", error);
      alert("Failed to delete instance");
    }
  };

  const handleResetPassword = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await apiPost("/api/instances/reset-password", {
        instance_id: instanceId,
        password: newPassword,
      });
      setIsResetPasswordModalOpen(false);
      setNewPassword("");
      alert("Password reset successfully!");
    } catch (error) {
      console.error("Failed to reset password:", error);
      alert("Failed to reset password");
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
        description={`Instance ID: ${instance.id}`}
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
              onClick={() => setIsResetPasswordModalOpen(true)}
              className="btn-secondary flex items-center gap-2"
              title="Reset Password"
            >
              <Key size={16} />
              Reset Password
            </button>
            <button
              onClick={handleDelete}
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
              <dt className="text-sm font-medium text-gray-500">Status</dt>
              <dd className="mt-1">
                <StatusBadge status={instance.state} />
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
              <dt className="text-sm font-medium text-gray-500">Image ID</dt>
              <dd className="mt-1 text-sm text-gray-900 font-mono">
                {instance.image_id || "N/A"}
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
                {new Date(instance.created_at).toLocaleString()}
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
              This will reset the root/administrator password for this instance.
              The instance may need to restart depending on the reset method available.
            </p>
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
    </DashboardLayout>
  );
}
