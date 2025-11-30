"use client";

import { Suspense, useEffect, useMemo, useState } from "react";
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

  useEffect(() => {
    const spNode = searchParams.get("node") || "";
    const spVM = searchParams.get("vm") || "";
    fetchNodes(spNode, spVM);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);

  useEffect(() => {
    if (selectedNode) {
      fetchInstances(selectedNode);
    }
  }, [selectedNode]);

  useEffect(() => {
    if (selectedNode && selectedVM) {
      fetchSnapshots(selectedNode, selectedVM);
    } else {
      setSnapshots([]);
    }
  }, [selectedNode, selectedVM]);

  const fetchNodes = async (preferredNode?: string, preferredVM?: string) => {
    try {
      const res = await fetch("/api/list-nodes", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });
      if (res.ok) {
        const data = await res.json();
        const list = data.nodes || [];
        setNodes(list);
        if (list.length > 0) {
          const nextNode = preferredNode && list.some((n: Node) => n.name === preferredNode)
            ? preferredNode
            : selectedNode || list[0].name;
          setSelectedNode(nextNode);
          if (preferredVM) {
            setSelectedVM(preferredVM);
          }
        }
      } else {
        toast.error("Failed to load nodes");
      }
    } catch (err) {
      console.error(err);
      toast.error("Failed to load nodes");
    }
  };

  const fetchInstances = async (nodeName: string) => {
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
          const exists = list.some((i) => i.id === selectedVM);
          if (!exists) {
            setSelectedVM(list[0].id);
          }
        } else {
          setSelectedVM("");
        }
      } else {
        toast.error("Failed to load instances");
      }
    } catch (err) {
      console.error(err);
      toast.error("Failed to load instances");
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
              onChange={(e) => setSelectedNode(e.target.value)}
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
              onChange={(e) => setSelectedVM(e.target.value)}
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
    <Suspense fallback={<div className="card text-center py-12"><p className="text-gray-500">Loading snapshots...</p></div>}>
      <SnapshotsContent />
    </Suspense>
  );
}
