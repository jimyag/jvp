import { useCallback, useEffect, useMemo, useState } from "react";
import { Download, KeyRound, Plus, Trash2, Upload } from "lucide-react";
import PageHeader from "@/components/PageHeader";
import Table from "@/components/Table";
import Modal from "@/components/Modal";
import ConfirmDialog from "@/components/ConfirmDialog";
import DropdownMenu from "@/components/DropdownMenu";
import SearchFilter from "@/components/SearchFilter";
import { Alert, Badge, CopyButton, EmptyState, Field, SegmentedControl, Spinner } from "@/components/ui";
import { useToast } from "@/components/ToastContainer";
import { api, errorMessage } from "@/lib/api";
import { formatDate, formatRelative } from "@/lib/format";
import type { KeyPair } from "@/lib/types";

function downloadText(fileName: string, content: string) {
  const blob = new Blob([content], { type: "text/plain" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = fileName;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

function CreateKeypairModal({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
  const toast = useToast();
  const [name, setName] = useState("");
  const [algorithm, setAlgorithm] = useState<"ed25519" | "rsa">("ed25519");
  const [saving, setSaving] = useState(false);
  const [created, setCreated] = useState<{ name: string; privateKey: string } | null>(null);
  const [downloaded, setDownloaded] = useState(false);

  const handleSubmit = async (e?: React.FormEvent) => {
    e?.preventDefault();
    if (!name.trim()) return;
    setSaving(true);
    try {
      const data = await api<{ keypair: KeyPair; private_key: string }>("/api/create-keypair", { name: name.trim(), algorithm });
      setCreated({ name: data.keypair?.name || name.trim(), privateKey: data.private_key });
      onCreated();
    } catch (err) {
      toast.error(errorMessage(err, "Failed to create key pair"));
    } finally {
      setSaving(false);
    }
  };

  const handleDownload = () => {
    if (!created) return;
    downloadText(`${created.name}.pem`, created.privateKey);
    setDownloaded(true);
  };

  if (created) {
    return (
      <Modal
        isOpen
        onClose={onClose}
        title="Save your private key"
        dismissible={downloaded}
        footer={
          <>
            <button className="btn-secondary" onClick={onClose}>
              {downloaded ? "Done" : "Close without saving"}
            </button>
            <button className="btn-primary" onClick={handleDownload}>
              <Download size={14} />
              Download {created.name}.pem
            </button>
          </>
        }
      >
        <div className="space-y-4">
          <Alert tone="warning" title="This is the only time the private key is shown">
            JVP does not store private keys. Download it now and keep it somewhere safe.
          </Alert>
          <div className="relative">
            <textarea className="input font-mono text-[11px] leading-snug" rows={10} value={created.privateKey} readOnly />
            <div className="absolute right-2 top-2">
              <CopyButton text={created.privateKey} label="Copy" />
            </div>
          </div>
          <p className="text-xs text-fg-muted">
            Then restrict its permissions: <code className="code">chmod 600 {created.name}.pem</code>
          </p>
        </div>
      </Modal>
    );
  }

  return (
    <Modal
      isOpen
      onClose={onClose}
      dismissible={!saving}
      title="Create key pair"
      description="JVP generates the key pair and keeps only the public key."
      footer={
        <>
          <button className="btn-secondary" onClick={onClose} disabled={saving}>
            Cancel
          </button>
          <button className="btn-primary" onClick={() => handleSubmit()} disabled={saving || !name.trim()}>
            {saving && <Spinner size={14} className="text-current" />}
            Create key pair
          </button>
        </>
      }
    >
      <form className="space-y-4" onSubmit={handleSubmit}>
        <Field label="Name" required>
          <input className="input" value={name} onChange={(e) => setName(e.target.value)} placeholder="my-laptop" autoFocus />
        </Field>
        <Field label="Algorithm" hint={algorithm === "ed25519" ? "Modern, short keys. Recommended." : "2048-bit RSA for older systems."}>
          <SegmentedControl
            value={algorithm}
            onChange={setAlgorithm}
            options={[
              { value: "ed25519", label: "Ed25519" },
              { value: "rsa", label: "RSA" },
            ]}
          />
        </Field>
      </form>
    </Modal>
  );
}

function ImportKeypairModal({ onClose, onImported }: { onClose: () => void; onImported: () => void }) {
  const toast = useToast();
  const [name, setName] = useState("");
  const [publicKey, setPublicKey] = useState("");
  const [saving, setSaving] = useState(false);
  const valid = name.trim() && publicKey.trim();

  const handleSubmit = async (e?: React.FormEvent) => {
    e?.preventDefault();
    if (!valid) return;
    setSaving(true);
    try {
      await api("/api/import-keypair", { name: name.trim(), public_key: publicKey.trim() });
      toast.success(`Key pair ${name.trim()} imported`);
      onImported();
    } catch (err) {
      toast.error(errorMessage(err, "Failed to import key pair"));
    } finally {
      setSaving(false);
    }
  };

  const loadFile = (file?: File) => {
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      const text = String(reader.result || "").trim();
      setPublicKey(text);
      if (!name) {
        const comment = text.split(/\s+/)[2];
        setName((comment || file.name.replace(/\.pub$/, "")).replace(/[^\w.-]+/g, "-"));
      }
    };
    reader.readAsText(file);
  };

  return (
    <Modal
      isOpen
      onClose={onClose}
      dismissible={!saving}
      title="Import public key"
      description="Use a key you already have, e.g. ~/.ssh/id_ed25519.pub."
      footer={
        <>
          <button className="btn-secondary" onClick={onClose} disabled={saving}>
            Cancel
          </button>
          <button className="btn-primary" onClick={() => handleSubmit()} disabled={saving || !valid}>
            {saving && <Spinner size={14} className="text-current" />}
            Import key
          </button>
        </>
      }
    >
      <form className="space-y-4" onSubmit={handleSubmit}>
        <Field label="Name" required>
          <input className="input" value={name} onChange={(e) => setName(e.target.value)} placeholder="my-laptop" autoFocus />
        </Field>
        <Field
          label="Public key"
          required
          hint={
            <label className="inline-flex cursor-pointer items-center gap-1">
              <input type="file" className="hidden" accept=".pub,text/plain" onChange={(e) => loadFile(e.target.files?.[0])} />
              <span className="link">Load from file…</span>
            </label>
          }
        >
          <textarea
            className="input font-mono text-xs"
            rows={5}
            value={publicKey}
            onChange={(e) => setPublicKey(e.target.value)}
            placeholder="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA… user@host"
          />
        </Field>
      </form>
    </Modal>
  );
}

export default function KeypairsPage() {
  const toast = useToast();
  const [keypairs, setKeypairs] = useState<KeyPair[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [query, setQuery] = useState("");
  const [modal, setModal] = useState<null | "create" | "import">(null);
  const [toDelete, setToDelete] = useState<KeyPair | null>(null);

  const fetchKeypairs = useCallback(
    async ({ silent = false }: { silent?: boolean } = {}) => {
      if (!silent) setRefreshing(true);
      try {
        const data = await api<{ keypairs: KeyPair[] }>("/api/describe-keypairs");
        setKeypairs(data.keypairs || []);
      } catch (err) {
        toast.error(errorMessage(err, "Failed to load key pairs"));
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    [toast]
  );

  useEffect(() => {
    fetchKeypairs({ silent: true });
  }, [fetchKeypairs]);

  const handleDelete = async () => {
    if (!toDelete) return;
    try {
      await api("/api/delete-keypair", { keypairID: toDelete.id });
      toast.success(`Key pair ${toDelete.name} deleted`);
      fetchKeypairs({ silent: true });
    } catch (err) {
      toast.error(errorMessage(err, "Failed to delete key pair"));
      throw err;
    }
  };

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return keypairs
      .filter((k) => !q || k.name.toLowerCase().includes(q) || k.fingerprint?.toLowerCase().includes(q))
      .sort((a, b) => a.name.localeCompare(b.name));
  }, [keypairs, query]);

  return (
    <>
      <PageHeader
        title="Key pairs"
        description="SSH public keys that can be injected into new instances."
        onRefresh={() => fetchKeypairs()}
        refreshing={refreshing}
        actions={
          <>
            <button className="btn-secondary" onClick={() => setModal("import")}>
              <Upload size={14} />
              Import
            </button>
            <button className="btn-primary" onClick={() => setModal("create")}>
              <Plus size={15} />
              Create key pair
            </button>
          </>
        }
      />

      {keypairs.length > 0 && (
        <div className="mb-4 flex justify-end">
          <SearchFilter value={query} onChange={setQuery} placeholder="Search name or fingerprint" className="sm:w-80" />
        </div>
      )}

      <Table
        rows={filtered}
        rowKey={(k) => k.id}
        loading={loading}
        loadingLabel="Loading key pairs…"
        empty={
          keypairs.length ? (
            <EmptyState title="No matching key pairs" />
          ) : (
            <EmptyState
              icon={<KeyRound size={20} />}
              title="No key pairs yet"
              description="Create a new key pair or import an existing public key to log in to instances over SSH."
              action={
                <div className="flex gap-2">
                  <button className="btn-secondary" onClick={() => setModal("import")}>
                    <Upload size={14} />
                    Import
                  </button>
                  <button className="btn-primary" onClick={() => setModal("create")}>
                    <Plus size={15} />
                    Create key pair
                  </button>
                </div>
              }
            />
          )
        }
        columns={[
          {
            key: "name",
            header: "Name",
            render: (k) => (
              <div className="flex items-center gap-2.5">
                <div className="flex h-8 w-8 items-center justify-center rounded-md bg-subtle text-fg-subtle">
                  <KeyRound size={15} />
                </div>
                <span className="font-medium">{k.name}</span>
              </div>
            ),
          },
          {
            key: "algorithm",
            header: "Type",
            render: (k) => (k.algorithm ? <Badge className="uppercase">{k.algorithm}</Badge> : <span className="text-fg-subtle">—</span>),
          },
          {
            key: "fingerprint",
            header: "Fingerprint",
            render: (k) => (
              <span className="flex items-center gap-1 font-mono text-xs text-fg-muted">
                <span className="max-w-[360px] truncate">{k.fingerprint}</span>
                <CopyButton text={k.fingerprint} />
              </span>
            ),
          },
          {
            key: "created",
            header: "Created",
            render: (k) => (
              <span className="whitespace-nowrap text-fg-muted" title={formatDate(k.created_at)}>
                {formatRelative(k.created_at)}
              </span>
            ),
          },
          {
            key: "actions",
            header: <span className="sr-only">Actions</span>,
            align: "right",
            render: (k) => (
              <div className="flex items-center justify-end gap-1">
                <CopyButton text={k.public_key} label="Public key" />
                <DropdownMenu
                  items={[
                    { label: "Download public key", icon: <Download size={14} />, onClick: () => downloadText(`${k.name}.pub`, k.public_key) },
                    { divider: true, label: "divider" },
                    { label: "Delete key pair", icon: <Trash2 size={14} />, danger: true, onClick: () => setToDelete(k) },
                  ]}
                />
              </div>
            ),
          },
        ]}
      />

      {modal === "create" && <CreateKeypairModal onClose={() => setModal(null)} onCreated={() => fetchKeypairs({ silent: true })} />}
      {modal === "import" && (
        <ImportKeypairModal
          onClose={() => setModal(null)}
          onImported={() => {
            setModal(null);
            fetchKeypairs({ silent: true });
          }}
        />
      )}

      <ConfirmDialog
        isOpen={Boolean(toDelete)}
        onClose={() => setToDelete(null)}
        onConfirm={handleDelete}
        title="Delete key pair?"
        message={
          <>
            <strong className="text-fg">{toDelete?.name}</strong> will no longer be offered for new instances. Keys already installed on
            existing instances are not removed.
          </>
        }
        confirmText="Delete"
      />
    </>
  );
}
