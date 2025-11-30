"use client";

import { Suspense, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import DashboardLayout from "@/components/DashboardLayout";
import Header from "@/components/Header";
import Table from "@/components/Table";
import Modal from "@/components/Modal";
import ConfirmDialog from "@/components/ConfirmDialog";
import { useToast } from "@/components/ToastContainer";
import { RefreshCw, Plus, History, Trash2 } from "lucide-react";

interface Node {
  name: string;
  uri: string;
  status: string;
}

interface Instance {
  id: string;
  name: string;
  node_name: string;
}

interface Snapshot {
  id: string;
  name: string;
  vm_name: string;
  node_name: string;
  created_at?: string;
  state?: string;
  parent?: string;
  memory?: boolean;
  disk_only?: boolean;
  description?: string;
  disks?: { target?: string; path?: string; format?: string }[];
}

function SnapshotsContent() {
  const toast = useToast();
  const searchParams = useSearchParams();
  const [nodes, setNodes] = useState<Node[]>([]);
  const [instances, setInstances] = useState<Instance[]>([]);
  const [snapshots, setSnapshots] = useState<Snapshot[]>([]);
  const [selectedNode, setSelectedNode] = useState("");
  const [selectedVM, setSelectedVM] = useState("");
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [revertDialogOpen, setRevertDialogOpen] = useState(false);
  const [targetSnapshot, setTargetSnapshot] = useState<Snapshot | null>(null);
  const [createForm, setCreateForm] = useState({
    snapshot_name: "",
    description: "",
    with_memory: false,
  });

  // 用于跟踪初始化状态
  const initDoneRef = useRef(false);
  const lastParamsRef = useRef("");

  // 更新 URL 参数（使用 window.history 避免触发 React 重渲染）
  const updateURL = (node: string, vm: string) => {
    const params = new URLSearchParams();
    if (node) params.set("node", node);
    if (vm) params.set("vm", vm);
    const newURL = params.toString() ? `/snapshots?${params.toString()}` : "/snapshots";
    window.history.replaceState(null, "", newURL);
  };

  // 使用 useSearchParams 监听 URL 参数变化
  const urlNode = searchParams.get("node") || "";
  const urlVM = searchParams.get("vm") || "";

  // 监听 URL 参数变化并初始化
  useEffect(() => {
    const currentParams = `${urlNode}|${urlVM}`;

    // 如果参数没变化，跳过
    if (lastParamsRef.current === currentParams) {
      return;
    }

    // 如果已经初始化过了，跳过
    if (initDoneRef.current) {
      return;
    }

    // 如果参数为空，先保存当前参数，但不初始化，等待参数更新
    if (!urlNode && !urlVM) {
      lastParamsRef.current = currentParams;
      return;
    }

    // 参数有效，执行初始化
    lastParamsRef.current = currentParams;
    initDoneRef.current = true;

    // 直接使用 searchParams 的值进行初始化
    initializeData(urlNode, urlVM);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [urlNode, urlVM]); // 依赖 URL 参数变化

  // 初始化数据加载
  const initializeData = async (urlNode: string, urlVM: string) => {
    try {
      // 1. 获取节点列表
      const nodesRes = await fetch("/api/list-nodes", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });
      if (!nodesRes.ok) {
        toast.error("Failed to load nodes");
        return;
      }
      const nodesData = await nodesRes.json();
      const nodeList = nodesData.nodes || [];
      setNodes(nodeList);

      if (nodeList.length === 0) {
        return;
      }

      // 2. 确定选中的节点
      const nodeExists = urlNode && nodeList.some((n: Node) => n.name === urlNode);
      const targetNode = nodeExists ? urlNode : nodeList[0].name;
      setSelectedNode(targetNode);

      // 3. 获取该节点的实例列表
      const instancesRes = await fetch("/api/instances/describe", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ node_name: targetNode }),
      });
      if (!instancesRes.ok) {
        toast.error("Failed to load instances");
        return;
      }
      const instancesData = await instancesRes.json();
      const instanceList: Instance[] = instancesData.instances || [];
      setInstances(instanceList);

      if (instanceList.length === 0) {
        updateURL(targetNode, "");
        return;
      }

      // 4. 确定选中的 VM（支持按 id 或 name 匹配）
      let targetVM: string;
      if (urlVM) {
        // 如果 URL 中有 VM 参数，优先使用它（即使列表中找不到也先设置）
        const matchedInstance = instanceList.find((i) => i.id === urlVM || i.name === urlVM);
        if (matchedInstance) {
          targetVM = matchedInstance.id;
        } else {
          // 如果在当前节点找不到该 VM，可能是节点不对或 VM 不存在
          // 此时不应该回退到第一个实例，而应该保持 URL 中的值或清空
          console.warn(`VM ${urlVM} not found in node ${targetNode}`);
          targetVM = urlVM; // 保持 URL 中的值，让用户看到参数
          setSelectedVM(targetVM);
          setSnapshots([]); // 清空快照列表
          updateURL(targetNode, targetVM);
          return; // 不继续获取快照
        }
      } else {
        // 没有 URL 参数时才使用第一个实例
        targetVM = instanceList[0].id;
      }

      setSelectedVM(targetVM);

      // 5. 获取快照
      fetchSnapshots(targetNode, targetVM);

      // 6. 更新 URL（保持参数一致）
      updateURL(targetNode, targetVM);

    } catch (err) {
      console.error(err);
      toast.error("Failed to initialize");
    }
  };

  // 用户手动切换节点时
  const handleNodeChange = async (nodeName: string) => {
    setSelectedNode(nodeName);
    setSelectedVM("");
    setInstances([]);
    setSnapshots([]);

    try {
      const res = await fetch("/api/instances/describe", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ node_name: nodeName }),
      });
      if (res.ok) {
        const data = await res.json();
        const list: Instance[] = data.instances || [];
        setInstances(list);
        if (list.length > 0) {
          const firstVM = list[0].id;
          setSelectedVM(firstVM);
          fetchSnapshots(nodeName, firstVM);
          updateURL(nodeName, firstVM);
        } else {
          updateURL(nodeName, "");
        }
      } else {
        toast.error("Failed to load instances");
      }
    } catch (err) {
      console.error(err);
      toast.error("Failed to load instances");
    }
  };

  // 用户手动切换 VM 时
  const handleVMChange = (vmId: string) => {
    setSelectedVM(vmId);
    if (selectedNode && vmId) {
      fetchSnapshots(selectedNode, vmId);
      updateURL(selectedNode, vmId);
    }
  };

  const fetchSnapshots = async (nodeName: string, vmName: string) => {
    setLoading(true);
    try {
      const res = await fetch("/api/list-snapshots", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ node_name: nodeName, vm_name: vmName }),
      });
      if (res.ok) {
        const data = await res.json();
        setSnapshots(data.snapshots || []);
      } else {
        toast.error("Failed to load snapshots");
      }
    } catch (err) {
      console.error(err);
      toast.error("Failed to load snapshots");
    } finally {
      setLoading(false);
    }
  };

  const handleCreateSnapshot = async () => {
    if (!selectedNode || !selectedVM) {
      toast.info("Select node and VM first");
      return;
    }
    setCreating(true);
    try {
      const res = await fetch("/api/create-snapshot", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          node_name: selectedNode,
          vm_name: selectedVM,
          snapshot_name: createForm.snapshot_name || undefined,
          description: createForm.description || undefined,
          with_memory: createForm.with_memory,
        }),
      });
      if (res.ok) {
        toast.success("Snapshot created");
        setCreateModalOpen(false);
        setCreateForm({ snapshot_name: "", description: "", with_memory: false });
        fetchSnapshots(selectedNode, selectedVM);
      } else {
        const data = await res.json().catch(() => ({}));
        toast.error(data?.message || "Create snapshot failed");
      }
    } catch (err) {
      console.error(err);
      toast.error("Create snapshot failed");
    } finally {
      setCreating(false);
    }
  };

  const handleDeleteSnapshot = async () => {
    if (!targetSnapshot) return;
    try {
      const res = await fetch("/api/delete-snapshot", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          node_name: selectedNode,
          vm_name: selectedVM,
          snapshot_name: targetSnapshot.name,
        }),
      });
      if (res.ok) {
        toast.success("Snapshot deleted");
        fetchSnapshots(selectedNode, selectedVM);
      } else {
        const data = await res.json().catch(() => ({}));
        toast.error(data?.message || "Delete snapshot failed");
      }
    } catch (err) {
      console.error(err);
      toast.error("Delete snapshot failed");
    } finally {
      setTargetSnapshot(null);
    }
  };

  const handleRevertSnapshot = async () => {
    if (!targetSnapshot) return;
    try {
      const res = await fetch("/api/revert-snapshot", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          node_name: selectedNode,
          vm_name: selectedVM,
          snapshot_name: targetSnapshot.name,
          start_after_revert: true,
        }),
      });
      if (res.ok) {
        toast.success("Reverted to snapshot");
      } else {
        const data = await res.json().catch(() => ({}));
        toast.error(data?.message || "Revert snapshot failed");
      }
    } catch (err) {
      console.error(err);
      toast.error("Revert snapshot failed");
    } finally {
      setTargetSnapshot(null);
    }
  };

  const columns = useMemo(
    () => [
      { key: "name", label: "Snapshot" },
      { key: "created_at", label: "Created At" },
      {
        key: "state",
        label: "State",
        render: (value: unknown) => (
          <span className="px-2 py-1 rounded-full text-xs bg-blue-50 text-blue-700">
            {String(value || "unknown")}
          </span>
        ),
      },
      {
        key: "memory",
        label: "Memory",
        render: (value: unknown, row: Snapshot) => (
          <span className="text-sm text-gray-800">
            {value ? "Include memory" : row.disk_only ? "Disk only" : "Disk"}
          </span>
        ),
      },
      {
        key: "description",
        label: "Description",
        render: (value: unknown) => <span className="text-sm text-gray-700">{value ? String(value) : "-"}</span>,
      },
      {
        key: "disks",
        label: "Disks",
        render: (_: unknown, row: Snapshot) => (
          <div className="flex flex-col text-xs text-gray-800">
            {row.disks && row.disks.length > 0
              ? row.disks.map((d) => (
                  <span key={`${d.target}-${d.path || d.format || Math.random()}`}>
                    {d.target}: {d.path || "n/a"} {d.format ? `(${d.format})` : ""}
                  </span>
                ))
              : <span>-</span>}
          </div>
        ),
      },
      {
        key: "parent",
        label: "Parent",
        render: (value: unknown) => <span>{value ? String(value) : "-"}</span>,
      },
      {
        key: "actions",
        label: "Actions",
        render: (_: unknown, row: Snapshot) => (
          <div className="flex gap-2">
            <button
              onClick={() => {
                setTargetSnapshot(row);
                setRevertDialogOpen(true);
              }}
              className="btn-secondary flex items-center gap-1"
            >
              <History size={16} />
              Revert
            </button>
            <button
              onClick={() => {
                setTargetSnapshot(row);
                setDeleteDialogOpen(true);
              }}
              className="btn-danger flex items-center gap-1"
            >
              <Trash2 size={16} />
              Delete
            </button>
          </div>
        ),
      },
    ],
    []
  );

  return (
    <DashboardLayout>
      <Header
        title="Snapshots"
        description="Manage VM snapshots (external overlays stored under _snapshots_)."
        action={
          <div className="flex gap-3">
            <button
              onClick={() => fetchSnapshots(selectedNode, selectedVM)}
              className="btn-secondary flex items-center gap-2"
              disabled={!selectedNode || !selectedVM || loading}
            >
              <RefreshCw size={16} />
              Refresh
            </button>
            <button
              onClick={() => setCreateModalOpen(true)}
              className="btn-primary flex items-center gap-2"
              disabled={!selectedNode || !selectedVM}
            >
              <Plus size={16} />
              Create Snapshot
            </button>
          </div>
        }
      />

      <div className="card mb-4">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Node
            </label>
            <select
              className="input"
              value={selectedNode}
              onChange={(e) => handleNodeChange(e.target.value)}
            >
              {nodes.map((node) => (
                <option key={node.name} value={node.name}>
                  {node.name}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              VM
            </label>
            <select
              className="input"
              value={selectedVM}
              onChange={(e) => handleVMChange(e.target.value)}
            >
              {instances.map((vm) => (
                <option key={vm.id} value={vm.id}>
                  {vm.name || vm.id}
                </option>
              ))}
            </select>
          </div>
          <div className="flex items-end">
            <p className="text-sm text-gray-600">
              Select node and VM to view snapshots.
            </p>
          </div>
        </div>
      </div>

      {loading ? (
        <div className="card text-center py-12">
          <p className="text-gray-500">Loading snapshots...</p>
        </div>
      ) : (
        <Table
          columns={columns}
          data={snapshots}
          emptyMessage={selectedVM ? "No snapshots found" : "Select a VM to view snapshots"}
          keyField="id"
        />
      )}

      <Modal
        isOpen={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
        title="Create Snapshot"
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Snapshot Name (optional)
            </label>
            <input
              type="text"
              className="input"
              value={createForm.snapshot_name}
              onChange={(e) =>
                setCreateForm((prev) => ({ ...prev, snapshot_name: e.target.value }))
              }
              placeholder="snap-001"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Description
            </label>
            <textarea
              className="input"
              rows={3}
              value={createForm.description}
              onChange={(e) =>
                setCreateForm((prev) => ({ ...prev, description: e.target.value }))
              }
              placeholder="Optional note about this snapshot"
            />
          </div>
          <label className="inline-flex items-center gap-2 text-sm text-gray-700">
            <input
              type="checkbox"
              className="rounded border-gray-300"
              checked={createForm.with_memory}
              onChange={(e) =>
                setCreateForm((prev) => ({ ...prev, with_memory: e.target.checked }))
              }
            />
            Include memory (may be slower and larger)
          </label>
          <div className="flex justify-end gap-3 pt-2">
            <button
              className="btn-secondary"
              onClick={() => setCreateModalOpen(false)}
            >
              Cancel
            </button>
            <button
              className="btn-primary"
              onClick={handleCreateSnapshot}
              disabled={creating}
            >
              {creating ? "Creating..." : "Create"}
            </button>
          </div>
        </div>
      </Modal>

      <ConfirmDialog
        isOpen={deleteDialogOpen}
        onClose={() => {
          setDeleteDialogOpen(false);
          setTargetSnapshot(null);
        }}
        onConfirm={handleDeleteSnapshot}
        title="Delete snapshot"
        message={`Delete snapshot ${targetSnapshot?.name || ""}? This removes the snapshot overlay.`}
        confirmText="Delete"
        variant="danger"
      />

      <ConfirmDialog
        isOpen={revertDialogOpen}
        onClose={() => {
          setRevertDialogOpen(false);
          setTargetSnapshot(null);
        }}
        onConfirm={handleRevertSnapshot}
        title="Revert to snapshot"
        message={`Revert VM ${selectedVM} to snapshot ${targetSnapshot?.name || ""}? VM will restart after revert.`}
        confirmText="Revert"
        variant="warning"
      />
    </DashboardLayout>
  );
}

export default function Page() {
  return (
    <Suspense fallback={
      <DashboardLayout>
        <div className="card text-center py-12">
          <p className="text-gray-500">Loading...</p>
        </div>
      </DashboardLayout>
    }>
      <SnapshotsContent />
    </Suspense>
  );
}
