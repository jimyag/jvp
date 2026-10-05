import { useCallback, useEffect, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import {
  AlertTriangle,
  Camera,
  Copy,
  Cpu,
  Disc3,
  Globe,
  HardDrive,
  History,
  KeyRound,
  MemoryStick,
  Monitor,
  Pencil,
  Play,
  Plus,
  Power,
  RotateCcw,
  Server,
  Square,
  Trash2,
} from "lucide-react";
import PageHeader from "@/components/PageHeader";
import Table from "@/components/Table";
import Modal from "@/components/Modal";
import ConfirmDialog from "@/components/ConfirmDialog";
import DropdownMenu from "@/components/DropdownMenu";
import {
  Alert,
  Badge,
  Card,
  CopyButton,
  DescriptionList,
  EmptyState,
  Field,
  LoadingState,
  Spinner,
  StatusBadge,
  Switch,
  Tabs,
  UsageBar,
} from "@/components/ui";
import { useToast } from "@/components/ToastContainer";
import { api, errorMessage } from "@/lib/api";
import { formatBytes, formatDate, formatMemoryMB, formatRelative, percent } from "@/lib/format";
import type { Instance, InstanceDisk, Snapshot } from "@/lib/types";
import { CloneSnapshotModal, CreateSnapshotModal } from "../snapshots/SnapshotModals";
import { actionProgressLabel, useInstanceActions } from "./useInstanceActions";

type TabId = "overview" | "disks" | "network" | "snapshots";

function EditInstanceModal({ instance, onClose, onSaved }: { instance: Instance; onClose: () => void; onSaved: () => void }) {
  const toast = useToast();
  const [name, setName] = useState(instance.name || "");
  const [vcpus, setVcpus] = useState(instance.vcpus || 1);
  const [memoryMB, setMemoryMB] = useState(instance.memory_mb || 1024);
  const [autostart, setAutostart] = useState(Boolean(instance.autostart));
  const [saving, setSaving] = useState(false);
  const running = instance.state === "running";

  const handleSave = async (e?: React.FormEvent) => {
    e?.preventDefault();
    setSaving(true);
    try {
      await api("/api/modify-instance-attribute", {
        node_name: instance.node_name,
        instance_id: instance.id,
        name: name.trim(),
        vcpus,
        memory_mb: memoryMB,
        autostart,
        live: running,
      });
      toast.success("Instance updated");
      onSaved();
    } catch (err) {
      toast.error(errorMessage(err, "Failed to update instance"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      isOpen
      onClose={onClose}
      dismissible={!saving}
      title="Edit instance"
      footer={
        <>
          <button className="btn-secondary" onClick={onClose} disabled={saving}>
            Cancel
          </button>
          <button className="btn-primary" onClick={() => handleSave()} disabled={saving || vcpus < 1 || memoryMB < 512}>
            {saving && <Spinner size={14} className="text-current" />}
            Save changes
          </button>
        </>
      }
    >
      <form className="space-y-4" onSubmit={handleSave}>
        {running && (
          <Alert tone="info">
            The instance is running. Changes are applied live where the hypervisor supports it; some may need a restart.
          </Alert>
        )}
        <Field label="Name">
          <input className="input" value={name} onChange={(e) => setName(e.target.value)} autoFocus />
        </Field>
        <div className="grid grid-cols-2 gap-4">
          <Field label="vCPUs">
            <input type="number" className="input" min={1} value={vcpus} onChange={(e) => setVcpus(Number(e.target.value))} />
          </Field>
          <Field label="Memory (MB)" hint={formatMemoryMB(memoryMB)}>
            <input
              type="number"
              className="input"
              min={512}
              step={512}
              value={memoryMB}
              onChange={(e) => setMemoryMB(Number(e.target.value))}
            />
          </Field>
        </div>
        <div className="rounded-lg border border-line p-4">
          <Switch
            checked={autostart}
            onChange={setAutostart}
            label="Start on node boot"
            description="Automatically start this instance when the host starts."
          />
        </div>
      </form>
    </Modal>
  );
}

function ResetPasswordModal({ instance, onClose }: { instance: Instance; onClose: () => void }) {
  const toast = useToast();
  const [username, setUsername] = useState("root");
  const [password, setPassword] = useState("");
  const [saving, setSaving] = useState(false);
  const valid = username.trim() && password.length >= 8;

  const handleSubmit = async (e?: React.FormEvent) => {
    e?.preventDefault();
    if (!valid) return;
    setSaving(true);
    try {
      await api("/api/reset-instance-password", {
        node_name: instance.node_name,
        instance_id: instance.id,
        users: [{ username: username.trim(), new_password: password }],
        auto_start: true,
      });
      toast.success(`Password reset for ${username.trim()}`);
      onClose();
    } catch (err) {
      toast.error(errorMessage(err, "Failed to reset password"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      isOpen
      onClose={onClose}
      dismissible={!saving}
      title="Reset password"
      footer={
        <>
          <button className="btn-secondary" onClick={onClose} disabled={saving}>
            Cancel
          </button>
          <button className="btn-primary" onClick={() => handleSubmit()} disabled={saving || !valid}>
            {saving && <Spinner size={14} className="text-current" />}
            Reset password
          </button>
        </>
      }
    >
      <form className="space-y-4" onSubmit={handleSubmit}>
        <Alert tone="warning">Depending on the guest, the instance may be restarted to apply the new password.</Alert>
        <Field label="Username" required>
          <input className="input" value={username} onChange={(e) => setUsername(e.target.value)} autoComplete="off" />
        </Field>
        <Field label="New password" required hint="At least 8 characters.">
          <input
            type="password"
            className="input"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="new-password"
            autoFocus
          />
        </Field>
      </form>
    </Modal>
  );
}

export default function InstanceDetailPage() {
  const toast = useToast();
  const navigate = useNavigate();
  const { nodeName = "", id: instanceId = "" } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  const tab = (searchParams.get("tab") as TabId) || "overview";

  const [instance, setInstance] = useState<Instance | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [notFound, setNotFound] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [snapshots, setSnapshots] = useState<Snapshot[]>([]);
  const [snapshotsLoading, setSnapshotsLoading] = useState(true);

  const [modal, setModal] = useState<null | "edit" | "password" | "snapshot">(null);
  const [cloneTarget, setCloneTarget] = useState<Snapshot | null>(null);
  const [snapshotAction, setSnapshotAction] = useState<{ type: "revert" | "delete"; snapshot: Snapshot } | null>(null);
  const [ejectTarget, setEjectTarget] = useState<InstanceDisk | null>(null);

  const setTab = (next: TabId) =>
    setSearchParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        if (next === "overview") p.delete("tab");
        else p.set("tab", next);
        return p;
      },
      { replace: true }
    );

  const fetchInstance = useCallback(
    // silent: 不显示刷新动画；background: 轮询请求，失败时不打扰用户
    async ({ silent = false, background = false }: { silent?: boolean; background?: boolean } = {}) => {
      if (!silent) setRefreshing(true);
      try {
        const data = await api<{ instances: Instance[] }>("/api/describe-instances", {
          node_name: nodeName,
          instance_ids: [instanceId],
        });
        const found = data.instances?.[0] || null;
        setInstance(found);
        setNotFound(!found);
        setLoadError(null);
      } catch (err) {
        if (!background) {
          const message = errorMessage(err, "Failed to load instance");
          setLoadError(message);
          toast.error(message);
        }
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    [nodeName, instanceId, toast]
  );

  const fetchSnapshots = useCallback(async () => {
    setSnapshotsLoading(true);
    try {
      const data = await api<{ snapshots: Snapshot[] }>("/api/list-snapshots", { node_name: nodeName, vm_name: instanceId });
      setSnapshots(data.snapshots || []);
    } catch {
      setSnapshots([]);
    } finally {
      setSnapshotsLoading(false);
    }
  }, [nodeName, instanceId]);

  useEffect(() => {
    setLoading(true);
    fetchInstance({ silent: true });
    fetchSnapshots();
  }, [fetchInstance, fetchSnapshots]);

  useEffect(() => {
    const timer = setInterval(() => {
      if (document.visibilityState === "visible") fetchInstance({ silent: true, background: true });
    }, 15000);
    return () => clearInterval(timer);
  }, [fetchInstance]);

  const { request, busy, dialog } = useInstanceActions({
    onChanged: (_, action) => {
      if (action === "terminate") {
        navigate(`/instances?node=${encodeURIComponent(nodeName)}`);
        return;
      }
      fetchInstance({ silent: true, background: true });
      setTimeout(() => fetchInstance({ silent: true, background: true }), 2500);
    },
  });

  const handleEject = async () => {
    if (!ejectTarget?.target) return;
    try {
      const data = await api<{ instance?: Instance }>("/api/eject-instance-media", {
        node_name: nodeName,
        instance_id: instanceId,
        target: ejectTarget.target,
      });
      if (data.instance) setInstance(data.instance);
      else fetchInstance({ silent: true });
      toast.success(`Media ejected from ${ejectTarget.target}`);
    } catch (err) {
      toast.error(errorMessage(err, "Failed to eject media"));
      throw err;
    }
  };

  const handleSnapshotAction = async () => {
    if (!snapshotAction) return;
    const { type, snapshot } = snapshotAction;
    try {
      if (type === "revert") {
        await api("/api/revert-snapshot", {
          node_name: nodeName,
          vm_name: instanceId,
          snapshot_name: snapshot.name,
          start_after_revert: true,
        });
        toast.success(`Reverted to ${snapshot.name}`);
        fetchInstance({ silent: true });
      } else {
        await api("/api/delete-snapshot", { node_name: nodeName, vm_name: instanceId, snapshot_name: snapshot.name });
        toast.success(`Snapshot ${snapshot.name} deleted`);
      }
      fetchSnapshots();
    } catch (err) {
      toast.error(errorMessage(err, `Failed to ${type} snapshot`));
      throw err;
    }
  };

  const listPath = `/instances?node=${encodeURIComponent(nodeName)}`;

  if (loading && !instance) {
    return <LoadingState label="Loading instance…" />;
  }

  if (loadError && !instance) {
    return (
      <div className="card">
        <EmptyState
          icon={<AlertTriangle size={20} />}
          title="Unable to load instance"
          description={loadError}
          action={
            <div className="flex gap-2">
              <Link to={listPath} className="btn-secondary">
                Back to instances
              </Link>
              <button className="btn-primary" onClick={() => fetchInstance()}>
                Try again
              </button>
            </div>
          }
        />
      </div>
    );
  }

  if (notFound || !instance) {
    return (
      <div className="card">
        <EmptyState
          icon={<Server size={20} />}
          title="Instance not found"
          description={`No instance "${instanceId}" exists on node ${nodeName}. It may have been terminated.`}
          action={
            <Link to={listPath} className="btn-primary">
              Back to instances
            </Link>
          }
        />
      </div>
    );
  }

  const running = instance.state === "running";
  const currentAction = busy[instance.id];
  const ips = Array.from(
    new Set([instance.ip_address, ...(instance.interfaces?.flatMap((i) => i.ips || []) || [])].filter(Boolean) as string[])
  );
  const disks = instance.disks || [];
  const interfaces = instance.interfaces || [];
  const consolePath = `/instances/${encodeURIComponent(nodeName)}/${encodeURIComponent(instanceId)}/console`;

  return (
    <>
      <PageHeader
        breadcrumbs={[{ label: "Instances", to: listPath }, { label: nodeName, to: listPath }, { label: instance.name || instance.id }]}
        title={instance.name || instance.id}
        icon={<Server size={18} />}
        meta={
          <>
            {currentAction ? (
              <Badge tone="warning" dot pulse>
                {actionProgressLabel[currentAction]}…
              </Badge>
            ) : (
              <StatusBadge status={instance.state} />
            )}
            <span className="flex items-center gap-1 font-mono text-xs text-fg-subtle">
              {instance.id}
              <CopyButton text={instance.id} />
            </span>
          </>
        }
        onRefresh={() => {
          fetchInstance();
          fetchSnapshots();
        }}
        refreshing={refreshing}
        actions={
          <>
            {running ? (
              <button className="btn-secondary" onClick={() => request(instance, "stop")} disabled={Boolean(currentAction)}>
                <Square size={14} />
                Stop
              </button>
            ) : (
              <button className="btn-primary" onClick={() => request(instance, "start")} disabled={Boolean(currentAction)}>
                <Play size={14} />
                Start
              </button>
            )}
            <button className="btn-secondary" onClick={() => request(instance, "reboot")} disabled={!running || Boolean(currentAction)}>
              <RotateCcw size={14} />
              Reboot
            </button>
            <Link to={consolePath} className="btn-secondary">
              <Monitor size={14} />
              Console
            </Link>
            <DropdownMenu
              triggerClassName="btn-secondary w-9 px-0"
              items={[
                { label: "Edit instance", icon: <Pencil size={14} />, onClick: () => setModal("edit") },
                { label: "Reset password", icon: <KeyRound size={14} />, onClick: () => setModal("password") },
                { label: "Create snapshot", icon: <Camera size={14} />, onClick: () => setModal("snapshot") },
                { divider: true, label: "divider" },
                {
                  label: "Terminate",
                  icon: <Trash2 size={14} />,
                  danger: true,
                  onClick: () => request(instance, "terminate"),
                  disabled: Boolean(currentAction),
                },
              ]}
            />
          </>
        }
      />

      <Tabs
        className="mb-6"
        value={tab}
        onChange={setTab}
        tabs={[
          { id: "overview", label: "Overview" },
          { id: "disks", label: "Disks", count: disks.length },
          { id: "network", label: "Network", count: interfaces.length },
          { id: "snapshots", label: "Snapshots", count: snapshotsLoading ? undefined : snapshots.length },
        ]}
      />

      {tab === "overview" && (
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            {[
              { icon: Cpu, label: "vCPUs", value: instance.vcpus },
              { icon: MemoryStick, label: "Memory", value: formatMemoryMB(instance.memory_mb) },
              {
                icon: Globe,
                label: "IP address",
                value: ips[0] ? (
                  <span className="flex items-center gap-1 font-mono text-base">
                    {ips[0]}
                    <CopyButton text={ips[0]} />
                  </span>
                ) : (
                  <span className="text-base text-fg-subtle">{running ? "Waiting for DHCP…" : "—"}</span>
                ),
              },
              {
                icon: Power,
                label: "Autostart",
                value: <span className="text-base">{instance.autostart ? "Enabled" : "Disabled"}</span>,
              },
            ].map(({ icon: Icon, label, value }) => (
              <div key={label} className="card p-4">
                <div className="flex items-center gap-1.5 text-xs font-medium text-fg-muted">
                  <Icon size={13} className="text-fg-subtle" />
                  {label}
                </div>
                <div className="mt-1.5 truncate text-xl font-semibold text-fg">{value}</div>
              </div>
            ))}
          </div>

          <div className="grid grid-cols-1 gap-6 xl:grid-cols-3">
            <Card title="Details" className="xl:col-span-2">
              <DescriptionList
                items={[
                  { label: "Instance ID", value: instance.id, mono: true, copy: instance.id },
                  { label: "Name", value: instance.name },
                  {
                    label: "Node",
                    value: (
                      <Link to={`/nodes/${encodeURIComponent(nodeName)}`} className="link">
                        {instance.node_name || nodeName}
                      </Link>
                    ),
                  },
                  { label: "Domain name", value: instance.domain_name, mono: true },
                  { label: "Domain UUID", value: instance.domain_uuid, mono: true, copy: instance.domain_uuid },
                  { label: "Template", value: instance.template_id || "None", mono: Boolean(instance.template_id) },
                  { label: "Created", value: instance.created_at ? `${formatDate(instance.created_at)}` : undefined },
                  {
                    label: "Started",
                    value: instance.started_at ? `${formatRelative(instance.started_at)} · ${formatDate(instance.started_at)}` : undefined,
                  },
                ]}
              />
            </Card>

            <Card title="Connect">
              <div className="space-y-4 text-sm">
                <div>
                  <div className="mb-1.5 text-xs font-medium text-fg-subtle">Console</div>
                  <Link to={consolePath} className="btn-secondary w-full">
                    <Monitor size={14} />
                    Open console
                  </Link>
                </div>
                <div>
                  <div className="mb-1.5 text-xs font-medium text-fg-subtle">SSH</div>
                  {running && ips[0] ? (
                    <div className="flex items-center gap-2 rounded-md border border-line bg-canvas px-3 py-2 font-mono text-xs">
                      <span className="min-w-0 flex-1 truncate">ssh &lt;user&gt;@{ips[0]}</span>
                      <CopyButton text={`ssh <user>@${ips[0]}`} />
                    </div>
                  ) : (
                    <p className="text-xs text-fg-muted">
                      {running ? "Available once the instance reports an IP address." : "Start the instance to connect over SSH."}
                    </p>
                  )}
                </div>
              </div>
            </Card>
          </div>
        </div>
      )}

      {tab === "disks" && (
        <Table
          rows={disks}
          rowKey={(d, i) => d.target || String(i)}
          empty={<EmptyState icon={<HardDrive size={20} />} title="No disks attached" />}
          columns={[
            {
              key: "target",
              header: "Device",
              render: (d) => (
                <div className="flex items-center gap-2">
                  {d.device === "cdrom" ? <Disc3 size={15} className="text-fg-subtle" /> : <HardDrive size={15} className="text-fg-subtle" />}
                  <span className="font-mono font-medium">{d.target}</span>
                  <Badge>{d.device || "disk"}</Badge>
                </div>
              ),
            },
            {
              key: "path",
              header: "Source",
              render: (d) =>
                d.path ? (
                  <span className="flex items-center gap-1 font-mono text-xs text-fg-muted" title={d.path}>
                    <span className="max-w-[420px] truncate">{d.path}</span>
                    <CopyButton text={d.path} />
                  </span>
                ) : (
                  <span className="text-fg-subtle">{d.device === "cdrom" ? "Empty drive" : "—"}</span>
                ),
            },
            { key: "format", header: "Format", render: (d) => <span className="text-fg-muted">{d.format || "—"}</span> },
            {
              key: "usage",
              header: "Usage",
              className: "w-56",
              render: (d) =>
                d.capacity_b ? (
                  <div className="space-y-1">
                    <div className="text-xs text-fg-muted">
                      {formatBytes(d.allocation_b)} / {formatBytes(d.capacity_b)}
                    </div>
                    <UsageBar value={percent(d.allocation_b || 0, d.capacity_b)} size="sm" />
                  </div>
                ) : (
                  <span className="text-fg-subtle">—</span>
                ),
            },
            {
              key: "actions",
              header: <span className="sr-only">Actions</span>,
              align: "right",
              render: (d) =>
                d.device === "cdrom" && d.path ? (
                  <button className="btn-secondary btn-sm" onClick={() => setEjectTarget(d)}>
                    <Disc3 size={13} />
                    Eject
                  </button>
                ) : null,
            },
          ]}
        />
      )}

      {tab === "network" && (
        <Table
          rows={interfaces}
          rowKey={(n) => `${n.name}-${n.mac}`}
          empty={<EmptyState icon={<Globe size={20} />} title="No network interfaces" />}
          columns={[
            { key: "name", header: "Interface", render: (n) => <span className="font-mono font-medium">{n.name || "—"}</span> },
            {
              key: "mac",
              header: "MAC address",
              render: (n) => (
                <span className="flex items-center gap-1 font-mono text-xs">
                  {n.mac}
                  <CopyButton text={n.mac} />
                </span>
              ),
            },
            {
              key: "ips",
              header: "IP addresses",
              render: (n) =>
                n.ips?.length ? (
                  <div className="space-y-0.5 font-mono text-xs">
                    {n.ips.map((ip) => (
                      <div key={ip}>{ip}</div>
                    ))}
                  </div>
                ) : (
                  <span className="text-fg-subtle">—</span>
                ),
            },
            {
              key: "source",
              header: "Attached to",
              render: (n) => (
                <span>
                  <span className="font-mono">{n.source}</span>
                  <span className="ml-1.5 text-xs text-fg-subtle">{n.type === "network" ? "libvirt network" : n.type}</span>
                </span>
              ),
            },
          ]}
        />
      )}

      {tab === "snapshots" && (
        <Card
          title="Snapshots"
          description="Restore points for this instance's disks."
          bodyClassName=""
          actions={
            <button className="btn-primary btn-sm" onClick={() => setModal("snapshot")}>
              <Plus size={13} />
              Create snapshot
            </button>
          }
        >
          <Table
            bare
            rows={snapshots}
            loading={snapshotsLoading}
            rowKey={(s) => s.id || s.name}
            empty={
              <EmptyState
                icon={<Camera size={20} />}
                title="No snapshots yet"
                description="Take a snapshot before risky changes so you can roll back."
              />
            }
            columns={[
              {
                key: "name",
                header: "Name",
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
                  <span className="text-fg-muted" title={formatDate(s.created_at)}>
                    {formatRelative(s.created_at)}
                  </span>
                ),
              },
              {
                key: "type",
                header: "Type",
                render: (s) => <Badge tone={s.memory ? "accent" : "neutral"}>{s.memory ? "Disk + memory" : "Disk only"}</Badge>,
              },
              { key: "state", header: "Guest state", render: (s) => <span className="capitalize text-fg-muted">{s.state || "—"}</span> },
              {
                key: "actions",
                header: <span className="sr-only">Actions</span>,
                align: "right",
                render: (s) => (
                  <div className="flex items-center justify-end gap-1">
                    <button className="btn-ghost btn-sm" onClick={() => setSnapshotAction({ type: "revert", snapshot: s })}>
                      <History size={13} />
                      Revert
                    </button>
                    <DropdownMenu
                      items={[
                        { label: "Clone to new instance", icon: <Copy size={14} />, onClick: () => setCloneTarget(s) },
                        { divider: true, label: "divider" },
                        {
                          label: "Delete snapshot",
                          icon: <Trash2 size={14} />,
                          danger: true,
                          onClick: () => setSnapshotAction({ type: "delete", snapshot: s }),
                        },
                      ]}
                    />
                  </div>
                ),
              },
            ]}
          />
        </Card>
      )}

      {modal === "edit" && (
        <EditInstanceModal
          instance={instance}
          onClose={() => setModal(null)}
          onSaved={() => {
            setModal(null);
            fetchInstance({ silent: true });
          }}
        />
      )}
      {modal === "password" && <ResetPasswordModal instance={instance} onClose={() => setModal(null)} />}
      {modal === "snapshot" && (
        <CreateSnapshotModal
          nodeName={nodeName}
          instance={instance}
          onClose={() => setModal(null)}
          onCreated={() => {
            setModal(null);
            fetchSnapshots();
            setTab("snapshots");
          }}
        />
      )}
      {cloneTarget && (
        <CloneSnapshotModal
          nodeName={nodeName}
          instance={instance}
          snapshotName={cloneTarget.name}
          onClose={() => setCloneTarget(null)}
          onCloned={(created) => {
            setCloneTarget(null);
            if (created?.id) navigate(`/instances/${encodeURIComponent(nodeName)}/${encodeURIComponent(created.id)}`);
          }}
        />
      )}

      <ConfirmDialog
        isOpen={Boolean(snapshotAction)}
        onClose={() => setSnapshotAction(null)}
        onConfirm={handleSnapshotAction}
        title={snapshotAction?.type === "revert" ? "Revert to snapshot?" : "Delete snapshot?"}
        message={
          snapshotAction?.type === "revert" ? (
            <>
              {instance.name || instance.id} will be restored to <strong className="text-fg">{snapshotAction.snapshot.name}</strong> and
              restarted. Changes made after the snapshot will be lost.
            </>
          ) : (
            <>
              Snapshot <strong className="text-fg">{snapshotAction?.snapshot.name}</strong> will be permanently deleted.
            </>
          )
        }
        confirmText={snapshotAction?.type === "revert" ? "Revert" : "Delete"}
        variant={snapshotAction?.type === "revert" ? "warning" : "danger"}
      />

      <ConfirmDialog
        isOpen={Boolean(ejectTarget)}
        onClose={() => setEjectTarget(null)}
        onConfirm={handleEject}
        title="Eject media?"
        message={
          <>
            The disc image will be removed from <span className="font-mono text-fg">{ejectTarget?.target}</span>. The image file itself is
            not deleted.
          </>
        }
        confirmText="Eject"
        variant="warning"
      />

      {dialog}
    </>
  );
}
