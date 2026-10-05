import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { AlertTriangle, Camera, Monitor, Play, Plus, RotateCcw, Server, Square, Trash2 } from "lucide-react";
import PageHeader from "@/components/PageHeader";
import Table from "@/components/Table";
import type { Column } from "@/components/Table";
import DropdownMenu from "@/components/DropdownMenu";
import NodeSelect from "@/components/NodeSelect";
import SearchFilter from "@/components/SearchFilter";
import { Badge, EmptyState, SegmentedControl, Spinner, StatusBadge } from "@/components/ui";
import { useToast } from "@/components/ToastContainer";
import { api, errorMessage } from "@/lib/api";
import { formatMemoryMB, shortId } from "@/lib/format";
import { useScopedNode } from "@/lib/nodes";
import type { Instance } from "@/lib/types";
import CreateInstanceModal from "./CreateInstanceModal";
import { actionProgressLabel, useInstanceActions } from "./useInstanceActions";

type StateFilter = "all" | "running" | "stopped";

const POLL_INTERVAL = 15000;

function instanceIPs(instance: Instance): string[] {
  const fromIfaces = instance.interfaces?.flatMap((i) => i.ips || []) || [];
  return Array.from(new Set([instance.ip_address, ...fromIfaces].filter(Boolean) as string[]));
}

export default function InstancesPage() {
  const toast = useToast();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const { nodes, currentNode, setCurrentNode, loading: nodesLoading } = useScopedNode();

  const [instances, setInstances] = useState<Instance[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [query, setQuery] = useState("");
  const [stateFilter, setStateFilter] = useState<StateFilter>("all");
  const [createOpen, setCreateOpen] = useState(searchParams.get("create") === "1");
  const [loadError, setLoadError] = useState<string | null>(null);
  const requestRef = useRef(0);

  const fetchInstances = useCallback(
    // silent: 不显示刷新动画；background: 轮询请求，失败时不打扰用户
    async ({ silent = false, background = false }: { silent?: boolean; background?: boolean } = {}) => {
      if (!currentNode) return;
      const requestId = ++requestRef.current;
      if (!silent) setRefreshing(true);
      try {
        const data = await api<{ instances: Instance[] }>("/api/describe-instances", { node_name: currentNode });
        if (requestId === requestRef.current) {
          setInstances(data.instances || []);
          setLoadError(null);
        }
      } catch (err) {
        if (!background && requestId === requestRef.current) {
          const message = errorMessage(err, "Failed to load instances");
          setLoadError(message);
          toast.error(message);
        }
      } finally {
        if (requestId === requestRef.current) {
          setLoading(false);
          setRefreshing(false);
        }
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
    setInstances([]);
    setLoadError(null);
    fetchInstances({ silent: true });
  }, [currentNode, nodesLoading, fetchInstances]);

  // 页面可见时定期刷新状态
  useEffect(() => {
    const timer = setInterval(() => {
      if (document.visibilityState === "visible") fetchInstances({ silent: true, background: true });
    }, POLL_INTERVAL);
    return () => clearInterval(timer);
  }, [fetchInstances]);

  const { request, busy, dialog } = useInstanceActions({
    onChanged: () => {
      fetchInstances({ silent: true, background: true });
      setTimeout(() => fetchInstances({ silent: true, background: true }), 2500);
    },
  });

  const counts = useMemo(
    () => ({
      all: instances.length,
      running: instances.filter((i) => i.state === "running").length,
      stopped: instances.filter((i) => i.state !== "running").length,
    }),
    [instances]
  );

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return instances
      .filter((i) => (stateFilter === "all" ? true : stateFilter === "running" ? i.state === "running" : i.state !== "running"))
      .filter(
        (i) =>
          !q ||
          i.id.toLowerCase().includes(q) ||
          i.name?.toLowerCase().includes(q) ||
          instanceIPs(i).some((ip) => ip.includes(q))
      )
      .sort((a, b) => (a.name || a.id).localeCompare(b.name || b.id));
  }, [instances, query, stateFilter]);

  const closeCreate = () => {
    setCreateOpen(false);
    if (searchParams.get("create")) {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          next.delete("create");
          return next;
        },
        { replace: true }
      );
    }
  };

  const detailPath = (i: Instance) => `/instances/${encodeURIComponent(i.node_name)}/${encodeURIComponent(i.id)}`;

  const columns: Column<Instance>[] = [
    {
      key: "name",
      header: "Name",
      render: (i) => (
        <div className="min-w-0">
          <Link to={detailPath(i)} onClick={(e) => e.stopPropagation()} className="font-medium text-fg hover:text-accent">
            {i.name || i.domain_name || i.id}
          </Link>
          <div className="whitespace-nowrap font-mono text-xs text-fg-subtle" title={i.id}>
            {shortId(i.id, 20)}
          </div>
        </div>
      ),
    },
    {
      key: "state",
      header: "Status",
      render: (i) =>
        busy[i.id] ? (
          <Badge tone="warning" dot pulse>
            {actionProgressLabel[busy[i.id]]}…
          </Badge>
        ) : (
          <StatusBadge status={i.state} />
        ),
    },
    {
      key: "ip",
      header: "IP address",
      render: (i) => {
        const ips = instanceIPs(i);
        if (ips.length === 0) return <span className="text-fg-subtle">—</span>;
        return (
          <div className="font-mono text-xs">
            {ips[0]}
            {ips.length > 1 && (
              <span className="ml-1.5 text-fg-subtle" title={ips.slice(1).join("\n")}>
                +{ips.length - 1}
              </span>
            )}
          </div>
        );
      },
    },
    {
      key: "spec",
      header: "Spec",
      render: (i) => (
        <span className="whitespace-nowrap text-fg-muted">
          {i.vcpus} vCPU · {formatMemoryMB(i.memory_mb)}
        </span>
      ),
    },
    {
      key: "autostart",
      header: "Autostart",
      render: (i) => (i.autostart ? <Badge tone="accent">On</Badge> : <span className="text-fg-subtle">Off</span>),
    },
    {
      key: "actions",
      header: <span className="sr-only">Actions</span>,
      align: "right",
      render: (i) => {
        const running = i.state === "running";
        const isBusy = Boolean(busy[i.id]);
        return (
          <div className="flex items-center justify-end gap-0.5" onClick={(e) => e.stopPropagation()}>
            {isBusy ? (
              <span className="flex h-8 w-8 items-center justify-center">
                <Spinner />
              </span>
            ) : running ? (
              <button className="btn-icon" title="Stop" onClick={() => request(i, "stop")}>
                <Square size={15} />
              </button>
            ) : (
              <button className="btn-icon hover:text-success" title="Start" onClick={() => request(i, "start")}>
                <Play size={15} />
              </button>
            )}
            <button className="btn-icon" title="Console" onClick={() => navigate(`${detailPath(i)}/console`)}>
              <Monitor size={15} />
            </button>
            <DropdownMenu
              items={[
                { label: "Reboot", icon: <RotateCcw size={14} />, onClick: () => request(i, "reboot"), disabled: !running || isBusy },
                {
                  label: "Snapshots",
                  icon: <Camera size={14} />,
                  onClick: () => navigate(`/snapshots?node=${encodeURIComponent(i.node_name)}&vm=${encodeURIComponent(i.id)}`),
                },
                { divider: true, label: "divider" },
                { label: "Terminate", icon: <Trash2 size={14} />, danger: true, onClick: () => request(i, "terminate"), disabled: isBusy },
              ]}
            />
          </div>
        );
      },
    },
  ];

  const noNodes = !nodesLoading && nodes.length === 0;

  return (
    <>
      <PageHeader
        title="Instances"
        description="Virtual machines running on the selected node."
        onRefresh={() => fetchInstances()}
        refreshing={refreshing}
        actions={
          <>
            <NodeSelect />
            <button className="btn-primary" onClick={() => setCreateOpen(true)} disabled={noNodes}>
              <Plus size={15} />
              Create instance
            </button>
          </>
        }
      />

      {noNodes ? (
        <div className="card">
          <EmptyState
            icon={<Server size={20} />}
            title="No nodes yet"
            description="Add a libvirt node before creating instances."
            action={
              <Link to="/nodes?add=1" className="btn-primary">
                Add node
              </Link>
            }
          />
        </div>
      ) : (
        <>
          <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <SegmentedControl
              value={stateFilter}
              onChange={setStateFilter}
              options={[
                { value: "all", label: <>All <span className="text-fg-subtle">{counts.all}</span></> },
                { value: "running", label: <>Running <span className="text-fg-subtle">{counts.running}</span></> },
                { value: "stopped", label: <>Stopped <span className="text-fg-subtle">{counts.stopped}</span></> },
              ]}
            />
            <SearchFilter value={query} onChange={setQuery} placeholder="Search by name, ID or IP" className="sm:w-80" />
          </div>

          <Table
            columns={columns}
            rows={filtered}
            rowKey={(i) => i.id}
            onRowClick={(i) => navigate(detailPath(i))}
            loading={loading}
            loadingLabel="Loading instances…"
            empty={
              loadError && instances.length === 0 ? (
                <EmptyState
                  icon={<AlertTriangle size={20} />}
                  title="Unable to load instances"
                  description={loadError}
                  action={
                    <button className="btn-secondary" onClick={() => fetchInstances()}>
                      Try again
                    </button>
                  }
                />
              ) : instances.length > 0 ? (
                <EmptyState
                  title="No matching instances"
                  description="Try a different search or filter."
                  action={
                    <button
                      className="btn-secondary"
                      onClick={() => {
                        setQuery("");
                        setStateFilter("all");
                      }}
                    >
                      Clear filters
                    </button>
                  }
                />
              ) : (
                <EmptyState
                  icon={<Server size={20} />}
                  title={`No instances on ${currentNode || "this node"}`}
                  description="Create an instance from a template, cloud image or installer ISO."
                  action={
                    <button className="btn-primary" onClick={() => setCreateOpen(true)}>
                      <Plus size={15} />
                      Create instance
                    </button>
                  }
                />
              )
            }
          />
        </>
      )}

      {createOpen && !nodesLoading && (
        <CreateInstanceModal
          nodes={nodes}
          defaultNode={currentNode}
          onClose={closeCreate}
          onCreated={(instance, nodeName) => {
            closeCreate();
            if (nodeName !== currentNode) {
              setCurrentNode(nodeName);
            } else {
              fetchInstances({ silent: true });
            }
            if (instance?.id) {
              navigate(`/instances/${encodeURIComponent(nodeName)}/${encodeURIComponent(instance.id)}`);
            }
          }}
        />
      )}

      {dialog}
    </>
  );
}
