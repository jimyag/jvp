import { useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { Info, Plus, Power, PowerOff, ServerCog, Trash2 } from "lucide-react";
import PageHeader from "@/components/PageHeader";
import Table from "@/components/Table";
import Modal from "@/components/Modal";
import ConfirmDialog from "@/components/ConfirmDialog";
import DropdownMenu from "@/components/DropdownMenu";
import { Alert, Badge, EmptyState, Field, Spinner, StatusBadge } from "@/components/ui";
import { useToast } from "@/components/ToastContainer";
import { api, errorMessage } from "@/lib/api";
import { useNodes } from "@/lib/nodes";
import type { Node } from "@/lib/types";

const NODE_TYPES: { value: string; label: string; description: string }[] = [
  { value: "remote", label: "Remote", description: "A libvirt host reached over the network (e.g. qemu+ssh)." },
  { value: "local", label: "Local", description: "The machine running this JVP server." },
  { value: "compute", label: "Compute", description: "Dedicated to running virtual machines." },
  { value: "storage", label: "Storage", description: "Dedicated to storage pools and volumes." },
  { value: "hybrid", label: "Hybrid", description: "Runs both compute and storage workloads." },
];

function AddNodeModal({ onClose, onCreated }: { onClose: () => void; onCreated: (name: string) => void }) {
  const toast = useToast();
  const [name, setName] = useState("");
  const [uri, setUri] = useState("");
  const [type, setType] = useState("remote");
  const [saving, setSaving] = useState(false);
  const valid = name.trim() && uri.trim();
  const isSSH = uri.includes("+ssh://") || type === "remote";

  const handleSubmit = async (e?: React.FormEvent) => {
    e?.preventDefault();
    if (!valid) return;
    setSaving(true);
    try {
      await api("/api/create-node", { name: name.trim(), uri: uri.trim(), type });
      toast.success(`Node ${name.trim()} added`);
      onCreated(name.trim());
    } catch (err) {
      toast.error(errorMessage(err, "Failed to add node"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      isOpen
      onClose={onClose}
      dismissible={!saving}
      title="Add node"
      description="Connect a libvirt host so JVP can manage its virtual machines."
      footer={
        <>
          <button className="btn-secondary" onClick={onClose} disabled={saving}>
            Cancel
          </button>
          <button className="btn-primary" onClick={() => handleSubmit()} disabled={saving || !valid}>
            {saving && <Spinner size={14} className="text-current" />}
            Add node
          </button>
        </>
      }
    >
      <form className="space-y-4" onSubmit={handleSubmit}>
        <Field label="Name" required hint="A unique, short identifier, e.g. node-1.">
          <input className="input" value={name} onChange={(e) => setName(e.target.value)} placeholder="node-1" autoFocus />
        </Field>
        <Field label="Libvirt URI" required>
          <input
            className="input font-mono text-[13px]"
            value={uri}
            onChange={(e) => setUri(e.target.value)}
            placeholder="qemu+ssh://root@192.168.1.100/system"
          />
        </Field>
        <Field label="Type" hint={NODE_TYPES.find((t) => t.value === type)?.description}>
          <select className="input" value={type} onChange={(e) => setType(e.target.value)}>
            {NODE_TYPES.map((t) => (
              <option key={t.value} value={t.value}>
                {t.label}
              </option>
            ))}
          </select>
        </Field>
        {isSSH && (
          <Alert tone="info" title="Passwordless SSH is required">
            <p className="mt-1">The JVP server must be able to log in to the host without a password:</p>
            <pre className="mt-2 overflow-x-auto rounded-md bg-surface/70 px-3 py-2 font-mono text-xs text-fg">
              {"ssh-keygen -t ed25519\nssh-copy-id root@<host>\nssh root@<host> virsh list"}
            </pre>
          </Alert>
        )}
      </form>
    </Modal>
  );
}

export default function NodesPage() {
  const toast = useToast();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const { nodes, loading, refresh } = useNodes();
  const [refreshing, setRefreshing] = useState(false);
  const [addOpen, setAddOpen] = useState(searchParams.get("add") === "1");
  const [toDelete, setToDelete] = useState<Node | null>(null);

  const handleRefresh = async () => {
    setRefreshing(true);
    try {
      await refresh();
    } finally {
      setRefreshing(false);
    }
  };

  const closeAdd = () => {
    setAddOpen(false);
    if (searchParams.get("add")) setSearchParams({}, { replace: true });
  };

  const setNodeEnabled = async (node: Node, enabled: boolean) => {
    try {
      await api(enabled ? "/api/enable-node" : "/api/disable-node", { name: node.name });
      toast.success(enabled ? `Node ${node.name} enabled` : `Node ${node.name} is now in maintenance`);
      refresh();
    } catch (err) {
      toast.error(errorMessage(err, "Failed to update node"));
    }
  };

  const handleDelete = async () => {
    if (!toDelete) return;
    try {
      await api("/api/delete-node", { name: toDelete.name });
      toast.success(`Node ${toDelete.name} removed`);
      refresh();
    } catch (err) {
      toast.error(errorMessage(err, "Failed to remove node"));
      throw err;
    }
  };

  return (
    <>
      <PageHeader
        title="Nodes"
        description="Libvirt hosts managed by this JVP server."
        onRefresh={handleRefresh}
        refreshing={refreshing}
        actions={
          <button className="btn-primary" onClick={() => setAddOpen(true)}>
            <Plus size={15} />
            Add node
          </button>
        }
      />

      <Table
        rows={nodes}
        rowKey={(n) => n.uuid || n.name}
        loading={loading}
        loadingLabel="Loading nodes…"
        onRowClick={(n) => navigate(`/nodes/${encodeURIComponent(n.name)}`)}
        empty={
          <EmptyState
            icon={<ServerCog size={20} />}
            title="No nodes yet"
            description="Add a libvirt host to start managing virtual machines."
            action={
              <button className="btn-primary" onClick={() => setAddOpen(true)}>
                <Plus size={15} />
                Add node
              </button>
            }
          />
        }
        columns={[
          {
            key: "name",
            header: "Name",
            render: (n) => (
              <div className="flex items-center gap-2.5">
                <div className="flex h-8 w-8 items-center justify-center rounded-md bg-subtle text-fg-subtle">
                  <ServerCog size={15} />
                </div>
                <span className="font-medium">{n.name}</span>
              </div>
            ),
          },
          { key: "state", header: "Status", render: (n) => <StatusBadge status={n.state} /> },
          { key: "type", header: "Type", render: (n) => <Badge className="capitalize">{n.type}</Badge> },
          {
            key: "uri",
            header: "URI",
            render: (n) => <span className="font-mono text-xs text-fg-muted">{n.uri}</span>,
          },
          {
            key: "actions",
            header: <span className="sr-only">Actions</span>,
            align: "right",
            render: (n) => (
              <div onClick={(e) => e.stopPropagation()}>
                <DropdownMenu
                  items={[
                    { label: "View details", icon: <Info size={14} />, onClick: () => navigate(`/nodes/${encodeURIComponent(n.name)}`) },
                    n.state === "maintenance"
                      ? { label: "Enable node", icon: <Power size={14} />, onClick: () => setNodeEnabled(n, true) }
                      : { label: "Enter maintenance", icon: <PowerOff size={14} />, onClick: () => setNodeEnabled(n, false) },
                    { divider: true, label: "divider" },
                    { label: "Remove node", icon: <Trash2 size={14} />, danger: true, onClick: () => setToDelete(n) },
                  ]}
                />
              </div>
            ),
          },
        ]}
      />

      {addOpen && (
        <AddNodeModal
          onClose={closeAdd}
          onCreated={() => {
            closeAdd();
            refresh();
          }}
        />
      )}

      <ConfirmDialog
        isOpen={Boolean(toDelete)}
        onClose={() => setToDelete(null)}
        onConfirm={handleDelete}
        title="Remove node?"
        message={
          <>
            <strong className="text-fg">{toDelete?.name}</strong> will be disconnected from JVP. Virtual machines on the host keep running and
            are not deleted.
          </>
        }
        confirmText="Remove"
      />
    </>
  );
}
