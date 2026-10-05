import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { Camera, Copy, History, Plus, Trash2 } from "lucide-react";
import PageHeader from "@/components/PageHeader";
import Table from "@/components/Table";
import ConfirmDialog from "@/components/ConfirmDialog";
import DropdownMenu from "@/components/DropdownMenu";
import NodeSelect from "@/components/NodeSelect";
import { Badge, EmptyState, StatusBadge } from "@/components/ui";
import { useToast } from "@/components/ToastContainer";
import { api, errorMessage } from "@/lib/api";
import { formatDate, formatRelative } from "@/lib/format";
import { useScopedNode } from "@/lib/nodes";
import type { Instance, Snapshot } from "@/lib/types";
import { CloneSnapshotModal, CreateSnapshotModal } from "./SnapshotModals";

export default function SnapshotsPage() {
  const toast = useToast();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const { nodes, currentNode, loading: nodesLoading } = useScopedNode();
  const vmParam = searchParams.get("vm") || "";

  const [instances, setInstances] = useState<Instance[]>([]);
  const [instancesLoading, setInstancesLoading] = useState(true);
  const [snapshots, setSnapshots] = useState<Snapshot[]>([]);
  const [snapshotsLoading, setSnapshotsLoading] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [cloneTarget, setCloneTarget] = useState<Snapshot | null>(null);
  const [action, setAction] = useState<{ type: "revert" | "delete"; snapshot: Snapshot } | null>(null);

  const setVM = useCallback(
    (vm: string) =>
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          if (vm) next.set("vm", vm);
          else next.delete("vm");
          return next;
        },
        { replace: true }
      ),
    [setSearchParams]
  );

  // 加载当前节点的实例
  useEffect(() => {
    if (nodesLoading) return;
    if (!currentNode) {
      setInstancesLoading(false);
      return;
    }
    let cancelled = false;
    setInstancesLoading(true);
    // 清空上一个节点的实例，避免用旧实例 ID 查询新节点的快照
    setInstances([]);
    api<{ instances: Instance[] }>("/api/describe-instances", { node_name: currentNode })
      .then((data) => {
        if (cancelled) return;
        const list = (data.instances || []).sort((a, b) => (a.name || a.id).localeCompare(b.name || b.id));
        setInstances(list);
      })
      .catch((err) => !cancelled && toast.error(errorMessage(err, "Failed to load instances")))
      .finally(() => !cancelled && setInstancesLoading(false));
    return () => {
      cancelled = true;
    };
  }, [currentNode, nodesLoading, toast]);

  const selected = useMemo(() => instances.find((i) => i.id === vmParam || i.name === vmParam), [instances, vmParam]);

  // URL 中的实例不在当前节点时，默认选择第一个实例
  useEffect(() => {
    if (instancesLoading) return;
    if (!selected && instances.length > 0) setVM(instances[0].id);
    if (instances.length === 0 && vmParam) setVM("");
  }, [instancesLoading, instances, selected, vmParam, setVM]);

  // 只采用最新一次请求的结果，避免切换实例后旧实例的快照覆盖列表
  const requestRef = useRef(0);
  const fetchSnapshots = useCallback(
    async ({ silent = false }: { silent?: boolean } = {}) => {
      if (!currentNode || !selected) return;
      const requestId = ++requestRef.current;
      if (silent) setSnapshotsLoading(true);
      else setRefreshing(true);
      try {
        const data = await api<{ snapshots: Snapshot[] }>("/api/list-snapshots", { node_name: currentNode, vm_name: selected.id });
        if (requestId === requestRef.current) setSnapshots(data.snapshots || []);
      } catch (err) {
        if (requestId === requestRef.current) toast.error(errorMessage(err, "Failed to load snapshots"));
      } finally {
        if (requestId === requestRef.current) {
          setSnapshotsLoading(false);
          setRefreshing(false);
        }
      }
    },
    [currentNode, selected, toast]
  );

  useEffect(() => {
    setSnapshots([]);
    fetchSnapshots({ silent: true });
  }, [fetchSnapshots]);

  const handleAction = async () => {
    if (!action || !selected) return;
    const { type, snapshot } = action;
    try {
      if (type === "revert") {
        await api("/api/revert-snapshot", {
          node_name: currentNode,
          vm_name: selected.id,
          snapshot_name: snapshot.name,
          start_after_revert: true,
        });
        toast.success(`${selected.name || selected.id} reverted to ${snapshot.name}`);
      } else {
        await api("/api/delete-snapshot", { node_name: currentNode, vm_name: selected.id, snapshot_name: snapshot.name });
        toast.success(`Snapshot ${snapshot.name} deleted`);
      }
      fetchSnapshots({ silent: true });
    } catch (err) {
      toast.error(errorMessage(err, `Failed to ${type} snapshot`));
      throw err;
    }
  };

  const noNodes = !nodesLoading && nodes.length === 0;
  const noInstances = !instancesLoading && instances.length === 0;

  return (
    <>
      <PageHeader
        title="Snapshots"
        description="Point-in-time restore points for instance disks."
        onRefresh={() => fetchSnapshots()}
        refreshing={refreshing}
        actions={
          <>
            <NodeSelect />
            <button className="btn-primary" onClick={() => setCreateOpen(true)} disabled={!selected}>
              <Plus size={15} />
              Create snapshot
            </button>
          </>
        }
      />

      {noNodes || noInstances ? (
        <div className="card">
          <EmptyState
            icon={<Camera size={20} />}
            title={noNodes ? "No nodes yet" : `No instances on ${currentNode}`}
            description="Snapshots are taken from instances. Create an instance first."
            action={
              <Link to={noNodes ? "/nodes?add=1" : "/instances?create=1"} className="btn-primary">
                {noNodes ? "Add node" : "Create instance"}
              </Link>
            }
          />
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-[260px_minmax(0,1fr)]">
          <aside className="card h-fit overflow-hidden">
            <div className="border-b border-line px-4 py-3 text-xs font-medium text-fg-subtle">Instances on {currentNode}</div>
            <div className="max-h-[60vh] overflow-y-auto p-1.5">
              {instancesLoading ? (
                <div className="px-3 py-6 text-center text-sm text-fg-subtle">Loading…</div>
              ) : (
                instances.map((i) => {
                  const active = selected?.id === i.id;
                  return (
                    <button
                      key={i.id}
                      type="button"
                      onClick={() => setVM(i.id)}
                      className={`flex w-full items-center justify-between gap-2 rounded-md px-2.5 py-2 text-left text-sm transition-colors ${
                        active ? "bg-accent-soft text-accent" : "text-fg hover:bg-subtle"
                      }`}
                    >
                      <span className="truncate font-medium">{i.name || i.id}</span>
                      <span className={`h-1.5 w-1.5 flex-shrink-0 rounded-full ${i.state === "running" ? "bg-success" : "bg-fg-subtle/50"}`} />
                    </button>
                  );
                })
              )}
            </div>
          </aside>

          <div className="min-w-0">
            {selected && (
              <div className="mb-3 flex items-center gap-2 text-sm">
                <Link
                  to={`/instances/${encodeURIComponent(currentNode)}/${encodeURIComponent(selected.id)}`}
                  className="font-medium text-fg hover:text-accent"
                >
                  {selected.name || selected.id}
                </Link>
                <StatusBadge status={selected.state} />
              </div>
            )}
            <Table
              rows={snapshots}
              loading={snapshotsLoading || instancesLoading}
              loadingLabel="Loading snapshots…"
              rowKey={(s) => s.id || s.name}
              empty={
                <EmptyState
                  icon={<Camera size={20} />}
                  title="No snapshots yet"
                  description="Take a snapshot before upgrades or risky changes so you can roll back."
                  action={
                    <button className="btn-primary" onClick={() => setCreateOpen(true)} disabled={!selected}>
                      <Plus size={15} />
                      Create snapshot
                    </button>
                  }
                />
              }
              columns={[
                {
                  key: "name",
                  header: "Snapshot",
                  render: (s) => (
                    <div>
                      <div className="font-medium">{s.name}</div>
                      {s.description && <div className="max-w-sm truncate text-xs text-fg-muted">{s.description}</div>}
                    </div>
                  ),
                },
                {
                  key: "created",
                  header: "Created",
                  render: (s) => (
                    <span className="whitespace-nowrap text-fg-muted" title={formatDate(s.created_at)}>
                      {formatRelative(s.created_at)}
                    </span>
                  ),
                },
                {
                  key: "type",
                  header: "Type",
                  render: (s) => <Badge tone={s.memory ? "accent" : "neutral"}>{s.memory ? "Disk + memory" : "Disk only"}</Badge>,
                },
                {
                  key: "disks",
                  header: "Disks",
                  render: (s) =>
                    s.disks?.length ? (
                      <div className="space-y-0.5 font-mono text-xs text-fg-muted">
                        {s.disks.map((d, idx) => (
                          <div key={`${d.target}-${idx}`} className="max-w-[260px] truncate" title={d.path}>
                            {d.target}
                            {d.format ? ` · ${d.format}` : ""}
                          </div>
                        ))}
                      </div>
                    ) : (
                      <span className="text-fg-subtle">—</span>
                    ),
                },
                { key: "parent", header: "Parent", render: (s) => <span className="text-fg-muted">{s.parent || "—"}</span> },
                {
                  key: "actions",
                  header: <span className="sr-only">Actions</span>,
                  align: "right",
                  render: (s) => (
                    <div className="flex items-center justify-end gap-1">
                      <button className="btn-ghost btn-sm" onClick={() => setAction({ type: "revert", snapshot: s })}>
                        <History size={13} />
                        Revert
                      </button>
                      <DropdownMenu
                        items={[
                          { label: "Clone to new instance", icon: <Copy size={14} />, onClick: () => setCloneTarget(s) },
                          { divider: true, label: "divider" },
                          { label: "Delete snapshot", icon: <Trash2 size={14} />, danger: true, onClick: () => setAction({ type: "delete", snapshot: s }) },
                        ]}
                      />
                    </div>
                  ),
                },
              ]}
            />
          </div>
        </div>
      )}

      {createOpen && selected && (
        <CreateSnapshotModal
          nodeName={currentNode}
          instance={selected}
          onClose={() => setCreateOpen(false)}
          onCreated={() => {
            setCreateOpen(false);
            fetchSnapshots({ silent: true });
          }}
        />
      )}
      {cloneTarget && selected && (
        <CloneSnapshotModal
          nodeName={currentNode}
          instance={selected}
          snapshotName={cloneTarget.name}
          onClose={() => setCloneTarget(null)}
          onCloned={(created) => {
            setCloneTarget(null);
            if (created?.id) navigate(`/instances/${encodeURIComponent(currentNode)}/${encodeURIComponent(created.id)}`);
          }}
        />
      )}

      <ConfirmDialog
        isOpen={Boolean(action)}
        onClose={() => setAction(null)}
        onConfirm={handleAction}
        title={action?.type === "revert" ? "Revert to snapshot?" : "Delete snapshot?"}
        message={
          action?.type === "revert" ? (
            <>
              {selected?.name || selected?.id} will be restored to <strong className="text-fg">{action.snapshot.name}</strong> and restarted.
              Changes made after the snapshot will be lost.
            </>
          ) : (
            <>
              Snapshot <strong className="text-fg">{action?.snapshot.name}</strong> will be permanently deleted.
            </>
          )
        }
        confirmText={action?.type === "revert" ? "Revert" : "Delete"}
        variant={action?.type === "revert" ? "warning" : "danger"}
      />
    </>
  );
}
