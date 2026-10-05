import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Database, Info, Play, Plus, RefreshCw, Square } from "lucide-react";
import PageHeader from "@/components/PageHeader";
import Table from "@/components/Table";
import Modal from "@/components/Modal";
import ConfirmDialog from "@/components/ConfirmDialog";
import DropdownMenu from "@/components/DropdownMenu";
import NodeSelect from "@/components/NodeSelect";
import { Badge, EmptyState, Field, Spinner, StatusBadge, UsageBar } from "@/components/ui";
import { useToast } from "@/components/ToastContainer";
import { api, errorMessage } from "@/lib/api";
import { formatBytes, percent } from "@/lib/format";
import { useScopedNode } from "@/lib/nodes";
import type { StoragePool } from "@/lib/types";

// 后端创建存储池时只接收目标路径，fs / netfs 需要的源设备、源主机等字段尚未支持，因此只提供 dir
const POOL_TYPES = [{ value: "dir", label: "Directory", hint: "A directory on the host filesystem." }];

function CreatePoolModal({ nodeName, onClose, onCreated }: { nodeName: string; onClose: () => void; onCreated: () => void }) {
  const toast = useToast();
  const [name, setName] = useState("");
  const [type, setType] = useState("dir");
  const [path, setPath] = useState("");
  const [saving, setSaving] = useState(false);
  const valid = name.trim() && path.trim();

  const handleSubmit = async (e?: React.FormEvent) => {
    e?.preventDefault();
    if (!valid) return;
    setSaving(true);
    try {
      await api("/api/create-storage-pool", { node_name: nodeName, name: name.trim(), type, path: path.trim() });
      toast.success(`Storage pool ${name.trim()} created`);
      onCreated();
    } catch (err) {
      toast.error(errorMessage(err, "Failed to create storage pool"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      isOpen
      onClose={onClose}
      dismissible={!saving}
      title="Create storage pool"
      description={`On node ${nodeName}`}
      footer={
        <>
          <button className="btn-secondary" onClick={onClose} disabled={saving}>
            Cancel
          </button>
          <button className="btn-primary" onClick={() => handleSubmit()} disabled={saving || !valid}>
            {saving && <Spinner size={14} className="text-current" />}
            Create pool
          </button>
        </>
      }
    >
      <form className="space-y-4" onSubmit={handleSubmit}>
        <Field label="Name" required>
          <input className="input" value={name} onChange={(e) => setName(e.target.value)} placeholder="images" autoFocus />
        </Field>
        <Field label="Type" hint={POOL_TYPES.find((t) => t.value === type)?.hint}>
          <select className="input" value={type} onChange={(e) => setType(e.target.value)}>
            {POOL_TYPES.map((t) => (
              <option key={t.value} value={t.value}>
                {t.label} ({t.value})
              </option>
            ))}
          </select>
        </Field>
        <Field label="Path" required hint="Absolute path on the node. It is created if it does not exist.">
          <input
            className="input font-mono text-[13px]"
            value={path}
            onChange={(e) => setPath(e.target.value)}
            placeholder="/var/lib/libvirt/images"
          />
        </Field>
      </form>
    </Modal>
  );
}

export default function StoragePoolsPage() {
  const toast = useToast();
  const navigate = useNavigate();
  const { nodes, currentNode, loading: nodesLoading } = useScopedNode();
  const [pools, setPools] = useState<StoragePool[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [toStop, setToStop] = useState<StoragePool | null>(null);

  // 只采用最新一次请求的结果，避免切换节点后旧节点的响应覆盖列表
  const requestRef = useRef(0);
  const fetchPools = useCallback(
    async ({ silent = false }: { silent?: boolean } = {}) => {
      if (!currentNode) return;
      const requestId = ++requestRef.current;
      if (!silent) setRefreshing(true);
      try {
        const data = await api<{ pools: StoragePool[] }>("/api/list-storage-pools", { node_name: currentNode });
        if (requestId === requestRef.current) setPools(data.pools || []);
      } catch (err) {
        if (requestId === requestRef.current) toast.error(errorMessage(err, "Failed to load storage pools"));
      } finally {
        if (requestId === requestRef.current) {
          setLoading(false);
          setRefreshing(false);
        }
      }
    },
    [currentNode, toast]
  );

  useEffect(() => {
    if (nodesLoading) return;
    if (!currentNode) {
      setLoading(false);
      return;
    }
    setLoading(true);
    setPools([]);
    fetchPools({ silent: true });
  }, [currentNode, nodesLoading, fetchPools]);

  const poolAction = async (pool: StoragePool, action: "start" | "stop" | "refresh") => {
    try {
      await api(`/api/${action}-storage-pool`, { node_name: currentNode, pool_name: pool.name });
      toast.success(
        action === "start" ? `Pool ${pool.name} started` : action === "stop" ? `Pool ${pool.name} stopped` : `Pool ${pool.name} rescanned`
      );
      fetchPools({ silent: true });
    } catch (err) {
      toast.error(errorMessage(err, `Failed to ${action} storage pool`));
      throw err;
    }
  };

  const detailPath = (pool: StoragePool) => `/storage-pools/${encodeURIComponent(pool.name)}?node=${encodeURIComponent(currentNode)}`;
  const noNodes = !nodesLoading && nodes.length === 0;

  return (
    <>
      <PageHeader
        title="Storage pools"
        description="Libvirt storage pools that hold instance disks, templates and ISOs."
        onRefresh={() => fetchPools()}
        refreshing={refreshing}
        actions={
          <>
            <NodeSelect />
            <button className="btn-primary" onClick={() => setCreateOpen(true)} disabled={!currentNode}>
              <Plus size={15} />
              Create pool
            </button>
          </>
        }
      />

      {noNodes ? (
        <div className="card">
          <EmptyState
            icon={<Database size={20} />}
            title="No nodes yet"
            description="Add a node before managing storage."
            action={
              <Link to="/nodes?add=1" className="btn-primary">
                Add node
              </Link>
            }
          />
        </div>
      ) : (
        <Table
          rows={pools}
          rowKey={(p) => p.uuid || p.name}
          loading={loading}
          loadingLabel="Loading storage pools…"
          onRowClick={(p) => navigate(detailPath(p))}
          empty={
            <EmptyState
              icon={<Database size={20} />}
              title={`No storage pools on ${currentNode}`}
              description="Create a pool to store instance disks and templates."
              action={
                <button className="btn-primary" onClick={() => setCreateOpen(true)}>
                  <Plus size={15} />
                  Create pool
                </button>
              }
            />
          }
          columns={[
            {
              key: "name",
              header: "Name",
              render: (p) => (
                <div className="flex items-center gap-2.5">
                  <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-md bg-subtle text-fg-subtle">
                    <Database size={15} />
                  </div>
                  <div className="min-w-0">
                    <div className="font-medium">{p.name}</div>
                    <div className="max-w-[280px] truncate font-mono text-xs text-fg-subtle" title={p.path}>
                      {p.path}
                    </div>
                  </div>
                </div>
              ),
            },
            { key: "state", header: "Status", render: (p) => <StatusBadge status={p.state} /> },
            { key: "type", header: "Type", render: (p) => (p.type ? <Badge>{p.type}</Badge> : <span className="text-fg-subtle">—</span>) },
            {
              key: "usage",
              header: "Usage",
              className: "w-[260px]",
              render: (p) => {
                const used = percent(p.allocation, p.capacity);
                return (
                  <div className="space-y-1">
                    <div className="flex justify-between text-xs text-fg-muted">
                      <span>
                        {formatBytes(p.allocation)} / {formatBytes(p.capacity)}
                      </span>
                      <span className="tabular-nums">{used.toFixed(0)}%</span>
                    </div>
                    <UsageBar value={used} size="sm" />
                  </div>
                );
              },
            },
            { key: "free", header: "Free", render: (p) => <span className="whitespace-nowrap text-fg-muted">{formatBytes(p.available)}</span> },
            { key: "volumes", header: "Volumes", align: "right", render: (p) => <span className="tabular-nums">{p.volume_count}</span> },
            {
              key: "actions",
              header: <span className="sr-only">Actions</span>,
              align: "right",
              render: (p) => {
                const active = p.state.toLowerCase() === "active";
                return (
                  <div onClick={(e) => e.stopPropagation()}>
                    <DropdownMenu
                      items={[
                        { label: "View volumes", icon: <Info size={14} />, onClick: () => navigate(detailPath(p)) },
                        { label: "Rescan volumes", icon: <RefreshCw size={14} />, onClick: () => poolAction(p, "refresh").catch(() => undefined), disabled: !active },
                        active
                          ? { label: "Stop pool", icon: <Square size={14} />, onClick: () => setToStop(p) }
                          : { label: "Start pool", icon: <Play size={14} />, onClick: () => poolAction(p, "start").catch(() => undefined) },
                      ]}
                    />
                  </div>
                );
              },
            },
          ]}
        />
      )}

      {createOpen && (
        <CreatePoolModal
          nodeName={currentNode}
          onClose={() => setCreateOpen(false)}
          onCreated={() => {
            setCreateOpen(false);
            fetchPools({ silent: true });
          }}
        />
      )}

      <ConfirmDialog
        isOpen={Boolean(toStop)}
        onClose={() => setToStop(null)}
        onConfirm={() => (toStop ? poolAction(toStop, "stop") : undefined)}
        title="Stop storage pool?"
        message={
          <>
            Instances using disks in <strong className="text-fg">{toStop?.name}</strong> may fail to start until the pool is started again.
          </>
        }
        confirmText="Stop pool"
        variant="warning"
      />
    </>
  );
}
