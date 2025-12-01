import { useState, useEffect, useCallback } from "react";
import Header from "@/components/Header";
import Table from "@/components/Table";
import { useToast } from "@/components/ToastContainer";
import { apiPost } from "@/lib/api";
import {
  Package,
  Server,
  HardDrive,
  Tag,
  Trash2,
  Plus,
  RefreshCw,
} from "lucide-react";

interface Template {
  id: string;
  name: string;
  description: string;
  node_name: string;
  pool_name: string;
  volume_name: string;
  size_gb: number;
  format: string;
  created_at: string;
  tags: string[];
}

interface ListTemplatesResponse {
  templates: Template[];
}

interface DownloadTask {
  id: string;
  node_name: string;
  pool_name: string;
  volume_name: string;
  status: string;
  error?: string;
}

interface RegisterTemplateResponse {
  template?: Template;
  download_task?: DownloadTask;
}

interface GetDownloadTaskResponse {
  task: DownloadTask;
}

interface ListDownloadTasksResponse {
  tasks: DownloadTask[] | null;
}

interface NodeItem {
  name: string;
  state: string;
}

interface StoragePoolItem {
  name: string;
}

const initialRegisterForm = {
  nodeName: "",
  poolName: "",
  volumeName: "",
  name: "",
  description: "",
  tags: "",
  osName: "",
  osVersion: "",
  osArch: "x86_64",
  cloudInit: true,
  virtio: true,
  qga: false,
  sourceType: "existing_volume",
  cloudUrl: "",
};

export default function TemplatesPage() {
  const toast = useToast();
  const [templates, setTemplates] = useState<Template[]>([]);
  const [loading, setLoading] = useState(true);
  const [filters, setFilters] = useState({ nodeName: "", poolName: "" });
  const [registerForm, setRegisterForm] = useState(initialRegisterForm);
  const [showRegisterModal, setShowRegisterModal] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [templateToDelete, setTemplateToDelete] = useState<Template | null>(null);
  const [deleteVolume, setDeleteVolume] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [nodes, setNodes] = useState<NodeItem[]>([]);
  const [loadingNodes, setLoadingNodes] = useState(false);
  const [filterPools, setFilterPools] = useState<StoragePoolItem[]>([]);
  const [registerPools, setRegisterPools] = useState<StoragePoolItem[]>([]);
  const [loadingFilterPools, setLoadingFilterPools] = useState(false);
  const [loadingRegisterPools, setLoadingRegisterPools] = useState(false);
  const [downloadTask, setDownloadTask] = useState<DownloadTask | null>(null);
  const [downloadTasks, setDownloadTasks] = useState<DownloadTask[]>([]);

  const fetchTemplates = useCallback(async () => {
    setLoading(true);
    try {
      const response = await apiPost<ListTemplatesResponse>("/api/list-templates", {
        node_name: filters.nodeName,
        pool_name: filters.poolName,
      });
      setTemplates(response.templates || []);
    } catch (error: any) {
      console.error("Failed to load templates:", error);
      toast.error(error?.message || "Failed to load templates");
    } finally {
      setLoading(false);
    }
  }, [filters, toast]);

  const fetchNodes = useCallback(async () => {
    setLoadingNodes(true);
    try {
      const response = await apiPost<{ nodes: NodeItem[] }>("/api/list-nodes", {});
      const nodeList = response.nodes || [];
      setNodes(nodeList);
      const defaultNodeName = nodeList[0]?.name || "";
      setRegisterForm((prev) => {
        if (prev.nodeName && nodeList.some((node) => node.name === prev.nodeName)) {
          return prev;
        }
        return {
          ...prev,
          nodeName: defaultNodeName,
        };
      });
      setFilters((prev) => {
        if (prev.nodeName && nodeList.some((node) => node.name === prev.nodeName)) {
          return prev;
        }
        return {
          ...prev,
          nodeName: defaultNodeName,
        };
      });
    } catch (error: any) {
      console.error("Failed to load nodes:", error);
      toast.error(error?.message || "Failed to load nodes");
    } finally {
      setLoadingNodes(false);
    }
  }, [toast]);

  const fetchFilterPools = useCallback(
    async (nodeName: string) => {
      if (!nodeName) {
        setFilterPools([]);
        setFilters((prev) => ({ ...prev, poolName: "" }));
        return;
      }
      setLoadingFilterPools(true);
      try {
        const response = await apiPost<{ pools: StoragePoolItem[] }>(
          "/api/list-storage-pools",
          {
            node_name: nodeName,
          }
        );
        const pools = response.pools || [];
        setFilterPools(pools);
        setFilters((prev) => ({
          ...prev,
          poolName: pools.some((pool) => pool.name === prev.poolName)
            ? prev.poolName
            : pools[0]?.name || "",
        }));
      } catch (error: any) {
        console.error("Failed to load storage pools:", error);
        toast.error(error?.message || "Failed to load storage pools");
      } finally {
        setLoadingFilterPools(false);
      }
    },
    [toast]
  );

  const fetchRegisterPools = useCallback(
    async (nodeName: string) => {
      if (!nodeName) {
        setRegisterPools([]);
        setRegisterForm((prev) => ({ ...prev, poolName: "" }));
        return;
      }
      setLoadingRegisterPools(true);
      try {
        const response = await apiPost<{ pools: StoragePoolItem[] }>(
          "/api/list-storage-pools",
          {
            node_name: nodeName,
          }
        );
        const pools = response.pools || [];
        setRegisterPools(pools);
        setRegisterForm((prev) => ({
          ...prev,
          poolName:
            prev.poolName && pools.some((pool) => pool.name === prev.poolName)
              ? prev.poolName
              : pools[0]?.name || "",
        }));
      } catch (error: any) {
        console.error("Failed to load storage pools:", error);
        toast.error(error?.message || "Failed to load storage pools");
      } finally {
        setLoadingRegisterPools(false);
      }
    },
    [toast]
  );

  useEffect(() => {
    fetchTemplates();
  }, [fetchTemplates]);

  useEffect(() => {
    fetchNodes();
  }, [fetchNodes]);

  useEffect(() => {
    fetchRegisterPools(registerForm.nodeName);
  }, [registerForm.nodeName, fetchRegisterPools]);

  useEffect(() => {
    fetchFilterPools(filters.nodeName);
  }, [filters.nodeName, fetchFilterPools]);

  const formattedDate = (value?: string) => {
    if (!value) return "N/A";
    try {
      return new Date(value).toLocaleString();
    } catch {
      return value;
    }
  };

  const openRegisterModal = () => {
    setRegisterForm((prev) => ({
      ...initialRegisterForm,
      nodeName: prev.nodeName || nodes[0]?.name || "",
      poolName: prev.poolName,
    }));
    setShowRegisterModal(true);
  };

  const pollDownloadTask = useCallback(async (taskId: string) => {
    try {
      const response = await apiPost<GetDownloadTaskResponse>("/api/get-download-task", {
        task_id: taskId,
      });

      const task = response.task;
      setDownloadTask(task);
      setDownloadTasks((prev) => {
        const exists = prev.some((t) => t.id === task.id);
        if (exists) {
          return prev.map((t) => (t.id === task.id ? task : t));
        }
        return [...prev, task];
      });

      if (task.status === "completed") {
        toast.success(`Download completed! Template "${task.volume_name}" registered successfully.`);
        setDownloadTask(null);
        setShowRegisterModal(false);
        setDownloadTasks((prev) => prev.filter((t) => t.id !== taskId));
        fetchTemplates();
        setRegisterForm({
          ...initialRegisterForm,
          nodeName: registerForm.nodeName,
          poolName: registerForm.poolName,
        });
      } else if (task.status === "failed") {
        toast.error(`Download failed: ${task.error || "Unknown error"}`);
        setDownloadTask(null);
        setDownloadTasks((prev) => prev.filter((t) => t.id !== taskId));
      } else {
        setTimeout(() => pollDownloadTask(taskId), 5000);
      }
    } catch (error: any) {
      console.error("Failed to poll download task:", error);
      setDownloadTasks((prev) => prev.filter((t) => t.id !== taskId));
      setDownloadTask(null);
    }
  }, [fetchTemplates, registerForm.nodeName, registerForm.poolName, toast]);

  const fetchAndResumeDownloadTasks = useCallback(async () => {
    try {
      const response = await apiPost<ListDownloadTasksResponse>("/api/list-download-tasks", {});
      const tasks = response.tasks || [];
      setDownloadTasks(tasks);

      for (const task of tasks) {
        if (task.status === "pending" || task.status === "running") {
          pollDownloadTask(task.id);
        }
      }
    } catch (error: any) {
      console.error("Failed to fetch download tasks:", error);
    }
  }, [pollDownloadTask]);

  useEffect(() => {
    fetchAndResumeDownloadTasks();
  }, [fetchAndResumeDownloadTasks]);

  const handleRegisterTemplate = async () => {
    if (!registerForm.name || !registerForm.volumeName) {
      toast.error("Template name and volume name are required");
      return;
    }
    if (!registerForm.nodeName || !registerForm.poolName) {
      toast.error("Node and storage pool selection is required");
      return;
    }
    if (
      registerForm.sourceType === "cloud_image" &&
      !registerForm.cloudUrl.trim()
    ) {
      toast.error("Download URL is required for cloud images");
      return;
    }

    setSubmitting(true);
    try {
      const source =
        registerForm.sourceType === "cloud_image"
          ? {
              type: "url",
              url: registerForm.cloudUrl.trim(),
            }
          : undefined;

      const payload = {
        node_name: registerForm.nodeName,
        pool_name: registerForm.poolName,
        volume_name: registerForm.volumeName,
        name: registerForm.name,
        description: registerForm.description,
        tags: registerForm.tags
          .split(",")
          .map((tag) => tag.trim())
          .filter(Boolean),
        os: {
          name: registerForm.osName,
          version: registerForm.osVersion,
          arch: registerForm.osArch,
        },
        features: {
          cloud_init: registerForm.cloudInit,
          virtio: registerForm.virtio,
          qemu_guest_agent: registerForm.qga,
        },
        source,
      };

      const response = await apiPost<RegisterTemplateResponse>(
        "/api/register-template",
        payload
      );

      if (response?.download_task) {
        const task = response.download_task;
        setDownloadTask(task);
        setDownloadTasks((prev) => [...prev, task]);
        toast.info("Download started. This may take a few minutes...");
        setTimeout(() => pollDownloadTask(task.id), 5000);
      } else if (response?.template) {
        setTemplates((prev) => [response.template!, ...prev]);
        toast.success(`Template ${registerForm.name} registered`);
        setShowRegisterModal(false);
        setRegisterForm({
          ...initialRegisterForm,
          nodeName: registerForm.nodeName,
          poolName: registerForm.poolName,
        });
      } else {
        fetchTemplates();
        toast.success(`Template ${registerForm.name} registered`);
        setShowRegisterModal(false);
      }
    } catch (error: any) {
      console.error("Failed to register template:", error);
      toast.error(error?.message || "Failed to register template");
    } finally {
      setSubmitting(false);
    }
  };

  const confirmDeleteTemplate = (template: Template) => {
    setTemplateToDelete(template);
    setDeleteVolume(false);
  };

  const handleDeleteTemplate = async () => {
    if (!templateToDelete) return;
    setDeleting(true);
    try {
      await apiPost("/api/delete-template", {
        template_id: templateToDelete.id,
        node_name: templateToDelete.node_name,
        delete_volume: deleteVolume,
      });
      toast.success(`Template ${templateToDelete.name} deleted`);
      setTemplateToDelete(null);
      fetchTemplates();
    } catch (error: any) {
      console.error("Failed to delete template:", error);
      toast.error(error?.message || "Failed to delete template");
    } finally {
      setDeleting(false);
    }
  };

  const columns = [
    {
      key: "name",
      label: "Template",
      render: (_: unknown, row: Template) => (
        <div className="flex flex-col">
          <div className="flex items-center gap-2">
            <Package className="w-4 h-4 text-blue-600" />
            <span className="font-medium">{row.name}</span>
          </div>
          <p className="text-xs text-gray-500 mt-1">{row.description || "No description"}</p>
        </div>
      ),
    },
    {
      key: "node_name",
      label: "Node",
      render: (value: unknown) => (
        <div className="flex items-center gap-2">
          <Server className="w-4 h-4 text-gray-500" />
          <span className="font-mono text-sm">{String(value || "local")}</span>
        </div>
      ),
    },
    {
      key: "pool_name",
      label: "Storage",
      render: (_: unknown, row: Template) => (
        <div>
          <div className="flex items-center gap-2">
            <HardDrive className="w-4 h-4 text-gray-500" />
            <span className="font-mono text-sm">{row.pool_name}</span>
          </div>
          <p className="text-xs text-gray-500 mt-1">{row.volume_name}</p>
        </div>
      ),
    },
    {
      key: "size_gb",
      label: "Size",
      render: (value: unknown, row: Template) => (
        <div>
          <div className="font-mono text-sm">{Number(value || 0)} GB</div>
          <p className="text-xs text-gray-500">{row.format?.toUpperCase()}</p>
        </div>
      ),
    },
    {
      key: "tags",
      label: "Tags",
      render: (value: unknown) => {
        const tags = (value as string[]) || [];
        if (!tags.length) {
          return <span className="text-xs text-gray-400">No tags</span>;
        }
        return (
          <div className="flex flex-wrap gap-1">
            {tags.map((tag) => (
              <span
                key={tag}
                className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-blue-50 text-blue-700 text-xs"
              >
                <Tag className="w-3 h-3" />
                {tag}
              </span>
            ))}
          </div>
        );
      },
    },
    {
      key: "created_at",
      label: "Created",
      render: (value: unknown) => (
        <span className="text-sm text-gray-600">{formattedDate(String(value || ""))}</span>
      ),
    },
    {
      key: "actions",
      label: "",
      render: (_: unknown, row: Template) => (
        <button
          onClick={() => confirmDeleteTemplate(row)}
          className="btn-danger text-xs flex items-center gap-1"
        >
          <Trash2 className="w-3 h-3" />
          Delete
        </button>
      ),
    },
  ];

  return (
    <>
      <div className="space-y-6">
        <Header
          title="Templates"
          description="Register storage volumes as reusable VM templates"
          onRefresh={fetchTemplates}
          action={
            <button className="btn-primary flex items-center gap-2" onClick={openRegisterModal}>
              <Plus className="w-4 h-4" />
              Register Template
            </button>
          }
        />

        {downloadTasks.length > 0 && (
          <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
            <h3 className="text-sm font-medium text-blue-800 mb-2 flex items-center gap-2">
              <RefreshCw className="w-4 h-4 animate-spin" />
              Active Downloads ({downloadTasks.length})
            </h3>
            <div className="space-y-2">
              {downloadTasks.map((task) => (
                <div key={task.id} className="flex items-center gap-3 text-sm">
                  <span className="text-blue-600 font-mono">{task.volume_name}</span>
                  <span className="text-blue-500">→</span>
                  <span className="text-blue-600">{task.node_name}/{task.pool_name}</span>
                  <span className="text-xs text-blue-500 capitalize">({task.status})</span>
                </div>
              ))}
            </div>
          </div>
        )}

        <div className="card space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="text-sm text-gray-600">Node Filter</label>
              <select
                value={filters.nodeName}
                onChange={(e) =>
                  setFilters((prev) => ({
                    ...prev,
                    nodeName: e.target.value,
                    poolName: "",
                  }))
                }
                className="w-full px-3 py-2 border border-gray-200 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                disabled={loadingNodes}
              >
                <option value="">All Nodes</option>
                {nodes.map((node) => (
                  <option key={node.name} value={node.name}>
                    {node.name}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="text-sm text-gray-600">Pool Filter</label>
              <select
                value={filters.poolName}
                onChange={(e) =>
                  setFilters((prev) => ({ ...prev, poolName: e.target.value }))
                }
                className="w-full px-3 py-2 border border-gray-200 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                disabled={!filters.nodeName || loadingFilterPools}
              >
                <option value="">All Pools</option>
                {filterPools.map((pool) => (
                  <option key={pool.name} value={pool.name}>
                    {pool.name}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <p className="text-xs text-gray-500 flex items-center gap-2">
            <RefreshCw className="w-4 h-4" />
            Leave filters blank to list templates across all nodes.
          </p>
        </div>

        {loading ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-center">
              <RefreshCw className="w-8 h-8 animate-spin text-primary mx-auto mb-2" />
              <p className="text-gray-600">Loading templates...</p>
            </div>
          </div>
        ) : (
          <Table data={templates} columns={columns} keyField="id" emptyMessage="No templates" />
        )}
      </div>

      {showRegisterModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full mx-4">
            <div className="p-6 space-y-5">
              <h2 className="text-xl font-semibold">Register Template</h2>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Node <span className="text-red-500">*</span>
                  </label>
                  <select
                    value={registerForm.nodeName}
                    onChange={(e) =>
                      setRegisterForm({
                        ...registerForm,
                        nodeName: e.target.value,
                        poolName: "",
                      })
                    }
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500"
                    disabled={nodes.length === 0 || loadingNodes}
                  >
                    {nodes.length === 0 && <option value="">No nodes available</option>}
                    {nodes.map((node) => (
                      <option key={node.name} value={node.name}>
                        {node.name} ({node.state})
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Storage Pool <span className="text-red-500">*</span>
                  </label>
                  <select
                    value={registerForm.poolName}
                    onChange={(e) =>
                      setRegisterForm({ ...registerForm, poolName: e.target.value })
                    }
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500"
                    disabled={registerPools.length === 0 || loadingRegisterPools}
                  >
                    {registerPools.length === 0 && (
                      <option value="">No pools available</option>
                    )}
                    {registerPools.map((pool) => (
                      <option key={pool.name} value={pool.name}>
                      {pool.name}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="md:col-span-2">
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Volume Name <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="text"
                    value={registerForm.volumeName}
                    onChange={(e) =>
                      setRegisterForm({ ...registerForm, volumeName: e.target.value })
                    }
                    placeholder="existing volume, e.g., ubuntu.qcow2"
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 font-mono text-sm"
                  />
                  <p className="text-xs text-gray-500 mt-1">
                    The volume must already exist inside the selected pool.
                  </p>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Source Type
                  </label>
                  <select
                    value={registerForm.sourceType}
                    onChange={(e) =>
                      setRegisterForm({ ...registerForm, sourceType: e.target.value })
                    }
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="existing_volume">Existing Volume</option>
                    <option value="cloud_image">Cloud Image (URL)</option>
                  </select>
                </div>
                {registerForm.sourceType === "cloud_image" && (
                  <div className="md:col-span-2">
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      Download URL <span className="text-red-500">*</span>
                    </label>
                    <input
                      type="text"
                      value={registerForm.cloudUrl}
                      onChange={(e) =>
                        setRegisterForm({ ...registerForm, cloudUrl: e.target.value })
                      }
                      placeholder="https://cloud-images.example.com/image.qcow2"
                      className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 font-mono text-sm"
                    />
                    <p className="text-xs text-gray-500 mt-1">
                      Provide the download URL for this cloud image.
                    </p>
                  </div>
                )}
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Template Name <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="text"
                    value={registerForm.name}
                    onChange={(e) =>
                      setRegisterForm({ ...registerForm, name: e.target.value })
                    }
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Tags
                  </label>
                  <input
                    type="text"
                    value={registerForm.tags}
                    onChange={(e) =>
                      setRegisterForm({ ...registerForm, tags: e.target.value })
                    }
                    placeholder="comma separated"
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                <div className="md:col-span-2">
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Description
                  </label>
                  <textarea
                    value={registerForm.description}
                    onChange={(e) =>
                      setRegisterForm({ ...registerForm, description: e.target.value })
                    }
                    rows={3}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    OS Name
                  </label>
                  <input
                    type="text"
                    value={registerForm.osName}
                    onChange={(e) =>
                      setRegisterForm({ ...registerForm, osName: e.target.value })
                    }
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    OS Version
                  </label>
                  <input
                    type="text"
                    value={registerForm.osVersion}
                    onChange={(e) =>
                      setRegisterForm({ ...registerForm, osVersion: e.target.value })
                    }
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Architecture
                  </label>
                  <input
                    type="text"
                    value={registerForm.osArch}
                    onChange={(e) =>
                      setRegisterForm({ ...registerForm, osArch: e.target.value })
                    }
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium text-gray-700">
                    Features
                  </label>
                  <label className="flex items-center gap-2 text-sm text-gray-600">
                    <input
                      type="checkbox"
                      checked={registerForm.cloudInit}
                      onChange={(e) =>
                        setRegisterForm({ ...registerForm, cloudInit: e.target.checked })
                      }
                    />
                    Cloud-init ready
                  </label>
                  <label className="flex items-center gap-2 text-sm text-gray-600">
                    <input
                      type="checkbox"
                      checked={registerForm.virtio}
                      onChange={(e) =>
                        setRegisterForm({ ...registerForm, virtio: e.target.checked })
                      }
                    />
                    Virtio drivers
                  </label>
                  <label className="flex items-center gap-2 text-sm text-gray-600">
                    <input
                      type="checkbox"
                      checked={registerForm.qga}
                      onChange={(e) =>
                        setRegisterForm({ ...registerForm, qga: e.target.checked })
                      }
                    />
                    QEMU guest agent
                  </label>
                </div>
              </div>

              {downloadTask && (
                <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                  <div className="flex items-center gap-3">
                    <RefreshCw className="w-5 h-5 animate-spin text-blue-600" />
                    <div className="flex-1">
                      <p className="text-sm font-medium text-blue-800">
                        {downloadTask.status === "pending" && "Preparing download..."}
                        {downloadTask.status === "running" && "Downloading image..."}
                      </p>
                      <p className="text-xs text-blue-600 mt-1">
                        {downloadTask.volume_name} - This may take several minutes
                      </p>
                    </div>
                  </div>
                </div>
              )}

              <div className="flex gap-3 pt-4">
                <button
                  className="btn-secondary flex-1"
                  onClick={() => setShowRegisterModal(false)}
                  disabled={submitting || !!downloadTask}
                >
                  Cancel
                </button>
                <button
                  className="btn-primary flex-1"
                  onClick={handleRegisterTemplate}
                  disabled={submitting || !!downloadTask}
                >
                  {submitting ? "Starting..." : downloadTask ? "Downloading..." : "Register Template"}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {templateToDelete && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4">
            <div className="p-6 space-y-4">
              <h2 className="text-xl font-semibold text-red-600">Delete Template</h2>
              <p className="text-sm text-gray-600">
                Are you sure you want to delete template{" "}
                <span className="font-semibold">{templateToDelete.name}</span>? This action cannot
                be undone.
              </p>
              <label className="flex items-center gap-2 text-sm text-gray-600">
                <input
                  type="checkbox"
                  checked={deleteVolume}
                  onChange={(e) => setDeleteVolume(e.target.checked)}
                />
                Also delete backing volume ({templateToDelete.volume_name})
              </label>
              <div className="flex gap-3 pt-2">
                <button
                  className="btn-secondary flex-1"
                  onClick={() => setTemplateToDelete(null)}
                  disabled={deleting}
                >
                  Cancel
                </button>
                <button
                  className="btn-danger flex-1"
                  onClick={handleDeleteTemplate}
                  disabled={deleting}
                >
                  {deleting ? "Deleting..." : "Delete"}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

