"use client";

import { useState, useEffect } from "react";
import DashboardLayout from "@/components/DashboardLayout";
import Header from "@/components/Header";
import Table from "@/components/Table";
import Modal from "@/components/Modal";
import { Plus, Trash2, Download, Upload } from "lucide-react";

interface KeyPair {
  id: string;
  name: string;
  fingerprint: string;
  public_key: string;
  created_at: string;
}

export default function KeyPairsPage() {
  const [keypairs, setKeypairs] = useState<KeyPair[]>([]);
  const [loading, setLoading] = useState(true);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isImportModalOpen, setIsImportModalOpen] = useState(false);
  const [privateKey, setPrivateKey] = useState("");
  const [createFormData, setCreateFormData] = useState({
    name: "",
    key_type: "rsa",
  });
  const [importFormData, setImportFormData] = useState({
    name: "",
    public_key: "",
  });

  const fetchKeypairs = async () => {
    setLoading(true);
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
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchKeypairs();
  }, []);

  const handleCreateKeypair = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const response = await fetch("/api/keypairs/create", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(createFormData),
      });

      if (response.ok) {
        const data = await response.json();
        setPrivateKey(data.private_key);
        fetchKeypairs();
        setCreateFormData({ name: "", key_type: "rsa" });
      }
    } catch (error) {
      console.error("Failed to create keypair:", error);
    }
  };

  const handleImportKeypair = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const response = await fetch("/api/keypairs/import", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(importFormData),
      });

      if (response.ok) {
        setIsImportModalOpen(false);
        fetchKeypairs();
        setImportFormData({ name: "", public_key: "" });
      }
    } catch (error) {
      console.error("Failed to import keypair:", error);
    }
  };

  const handleDownloadPrivateKey = () => {
    const blob = new Blob([privateKey], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${createFormData.name}.pem`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const handleDelete = async (keypairName: string) => {
    if (!confirm("Are you sure you want to delete this key pair?")) return;

    try {
      await fetch("/api/keypairs/delete", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ keypair_id: keypairName }),
      });
      fetchKeypairs();
    } catch (error) {
      console.error("Failed to delete keypair:", error);
    }
  };

  const columns = [
    { key: "name", label: "Name" },
    {
      key: "fingerprint",
      label: "Fingerprint",
      render: (value: unknown) => (
        <code className="text-xs bg-gray-100 px-2 py-1 rounded">
          {String(value)}
        </code>
      ),
    },
    {
      key: "created_at",
      label: "Created",
      render: (value: unknown) => {
        if (!value) return "-";
        return new Date(String(value)).toLocaleDateString();
      },
    },
    {
      key: "actions",
      label: "Actions",
      render: (_: unknown, row: Record<string, unknown>) => {
        const keypair = row as unknown as KeyPair;
        return (
          <div className="flex gap-2">
            <button
              onClick={() => handleDelete(keypair.name)}
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
        title="Key Pairs"
        description="Manage your SSH key pairs"
        action={
          <div className="flex gap-3">
            <button
              onClick={() => setIsImportModalOpen(true)}
              className="btn-secondary flex items-center gap-2"
            >
              <Upload size={16} />
              Import Key
            </button>
            <button
              onClick={() => setIsCreateModalOpen(true)}
              className="btn-primary flex items-center gap-2"
            >
              <Plus size={16} />
              Create Key Pair
            </button>
          </div>
        }
        onRefresh={fetchKeypairs}
      />

      {loading ? (
        <div className="card text-center py-12">
          <p className="text-gray-500">Loading key pairs...</p>
        </div>
      ) : (
        <Table
          columns={columns}
          data={keypairs}
          emptyMessage="No key pairs found. Create or import your first key pair to get started."
        />
      )}

      <Modal
        isOpen={isCreateModalOpen}
        onClose={() => {
          setIsCreateModalOpen(false);
          setPrivateKey("");
        }}
        title="Create New Key Pair"
      >
        {!privateKey ? (
          <form onSubmit={handleCreateKeypair} className="space-y-6">
            <div>
              <label className="label">Key Pair Name</label>
              <input
                type="text"
                className="input"
                value={createFormData.name}
                onChange={(e) =>
                  setCreateFormData({ ...createFormData, name: e.target.value })
                }
                required
                placeholder="my-keypair"
              />
            </div>

            <div>
              <label className="label">Key Type</label>
              <select
                className="input"
                value={createFormData.key_type}
                onChange={(e) =>
                  setCreateFormData({ ...createFormData, key_type: e.target.value })
                }
              >
                <option value="rsa">RSA (2048 bits)</option>
                <option value="ed25519">Ed25519</option>
              </select>
            </div>

            <div className="flex justify-end gap-3 pt-4">
              <button
                type="button"
                onClick={() => setIsCreateModalOpen(false)}
                className="btn-secondary"
              >
                Cancel
              </button>
              <button type="submit" className="btn-primary">
                Create Key Pair
              </button>
            </div>
          </form>
        ) : (
          <div className="space-y-6">
            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
              <p className="text-sm text-yellow-800 font-medium mb-2">
                Important: Download your private key now
              </p>
              <p className="text-xs text-yellow-700">
                This is the only time you will be able to download the private key.
                Store it securely.
              </p>
            </div>

            <div>
              <label className="label">Private Key</label>
              <textarea
                className="input font-mono text-xs"
                rows={10}
                value={privateKey}
                readOnly
              />
            </div>

            <div className="flex justify-end gap-3 pt-4">
              <button
                onClick={() => {
                  setIsCreateModalOpen(false);
                  setPrivateKey("");
                }}
                className="btn-secondary"
              >
                Close
              </button>
              <button
                onClick={handleDownloadPrivateKey}
                className="btn-primary flex items-center gap-2"
              >
                <Download size={16} />
                Download Private Key
              </button>
            </div>
          </div>
        )}
      </Modal>

      <Modal
        isOpen={isImportModalOpen}
        onClose={() => setIsImportModalOpen(false)}
        title="Import Key Pair"
      >
        <form onSubmit={handleImportKeypair} className="space-y-6">
          <div>
            <label className="label">Key Pair Name</label>
            <input
              type="text"
              className="input"
              value={importFormData.name}
              onChange={(e) =>
                setImportFormData({ ...importFormData, name: e.target.value })
              }
              required
              placeholder="my-keypair"
            />
          </div>

          <div>
            <label className="label">Public Key</label>
            <textarea
              className="input font-mono text-xs"
              rows={6}
              value={importFormData.public_key}
              onChange={(e) =>
                setImportFormData({ ...importFormData, public_key: e.target.value })
              }
              required
              placeholder="ssh-rsa AAAAB3NzaC1yc2E..."
            />
            <p className="text-xs text-gray-500 mt-1">
              Paste your public key (usually from ~/.ssh/id_rsa.pub)
            </p>
          </div>

          <div className="flex justify-end gap-3 pt-4">
            <button
              type="button"
              onClick={() => setIsImportModalOpen(false)}
              className="btn-secondary"
            >
              Cancel
            </button>
            <button type="submit" className="btn-primary">
              Import Key Pair
            </button>
          </div>
        </form>
      </Modal>
    </DashboardLayout>
  );
}
