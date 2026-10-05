import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { ArrowUpRight, Cpu, Database, Layers, MemoryStick, Plus, Server, ServerCog } from "lucide-react";
import PageHeader from "@/components/PageHeader";
import Table from "@/components/Table";
import { Alert, Card, EmptyState, StatCard, StatusBadge, UsageBar } from "@/components/ui";
import { api, errorMessage } from "@/lib/api";
import { formatBytes, formatMemoryMB, percent } from "@/lib/format";
import { useNodes } from "@/lib/nodes";
import type { Instance, Node, StoragePool } from "@/lib/types";

interface NodeOverview {
  node: Node;
  instances: Instance[];
  pools: StoragePool[];
  error?: string;
}

export default function OverviewPage() {
  const navigate = useNavigate();
  const { nodes, loading: nodesLoading, error: nodesError, refresh: refreshNodes, setCurrentNode } = useNodes();
  const [overview, setOverview] = useState<NodeOverview[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  useEffect(() => {
    if (nodesLoading) return;
    let cancelled = false;
    setLoading(true);
    Promise.all(
      nodes.map(async (node): Promise<NodeOverview> => {
        if (node.state !== "online") return { node, instances: [], pools: [] };
        try {
          const [instancesRes, poolsRes] = await Promise.all([
            api<{ instances: Instance[] }>("/api/describe-instances", { node_name: node.name }),
            api<{ pools: StoragePool[] }>("/api/list-storage-pools", { node_name: node.name }),
          ]);
          return { node, instances: instancesRes.instances || [], pools: poolsRes.pools || [] };
        } catch (err) {
          return { node, instances: [], pools: [], error: errorMessage(err) };
        }
      })
    ).then((result) => {
      if (cancelled) return;
      setOverview(result);
      setLoading(false);
    });
    return () => {
      cancelled = true;
    };
  }, [nodes, nodesLoading]);

  const handleRefresh = async () => {
    setRefreshing(true);
    try {
      await refreshNodes();
    } finally {
      setRefreshing(false);
    }
  };

  const totals = useMemo(() => {
    const instances = overview.flatMap((o) => o.instances);
    const running = instances.filter((i) => i.state === "running");
    const pools = overview.flatMap((o) => o.pools);
    return {
      nodesOnline: nodes.filter((n) => n.state === "online").length,
      instances: instances.length,
      running: running.length,
      vcpus: running.reduce((sum, i) => sum + (i.vcpus || 0), 0),
      memoryMB: running.reduce((sum, i) => sum + (i.memory_mb || 0), 0),
      storageUsed: pools.reduce((sum, p) => sum + (p.allocation || 0), 0),
      storageTotal: pools.reduce((sum, p) => sum + (p.capacity || 0), 0),
    };
  }, [overview, nodes]);

  const openNodeScoped = (nodeName: string, path: string) => {
    setCurrentNode(nodeName);
    navigate(`${path}?node=${encodeURIComponent(nodeName)}`);
  };

  const isEmpty = !nodesLoading && nodes.length === 0;

  return (
    <>
      <PageHeader
        title="Overview"
        description="Health and capacity across all virtualization nodes."
        onRefresh={handleRefresh}
        refreshing={refreshing}
        actions={
          <Link to="/instances?create=1" className="btn-primary">
            <Plus size={15} />
            Create instance
          </Link>
        }
      />

      {nodesError && (
        <Alert tone="danger" title="Unable to load nodes" className="mb-6">
          {nodesError}
        </Alert>
      )}

      {isEmpty ? (
        <div className="card">
          <EmptyState
            icon={<ServerCog size={20} />}
            title="Add your first node"
            description="JVP manages virtual machines on libvirt hosts. Connect a node to start creating instances."
            action={
              <Link to="/nodes?add=1" className="btn-primary">
                <Plus size={15} />
                Add node
              </Link>
            }
          />
        </div>
      ) : (
        <div className="space-y-6">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <StatCard
              label="Nodes"
              icon={<ServerCog size={16} />}
              value={
                <>
                  {totals.nodesOnline}
                  <span className="text-base font-normal text-fg-subtle"> / {nodes.length}</span>
                </>
              }
              hint="online"
            />
            <StatCard
              label="Instances"
              icon={<Server size={16} />}
              value={
                <>
                  {loading ? "—" : totals.running}
                  <span className="text-base font-normal text-fg-subtle"> / {loading ? "—" : totals.instances}</span>
                </>
              }
              hint="running"
            />
            <StatCard
              label="Allocated to running instances"
              icon={<Cpu size={16} />}
              value={loading ? "—" : `${totals.vcpus} vCPU`}
              hint={
                <span className="inline-flex items-center gap-1">
                  <MemoryStick size={12} />
                  {loading ? "—" : formatMemoryMB(totals.memoryMB)} memory
                </span>
              }
            />
            <StatCard
              label="Storage"
              icon={<Database size={16} />}
              value={loading ? "—" : formatBytes(totals.storageUsed)}
              hint={loading ? undefined : `of ${formatBytes(totals.storageTotal)} across all pools`}
            >
              <UsageBar value={percent(totals.storageUsed, totals.storageTotal)} size="sm" />
            </StatCard>
          </div>

          <Card
            title="Nodes"
            description="Click a node to manage its instances."
            bodyClassName=""
            actions={
              <Link to="/nodes" className="btn-ghost btn-sm">
                Manage nodes
                <ArrowUpRight size={13} />
              </Link>
            }
          >
            <Table
              bare
              loading={loading || nodesLoading}
              rows={overview}
              rowKey={(o) => o.node.name}
              onRowClick={(o) => openNodeScoped(o.node.name, "/instances")}
              columns={[
                {
                  key: "name",
                  header: "Node",
                  render: (o) => (
                    <div>
                      <div className="font-medium">{o.node.name}</div>
                      <div className="max-w-[260px] truncate font-mono text-xs text-fg-subtle">{o.node.uri}</div>
                    </div>
                  ),
                },
                { key: "state", header: "Status", render: (o) => <StatusBadge status={o.node.state} /> },
                {
                  key: "instances",
                  header: "Instances",
                  render: (o) =>
                    o.node.state !== "online" ? (
                      <span className="text-fg-subtle">—</span>
                    ) : o.error ? (
                      <span className="text-xs text-danger" title={o.error}>
                        Unavailable
                      </span>
                    ) : (
                      <span className="tabular-nums">
                        <span className="font-medium">{o.instances.filter((i) => i.state === "running").length}</span>
                        <span className="text-fg-subtle"> running · {o.instances.length} total</span>
                      </span>
                    ),
                },
                {
                  key: "storage",
                  header: "Storage",
                  className: "w-[280px]",
                  render: (o) => {
                    const used = o.pools.reduce((s, p) => s + p.allocation, 0);
                    const total = o.pools.reduce((s, p) => s + p.capacity, 0);
                    if (!total) return <span className="text-fg-subtle">—</span>;
                    return (
                      <div className="space-y-1">
                        <div className="flex justify-between text-xs text-fg-muted">
                          <span>
                            {formatBytes(used)} / {formatBytes(total)}
                          </span>
                          <span>{o.pools.length} {o.pools.length === 1 ? "pool" : "pools"}</span>
                        </div>
                        <UsageBar value={percent(used, total)} size="sm" />
                      </div>
                    );
                  },
                },
              ]}
            />
          </Card>

          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            {[
              { to: "/instances?create=1", icon: Server, title: "Launch an instance", text: "Create a VM from a template or ISO." },
              { to: "/templates?register=1", icon: Layers, title: "Add a template", text: "Download a cloud image or register a volume." },
              { to: "/nodes?add=1", icon: ServerCog, title: "Connect a node", text: "Manage another libvirt host." },
            ].map(({ to, icon: Icon, title, text }) => (
              <Link
                key={to}
                to={to}
                className="card group flex items-start gap-3 p-4 transition-colors hover:border-line-strong hover:bg-subtle/40"
              >
                <div className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-md bg-accent-soft text-accent">
                  <Icon size={17} />
                </div>
                <div className="min-w-0">
                  <div className="flex items-center gap-1 text-sm font-medium text-fg">
                    {title}
                    <ArrowUpRight size={13} className="text-fg-subtle opacity-0 transition-opacity group-hover:opacity-100" />
                  </div>
                  <p className="mt-0.5 text-xs text-fg-muted">{text}</p>
                </div>
              </Link>
            ))}
          </div>
        </div>
      )}
    </>
  );
}
