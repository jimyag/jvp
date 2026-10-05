import { useCallback, useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { Cable, Network, Play, Plus, Square, Trash2 } from "lucide-react";
import PageHeader from "@/components/PageHeader";
import Table from "@/components/Table";
import Modal from "@/components/Modal";
import ConfirmDialog from "@/components/ConfirmDialog";
import DropdownMenu from "@/components/DropdownMenu";
import NodeSelect from "@/components/NodeSelect";
import { Alert, Badge, Checkbox, EmptyState, Field, SegmentedControl, Spinner, StatusBadge, Tabs } from "@/components/ui";
import { useToast } from "@/components/ToastContainer";
import { api, errorMessage } from "@/lib/api";
import { netmaskToPrefix } from "@/lib/format";
import { useScopedNode } from "@/lib/nodes";
import type { HostBridge, LibvirtNetwork } from "@/lib/types";

type TabId = "networks" | "bridges";

type ConfirmTarget =
  | { kind: "stop-network"; network: LibvirtNetwork }
  | { kind: "delete-network"; network: LibvirtNetwork }
  | { kind: "delete-bridge"; bridge: HostBridge };

interface AvailableInterface {
  name: string;
  mac: string;
  state: string;
  bound_to?: string;
}

function dhcpRange(gateway: string): [string, string] {
  const parts = gateway.split(".");
  if (parts.length !== 4) return ["", ""];
  const prefix = parts.slice(0, 3).join(".");
  return [`${prefix}.100`, `${prefix}.200`];
}

function CreateNetworkModal({ nodeName, onClose, onCreated }: { nodeName: string; onClose: () => void; onCreated: () => void }) {
  const toast = useToast();
  const [name, setName] = useState("");
  const [mode, setMode] = useState<"nat" | "isolated">("nat");
  const [gateway, setGateway] = useState("192.168.100.1");
  const [netmask, setNetmask] = useState("255.255.255.0");
  const [dhcpStart, setDhcpStart] = useState("192.168.100.100");
  const [dhcpEnd, setDhcpEnd] = useState("192.168.100.200");
  const [dhcpTouched, setDhcpTouched] = useState(false);
  const [autostart, setAutostart] = useState(true);
  const [saving, setSaving] = useState(false);

  const updateGateway = (value: string) => {
    setGateway(value);
    if (!dhcpTouched) {
      const [start, end] = dhcpRange(value);
      if (start) {
        setDhcpStart(start);
        setDhcpEnd(end);
      }
    }
  };

  const handleSubmit = async (e?: React.FormEvent) => {
    e?.preventDefault();
    if (!name.trim()) return;
    setSaving(true);
    try {
      await api("/api/create-network", {
        node_name: nodeName,
        name: name.trim(),
        mode,
        ip_address: gateway.trim(),
        netmask,
        dhcp_start: dhcpStart.trim(),
        dhcp_end: dhcpEnd.trim(),
        autostart,
      });
      toast.success(`Network ${name.trim()} created`);
      onCreated();
    } catch (err) {
      toast.error(errorMessage(err, "Failed to create network"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      isOpen
      onClose={onClose}
      dismissible={!saving}
      title="Create libvirt network"
      description={`A virtual network with DHCP on node ${nodeName}.`}
      footer={
        <>
          <button className="btn-secondary" onClick={onClose} disabled={saving}>
            Cancel
          </button>
          <button className="btn-primary" onClick={() => handleSubmit()} disabled={saving || !name.trim()}>
            {saving && <Spinner size={14} className="text-current" />}
            Create network
          </button>
        </>
      }
    >
      <form className="space-y-4" onSubmit={handleSubmit}>
        <Field label="Name" required>
          <input className="input" value={name} onChange={(e) => setName(e.target.value)} placeholder="private" autoFocus />
        </Field>
        <Field
          label="Mode"
          hint={mode === "nat" ? "Instances reach the outside world through the host (NAT)." : "Instances can only talk to each other and the host."}
        >
          <SegmentedControl
            value={mode}
            onChange={setMode}
            options={[
              { value: "nat", label: "NAT" },
              { value: "isolated", label: "Isolated" },
            ]}
          />
        </Field>
        <div className="grid grid-cols-2 gap-4">
          <Field label="Gateway IP">
            <input className="input font-mono text-[13px]" value={gateway} onChange={(e) => updateGateway(e.target.value)} />
          </Field>
          <Field label="Netmask">
            <select className="input" value={netmask} onChange={(e) => setNetmask(e.target.value)}>
              <option value="255.255.255.0">/24 · 254 hosts</option>
              <option value="255.255.255.128">/25 · 126 hosts</option>
              <option value="255.255.0.0">/16 · 65,534 hosts</option>
            </select>
          </Field>
          <Field label="DHCP start">
            <input
              className="input font-mono text-[13px]"
              value={dhcpStart}
              onChange={(e) => {
                setDhcpTouched(true);
                setDhcpStart(e.target.value);
              }}
            />
          </Field>
          <Field label="DHCP end">
            <input
              className="input font-mono text-[13px]"
              value={dhcpEnd}
              onChange={(e) => {
                setDhcpTouched(true);
                setDhcpEnd(e.target.value);
              }}
            />
          </Field>
        </div>
        <Checkbox checked={autostart} onChange={setAutostart} label="Start automatically when the node boots" />
      </form>
    </Modal>
  );
}

function CreateBridgeModal({ nodeName, onClose, onCreated }: { nodeName: string; onClose: () => void; onCreated: () => void }) {
  const toast = useToast();
  const [name, setName] = useState("");
  const [stp, setStp] = useState(false);
  const [selected, setSelected] = useState<string[]>([]);
  const [interfaces, setInterfaces] = useState<AvailableInterface[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api<{ interfaces: AvailableInterface[] }>("/api/list-available-interfaces", { node_name: nodeName })
      .then((data) => setInterfaces(data.interfaces || []))
      .catch(() => setInterfaces([]))
      .finally(() => setLoading(false));
  }, [nodeName]);

  const toggle = (iface: string, checked: boolean) =>
    setSelected((prev) => (checked ? [...prev, iface] : prev.filter((i) => i !== iface)));

  const handleSubmit = async (e?: React.FormEvent) => {
    e?.preventDefault();
    if (!name.trim()) return;
    setSaving(true);
    try {
      await api("/api/create-bridge", { node_name: nodeName, bridge_name: name.trim(), stp, interfaces: selected });
      toast.success(`Bridge ${name.trim()} created`);
      onCreated();
    } catch (err) {
      toast.error(errorMessage(err, "Failed to create bridge"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      isOpen
      onClose={onClose}
      dismissible={!saving}
      title="Create host bridge"
      description={`A Linux bridge on node ${nodeName} that connects instances to the physical network.`}
      footer={
        <>
          <button className="btn-secondary" onClick={onClose} disabled={saving}>
            Cancel
          </button>
          <button className="btn-primary" onClick={() => handleSubmit()} disabled={saving || !name.trim()}>
            {saving && <Spinner size={14} className="text-current" />}
            Create bridge
          </button>
        </>
      }
    >
      <form className="space-y-4" onSubmit={handleSubmit}>
        <Field label="Bridge name" required>
          <input className="input font-mono text-[13px]" value={name} onChange={(e) => setName(e.target.value)} placeholder="br0" autoFocus />
        </Field>
        <Field label="Attach interfaces" hint="Physical interfaces to enslave to the bridge. Optional.">
          {loading ? (
            <div className="flex items-center gap-2 py-2 text-sm text-fg-subtle">
              <Spinner /> Loading interfaces…
            </div>
          ) : interfaces.length === 0 ? (
            <p className="py-2 text-sm text-fg-muted">No available interfaces.</p>
          ) : (
            <div className="max-h-48 divide-y divide-line overflow-y-auto rounded-md border border-line">
              {interfaces.map((iface) => {
                const bound = Boolean(iface.bound_to);
                return (
                  <label
                    key={iface.name}
                    className={`flex items-center gap-3 px-3 py-2 text-sm ${bound ? "cursor-not-allowed opacity-50" : "cursor-pointer hover:bg-subtle/60"}`}
                  >
                    <input
                      type="checkbox"
                      checked={selected.includes(iface.name)}
                      disabled={bound}
                      onChange={(e) => toggle(iface.name, e.target.checked)}
                    />
                    <span className="font-mono font-medium">{iface.name}</span>
                    <span className="font-mono text-xs text-fg-subtle">{iface.mac}</span>
                    {bound && <span className="text-xs text-warning">in {iface.bound_to}</span>}
                    <span className="ml-auto">
                      <StatusBadge status={iface.state} />
                    </span>
                  </label>
                );
              })}
            </div>
          )}
        </Field>
        {selected.length > 0 && (
          <Alert tone="warning">
            Moving an interface into a bridge briefly interrupts its connectivity. Don't select the interface you use to reach this node
            unless you know the IP will move to the bridge.
          </Alert>
        )}
        <Checkbox checked={stp} onChange={setStp} label="Enable Spanning Tree Protocol (STP)" />
      </form>
    </Modal>
  );
}

export default function NetworksPage() {
  const toast = useToast();
  const [searchParams, setSearchParams] = useSearchParams();
  const tab = (searchParams.get("tab") as TabId) || "networks";
  const { nodes, currentNode, loading: nodesLoading } = useScopedNode();

  const [networks, setNetworks] = useState<LibvirtNetwork[]>([]);
  const [bridges, setBridges] = useState<HostBridge[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [confirm, setConfirm] = useState<ConfirmTarget | null>(null);

  const setTab = (next: TabId) =>
    setSearchParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        if (next === "networks") p.delete("tab");
        else p.set("tab", next);
        return p;
      },
      { replace: true }
    );

  const fetchAll = useCallback(
    async ({ silent = false }: { silent?: boolean } = {}) => {
      if (!currentNode) return;
      if (!silent) setRefreshing(true);
      try {
        const [netData, brData] = await Promise.all([
          api<{ networks: LibvirtNetwork[] }>("/api/list-networks", { node_name: currentNode }),
          api<{ bridges: HostBridge[] }>("/api/list-bridges", { node_name: currentNode }),
        ]);
        setNetworks(netData.networks || []);
        setBridges(brData.bridges || []);
      } catch (err) {
        toast.error(errorMessage(err, "Failed to load networks"));
      } finally {
        setLoading(false);
        setRefreshing(false);
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
    setNetworks([]);
    setBridges([]);
    fetchAll({ silent: true });
  }, [currentNode, nodesLoading, fetchAll]);

  const startNetwork = async (network: LibvirtNetwork) => {
    try {
      await api("/api/start-network", { node_name: currentNode, network_name: network.name });
      toast.success(`Network ${network.name} started`);
      fetchAll({ silent: true });
    } catch (err) {
      toast.error(errorMessage(err, "Failed to start network"));
    }
  };

  const handleConfirm = async () => {
    if (!confirm) return;
    try {
      if (confirm.kind === "stop-network") {
        await api("/api/stop-network", { node_name: currentNode, network_name: confirm.network.name });
        toast.success(`Network ${confirm.network.name} stopped`);
      } else if (confirm.kind === "delete-network") {
        await api("/api/delete-network", { node_name: currentNode, network_name: confirm.network.name });
        toast.success(`Network ${confirm.network.name} deleted`);
      } else {
        await api("/api/delete-bridge", { node_name: currentNode, bridge_name: confirm.bridge.name });
        toast.success(`Bridge ${confirm.bridge.name} deleted`);
      }
      fetchAll({ silent: true });
    } catch (err) {
      toast.error(errorMessage(err, "Operation failed"));
      throw err;
    }
  };

  const noNodes = !nodesLoading && nodes.length === 0;
  const confirmTarget = !confirm ? "" : confirm.kind === "delete-bridge" ? confirm.bridge.name : confirm.network.name;

  return (
    <>
      <PageHeader
        title="Networks"
        description="Virtual networks and host bridges that instances connect to."
        onRefresh={() => fetchAll()}
        refreshing={refreshing}
        actions={
          <>
            <NodeSelect />
            <button className="btn-primary" onClick={() => setCreateOpen(true)} disabled={!currentNode}>
              <Plus size={15} />
              {tab === "networks" ? "Create network" : "Create bridge"}
            </button>
          </>
        }
      />

      {noNodes ? (
        <div className="card">
          <EmptyState
            icon={<Network size={20} />}
            title="No nodes yet"
            description="Add a node before managing networks."
            action={
              <Link to="/nodes?add=1" className="btn-primary">
                Add node
              </Link>
            }
          />
        </div>
      ) : (
        <>
          <Tabs
            className="mb-4"
            value={tab}
            onChange={setTab}
            tabs={[
              { id: "networks", label: "Libvirt networks", count: loading ? undefined : networks.length },
              { id: "bridges", label: "Host bridges", count: loading ? undefined : bridges.length },
            ]}
          />

          {tab === "networks" ? (
            <Table
              rows={networks}
              rowKey={(n) => n.uuid || n.name}
              loading={loading}
              loadingLabel="Loading networks…"
              empty={
                <EmptyState
                  icon={<Network size={20} />}
                  title="No libvirt networks"
                  description="Create a NAT network to give instances private IPs with DHCP."
                  action={
                    <button className="btn-primary" onClick={() => setCreateOpen(true)}>
                      <Plus size={15} />
                      Create network
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
                        <Network size={15} />
                      </div>
                      <div>
                        <div className="font-medium">{n.name}</div>
                        {n.bridge && <div className="font-mono text-xs text-fg-subtle">{n.bridge}</div>}
                      </div>
                    </div>
                  ),
                },
                { key: "state", header: "Status", render: (n) => <StatusBadge status={n.state} /> },
                {
                  key: "mode",
                  header: "Mode",
                  render: (n) => (
                    <div className="flex flex-wrap gap-1">
                      <Badge tone={n.mode === "nat" ? "accent" : "neutral"} className="uppercase">
                        {n.mode}
                      </Badge>
                      {n.autostart && <Badge>autostart</Badge>}
                    </div>
                  ),
                },
                {
                  key: "subnet",
                  header: "Subnet",
                  render: (n) =>
                    n.ip_address ? (
                      <span className="font-mono text-xs">
                        {n.ip_address}
                        {netmaskToPrefix(n.netmask) !== null ? `/${netmaskToPrefix(n.netmask)}` : ""}
                      </span>
                    ) : (
                      <span className="text-fg-subtle">—</span>
                    ),
                },
                {
                  key: "dhcp",
                  header: "DHCP range",
                  render: (n) =>
                    n.dhcp_start ? (
                      <span className="font-mono text-xs text-fg-muted">
                        {n.dhcp_start} – {n.dhcp_end}
                      </span>
                    ) : (
                      <span className="text-fg-subtle">—</span>
                    ),
                },
                {
                  key: "actions",
                  header: <span className="sr-only">Actions</span>,
                  align: "right",
                  render: (n) => {
                    const active = n.state.toLowerCase() === "active";
                    return (
                      <DropdownMenu
                        items={[
                          active
                            ? { label: "Stop network", icon: <Square size={14} />, onClick: () => setConfirm({ kind: "stop-network", network: n }) }
                            : { label: "Start network", icon: <Play size={14} />, onClick: () => startNetwork(n) },
                          { divider: true, label: "divider" },
                          { label: "Delete network", icon: <Trash2 size={14} />, danger: true, onClick: () => setConfirm({ kind: "delete-network", network: n }) },
                        ]}
                      />
                    );
                  },
                },
              ]}
            />
          ) : (
            <Table
              rows={bridges}
              rowKey={(b) => b.name}
              loading={loading}
              loadingLabel="Loading bridges…"
              empty={
                <EmptyState
                  icon={<Cable size={20} />}
                  title="No host bridges"
                  description="Create a bridge to put instances directly on the physical network."
                  action={
                    <button className="btn-primary" onClick={() => setCreateOpen(true)}>
                      <Plus size={15} />
                      Create bridge
                    </button>
                  }
                />
              }
              columns={[
                {
                  key: "name",
                  header: "Name",
                  render: (b) => (
                    <div className="flex items-center gap-2.5">
                      <div className="flex h-8 w-8 items-center justify-center rounded-md bg-subtle text-fg-subtle">
                        <Cable size={15} />
                      </div>
                      <div>
                        <div className="font-mono font-medium">{b.name}</div>
                        {b.mac && <div className="font-mono text-xs text-fg-subtle">{b.mac}</div>}
                      </div>
                    </div>
                  ),
                },
                { key: "state", header: "Status", render: (b) => <StatusBadge status={b.state} /> },
                {
                  key: "ips",
                  header: "Addresses",
                  render: (b) =>
                    b.ips?.length ? (
                      <div className="space-y-0.5 font-mono text-xs">
                        {b.ips.map((ip) => (
                          <div key={ip}>{ip}</div>
                        ))}
                      </div>
                    ) : (
                      <span className="text-fg-subtle">—</span>
                    ),
                },
                {
                  key: "interfaces",
                  header: "Members",
                  render: (b) => {
                    const members = b.interfaces || [];
                    if (!members.length) return <span className="text-fg-subtle">—</span>;
                    const physical = members.filter((i) => !i.startsWith("vnet") && !i.startsWith("veth") && !i.startsWith("tap"));
                    const virtualCount = members.length - physical.length;
                    return (
                      <div className="flex flex-wrap items-center gap-1">
                        {physical.map((i) => (
                          <Badge key={i} tone="info" className="font-mono">
                            {i}
                          </Badge>
                        ))}
                        {virtualCount > 0 && (
                          <span className="text-xs text-fg-subtle" title={members.filter((i) => !physical.includes(i)).join(", ")}>
                            +{virtualCount} instance {virtualCount === 1 ? "port" : "ports"}
                          </span>
                        )}
                      </div>
                    );
                  },
                },
                {
                  key: "opts",
                  header: "Options",
                  render: (b) => (
                    <div className="flex gap-1 text-xs text-fg-muted">
                      {b.mtu ? <span>MTU {b.mtu}</span> : null}
                      {b.stp && <Badge>STP</Badge>}
                    </div>
                  ),
                },
                {
                  key: "actions",
                  header: <span className="sr-only">Actions</span>,
                  align: "right",
                  render: (b) => (
                    <DropdownMenu
                      items={[{ label: "Delete bridge", icon: <Trash2 size={14} />, danger: true, onClick: () => setConfirm({ kind: "delete-bridge", bridge: b }) }]}
                    />
                  ),
                },
              ]}
            />
          )}
        </>
      )}

      {createOpen && tab === "networks" && (
        <CreateNetworkModal
          nodeName={currentNode}
          onClose={() => setCreateOpen(false)}
          onCreated={() => {
            setCreateOpen(false);
            fetchAll({ silent: true });
          }}
        />
      )}
      {createOpen && tab === "bridges" && (
        <CreateBridgeModal
          nodeName={currentNode}
          onClose={() => setCreateOpen(false)}
          onCreated={() => {
            setCreateOpen(false);
            fetchAll({ silent: true });
          }}
        />
      )}

      <ConfirmDialog
        isOpen={Boolean(confirm)}
        onClose={() => setConfirm(null)}
        onConfirm={handleConfirm}
        title={
          confirm?.kind === "stop-network" ? "Stop network?" : confirm?.kind === "delete-network" ? "Delete network?" : "Delete bridge?"
        }
        message={
          confirm?.kind === "stop-network" ? (
            <>
              Instances attached to <strong className="text-fg">{confirmTarget}</strong> will lose network connectivity until it is started
              again.
            </>
          ) : (
            <>
              <strong className="text-fg">{confirmTarget}</strong> will be permanently deleted. Instances attached to it will lose network
              connectivity.
            </>
          )
        }
        confirmText={confirm?.kind === "stop-network" ? "Stop" : "Delete"}
        variant={confirm?.kind === "stop-network" ? "warning" : "danger"}
      />
    </>
  );
}
