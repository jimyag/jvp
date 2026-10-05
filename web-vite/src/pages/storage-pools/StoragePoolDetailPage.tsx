import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { Database, HardDrive, Maximize2, Play, Plus, RefreshCw, Square, Trash2 } from "lucide-react";
import PageHeader from "@/components/PageHeader";
import Table from "@/components/Table";
import Modal from "@/components/Modal";
import ConfirmDialog from "@/components/ConfirmDialog";
import DropdownMenu from "@/components/DropdownMenu";
import SearchFilter from "@/components/SearchFilter";
import { Alert, Badge, Card, Checkbox, CopyButton, EmptyState, Field, LoadingState, Spinner, StatCard, StatusBadge, UsageBar } from "@/components/ui";
import { useToast } from "@/components/ToastContainer";
import { api, errorMessage } from "@/lib/api";
import { formatBytes, percent } from "@/lib/format";
import type { StoragePool, Volume } from "@/lib/types";

function CreateVolumeModal({
  nodeName,
  poolName,
  onClose,
  onCreated,
}: {
  nodeName: string;
  poolName: string;
  onClose: () => void;
  onCreated: () => void;
}) {
  const toast = useToast();
  const [name, setName] = useState("");
  const [size, setSize] = useState(20);
  const [format, setFormat] = useState("qcow2");
  const [saving, setSaving] = useState(false);

  const handleSubmit = async (e?: React.FormEvent) => {
    e?.preventDefault();
    if (size < 1) return;
    setSaving(true);
    try {
      await api("/api/create-volume", {
        node_name: nodeName,
        pool_name: poolName,
        name: name.trim() || undefined,
        size_gb: size,
        format,
      });
      toast.success("Volume created");
      onCreated();
    } catch (err) {
      toast.error(errorMessage(err, "Failed to create volume"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      isOpen
      onClose={onClose}
      dismissible={!saving}
      title="Create volume"
      description={`In pool ${poolName}`}
      footer={
        <>
          <button className="btn-secondary" onClick={onClose} disabled={saving}>
            Cancel
          </button>
          <button className="btn-primary" onClick={() => handleSubmit()} disabled={saving || size < 1}>
            {saving && <Spinner size={14} className="text-current" />}
            Create volume
          </button>
        </>
      }
    >
      <form className="space-y-4" onSubmit={handleSubmit}>
        <Field label="Name" hint="Leave empty to generate a unique ID.">
          <input className="input" value={name} onChange={(e) => setName(e.target.value)} placeholder="data-disk" autoFocus />
        </Field>
        <div className="grid grid-cols-2 gap-4">
          <Field label="Size (GB)" required>
            <input type="number" className="input" min={1} value={size} onChange={(e) => setSize(Number(e.target.value))} />
          </Field>
          <Field label="Format" hint={format === "qcow2" ? "Thin-provisioned, supports snapshots." : "Fully allocated, fastest I/O."}>
            <select className="input" value={format} onChange={(e) => setFormat(e.target.value)}>
              <option value="qcow2">qcow2</option>
              <option value="raw">raw</option>
            </select>
          </Field>
        </div>
      </form>
    </Modal>
  );
}

function ResizeVolumeModal({
  nodeName,
  poolName,
  volume,
  onClose,
  onResized,
}: {
  nodeName: string;
  poolName: string;
  volume: Volume;
  onClose: () => void;
  onResized: () => void;
}) {
  const toast = useToast();
  const [size, setSize] = useState(volume.size_gb + 10);
  const [saving, setSaving] = useState(false);
  const valid = size > volume.size_gb;

  const handleSubmit = async (e?: React.FormEvent) => {
    e?.preventDefault();
    if (!valid) return;
    setSaving(true);
    try {
      await api("/api/resize-volume", { node_name: nodeName, pool_name: poolName, volume_id: volume.volume_id, new_size_gb: size });
      toast.success(`${volume.name} resized to ${size} GB`);
      onResized();
    } catch (err) {
      toast.error(errorMessage(err, "Failed to resize volume"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      isOpen
      onClose={onClose}
      dismissible={!saving}
      size="sm"
      title="Resize volume"
      description={volume.name}
      footer={
        <>
          <button className="btn-secondary" onClick={onClose} disabled={saving}>
            Cancel
          </button>
          <button className="btn-primary" onClick={() => handleSubmit()} disabled={saving || !valid}>
            {saving && <Spinner size={14} className="text-current" />}
            Resize
          </button>
        </>
      }
    >
      <form className="space-y-4" onSubmit={handleSubmit}>
        <Field label="New size (GB)" hint={`Current size is ${volume.size_gb} GB. Volumes can only grow.`}>
          <input
            type="number"
            className="input"
            min={volume.size_gb + 1}
            value={size}
            onChange={(e) => setSize(Number(e.target.value))}
            autoFocus
          />
        </Field>
        <Alert tone="info">After resizing, extend the partition and filesystem inside the guest to use the new space.</Alert>
      </form>
    </Modal>
  );
}

export default function StoragePoolDetailPage() {
  const toast = useToast();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const nodeName = searchParams.get("node") || "";
  const { poolName = "" } = useParams();

  const [pool, setPool] = useState<StoragePool | null>(null);
  const [volumes, setVolumes] = useState<Volume[]>([]);
  const [loading, setLoading] = useState(true);
  const [volumesLoading, setVolumesLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [query, setQuery] = useState("");

  const [createOpen, setCreateOpen] = useState(false);
  const [resizeTarget, setResizeTarget] = useState<Volume | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Volume | null>(null);
  const [confirm, setConfirm] = useState<null | "stop" | "delete">(null);
  const [deleteVolumes, setDeleteVolumes] = useState(false);

  const listPath = `/storage-pools?node=${encodeURIComponent(nodeName)}`;

  const fetchPool = useCallback(async () => {
    try {
      const data = await api<{ pool: StoragePool }>("/api/describe-storage-pool", { node_name: nodeName, pool_name: poolName });
      setPool(data.pool);
    } catch (err) {
      setPool(null);
      toast.error(errorMessage(err, "Failed to load storage pool"));
    } finally {
      setLoading(false);
    }
  }, [nodeName, poolName, toast]);

  const fetchVolumes = useCallback(async () => {
    setVolumesLoading(true);
    try {
      const data = await api<{ volumes: Volume[] }>("/api/list-volumes", { node_name: nodeName, pool_name: poolName });
      setVolumes(data.volumes || []);
    } catch (err) {
      setVolumes([]);
      toast.error(errorMessage(err, "Failed to load volumes"));
    } finally {
      setVolumesLoading(false);
    }
  }, [nodeName, poolName, toast]);

  useEffect(() => {
    if (!nodeName || !poolName) {
      setLoading(false);
      return;
    }
    fetchPool();
    fetchVolumes();
  }, [nodeName, poolName, fetchPool, fetchVolumes]);

  const reload = async () => {
    await Promise.all([fetchPool(), fetchVolumes()]);
  };

  const handleRescan = async () => {
    setRefreshing(true);
    try {
      await api("/api/refresh-storage-pool", { node_name: nodeName, pool_name: poolName });
      await reload();
    } catch (err) {
      toast.error(errorMessage(err, "Failed to rescan pool"));
    } finally {
      setRefreshing(false);
    }
  };

  const poolAction = async (action: "start" | "stop") => {
    try {
      await api(`/api/${action}-storage-pool`, { node_name: nodeName, pool_name: poolName });
      toast.success(action === "start" ? "Pool started" : "Pool stopped");
      await reload();
    } catch (err) {
      toast.error(errorMessage(err, `Failed to ${action} pool`));
      throw err;
    }
  };

  const handleDeletePool = async () => {
    try {
      await api("/api/delete-storage-pool", { node_name: nodeName, pool_name: poolName, delete_volumes: deleteVolumes });
      toast.success(`Storage pool ${poolName} deleted`);
      navigate(listPath);
    } catch (err) {
      toast.error(errorMessage(err, "Failed to delete storage pool"));
      throw err;
    }
  };

  const handleDeleteVolume = async () => {
    if (!deleteTarget) return;
    try {
      await api("/api/delete-volume", { node_name: nodeName, pool_name: poolName, volume_id: deleteTarget.volume_id });
      toast.success(`Volume ${deleteTarget.name} deleted`);
      reload();
    } catch (err) {
      toast.error(errorMessage(err, "Failed to delete volume"));
      throw err;
    }
  };

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return volumes
      .filter((v) => !q || v.name.toLowerCase().includes(q) || v.path.toLowerCase().includes(q) || v.volume_id.toLowerCase().includes(q))
      .sort((a, b) => a.name.localeCompare(b.name));
  }, [volumes, query]);

  if (loading) return <LoadingState label="Loading storage pool…" />;

  if (!pool) {
    return (
      <div className="card">
        <EmptyState
          icon={<Database size={20} />}
          title="Storage pool not found"
          description={nodeName ? `Pool "${poolName}" does not exist on node ${nodeName}.` : "No node was specified for this pool."}
          action={
            <Link to={listPath} className="btn-primary">
              Back to storage pools
            </Link>
          }
        />
      </div>
    );
  }

  const active = pool.state.toLowerCase() === "active";
  const used = percent(pool.allocation, pool.capacity);

  return (
    <>
      <PageHeader
        breadcrumbs={[{ label: "Storage", to: listPath }, { label: nodeName, to: listPath }, { label: pool.name }]}
        title={pool.name}
        icon={<Database size={18} />}
        meta={
          <>
            <StatusBadge status={pool.state} />
            {pool.type && <Badge>{pool.type}</Badge>}
            <span className="flex items-center gap-1 font-mono text-xs text-fg-subtle">
              {pool.path}
              <CopyButton text={pool.path} />
            </span>
          </>
        }
        actions={
          <>
            <button className="btn-secondary" onClick={handleRescan} disabled={refreshing || !active}>
              <RefreshCw size={14} className={refreshing ? "animate-spin" : ""} />
              Rescan
            </button>
            {active ? (
              <button className="btn-secondary" onClick={() => setConfirm("stop")}>
                <Square size={14} />
                Stop
              </button>
            ) : (
              <button className="btn-primary" onClick={() => poolAction("start").catch(() => undefined)}>
                <Play size={14} />
                Start
              </button>
            )}
            <DropdownMenu
              triggerClassName="btn-secondary w-9 px-0"
              items={[
                {
                  label: "Delete pool",
                  icon: <Trash2 size={14} />,
                  danger: true,
                  onClick: () => {
                    setDeleteVolumes(false);
                    setConfirm("delete");
                  },
                },
              ]}
            />
          </>
        }
      />

      <div className="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
        <StatCard label="Used" value={formatBytes(pool.allocation)} hint={`${used.toFixed(1)}% of ${formatBytes(pool.capacity)}`}>
          <UsageBar value={used} size="sm" />
        </StatCard>
        <StatCard label="Free" value={formatBytes(pool.available)} hint="Available for new volumes" />
        <StatCard label="Volumes" value={pool.volume_count} hint="Disks, images and ISOs" />
      </div>

      {!active && (
        <Alert tone="warning" className="mb-6" title="This pool is not active">
          Start the pool to create volumes or launch instances from it.
        </Alert>
      )}

      <Card
        title="Volumes"
        bodyClassName=""
        actions={
          <>
            <SearchFilter value={query} onChange={setQuery} placeholder="Filter volumes" className="hidden w-56 sm:block" />
            <button className="btn-primary" onClick={() => setCreateOpen(true)} disabled={!active}>
              <Plus size={14} />
              Create volume
            </button>
          </>
        }
      >
        <Table
          bare
          rows={filtered}
          loading={volumesLoading}
          rowKey={(v) => v.volume_id}
          empty={
            volumes.length ? (
              <EmptyState title="No matching volumes" />
            ) : (
              <EmptyState icon={<HardDrive size={20} />} title="No volumes in this pool" description="Create a volume or register a template to add one." />
            )
          }
          columns={[
            {
              key: "name",
              header: "Name",
              render: (v) => (
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{v.name}</span>
                  </div>
                  <div className="flex max-w-[460px] items-center gap-1 font-mono text-xs text-fg-subtle" title={v.path}>
                    <span className="truncate">{v.path}</span>
                    <CopyButton text={v.path} />
                  </div>
                </div>
              ),
            },
            { key: "format", header: "Format", render: (v) => <Badge>{v.format || "—"}</Badge> },
            { key: "size", header: "Size", render: (v) => <span className="whitespace-nowrap">{v.size_gb} GB</span> },
            {
              key: "allocated",
              header: "Allocated",
              className: "w-48",
              render: (v) => (
                <div className="space-y-1">
                  <div className="text-xs text-fg-muted">
                    {formatBytes(v.allocation_b)}
                    {v.capacity_b > 0 && ` · ${percent(v.allocation_b, v.capacity_b).toFixed(0)}%`}
                  </div>
                  <UsageBar value={percent(v.allocation_b, v.capacity_b)} size="sm" />
                </div>
              ),
            },
            {
              key: "actions",
              header: <span className="sr-only">Actions</span>,
              align: "right",
              render: (v) => (
                <DropdownMenu
                  items={[
                    { label: "Resize", icon: <Maximize2 size={14} />, onClick: () => setResizeTarget(v) },
                    { divider: true, label: "divider" },
                    { label: "Delete volume", icon: <Trash2 size={14} />, danger: true, onClick: () => setDeleteTarget(v) },
                  ]}
                />
              ),
            },
          ]}
        />
      </Card>

      {createOpen && (
        <CreateVolumeModal
          nodeName={nodeName}
          poolName={poolName}
          onClose={() => setCreateOpen(false)}
          onCreated={() => {
            setCreateOpen(false);
            reload();
          }}
        />
      )}
      {resizeTarget && (
        <ResizeVolumeModal
          nodeName={nodeName}
          poolName={poolName}
          volume={resizeTarget}
          onClose={() => setResizeTarget(null)}
          onResized={() => {
            setResizeTarget(null);
            reload();
          }}
        />
      )}

      <ConfirmDialog
        isOpen={Boolean(deleteTarget)}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDeleteVolume}
        title="Delete volume?"
        message={
          <>
            <strong className="text-fg">{deleteTarget?.name}</strong> and all data on it will be permanently deleted. Make sure no instance or
            template uses it.
          </>
        }
        confirmText="Delete volume"
      />
      <ConfirmDialog
        isOpen={confirm === "stop"}
        onClose={() => setConfirm(null)}
        onConfirm={() => poolAction("stop")}
        title="Stop storage pool?"
        message="Instances with disks in this pool may fail to start until the pool is started again."
        confirmText="Stop pool"
        variant="warning"
      />
      <ConfirmDialog
        isOpen={confirm === "delete"}
        onClose={() => setConfirm(null)}
        onConfirm={handleDeletePool}
        title="Delete storage pool?"
        message={
          <>
            The pool definition <strong className="text-fg">{pool.name}</strong> will be removed from libvirt.
          </>
        }
        confirmText="Delete pool"
      >
        <Checkbox
          checked={deleteVolumes}
          onChange={setDeleteVolumes}
          label="Also delete all volumes and the pool directory"
          description="Permanently removes every disk, template and ISO stored in this pool."
        />
      </ConfirmDialog>
    </>
  );
}
