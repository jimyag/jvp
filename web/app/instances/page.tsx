"use client";

import { useState, useEffect, useMemo } from "react";
import DashboardLayout from "@/components/DashboardLayout";
import Header from "@/components/Header";
import Table from "@/components/Table";
import StatusBadge from "@/components/StatusBadge";
import Modal from "@/components/Modal";
import ConfirmDialog from "@/components/ConfirmDialog";
import SearchFilter from "@/components/SearchFilter";
import { useToast } from "@/components/ToastContainer";
import { Play, Square, RefreshCw, Trash2, Plus } from "lucide-react";

interface Instance {
  id: string;
  name: string;
  state: string;
  node_name: string;
  template_id?: string;
  vcpus: number;
  memory_mb: number;
  created_at: string;
  domain_uuid?: string;
  domain_name?: string;
}

interface Node {
  name: string;
  uri: string;
  status: string;
}

interface StoragePool {
  name: string;
  state: string;
  path: string;
}

interface Template {
  id: string;
  name: string;
  format: string;
  size_gb: number;
}

interface KeyPair {
  id: string;
  name: string;
  fingerprint: string;
}

export default function InstancesPage() {
  const toast = useToast();
  const [instances, setInstances] = useState<Instance[]>([]);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [storagePools, setStoragePools] = useState<StoragePool[]>([]);
  const [templates, setTemplates] = useState<Template[]>([]);
  const [keypairs, setKeypairs] = useState<KeyPair[]>([]);
  const [loading, setLoading] = useState(true);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [instanceToDelete, setInstanceToDelete] = useState<{id: string, nodeName: string} | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedNode, setSelectedNode] = useState<string>("");
  const [currentStep, setCurrentStep] = useState(0);
  const [keypairInputMethod, setKeypairInputMethod] = useState<"select" | "upload" | "manual">("select");
  const [manualPublicKey, setManualPublicKey] = useState("");
  const [formData, setFormData] = useState({
    node_name: "",
    pool_name: "",
    template_id: "",
    size_gb: 20,
    memory_mb: 2048,
    vcpus: 2,
    network_type: "bridge",
    network_source: "br0",
    keypair_ids: [] as string[],
    hostname: "",
    timezone: "Asia/Shanghai",
    disable_root: false,
    packages: [] as string[],
    run_cmd: [] as string[],
    username: "",
    user_password: "",
    user_sudo: "ALL=(ALL) NOPASSWD:ALL",
    user_groups: "sudo",
    user_shell: "/bin/bash",
  });

  const filteredInstances = useMemo(() => {
    let filtered = instances;
    if (selectedNode) {
      filtered = filtered.filter(i => i.node_name === selectedNode);
    }
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(
        (instance) =>
          instance.id.toLowerCase().includes(query) ||
          instance.name?.toLowerCase().includes(query) ||
          instance.state.toLowerCase().includes(query) ||
          instance.node_name?.toLowerCase().includes(query)
      );
    }
    return filtered;
  }, [instances, searchQuery, selectedNode]);

  const fetchNodes = async () => {
    try {
      const response = await fetch("/api/list-nodes", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });
      if (response.ok) {
        const data = await response.json();
        setNodes(data.nodes || []);
        if (data.nodes?.length > 0 && !selectedNode) {
          setSelectedNode(data.nodes[0].name);
        }
      }
    } catch (error) {
      console.error("Failed to fetch nodes:", error);
    }
  };

  const fetchInstances = async () => {
    if (!selectedNode) return;
    setLoading(true);
    try {
      const response = await fetch("/api/instances/describe", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ node_name: selectedNode }),
      });
      if (response.ok) {
        const data = await response.json();
        setInstances(data.instances || []);
      } else {
        toast.error("Failed to load instances");
      }
    } catch (error) {
      console.error("Failed to fetch instances:", error);
      toast.error("Failed to load instances. Please check if backend is running.");
    } finally {
      setLoading(false);
    }
  };

  const fetchStoragePools = async (nodeName: string) => {
    if (!nodeName) return;
    try {
      const response = await fetch("/api/list-storage-pools", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ node_name: nodeName }),
      });
      if (response.ok) {
        const data = await response.json();
        setStoragePools(data.pools || []);
      }
    } catch (error) {
      console.error("Failed to fetch storage pools:", error);
    }
  };

  const fetchTemplates = async (nodeName: string, poolName: string) => {
    if (!nodeName || !poolName) return;
    try {
      const response = await fetch("/api/list-templates", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ node_name: nodeName, pool_name: poolName }),
      });
      if (response.ok) {
        const data = await response.json();
        setTemplates(data.templates || []);
      }
    } catch (error) {
      console.error("Failed to fetch templates:", error);
    }
  };

  const fetchKeypairs = async () => {
    try {
      const response = await fetch("/api/keypairs/describe", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });
      if (response.ok) {
        const data = await response.json();
        setKeypairs(data.keypairs || []);
      }
    } catch (error) {
      console.error("Failed to fetch keypairs:", error);
    }
  };

  useEffect(() => {
    fetchNodes();
    fetchKeypairs();
  }, []);

  useEffect(() => {
    if (selectedNode) {
      fetchInstances();
    }
  }, [selectedNode]);

  useEffect(() => {
    if (formData.node_name) {
      fetchStoragePools(formData.node_name);
    }
  }, [formData.node_name]);

  useEffect(() => {
    if (formData.node_name && formData.pool_name) {
      fetchTemplates(formData.node_name, formData.pool_name);
    }
  }, [formData.node_name, formData.pool_name]);

  const handleCreateInstance = async (e: React.FormEvent) => {
    e.preventDefault();

    // 只有在最后一步才能提交
    if (currentStep < 2) {
      return;
    }

    try {
      let finalKeypairIds = [...formData.keypair_ids];
      if (keypairInputMethod === "manual" && manualPublicKey.trim()) {
        try {
          const importResponse = await fetch("/api/keypairs/import", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              name: `keypair-${Date.now()}`,
              public_key: manualPublicKey.trim(),
            }),
          });

          if (importResponse.ok) {
            const importData = await importResponse.json();
            finalKeypairIds = [importData.id];
            toast.success("Public key imported successfully!");
          } else {
            const error = await importResponse.json();
            toast.error(`Failed to import public key: ${error.message || "Unknown error"}`);
            return;
          }
        } catch (error) {
          console.error("Failed to import public key:", error);
          toast.error("Failed to import public key. Please try again.");
          return;
        }
      }

      const userData: any = {};
      const structuredUserData: any = {};

      if (formData.hostname) {
        structuredUserData.hostname = formData.hostname;
      }
      if (formData.timezone) {
        structuredUserData.timezone = formData.timezone;
      }
      structuredUserData.disable_root = formData.disable_root;

      if (formData.username) {
        structuredUserData.users = [{
          name: formData.username,
          plain_text_passwd: formData.user_password || undefined,
          sudo: formData.user_sudo,
          groups: formData.user_groups,
          shell: formData.user_shell,
        }];
      }

      if (formData.packages.length > 0) {
        structuredUserData.packages = formData.packages;
      }
      if (formData.run_cmd.length > 0) {
        structuredUserData.run_cmd = formData.run_cmd;
      }

      const hasUserData = Object.keys(structuredUserData).length > 0;
      if (hasUserData) {
        userData.structured_user_data = structuredUserData;
      }

      const requestBody: any = {
        node_name: formData.node_name,
        pool_name: formData.pool_name,
        template_id: formData.template_id || undefined,
        size_gb: formData.size_gb,
        memory_mb: formData.memory_mb,
        vcpus: formData.vcpus,
        network_type: formData.network_type,
        network_source: formData.network_source,
        keypair_ids: finalKeypairIds.length > 0 ? finalKeypairIds : undefined,
      };

      if (hasUserData) {
        requestBody.user_data = userData;
      }

      const response = await fetch("/api/instances/run", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(requestBody),
      });

      if (response.ok) {
        setIsCreateModalOpen(false);
        setCurrentStep(0);
        setSelectedNode(formData.node_name);
        fetchInstances();
        fetchKeypairs();
        setFormData({
          node_name: "",
          pool_name: "",
          template_id: "",
          size_gb: 20,
          memory_mb: 2048,
          vcpus: 2,
          network_type: "bridge",
          network_source: "br0",
          keypair_ids: [],
          hostname: "",
          timezone: "Asia/Shanghai",
          disable_root: false,
          packages: [],
          run_cmd: [],
          username: "",
          user_password: "",
          user_sudo: "ALL=(ALL) NOPASSWD:ALL",
          user_groups: "sudo",
          user_shell: "/bin/bash",
        });
        setKeypairInputMethod("select");
        setManualPublicKey("");
        toast.success("Instance created successfully!");
      } else {
        const error = await response.json();
        toast.error(`Failed to create instance: ${error.message || "Unknown error"}`);
      }
    } catch (error) {
      console.error("Failed to create instance:", error);
      toast.error("Failed to create instance. Please try again.");
    }
  };

  const handleAction = async (instance: Instance, action: string) => {
    try {
      const response = await fetch(`/api/instances/${action}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          node_name: instance.node_name,
          instance_ids: [instance.id]
        }),
      });

      if (response.ok) {
        const actionName = action === "start" ? "started" : action === "stop" ? "stopped" : "rebooted";
        toast.success(`Instance ${actionName} successfully!`);
        setTimeout(() => {
          fetchInstances();
        }, 2000);
      } else {
        const error = await response.json();
        toast.error(`Failed to ${action} instance: ${error.message || "Unknown error"}`);
      }
    } catch (error) {
      console.error(`Failed to ${action} instance:`, error);
      toast.error(`Failed to ${action} instance. Please try again.`);
    }
  };

  const handleDeleteClick = (instance: Instance) => {
    setInstanceToDelete({id: instance.id, nodeName: instance.node_name});
    setIsDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = async () => {
    if (!instanceToDelete) return;
    try {
      const response = await fetch("/api/instances/terminate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          node_name: instanceToDelete.nodeName,
          instance_ids: [instanceToDelete.id]
        }),
      });

      if (response.ok) {
        fetchInstances();
        toast.success("Instance terminated successfully!");
      } else {
        const error = await response.json();
        toast.error(`Failed to terminate instance: ${error.message || "Unknown error"}`);
      }
    } catch (error) {
      console.error("Failed to delete instance:", error);
      toast.error("Failed to terminate instance. Please try again.");
    }
  };

  const columns = [
    {
      key: "id",
      label: "ID",
      render: (value: unknown, row: any) => (
        <a href={`/instances/${row.node_name}/${value}`} className="text-accent hover:underline font-mono text-xs">
          {String(value).substring(0, 12)}...
        </a>
      ),
    },
    {
      key: "name",
      label: "Name",
      render: (_: unknown, row: any) => (
        <a href={`/instances/${row.node_name}/${row.id}`} className="text-primary hover:text-accent font-medium">
          {row.name || row.domain_name || "N/A"}
        </a>
      ),
    },
    {
      key: "node_name",
      label: "Node",
      render: (value: unknown) => <span className="text-gray-600">{String(value)}</span>,
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
      key: "template_id",
      label: "Template",
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
                onClick={() => handleAction(instance, "stop")}
                className="p-2 text-gray-600 hover:text-red-600 transition-colors"
                title="Stop"
              >
                <Square size={16} />
              </button>
            ) : (
              <button
                onClick={() => handleAction(instance, "start")}
                className="p-2 text-gray-600 hover:text-green-600 transition-colors"
                title="Start"
              >
                <Play size={16} />
              </button>
            )}
            <button
              onClick={() => handleAction(instance, "reboot")}
              className="p-2 text-gray-600 hover:text-blue-600 transition-colors"
              title="Reboot"
            >
              <RefreshCw size={16} />
            </button>
            <button
              onClick={() => handleDeleteClick(instance)}
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

      {/* Node selector */}
      <div className="mb-4 flex gap-4 items-center">
        <label className="text-sm font-medium text-gray-700">Node:</label>
        <select
          className="input w-48"
          value={selectedNode}
          onChange={(e) => setSelectedNode(e.target.value)}
        >
          {nodes.map((node) => (
            <option key={node.name} value={node.name}>
              {node.name}
            </option>
          ))}
        </select>
      </div>

      {loading ? (
        <div className="card text-center py-12">
          <p className="text-gray-500">Loading instances...</p>
        </div>
      ) : (
        <>
          <div className="mb-4">
            <SearchFilter
              onSearch={setSearchQuery}
              placeholder="Search instances by ID, name, state, or node..."
            />
          </div>
          <Table
            columns={columns}
            data={filteredInstances}
            emptyMessage={
              searchQuery
                ? "No instances match your search criteria."
                : "No instances found. Create your first instance to get started."
            }
          />
        </>
      )}

      <Modal
        isOpen={isCreateModalOpen}
        onClose={() => {
          setIsCreateModalOpen(false);
          setCurrentStep(0);
        }}
        title="Create New Instance"
        maxWidth="xl"
      >
        <form
          onSubmit={handleCreateInstance}
          onKeyDown={(e) => {
            // 阻止 Enter 键在非最后一步时提交表单
            if (e.key === "Enter" && currentStep < 2) {
              e.preventDefault();
            }
          }}
          className="space-y-6"
        >
          {/* Step indicator */}
          <div className="flex items-center justify-between border-b pb-4">
            {["Basic", "User & System", "Advanced"].map((step, index) => (
              <div key={step} className="flex items-center">
                <button
                  type="button"
                  onClick={() => setCurrentStep(index)}
                  disabled={index === 0 ? false : (index === 1 ? !formData.node_name || !formData.pool_name : false)}
                  className={`flex items-center justify-center w-8 h-8 rounded-full border-2 transition-colors ${
                    currentStep >= index
                      ? "border-accent bg-accent text-white"
                      : "border-gray-300 text-gray-400"
                  } ${index > 0 && (!formData.node_name || !formData.pool_name) ? "cursor-not-allowed opacity-50" : "cursor-pointer hover:opacity-80"}`}
                >
                  {index + 1}
                </button>
                <span
                  className={`ml-2 text-sm font-medium ${
                    currentStep >= index ? "text-accent" : "text-gray-400"
                  }`}
                >
                  {step}
                </span>
                {index < 2 && (
                  <div
                    className={`w-12 h-0.5 mx-2 ${
                      currentStep > index ? "bg-accent" : "bg-gray-300"
                    }`}
                  />
                )}
              </div>
            ))}
          </div>

          {/* Step 1: Basic Configuration */}
          {currentStep === 0 && (
            <div className="space-y-4">
              <h3 className="font-semibold text-gray-900">Basic Configuration</h3>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="label">Node *</label>
                  <select
                    className="input"
                    value={formData.node_name}
                    onChange={(e) =>
                      setFormData({ ...formData, node_name: e.target.value, pool_name: "", template_id: "" })
                    }
                    required
                  >
                    <option value="">Select Node</option>
                    {nodes.map((node) => (
                      <option key={node.name} value={node.name}>
                        {node.name}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="label">Storage Pool *</label>
                  <select
                    className="input"
                    value={formData.pool_name}
                    onChange={(e) =>
                      setFormData({ ...formData, pool_name: e.target.value, template_id: "" })
                    }
                    required
                    disabled={!formData.node_name}
                  >
                    <option value="">Select Storage Pool</option>
                    {storagePools.map((pool) => (
                      <option key={pool.name} value={pool.name}>
                        {pool.name}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <div>
                <label className="label">Template</label>
                <select
                  className="input"
                  value={formData.template_id}
                  onChange={(e) =>
                    setFormData({ ...formData, template_id: e.target.value })
                  }
                  disabled={!formData.pool_name}
                >
                  <option value="">No Template (Empty Disk)</option>
                  {templates.map((template) => (
                    <option key={template.id} value={template.id}>
                      {template.name} ({template.size_gb}GB, {template.format})
                    </option>
                  ))}
                </select>
                <p className="text-xs text-gray-500 mt-1">
                  Select a template to create instance from, or leave empty for blank disk
                </p>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="label">vCPUs *</label>
                  <input
                    type="number"
                    className="input"
                    value={formData.vcpus}
                    onChange={(e) =>
                      setFormData({ ...formData, vcpus: Number(e.target.value) })
                    }
                    min="1"
                    max="32"
                    required
                  />
                </div>

                <div>
                  <label className="label">Memory (MB) *</label>
                  <input
                    type="number"
                    className="input"
                    value={formData.memory_mb}
                    onChange={(e) =>
                      setFormData({ ...formData, memory_mb: Number(e.target.value) })
                    }
                    min="512"
                    step="512"
                    required
                  />
                  <p className="text-xs text-gray-500 mt-1">
                    {(formData.memory_mb / 1024).toFixed(1)} GB
                  </p>
                </div>
              </div>

              <div>
                <label className="label">Disk Size (GB) *</label>
                <input
                  type="number"
                  className="input"
                  value={formData.size_gb}
                  onChange={(e) =>
                    setFormData({ ...formData, size_gb: Number(e.target.value) })
                  }
                  min="10"
                  required
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="label">Network Type</label>
                  <select
                    className="input"
                    value={formData.network_type}
                    onChange={(e) =>
                      setFormData({ ...formData, network_type: e.target.value })
                    }
                  >
                    <option value="bridge">Bridge</option>
                    <option value="network">NAT Network</option>
                  </select>
                </div>

                <div>
                  <label className="label">Network Source</label>
                  <input
                    type="text"
                    className="input"
                    value={formData.network_source}
                    onChange={(e) =>
                      setFormData({ ...formData, network_source: e.target.value })
                    }
                    placeholder={formData.network_type === "bridge" ? "br0" : "default"}
                  />
                  <p className="text-xs text-gray-500 mt-1">
                    {formData.network_type === "bridge"
                      ? "Bridge interface name (e.g., br0, virbr0)"
                      : "Libvirt network name (e.g., default)"}
                  </p>
                </div>
              </div>

              <div>
                <label className="label">Key Pairs</label>
                <div className="flex gap-4 mb-3">
                  <label className="flex items-center">
                    <input
                      type="radio"
                      name="keypairMethod"
                      value="select"
                      checked={keypairInputMethod === "select"}
                      onChange={(e) => setKeypairInputMethod(e.target.value as "select" | "upload" | "manual")}
                      className="mr-2"
                    />
                    <span className="text-sm">Select Existing</span>
                  </label>
                  <label className="flex items-center">
                    <input
                      type="radio"
                      name="keypairMethod"
                      value="upload"
                      checked={keypairInputMethod === "upload"}
                      onChange={(e) => setKeypairInputMethod(e.target.value as "select" | "upload" | "manual")}
                      className="mr-2"
                    />
                    <span className="text-sm">Upload File</span>
                  </label>
                  <label className="flex items-center">
                    <input
                      type="radio"
                      name="keypairMethod"
                      value="manual"
                      checked={keypairInputMethod === "manual"}
                      onChange={(e) => setKeypairInputMethod(e.target.value as "select" | "upload" | "manual")}
                      className="mr-2"
                    />
                    <span className="text-sm">Manual Input</span>
                  </label>
                </div>

                {keypairInputMethod === "select" && (
                  <select
                    className="input"
                    value={formData.keypair_ids[0] || ""}
                    onChange={(e) => {
                      setFormData({
                        ...formData,
                        keypair_ids: e.target.value ? [e.target.value] : []
                      });
                    }}
                  >
                    <option value="">Select Key Pair (Optional)</option>
                    {keypairs.map((keypair) => (
                      <option key={keypair.id} value={keypair.id}>
                        {keypair.name}
                      </option>
                    ))}
                  </select>
                )}

                {keypairInputMethod === "upload" && (
                  <input
                    type="file"
                    className="input"
                    accept=".pub,.pem,.key"
                    onChange={(e) => {
                      const file = e.target.files?.[0];
                      if (file) {
                        const reader = new FileReader();
                        reader.onload = (event) => {
                          const content = event.target?.result as string;
                          setManualPublicKey(content);
                        };
                        reader.readAsText(file);
                      }
                    }}
                  />
                )}

                {keypairInputMethod === "manual" && (
                  <textarea
                    className="input font-mono text-xs"
                    rows={4}
                    value={manualPublicKey}
                    onChange={(e) => setManualPublicKey(e.target.value)}
                    placeholder="ssh-rsa AAAAB3NzaC1yc2E..."
                  />
                )}
              </div>
            </div>
          )}

          {/* Step 2: User and System Configuration */}
          {currentStep === 1 && (
            <div className="space-y-4">
              <h3 className="font-semibold text-gray-900">User and System Configuration</h3>

              <div>
                <label className="label">Hostname</label>
                <input
                  type="text"
                  className="input"
                  value={formData.hostname}
                  onChange={(e) =>
                    setFormData({ ...formData, hostname: e.target.value })
                  }
                  placeholder="my-server"
                />
              </div>

              <div>
                <label className="label">Timezone</label>
                <select
                  className="input"
                  value={formData.timezone}
                  onChange={(e) =>
                    setFormData({ ...formData, timezone: e.target.value })
                  }
                >
                  <option value="Asia/Shanghai">Asia/Shanghai</option>
                  <option value="UTC">UTC</option>
                  <option value="America/New_York">America/New_York</option>
                  <option value="Europe/London">Europe/London</option>
                  <option value="Asia/Tokyo">Asia/Tokyo</option>
                </select>
              </div>

              <div className="border-t pt-4">
                <h4 className="font-medium text-gray-900 mb-3">Create User (Optional)</h4>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="label">Username</label>
                    <input
                      type="text"
                      className="input"
                      value={formData.username}
                      onChange={(e) =>
                        setFormData({ ...formData, username: e.target.value })
                      }
                      placeholder="ubuntu"
                    />
                  </div>

                  <div>
                    <label className="label">Password</label>
                    <input
                      type="password"
                      className="input"
                      value={formData.user_password}
                      onChange={(e) =>
                        setFormData({ ...formData, user_password: e.target.value })
                      }
                      placeholder="User password"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4 mt-4">
                  <div>
                    <label className="label">Groups</label>
                    <input
                      type="text"
                      className="input"
                      value={formData.user_groups}
                      onChange={(e) =>
                        setFormData({ ...formData, user_groups: e.target.value })
                      }
                      placeholder="sudo"
                    />
                  </div>

                  <div>
                    <label className="label">Shell</label>
                    <select
                      className="input"
                      value={formData.user_shell}
                      onChange={(e) =>
                        setFormData({ ...formData, user_shell: e.target.value })
                      }
                    >
                      <option value="/bin/bash">/bin/bash</option>
                      <option value="/bin/sh">/bin/sh</option>
                      <option value="/bin/zsh">/bin/zsh</option>
                    </select>
                  </div>
                </div>

                <div className="mt-4">
                  <label className="label">Sudo Permissions</label>
                  <input
                    type="text"
                    className="input"
                    value={formData.user_sudo}
                    onChange={(e) =>
                      setFormData({ ...formData, user_sudo: e.target.value })
                    }
                    placeholder="ALL=(ALL) NOPASSWD:ALL"
                  />
                </div>
              </div>

              <div className="flex items-center">
                <input
                  type="checkbox"
                  id="disable_root"
                  checked={formData.disable_root}
                  onChange={(e) =>
                    setFormData({ ...formData, disable_root: e.target.checked })
                  }
                  className="mr-2"
                />
                <label htmlFor="disable_root" className="text-sm text-gray-700">
                  Disable root login
                </label>
              </div>
            </div>
          )}

          {/* Step 3: Advanced Configuration */}
          {currentStep === 2 && (
            <div className="space-y-4">
              <h3 className="font-semibold text-gray-900">Advanced Configuration</h3>

              <div>
                <label className="label">Packages to Install</label>
                <textarea
                  className="input"
                  rows={3}
                  value={formData.packages.join("\n")}
                  onChange={(e) => {
                    const pkgs = e.target.value
                      .split("\n")
                      .map((s) => s.trim())
                      .filter(Boolean);
                    setFormData({ ...formData, packages: pkgs });
                  }}
                  placeholder="One package per line, e.g.:&#10;nginx&#10;git&#10;curl"
                />
              </div>

              <div>
                <label className="label">Run Commands</label>
                <textarea
                  className="input"
                  rows={4}
                  value={formData.run_cmd.join("\n")}
                  onChange={(e) => {
                    const cmds = e.target.value
                      .split("\n")
                      .map((s) => s.trim())
                      .filter(Boolean);
                    setFormData({ ...formData, run_cmd: cmds });
                  }}
                  placeholder="One command per line, e.g.:&#10;systemctl enable nginx&#10;systemctl start nginx"
                />
              </div>

              <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                <p className="text-sm text-blue-800">
                  <strong>Note:</strong> Advanced configuration is optional. You can skip and create the instance directly.
                </p>
              </div>
            </div>
          )}

          {/* Buttons */}
          <div className="flex justify-between pt-4 border-t">
            <div>
              {currentStep > 0 && (
                <button
                  type="button"
                  onClick={() => setCurrentStep(currentStep - 1)}
                  className="btn-secondary"
                >
                  Previous
                </button>
              )}
            </div>

            <div className="flex gap-3">
              <button
                type="button"
                onClick={() => {
                  setIsCreateModalOpen(false);
                  setCurrentStep(0);
                }}
                className="btn-secondary"
              >
                Cancel
              </button>

              {currentStep < 2 ? (
                <button
                  type="button"
                  onClick={() => setCurrentStep(currentStep + 1)}
                  className="btn-primary"
                  disabled={currentStep === 0 && (!formData.node_name || !formData.pool_name)}
                >
                  Next
                </button>
              ) : (
                <button type="submit" className="btn-primary">
                  Create Instance
                </button>
              )}
            </div>
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
      />
    </DashboardLayout>
  );
}
