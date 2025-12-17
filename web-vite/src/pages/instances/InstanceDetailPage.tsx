import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import Header from "@/components/Header";
import StatusBadge from "@/components/StatusBadge";
import ConfirmDialog from "@/components/ConfirmDialog";
import Table from "@/components/Table";
import { useToast } from "@/components/ToastContainer";
import { Play, Square, RefreshCw, Trash2, ArrowLeft, Key, Edit, Monitor, Info, Cpu, HardDrive, Settings, Network } from "lucide-react";
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
  started_at?: string;
  updated_at?: string;
  domain_uuid?: string;
  domain_name?: string;
  interfaces?: InstanceInterface[];
  disks?: { target?: string; path?: string; format?: string; capacity_b?: number; allocation_b?: number }[];
}

type InstanceInterface = {
  name: string;
  type: string;
  source: string;
  mac: string;
  ips?: string[];
};

export default function InstanceDetailPage() {
  const toast = useToast();
  const params = useParams();
  const navigate = useNavigate();
  const { nodeName, id: instanceId } = params;

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
      const response = await fetch("/api/describe-instances", {
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
    if (nodeName && instanceId) {
      fetchInstance();
    }
  }, [nodeName, instanceId]);

  const handleAction = async (action: string) => {
    try {
      const response = await fetch(`/api/${action}-instances`, {
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
      const response = await fetch("/api/terminate-instances", {
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
        navigate("/instances");
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
      const response = await fetch("/api/reset-instance-password", {
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
      const response = await fetch("/api/modify-instance-attribute", {
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

  const disksColumns = [
    { key: "target", label: "Target" },
    { key: "path", label: "Path" },
    { key: "format", label: "Format" },
    {
      key: "capacity_b",
      label: "Capacity",
      render: (value: unknown) => {
        const cap = Number(value || 0);
        return <span>{cap > 0 ? `${(cap / 1024 / 1024 / 1024).toFixed(2)} GB` : "-"}</span>;
      },
    },
    {
      key: "allocation_b",
      label: "Allocated",
      render: (value: unknown, row: any) => {
        const alloc = Number(value || 0);
        const cap = Number(row.capacity_b || 0);
        const pct = cap > 0 ? ((alloc / cap) * 100).toFixed(0) : "";
        return (
          <span>
            {alloc > 0 ? `${(alloc / 1024 / 1024 / 1024).toFixed(2)} GB` : "-"}
            {pct ? ` (${pct}%)` : ""}
          </span>
        );
      },
    },
  ];

  if (loading) {
    return (
      <div className="card text-center py-12">
        <p className="text-gray-500">Loading instance details...</p>
      </div>
    );
  }

  if (!instance) {
    return (
      <div className="card text-center py-12">
        <p className="text-gray-500">Instance not found</p>
        <button onClick={() => navigate("/instances")} className="btn-primary mt-4">
          Back to Instances
        </button>
      </div>
    );
  }

  return (
    <>
      <Header
        title={instance.name}
        description={`Instance ID: ${instance.id} | Node: ${nodeName}`}
        action={
          <button
            onClick={() => navigate("/instances")}
            className="btn-secondary flex items-center gap-2"
          >
            <ArrowLeft size={16} />
            Back to List
          </button>
        }
        onRefresh={fetchInstance}
      />

      {/* Status and Actions */}
      <div className="bg-white border-l-4 border-accent rounded-lg p-6 mb-6 shadow-sm">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-accent/10 rounded-lg">
              <Monitor className="w-5 h-5 text-accent" />
            </div>
            <div>
              <h2 className="text-xl font-bold text-primary mb-2">Status</h2>
              <StatusBadge status={instance.state} />
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            {instance.state === "running" ? (
              <button
                onClick={() => handleAction("stop")}
                className="btn-secondary flex items-center gap-2 px-4 py-2 border-2 border-primary hover:bg-primary hover:text-white transition-all"
                title="Stop"
              >
                <Square size={18} className="text-red-600" />
                <span>Stop</span>
              </button>
            ) : (
              <button
                onClick={() => handleAction("start")}
                className="btn-primary flex items-center gap-2 px-4 py-2 border-2 border-accent hover:bg-accent-dark hover:border-accent-dark transition-all"
                title="Start"
              >
                <Play size={18} className="text-white" />
                <span>Start</span>
              </button>
            )}
            <button
              onClick={() => handleAction("reboot")}
              className="btn-secondary flex items-center gap-2 px-4 py-2 border-2 border-primary hover:bg-primary hover:text-white transition-all"
              title="Reboot"
            >
              <RefreshCw size={18} className="text-blue-600" />
              <span>Reboot</span>
            </button>
            <button
              onClick={handleEditClick}
              className="btn-secondary flex items-center gap-2 px-4 py-2 border-2 border-primary hover:bg-primary hover:text-white transition-all"
              title="Edit Instance"
            >
              <Edit size={18} className="text-purple-600" />
              <span>Edit</span>
            </button>
            <button
              onClick={() => navigate(`/instances/${nodeName}/${instanceId}/console`)}
              className="btn-secondary flex items-center gap-2 px-4 py-2 border-2 border-primary hover:bg-primary hover:text-white transition-all"
              title="Console"
            >
              <Monitor size={18} className="text-accent" />
              <span>Console</span>
            </button>
            <button
              onClick={() => setIsResetPasswordModalOpen(true)}
              className="btn-secondary flex items-center gap-2 px-4 py-2 border-2 border-primary hover:bg-primary hover:text-white transition-all"
              title="Reset Password"
            >
              <Key size={18} className="text-yellow-600" />
              <span>Reset Password</span>
            </button>
            <button
              onClick={handleDeleteClick}
              className="btn-danger flex items-center gap-2 px-4 py-2 border-2 border-coral hover:bg-red-600 hover:border-red-600 transition-all"
              title="Delete"
            >
              <Trash2 size={18} className="text-white" />
              <span>Delete</span>
            </button>
          </div>
        </div>
      </div>

      <div className="bg-white border-l-4 border-blue-500 rounded-lg p-6 shadow-sm mb-6">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-blue-50 rounded-lg">
              <HardDrive className="w-5 h-5 text-blue-600" />
            </div>
            <h2 className="text-lg font-semibold text-gray-900">Disks</h2>
          </div>
        </div>
        <Table
          columns={disksColumns}
          data={instance.disks || []}
          emptyMessage="No disks"
          keyField="target"
        />
      </div>

      {/* Instance Details - Integrated Card */}
      <div className="bg-white rounded-lg p-6 shadow-sm">
        {/* First Row: Basic Information and Resources */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
          {/* Basic Information */}
          <div className="space-y-3">
            <div className="flex items-center gap-2 mb-4 pb-3 border-b border-gray-200">
              <div className="p-1.5 bg-blue-50 rounded-lg">
                <Info className="w-4 h-4 text-blue-600" />
              </div>
              <h3 className="text-sm font-semibold text-gray-700 uppercase tracking-wide">Basic Information</h3>
            </div>
            <dl className="grid grid-cols-2 gap-x-4 gap-y-2.5">
              <div>
                <dt className="text-xs font-medium text-gray-500">Instance ID</dt>
                <dd className="mt-0.5 text-sm text-gray-900 font-mono break-all">{instance.id}</dd>
              </div>
              <div>
                <dt className="text-xs font-medium text-gray-500">Name</dt>
                <dd className="mt-0.5 text-sm text-gray-900">{instance.name}</dd>
              </div>
              <div>
                <dt className="text-xs font-medium text-gray-500">Node</dt>
                <dd className="mt-0.5 text-sm text-gray-900">{instance.node_name || nodeName}</dd>
              </div>
              <div>
                <dt className="text-xs font-medium text-gray-500">Status</dt>
                <dd className="mt-0.5">
                  <StatusBadge status={instance.state} />
                </dd>
              </div>
              <div>
                <dt className="text-xs font-medium text-gray-500">Auto Start</dt>
                <dd className="mt-0.5 text-sm text-gray-900">
                  {instance.autostart ? "Enabled" : "Disabled"}
                </dd>
              </div>
              {instance.started_at && (
                <div>
                  <dt className="text-xs font-medium text-gray-500">Started At</dt>
                  <dd className="mt-0.5 text-sm text-gray-900">
                    {new Date(instance.started_at).toLocaleString()}
                  </dd>
                </div>
              )}
              {instance.updated_at && (
                <div>
                  <dt className="text-xs font-medium text-gray-500">Updated At</dt>
                  <dd className="mt-0.5 text-sm text-gray-900">
                    {new Date(instance.updated_at).toLocaleString()}
                  </dd>
                </div>
              )}
            </dl>
          </div>

          {/* Resources */}
          <div className="space-y-3">
            <div className="flex items-center gap-2 mb-4 pb-3 border-b border-gray-200">
              <div className="p-1.5 bg-green-50 rounded-lg">
                <Cpu className="w-4 h-4 text-green-600" />
              </div>
              <h3 className="text-sm font-semibold text-gray-700 uppercase tracking-wide">Resources</h3>
            </div>
            <dl className="space-y-2.5">
              <div>
                <dt className="text-xs font-medium text-gray-500">vCPUs</dt>
                <dd className="mt-0.5 text-sm text-gray-900">{instance.vcpus} cores</dd>
              </div>
              <div>
                <dt className="text-xs font-medium text-gray-500">Memory</dt>
                <dd className="mt-0.5 text-sm text-gray-900">
                  {(instance.memory_mb / 1024).toFixed(2)} GB
                </dd>
              </div>
              {instance.volume_id && (
                <div>
                  <dt className="text-xs font-medium text-gray-500">Volume ID</dt>
                  <dd className="mt-0.5 text-sm text-gray-900 font-mono break-all">{instance.volume_id}</dd>
                </div>
              )}
            </dl>
          </div>
        </div>

        {/* Second Row: Network Interfaces and Configuration */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {/* Network Interfaces */}
          {instance.interfaces && instance.interfaces.length > 0 ? (
            <div className="space-y-3">
              <div className="flex items-center gap-2 mb-4 pb-3 border-b border-gray-200">
                <div className="p-1.5 bg-accent/10 rounded-lg">
                  <Network className="w-4 h-4 text-accent" />
                </div>
                <h3 className="text-sm font-semibold text-gray-700 uppercase tracking-wide">Network Interfaces</h3>
              </div>
              <div className="space-y-4">
                {instance.interfaces.map((iface) => (
                  <div key={`${iface.name}-${iface.mac}`} className="space-y-2.5">
                    <div>
                      <dt className="text-xs font-medium text-gray-500">Interface Name</dt>
                      <dd className="mt-0.5 text-sm text-gray-900 font-mono font-semibold">{iface.name}</dd>
                    </div>
                    <div>
                      <dt className="text-xs font-medium text-gray-500">MAC Address</dt>
                      <dd className="mt-0.5 text-sm text-gray-900 font-mono">{iface.mac}</dd>
                    </div>
                    {iface.ips && iface.ips.length > 0 && (
                      <div>
                        <dt className="text-xs font-medium text-gray-500">IP Address</dt>
                        <dd className="mt-0.5 text-sm text-gray-900 font-mono">
                          {iface.ips.map((ip, idx) => (
                            <span key={ip} className={idx > 0 ? "block mt-0.5" : ""}>{ip}</span>
                          ))}
                        </dd>
                      </div>
                    )}
                    <div>
                      <dt className="text-xs font-medium text-gray-500">Network Mode</dt>
                      <dd className="mt-0.5 text-sm text-gray-900 capitalize">{iface.type}</dd>
                    </div>
                    <div>
                      <dt className="text-xs font-medium text-gray-500">Network Source</dt>
                      <dd className="mt-0.5 text-sm text-gray-900 font-mono">{iface.source}</dd>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <div className="space-y-3">
              <div className="flex items-center gap-2 mb-4 pb-3 border-b border-gray-200">
                <div className="p-1.5 bg-accent/10 rounded-lg">
                  <Network className="w-4 h-4 text-accent" />
                </div>
                <h3 className="text-sm font-semibold text-gray-700 uppercase tracking-wide">Network Interfaces</h3>
              </div>
              <p className="text-sm text-gray-500">No network interfaces</p>
            </div>
          )}

          {/* Configuration */}
          <div className="space-y-3">
            <div className="flex items-center gap-2 mb-4 pb-3 border-b border-gray-200">
              <div className="p-1.5 bg-purple-50 rounded-lg">
                <Settings className="w-4 h-4 text-purple-600" />
              </div>
              <h3 className="text-sm font-semibold text-gray-700 uppercase tracking-wide">Configuration</h3>
            </div>
            <dl className="space-y-2.5">
              <div>
                <dt className="text-xs font-medium text-gray-500">Template ID</dt>
                <dd className="mt-0.5 text-sm text-gray-900 font-mono break-all">
                  {instance.template_id || "N/A"}
                </dd>
              </div>
              <div>
                <dt className="text-xs font-medium text-gray-500">Domain UUID</dt>
                <dd className="mt-0.5 text-sm text-gray-900 font-mono break-all">
                  {instance.domain_uuid || "N/A"}
                </dd>
              </div>
              <div>
                <dt className="text-xs font-medium text-gray-500">Key Pair</dt>
                <dd className="mt-0.5 text-sm text-gray-900">
                  {instance.keypair_name || "None"}
                </dd>
              </div>
            </dl>
          </div>
        </div>
      </div>

      {/* SSH Connection Info */}
      {instance.ip_address && instance.state === "running" && (
        <div className="bg-white border-l-4 border-accent rounded-lg p-6 shadow-sm mt-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 bg-accent/10 rounded-lg">
              <Network className="w-5 h-5 text-accent" />
            </div>
            <h2 className="text-xl font-bold text-primary">Connection</h2>
          </div>
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
    </>
  );
}

