"use client";

import { useState, useEffect } from "react";
import DashboardLayout from "@/components/DashboardLayout";
import Header from "@/components/Header";
import Table from "@/components/Table";
import StatusBadge from "@/components/StatusBadge";
import Modal from "@/components/Modal";
import { Plus, Trash2, Upload } from "lucide-react";

interface Image {
  id: string;
  name: string;
  description?: string;
  pool?: string;
  path?: string;
  size_gb: number;
  format?: string;
  state: string;
  created_at: string;
}

export default function ImagesPage() {
  const [images, setImages] = useState<Image[]>([]);
  const [loading, setLoading] = useState(true);
  const [isRegisterModalOpen, setIsRegisterModalOpen] = useState(false);
  const [formData, setFormData] = useState({
    name: "",
    url: "",
    os_type: "linux",
    description: "",
  });

  const fetchImages = async () => {
    setLoading(true);
    try {
      const response = await fetch("/api/images/describe", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });
      if (response.ok) {
        const data = await response.json();
        setImages(data.images || []);
      }
    } catch (error) {
      console.error("Failed to fetch images:", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchImages();
  }, []);

  const handleRegisterImage = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const response = await fetch("/api/images/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(formData),
      });

      if (response.ok) {
        setIsRegisterModalOpen(false);
        fetchImages();
        setFormData({
          name: "",
          url: "",
          os_type: "linux",
          description: "",
        });
      }
    } catch (error) {
      console.error("Failed to register image:", error);
    }
  };

  const handleDelete = async (imageId: string) => {
    if (!confirm("Are you sure you want to delete this image?")) return;

    try {
      await fetch("/api/images/deregister", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ imageID: imageId }),
      });
      fetchImages();
    } catch (error) {
      console.error("Failed to delete image:", error);
    }
  };

  const columns = [
    {
      key: "id",
      label: "ID",
      render: (value: unknown) => (
        <span className="font-mono text-xs">
          {String(value).substring(0, 12)}...
        </span>
      ),
    },
    {
      key: "name",
      label: "Name",
      render: (_: unknown, row: any) => (
        <span className="font-medium">
          {row.name}
        </span>
      ),
    },
    {
      key: "state",
      label: "Status",
      render: (value: unknown) => <StatusBadge status={String(value)} />,
    },
    {
      key: "format",
      label: "Format",
      render: (value: unknown) => <span>{value ? String(value).toUpperCase() : "QCOW2"}</span>,
    },
    {
      key: "size_gb",
      label: "Size",
      render: (value: unknown) => <span>{Number(value).toFixed(2)} GB</span>,
    },
    {
      key: "created_at",
      label: "Created",
      render: (value: unknown) => {
        if (!value) return <span>-</span>;
        return <span>{new Date(String(value)).toLocaleDateString()}</span>;
      },
    },
    {
      key: "actions",
      label: "Actions",
      render: (_: unknown, row: Record<string, unknown>) => {
        const image = row as unknown as Image;
        return (
          <div className="flex gap-2">
            <button
              onClick={() => handleDelete(image.id)}
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
        title="Images"
        description="Manage your system images"
        action={
          <button
            onClick={() => setIsRegisterModalOpen(true)}
            className="btn-primary flex items-center gap-2"
          >
            <Upload size={16} />
            Register Image
          </button>
        }
        onRefresh={fetchImages}
      />

      {loading ? (
        <div className="card text-center py-12">
          <p className="text-gray-500">Loading images...</p>
        </div>
      ) : (
        <Table
          columns={columns}
          data={images}
          emptyMessage="No images found. Register your first image to get started."
        />
      )}

      <Modal
        isOpen={isRegisterModalOpen}
        onClose={() => setIsRegisterModalOpen(false)}
        title="Register New Image"
        maxWidth="lg"
      >
        <form onSubmit={handleRegisterImage} className="space-y-6">
          <div>
            <label className="label">Image Name</label>
            <input
              type="text"
              className="input"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              required
              placeholder="ubuntu-22.04-server"
            />
          </div>

          <div>
            <label className="label">Image URL</label>
            <input
              type="url"
              className="input"
              value={formData.url}
              onChange={(e) => setFormData({ ...formData, url: e.target.value })}
              required
              placeholder="https://cloud-images.ubuntu.com/..."
            />
            <p className="text-xs text-gray-500 mt-1">
              URL to the image file (ISO, QCOW2, etc.)
            </p>
          </div>

          <div>
            <label className="label">OS Type</label>
            <select
              className="input"
              value={formData.os_type}
              onChange={(e) =>
                setFormData({ ...formData, os_type: e.target.value })
              }
            >
              <option value="linux">Linux</option>
              <option value="windows">Windows</option>
              <option value="other">Other</option>
            </select>
          </div>

          <div>
            <label className="label">Description (Optional)</label>
            <textarea
              className="input"
              rows={3}
              value={formData.description}
              onChange={(e) =>
                setFormData({ ...formData, description: e.target.value })
              }
              placeholder="Brief description of this image"
            />
          </div>

          <div className="flex justify-end gap-3 pt-4">
            <button
              type="button"
              onClick={() => setIsRegisterModalOpen(false)}
              className="btn-secondary"
            >
              Cancel
            </button>
            <button type="submit" className="btn-primary">
              Register Image
            </button>
          </div>
        </form>
      </Modal>
    </DashboardLayout>
  );
}
