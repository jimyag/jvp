"use client";

import { useState, useEffect } from "react";
import DashboardLayout from "@/components/DashboardLayout";
import Header from "@/components/Header";
import Table from "@/components/Table";
import { useToast } from "@/components/ToastContainer";
import { RefreshCw, Package } from "lucide-react";

interface VMTemplate {
  id: string;
  name: string;
  description: string;
  sourceVM: string;
  vcpus: number;
  memory: number;
  diskSize: number;
  createdAt: string;
}

export default function TemplatesPage() {
  const toast = useToast();
  const [templates, setTemplates] = useState<VMTemplate[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchTemplates = async () => {
    setLoading(true);
    try {
      const response = await fetch("/api/vm-templates", {
        method: "GET",
        headers: { "Content-Type": "application/json" },
      });
      if (response.ok) {
        const data = await response.json();
        setTemplates(data.templates || []);
      } else {
        toast.error("Failed to load VM templates");
      }
    } catch (error) {
      console.error("Failed to fetch templates:", error);
      toast.error("Failed to load VM templates. Please check if backend is running.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTemplates();
  }, []);

  const formatBytes = (mb: number): string => {
    if (mb < 1024) return `${mb} MB`;
    return `${(mb / 1024).toFixed(2)} GB`;
  };

  const formatDate = (dateStr: string): string => {
    if (!dateStr) return "N/A";
    try {
      const date = new Date(dateStr);
      return date.toLocaleString();
    } catch {
      return dateStr;
    }
  };

  const columns = [
    {
      key: "id",
      label: "Template ID",
      render: (value: unknown) => (
        <span className="font-mono text-xs">
          {String(value).substring(0, 12)}...
        </span>
      ),
    },
    {
      key: "name",
      label: "Name",
      render: (_: unknown, row: VMTemplate) => (
        <div className="flex items-center gap-2">
          <Package className="w-4 h-4 text-blue-600" />
          <span className="font-medium">{row.name}</span>
        </div>
      ),
    },
    {
      key: "sourceVM",
      label: "Source VM",
      render: (value: unknown) => (
        <span className="text-gray-700">{String(value)}</span>
      ),
    },
    {
      key: "description",
      label: "Description",
      render: (value: unknown) => (
        <span className="text-sm text-gray-600">{String(value)}</span>
      ),
    },
    {
      key: "vcpus",
      label: "vCPUs",
      render: (value: unknown) => (
        <span className="font-mono">{String(value)}</span>
      ),
    },
    {
      key: "memory",
      label: "Memory",
      render: (value: unknown) => (
        <span className="font-mono">{formatBytes(Number(value))}</span>
      ),
    },
    {
      key: "diskSize",
      label: "Disk Size",
      render: (value: unknown) => (
        <span className="font-mono">{Number(value)} GB</span>
      ),
    },
    {
      key: "createdAt",
      label: "Created At",
      render: (value: unknown) => (
        <span className="text-sm text-gray-600">{formatDate(String(value))}</span>
      ),
    },
  ];

  return (
    <DashboardLayout>
      <div className="space-y-6">
        <Header
          title="VM Templates"
          description="Virtual machines with snapshots that can be used to clone new instances"
          onRefresh={fetchTemplates}
        />

        {loading ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-center">
              <RefreshCw className="w-8 h-8 animate-spin text-primary mx-auto mb-2" />
              <p className="text-gray-600">Loading VM templates...</p>
            </div>
          </div>
        ) : templates.length === 0 ? (
          <div className="bg-white rounded-lg border border-gray-200 p-12">
            <div className="text-center">
              <Package className="w-12 h-12 text-gray-400 mx-auto mb-4" />
              <h3 className="text-lg font-medium text-gray-900 mb-2">
                No VM Templates Found
              </h3>
              <p className="text-gray-600 mb-4">
                VM templates are virtual machines with snapshots that can be cloned.
              </p>
              <p className="text-sm text-gray-500">
                Create a snapshot of an instance to make it available as a template.
              </p>
            </div>
          </div>
        ) : (
          <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
            <Table
              data={templates}
              columns={columns}
              emptyMessage="No VM templates available"
            />
          </div>
        )}
      </div>
    </DashboardLayout>
  );
}
