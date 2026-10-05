import { useCallback, useEffect, useState } from "react";
import type { ReactNode } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { Check, Cpu, Database, MemoryStick, Network, Power, PowerOff, Server, ServerCog, X } from "lucide-react";
import PageHeader from "@/components/PageHeader";
import Table from "@/components/Table";
import type { Column } from "@/components/Table";
import { Badge, Card, CopyButton, DescriptionList, EmptyState, LoadingState, StatCard, StatusBadge, Tabs, UsageBar } from "@/components/ui";
import { useToast } from "@/components/ToastContainer";
import { api, errorMessage } from "@/lib/api";
import { formatBytes, formatMemoryMB } from "@/lib/format";
import { useNodes } from "@/lib/nodes";
import type { Node } from "@/lib/types";

interface NodeSummary {
  cpu: {
    cores: number;
    threads: number;
    model: string;
    vendor: string;
    frequency: number;
    arch: string;
    cache_size: number;
    flags: string[];
  };
  memory: {
    total: number;
    available: number;
    used: number;
    usage_percent: number;
    swap_total: number;
    swap_used: number;
  };
  numa?: { node_count: number };
  hugepages?: { enabled: boolean; page_sizes?: { size: string; total: number; free: number; used: number }[] };
  virtualization?: { vtx: boolean; ept: boolean; iommu: boolean; nested_virt: boolean };
}

interface PCIDevice {
  address: string;
  vendor: string;
  device: string;
  class?: string;
  driver?: string;
  iommu_group: number;
  memory?: number;
}

interface USBDevice {
  bus: number;
  device: number;
  vendor_id: string;
  product_id: string;
  vendor: string;
  product: string;
}

interface NetInterface {
  name: string;
  mac: string;
  speed?: string;
  state?: string;
  ip?: string[];
  bound_to?: string;
}

interface PhysicalDisk {
  name: string;
  type: string;
  size: number;
  model?: string;
  serial?: string;
  interface?: string;
  smart?: { health?: string; temperature?: number };
}

interface NodeVM {
  uuid: string;
  name: string;
  state: string;
  cpus: number;
  memory: number; // KB
}

type TabId = "overview" | "pci" | "gpu" | "usb" | "net" | "disks" | "vms";

const tabEndpoints: Record<Exclude<TabId, "overview">, { endpoint: string; key: string }> = {
  pci: { endpoint: "/api/describe-node-pci", key: "devices" },
  gpu: { endpoint: "/api/describe-node-gpu", key: "devices" },
  usb: { endpoint: "/api/describe-node-usb", key: "devices" },
  net: { endpoint: "/api/describe-node-net", key: "interfaces" },
  disks: { endpoint: "/api/describe-node-disks", key: "disks" },
  vms: { endpoint: "/api/describe-node-vms", key: "vms" },
};

function Feature({ enabled, label }: { enabled?: boolean; label: string }) {
  return (
    <div className="flex items-center gap-2.5 rounded-md border border-line px-3 py-2.5">
      <span
        className={`flex h-5 w-5 items-center justify-center rounded-full ${enabled ? "bg-success-soft text-success" : "bg-subtle text-fg-subtle"}`}
      >
        {enabled ? <Check size={12} strokeWidth={3} /> : <X size={12} strokeWidth={3} />}
      </span>
      <span className={`text-sm ${enabled ? "text-fg" : "text-fg-muted"}`}>{label}</span>
    </div>
  );
}

function DeviceTable<T>({
  rows,
  columns,
  emptyLabel,
  rowKey,
}: {
  rows: T[];
  columns: Column<T>[];
  emptyLabel: string;
  rowKey: (row: T, index: number) => string;
}) {
  return <Table rows={rows} columns={columns} rowKey={rowKey} empty={<EmptyState title={emptyLabel} />} />;
}

export default function NodeDetailPage() {
  const toast = useToast();
  const { name: nodeName = "" } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  const tab = (searchParams.get("tab") as TabId) || "overview";
  const { refresh: refreshNodes, setCurrentNode } = useNodes();

  const [node, setNode] = useState<Node | null>(null);
  const [summary, setSummary] = useState<NodeSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [summaryError, setSummaryError] = useState("");
  const [actionLoading, setActionLoading] = useState(false);
  const [deviceData, setDeviceData] = useState<Partial<Record<TabId, unknown[]>>>({});
  const [deviceLoading, setDeviceLoading] = useState(false);

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

  const load = useCallback(async () => {
    try {
      const data = await api<Node>("/api/describe-node", { name: nodeName });
      setNode(data);
    } catch (err) {
      setNode(null);
      toast.error(errorMessage(err, "Failed to load node"));
      setLoading(false);
      return;
    }
    try {
      const data = await api<NodeSummary>("/api/describe-node-summary", { name: nodeName });
      setSummary(data);
      setSummaryError("");
    } catch (err) {
      setSummary(null);
      setSummaryError(errorMessage(err, "Failed to load hardware summary"));
    } finally {
      setLoading(false);
    }
  }, [nodeName, toast]);

  useEffect(() => {
    setLoading(true);
    setDeviceData({});
    load();
  }, [load]);

  const loadDevices = useCallback(
    async (target: TabId, force = false) => {
      if (target === "overview") return;
      if (!force && deviceData[target]) return;
      const { endpoint, key } = tabEndpoints[target];
      setDeviceLoading(true);
      try {
        const data = await api<Record<string, unknown[]>>(endpoint, { name: nodeName });
        setDeviceData((prev) => ({ ...prev, [target]: data[key] || [] }));
      } catch (err) {
        toast.error(errorMessage(err, "Failed to load devices"));
        setDeviceData((prev) => ({ ...prev, [target]: [] }));
      } finally {
        setDeviceLoading(false);
      }
    },
    [deviceData, nodeName, toast]
  );

  useEffect(() => {
    loadDevices(tab);
  }, [tab, loadDevices]);

  const handleRefresh = async () => {
    setRefreshing(true);
    await load();
    if (tab !== "overview") await loadDevices(tab, true);
    setRefreshing(false);
  };

  const setEnabled = async (enabled: boolean) => {
    setActionLoading(true);
    try {
      await api(enabled ? "/api/enable-node" : "/api/disable-node", { name: nodeName });
      toast.success(enabled ? "Node enabled" : "Node is now in maintenance mode");
      await load();
      refreshNodes();
    } catch (err) {
      toast.error(errorMessage(err, "Failed to update node"));
    } finally {
      setActionLoading(false);
    }
  };

  if (loading && !node) return <LoadingState label="Loading node…" />;

  if (!node) {
    return (
      <div className="card">
        <EmptyState
          icon={<ServerCog size={20} />}
          title="Node not found"
          description={`No node named "${nodeName}" is registered.`}
          action={
            <Link to="/nodes" className="btn-primary">
              Back to nodes
            </Link>
          }
        />
      </div>
    );
  }

  const rows = (deviceData[tab] || []) as never[];
  const scoped = (path: string) => `${path}?node=${encodeURIComponent(node.name)}`;

  let tabContent: ReactNode = null;
  if (tab !== "overview" && deviceLoading && !deviceData[tab]) {
    tabContent = <LoadingState label="Loading…" className="card" />;
  } else if (tab === "pci" || tab === "gpu") {
    tabContent = (
      <DeviceTable<PCIDevice>
        rows={rows}
        rowKey={(d) => d.address}
        emptyLabel={tab === "gpu" ? "No GPUs found" : "No PCI devices found"}
        columns={[
          { key: "address", header: "Address", render: (d) => <span className="font-mono text-xs">{d.address}</span> },
          {
            key: "device",
            header: "Device",
            render: (d) => (
              <div>
                <div className="font-medium">{d.device || "Unknown device"}</div>
                <div className="text-xs text-fg-muted">{d.vendor}</div>
              </div>
            ),
          },
          ...(tab === "gpu"
            ? [{ key: "memory", header: "Memory", render: (d: PCIDevice) => (d.memory ? formatMemoryMB(d.memory) : "—") }]
            : [{ key: "class", header: "Class", render: (d: PCIDevice) => <span className="text-fg-muted">{d.class || "—"}</span> }]),
          { key: "driver", header: "Driver", render: (d) => <span className="font-mono text-xs text-fg-muted">{d.driver || "—"}</span> },
          {
            key: "iommu",
            header: "IOMMU group",
            render: (d) => (d.iommu_group >= 0 ? <Badge tone="info">{d.iommu_group}</Badge> : <span className="text-fg-subtle">—</span>),
          },
        ]}
      />
    );
  } else if (tab === "usb") {
    tabContent = (
      <DeviceTable<USBDevice>
        rows={rows}
        rowKey={(d, i) => `${d.bus}-${d.device}-${i}`}
        emptyLabel="No USB devices found"
        columns={[
          {
            key: "product",
            header: "Device",
            render: (d) => (
              <div>
                <div className="font-medium">{d.product || "Unknown device"}</div>
                <div className="text-xs text-fg-muted">{d.vendor}</div>
              </div>
            ),
          },
          {
            key: "ids",
            header: "Vendor:Product",
            render: (d) => (
              <span className="font-mono text-xs">
                {d.vendor_id || "????"}:{d.product_id || "????"}
              </span>
            ),
          },
          { key: "bus", header: "Bus / Device", render: (d) => <span className="tabular-nums text-fg-muted">{d.bus} / {d.device}</span> },
        ]}
      />
    );
  } else if (tab === "net") {
    tabContent = (
      <DeviceTable<NetInterface>
        rows={rows}
        rowKey={(d) => d.name}
        emptyLabel="No network interfaces found"
        columns={[
          { key: "name", header: "Interface", render: (d) => <span className="font-mono font-medium">{d.name}</span> },
          { key: "state", header: "State", render: (d) => <StatusBadge status={d.state || "unknown"} /> },
          { key: "mac", header: "MAC", render: (d) => <span className="font-mono text-xs text-fg-muted">{d.mac || "—"}</span> },
          {
            key: "ip",
            header: "Addresses",
            render: (d) =>
              d.ip?.length ? (
                <div className="space-y-0.5 font-mono text-xs">
                  {d.ip.map((ip) => (
                    <div key={ip}>{ip}</div>
                  ))}
                </div>
              ) : (
                <span className="text-fg-subtle">—</span>
              ),
          },
          { key: "speed", header: "Speed", render: (d) => <span className="text-fg-muted">{d.speed || "—"}</span> },
          { key: "bridge", header: "Bridge", render: (d) => (d.bound_to ? <Badge>{d.bound_to}</Badge> : <span className="text-fg-subtle">—</span>) },
        ]}
      />
    );
  } else if (tab === "disks") {
    tabContent = (
      <DeviceTable<PhysicalDisk>
        rows={rows}
        rowKey={(d) => d.name}
        emptyLabel="No physical disks found"
        columns={[
          {
            key: "name",
            header: "Disk",
            render: (d) => (
              <div>
                <div className="font-mono font-medium">{d.name}</div>
                {d.model && <div className="text-xs text-fg-muted">{d.model}</div>}
              </div>
            ),
          },
          {
            key: "type",
            header: "Type",
            render: (d) => <Badge tone={d.type === "NVMe" ? "accent" : d.type === "SSD" ? "info" : "neutral"}>{d.type || "—"}</Badge>,
          },
          { key: "size", header: "Size", render: (d) => (d.size > 0 ? formatBytes(d.size) : "—") },
          { key: "serial", header: "Serial", render: (d) => <span className="font-mono text-xs text-fg-muted">{d.serial || "—"}</span> },
          {
            key: "health",
            header: "Health",
            render: (d) =>
              d.smart?.health ? (
                <span className="text-fg-muted">
                  {d.smart.health}
                  {d.smart.temperature ? ` · ${d.smart.temperature}°C` : ""}
                </span>
              ) : (
                <span className="text-fg-subtle">—</span>
              ),
          },
        ]}
      />
    );
  } else if (tab === "vms") {
    tabContent = (
      <DeviceTable<NodeVM>
        rows={rows}
        rowKey={(d) => d.uuid}
        emptyLabel="No virtual machines on this node"
        columns={[
          {
            key: "name",
            header: "Name",
            render: (d) => (
              <Link
                to={`/instances/${encodeURIComponent(node.name)}/${encodeURIComponent(d.name)}`}
                className="font-medium text-fg hover:text-accent"
              >
                {d.name}
              </Link>
            ),
          },
          { key: "state", header: "State", render: (d) => <StatusBadge status={d.state} /> },
          { key: "cpus", header: "vCPUs", render: (d) => (d.cpus > 0 ? d.cpus : "—") },
          { key: "memory", header: "Memory", render: (d) => (d.memory > 0 ? formatBytes(d.memory * 1024) : "—") },
          { key: "uuid", header: "UUID", render: (d) => <span className="font-mono text-xs text-fg-subtle">{d.uuid}</span> },
        ]}
      />
    );
  } else if (tab === "overview") {
    tabContent = (
      <div className="space-y-6">
        {summary && (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <StatCard label="CPU" icon={<Cpu size={16} />} value={`${summary.cpu.threads || summary.cpu.cores} threads`} hint={`${summary.cpu.cores} cores · ${summary.cpu.arch}`} />
            <StatCard
              label="Memory"
              icon={<MemoryStick size={16} />}
              value={formatBytes(summary.memory.used)}
              hint={`of ${formatBytes(summary.memory.total)} · ${summary.memory.usage_percent.toFixed(0)}% used`}
            >
              <UsageBar value={summary.memory.usage_percent} size="sm" />
            </StatCard>
            <StatCard
              label="Swap"
              icon={<Database size={16} />}
              value={formatBytes(summary.memory.swap_used)}
              hint={`of ${formatBytes(summary.memory.swap_total)}`}
            />
            <StatCard label="NUMA nodes" icon={<Server size={16} />} value={summary.numa?.node_count ?? "—"} hint={summary.hugepages?.enabled ? "Huge pages enabled" : "Huge pages disabled"} />
          </div>
        )}

        <div className="grid grid-cols-1 gap-6 xl:grid-cols-2">
          <Card title="Node">
            <DescriptionList
              items={[
                { label: "Name", value: node.name },
                { label: "Status", value: <StatusBadge status={node.state} /> },
                { label: "Type", value: <span className="capitalize">{node.type}</span> },
                { label: "UUID", value: node.uuid, mono: true, copy: node.uuid },
                { label: "Libvirt URI", value: node.uri, mono: true, copy: node.uri, span: true },
              ]}
            />
          </Card>

          {summary ? (
            <Card title="Processor">
              <DescriptionList
                items={[
                  { label: "Model", value: summary.cpu.model, span: true },
                  { label: "Vendor", value: summary.cpu.vendor },
                  { label: "Frequency", value: summary.cpu.frequency ? `${summary.cpu.frequency} MHz` : undefined },
                  { label: "Cache", value: summary.cpu.cache_size ? `${summary.cpu.cache_size} KB` : undefined },
                  { label: "Architecture", value: summary.cpu.arch },
                ]}
              />
            </Card>
          ) : (
            <Card title="Hardware">
              <p className="text-sm text-fg-muted">{summaryError || "Hardware information is not available."}</p>
            </Card>
          )}
        </div>

        {summary?.virtualization && (
          <Card title="Virtualization features">
            <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
              <Feature enabled={summary.virtualization.vtx} label="VT-x / AMD-V" />
              <Feature enabled={summary.virtualization.ept} label="EPT / NPT" />
              <Feature enabled={summary.virtualization.iommu} label="IOMMU" />
              <Feature enabled={summary.virtualization.nested_virt} label="Nested virtualization" />
            </div>
          </Card>
        )}

        {summary?.cpu.flags?.length ? (
          <Card title="CPU flags" description={`${summary.cpu.flags.length} flags`}>
            <div className="flex max-h-40 flex-wrap gap-1 overflow-y-auto">
              {summary.cpu.flags.map((flag) => (
                <span key={flag} className="code">
                  {flag}
                </span>
              ))}
            </div>
          </Card>
        ) : null}
      </div>
    );
  }

  return (
    <>
      <PageHeader
        breadcrumbs={[{ label: "Nodes", to: "/nodes" }, { label: node.name }]}
        title={node.name}
        icon={<ServerCog size={18} />}
        meta={
          <>
            <StatusBadge status={node.state} />
            <Badge className="capitalize">{node.type}</Badge>
            <span className="flex items-center gap-1 font-mono text-xs text-fg-subtle">
              {node.uri}
              <CopyButton text={node.uri} />
            </span>
          </>
        }
        onRefresh={handleRefresh}
        refreshing={refreshing}
        actions={
          <>
            <Link to={scoped("/instances")} className="btn-secondary" onClick={() => setCurrentNode(node.name)}>
              <Server size={14} />
              Instances
            </Link>
            <Link to={scoped("/networks")} className="btn-secondary" onClick={() => setCurrentNode(node.name)}>
              <Network size={14} />
              Networks
            </Link>
            <Link to={scoped("/storage-pools")} className="btn-secondary" onClick={() => setCurrentNode(node.name)}>
              <Database size={14} />
              Storage
            </Link>
            {node.state === "maintenance" ? (
              <button className="btn-primary" onClick={() => setEnabled(true)} disabled={actionLoading}>
                <Power size={14} />
                Enable
              </button>
            ) : (
              <button className="btn-secondary" onClick={() => setEnabled(false)} disabled={actionLoading}>
                <PowerOff size={14} />
                Maintenance
              </button>
            )}
          </>
        }
      />

      <Tabs
        className="mb-6"
        value={tab}
        onChange={setTab}
        tabs={[
          { id: "overview", label: "Overview" },
          { id: "vms", label: "Virtual machines", count: deviceData.vms?.length },
          { id: "net", label: "Network", count: deviceData.net?.length },
          { id: "disks", label: "Disks", count: deviceData.disks?.length },
          { id: "pci", label: "PCI", count: deviceData.pci?.length },
          { id: "gpu", label: "GPU", count: deviceData.gpu?.length },
          { id: "usb", label: "USB", count: deviceData.usb?.length },
        ]}
      />

      {tabContent}
    </>
  );
}
