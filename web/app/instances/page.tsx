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
  vcpus: number;
  memory_mb: number;
  image_id?: string;
  volume_id?: string;
  created_at: string;
  domain_uuid?: string;
  domain_name?: string;
}

interface Image {
  id: string;
  name: string;
  state: string;
}

interface KeyPair {
  id: string;
  name: string;
  fingerprint: string;
}

export default function InstancesPage() {
  const toast = useToast();
  const [instances, setInstances] = useState<Instance[]>([]);
  const [images, setImages] = useState<Image[]>([]);
  const [keypairs, setKeypairs] = useState<KeyPair[]>([]);
  const [loading, setLoading] = useState(true);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [instanceToDelete, setInstanceToDelete] = useState<string>("");
  const [searchQuery, setSearchQuery] = useState("");
  const [currentStep, setCurrentStep] = useState(0);
  const [keypairInputMethod, setKeypairInputMethod] = useState<"select" | "upload" | "manual">("select");
  const [manualPublicKey, setManualPublicKey] = useState("");
  const [formData, setFormData] = useState({
    // 基础配置
    image_id: "",
    size_gb: 20,
    memory_mb: 2048,
    vcpus: 2,
    keypair_ids: [] as string[],

    // UserData 配置
    hostname: "",
    timezone: "Asia/Shanghai",
    disable_root: false,
    packages: [] as string[],
    run_cmd: [] as string[],

    // 用户配置
    username: "",
    user_password: "",
    user_sudo: "ALL=(ALL) NOPASSWD:ALL",
    user_groups: "sudo",
    user_shell: "/bin/bash",
  });

  const filteredInstances = useMemo(() => {
    if (!searchQuery) return instances;

    const query = searchQuery.toLowerCase();
    return instances.filter(
      (instance) =>
        instance.id.toLowerCase().includes(query) ||
        instance.name?.toLowerCase().includes(query) ||
        instance.state.toLowerCase().includes(query)
    );
  }, [instances, searchQuery]);

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

  const fetchImages = async () => {
    try {
      const response = await fetch("/api/images/describe", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });
      if (response.ok) {
        const data = await response.json();
        setImages(data.images || []);
      }
    } catch (error) {
      console.error("Failed to fetch images:", error);
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
    fetchInstances();
    fetchImages();
    fetchKeypairs();
  }, []);

  const handleCreateInstance = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      // 如果是手动输入或上传公钥，先创建密钥对
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

      // 构建 UserData 配置
      const userData: any = {};

      // 构建结构化 UserData
      const structuredUserData: any = {};

      if (formData.hostname) {
        structuredUserData.hostname = formData.hostname;
      }

      if (formData.timezone) {
        structuredUserData.timezone = formData.timezone;
      }

      structuredUserData.disable_root = formData.disable_root;

      // 添加用户配置
      if (formData.username) {
        structuredUserData.users = [{
          name: formData.username,
          plain_text_passwd: formData.user_password || undefined,
          sudo: formData.user_sudo,
          groups: formData.user_groups,
          shell: formData.user_shell,
          ssh_authorized_keys: formData.keypair_ids.length > 0 ? formData.keypair_ids : undefined,
        }];
      }

      // 添加软件包
      if (formData.packages.length > 0) {
        structuredUserData.packages = formData.packages;
      }

      // 添加运行命令
      if (formData.run_cmd.length > 0) {
        structuredUserData.run_cmd = formData.run_cmd;
      }

      // 只有当有配置时才添加 user_data
      const hasUserData = Object.keys(structuredUserData).length > 0;
      if (hasUserData) {
        userData.structured_user_data = structuredUserData;
      }

      const requestBody: any = {
        image_id: formData.image_id,
        size_gb: formData.size_gb,
        memory_mb: formData.memory_mb,
        vcpus: formData.vcpus,
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
        fetchInstances();
        fetchKeypairs(); // 刷新密钥对列表
        // 重置表单
        setFormData({
          image_id: "",
          size_gb: 20,
          memory_mb: 2048,
          vcpus: 2,
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

  const handleAction = async (instanceId: string, action: string) => {
    try {
      const response = await fetch(`/api/instances/${action}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ instanceIDs: [instanceId] }),
      });

      if (response.ok) {
        const actionName = action === "start" ? "started" : action === "stop" ? "stopped" : "rebooted";
        toast.success(`Instance ${actionName} successfully!`);

        // 延迟刷新以等待后端状态更新
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

  const handleDeleteClick = (instanceId: string) => {
    setInstanceToDelete(instanceId);
    setIsDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = async () => {
    try {
      const response = await fetch("/api/instances/terminate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ instanceIDs: [instanceToDelete] }),
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
              onClick={() => handleAction(instance.id, "reboot")}
              className="p-2 text-gray-600 hover:text-blue-600 transition-colors"
              title="Reboot"
            >
              <RefreshCw size={16} />
            </button>
            <button
              onClick={() => handleDeleteClick(instance.id)}
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
        <>
          <div className="mb-4">
            <SearchFilter
              onSearch={setSearchQuery}
              placeholder="Search instances by ID, name, or state..."
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
        <form onSubmit={handleCreateInstance} className="space-y-6">
          {/* 步骤指示器 */}
          <div className="flex items-center justify-between border-b pb-4">
            {["Basic", "User & System", "Advanced"].map((step, index) => (
              <div key={step} className="flex items-center">
                <div
                  className={`flex items-center justify-center w-8 h-8 rounded-full border-2 transition-colors ${
                    currentStep >= index
                      ? "border-accent bg-accent text-white"
                      : "border-gray-300 text-gray-400"
                  }`}
                >
                  {index + 1}
                </div>
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

          {/* Step 1: 基础配置 */}
          {currentStep === 0 && (
            <div className="space-y-4">
              <h3 className="font-semibold text-gray-900">基础配置</h3>

              <div>
                <label className="label">镜像 (Image) *</label>
                <select
                  className="input"
                  value={formData.image_id}
                  onChange={(e) =>
                    setFormData({ ...formData, image_id: e.target.value })
                  }
                  required
                >
                  <option value="">请选择镜像</option>
                  {images.map((image) => (
                    <option key={image.id} value={image.id}>
                      {image.name} ({image.id})
                    </option>
                  ))}
                </select>
                <p className="text-xs text-gray-500 mt-1">
                  选择用于创建实例的系统镜像
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
                  <label className="label">内存 (Memory MB) *</label>
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
                <label className="label">磁盘大小 (Disk Size GB) *</label>
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

              <div>
                <label className="label">密钥对 (Key Pairs)</label>

                {/* 输入方式选择 */}
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
                    <span className="text-sm">选择已有</span>
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
                    <span className="text-sm">上传文件</span>
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
                    <span className="text-sm">手动输入</span>
                  </label>
                </div>

                {/* 选择已有密钥对 */}
                {keypairInputMethod === "select" && (
                  <>
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
                      <option value="">请选择密钥对（可选）</option>
                      {keypairs.map((keypair) => (
                        <option key={keypair.id} value={keypair.id}>
                          {keypair.name}
                        </option>
                      ))}
                    </select>
                    <p className="text-xs text-gray-500 mt-1">
                      选择用于SSH登录的密钥对
                    </p>
                  </>
                )}

                {/* 上传公钥文件 */}
                {keypairInputMethod === "upload" && (
                  <>
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
                    <p className="text-xs text-gray-500 mt-1">
                      上传公钥文件 (*.pub, *.pem, *.key)
                    </p>
                  </>
                )}

                {/* 手动输入公钥 */}
                {keypairInputMethod === "manual" && (
                  <>
                    <textarea
                      className="input font-mono text-xs"
                      rows={4}
                      value={manualPublicKey}
                      onChange={(e) => setManualPublicKey(e.target.value)}
                      placeholder="ssh-rsa AAAAB3NzaC1yc2E..."
                    />
                    <p className="text-xs text-gray-500 mt-1">
                      粘贴公钥内容 (通常来自 ~/.ssh/id_rsa.pub)
                    </p>
                  </>
                )}
              </div>
            </div>
          )}

          {/* Step 2: 用户和系统配置 */}
          {currentStep === 1 && (
            <div className="space-y-4">
              <h3 className="font-semibold text-gray-900">用户和系统配置</h3>

              <div>
                <label className="label">主机名 (Hostname)</label>
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
                <label className="label">时区 (Timezone)</label>
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
                <h4 className="font-medium text-gray-900 mb-3">创建用户 (Optional)</h4>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="label">用户名 (Username)</label>
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
                    <label className="label">密码 (Password)</label>
                    <input
                      type="password"
                      className="input"
                      value={formData.user_password}
                      onChange={(e) =>
                        setFormData({ ...formData, user_password: e.target.value })
                      }
                      placeholder="用户密码"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4 mt-4">
                  <div>
                    <label className="label">用户组 (Groups)</label>
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
                  <label className="label">Sudo 权限</label>
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
                  禁用 root 登录 (Disable root login)
                </label>
              </div>
            </div>
          )}

          {/* Step 3: 高级配置 */}
          {currentStep === 2 && (
            <div className="space-y-4">
              <h3 className="font-semibold text-gray-900">高级配置</h3>

              <div>
                <label className="label">安装软件包 (Packages)</label>
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
                  placeholder="每行一个软件包名称,例如:&#10;nginx&#10;git&#10;curl"
                />
                <p className="text-xs text-gray-500 mt-1">
                  实例创建后将自动安装这些软件包,每行一个
                </p>
              </div>

              <div>
                <label className="label">启动命令 (Run Commands)</label>
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
                  placeholder="每行一个命令,例如:&#10;systemctl enable nginx&#10;systemctl start nginx&#10;echo 'Setup complete'"
                />
                <p className="text-xs text-gray-500 mt-1">
                  实例启动后将执行这些命令,每行一个
                </p>
              </div>

              <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                <p className="text-sm text-blue-800">
                  <strong>提示:</strong> 高级配置是可选的。如果不需要,可以直接点击&ldquo;创建实例&rdquo;按钮。
                </p>
              </div>
            </div>
          )}

          {/* 按钮 */}
          <div className="flex justify-between pt-4 border-t">
            <div>
              {currentStep > 0 && (
                <button
                  type="button"
                  onClick={() => setCurrentStep(currentStep - 1)}
                  className="btn-secondary"
                >
                  上一步
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
                取消
              </button>

              {currentStep < 2 ? (
                <button
                  type="button"
                  onClick={() => setCurrentStep(currentStep + 1)}
                  className="btn-primary"
                >
                  下一步
                </button>
              ) : (
                <button type="submit" className="btn-primary">
                  创建实例
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
