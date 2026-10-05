import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { Download, Layers, Plus, Trash2 } from "lucide-react";
import PageHeader from "@/components/PageHeader";
import Table from "@/components/Table";
import ConfirmDialog from "@/components/ConfirmDialog";
import DropdownMenu from "@/components/DropdownMenu";
import NodeSelect from "@/components/NodeSelect";
import SearchFilter from "@/components/SearchFilter";
import { Alert, Badge, Checkbox, EmptyState, Spinner } from "@/components/ui";
import { useToast } from "@/components/ToastContainer";
import { api, errorMessage } from "@/lib/api";
import { useScopedNode } from "@/lib/nodes";
import { isRecommendedWindowsCloudImage } from "@/lib/templates";
import type { DownloadTask, StoragePool, Template } from "@/lib/types";
import RegisterTemplateModal from "./RegisterTemplateModal";

const ALL_POOLS = "";

export default function TemplatesPage() {
  const toast = useToast();
  const [searchParams, setSearchParams] = useSearchParams();
  const { nodes, currentNode, loading: nodesLoading } = useScopedNode();
  const poolFilter = searchParams.get("pool") || ALL_POOLS;

  const [pools, setPools] = useState<StoragePool[]>([]);
  const [templates, setTemplates] = useState<Template[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [query, setQuery] = useState("");
  const [registerOpen, setRegisterOpen] = useState(searchParams.get("register") === "1");
  const [toDelete, setToDelete] = useState<Template | null>(null);
  const [deleteVolume, setDeleteVolume] = useState(false);
  const [tasks, setTasks] = useState<DownloadTask[]>([]);
  const notifiedRef = useRef(new Set<string>());

  const setPoolFilter = (pool: string) =>
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        if (pool) next.set("pool", pool);
        else next.delete("pool");
        return next;
      },
      { replace: true }
    );

  // 只采用最新一次请求的结果，避免切换节点或存储池后旧响应覆盖列表
  const requestRef = useRef(0);
  const fetchTemplates = useCallback(
    async ({ silent = false }: { silent?: boolean } = {}) => {
      if (!currentNode) return;
      const requestId = ++requestRef.current;
      let redirected = false;
      const isLatest = () => requestId === requestRef.current && !redirected;
      if (!silent) setRefreshing(true);
      try {
        const poolData = await api<{ pools: StoragePool[] }>("/api/list-storage-pools", { node_name: currentNode });
        if (!isLatest()) return;
        const poolList = poolData.pools || [];
        setPools(poolList);
        // URL 中的存储池不属于当前节点（例如刚切换节点）时，回退到全部存储池
        if (poolFilter && !poolList.some((p) => p.name === poolFilter)) {
          setSearchParams(
            (prev) => {
              const next = new URLSearchParams(prev);
              next.delete("pool");
              return next;
            },
            { replace: true }
          );
          // 由 poolFilter 变化触发的下一次请求负责加载列表和收尾状态
          redirected = true;
          return;
        }
        // 后端按存储池查询模板，"全部" 时并行查询每个存储池后合并
        const targets = poolFilter ? poolList.filter((p) => p.name === poolFilter) : poolList;
        const results = await Promise.all(
          targets.map((p) =>
            api<{ templates: Template[] }>("/api/list-templates", { node_name: currentNode, pool_name: p.name })
              .then((d) => d.templates || [])
              .catch(() => [] as Template[])
          )
        );
        if (isLatest()) setTemplates(results.flat());
      } catch (err) {
        if (isLatest()) toast.error(errorMessage(err, "Failed to load templates"));
      } finally {
        if (isLatest()) {
          setLoading(false);
          setRefreshing(false);
        }
      }
    },
    [currentNode, poolFilter, setSearchParams, toast]
  );

  useEffect(() => {
    if (nodesLoading) return;
    if (!currentNode) {
      setLoading(false);
      return;
    }
    setLoading(true);
    setTemplates([]);
    fetchTemplates({ silent: true });
  }, [currentNode, nodesLoading, fetchTemplates]);

  // 恢复进行中的下载任务
  useEffect(() => {
    api<{ tasks: DownloadTask[] | null }>("/api/list-download-tasks")
      .then((data) => setTasks((data.tasks || []).filter((t) => t.status === "pending" || t.status === "running")))
      .catch(() => undefined);
  }, []);

  // 统一轮询所有进行中的下载任务
  const activeTaskIds = tasks.map((t) => t.id).join(",");
  useEffect(() => {
    if (!activeTaskIds) return;
    const timer = setInterval(async () => {
      const ids = activeTaskIds.split(",");
      const updates = await Promise.all(
        ids.map((id) =>
          api<{ task: DownloadTask }>("/api/get-download-task", { task_id: id })
            .then((d) => d.task)
            .catch(() => null)
        )
      );
      let finished = false;
      for (const task of updates) {
        if (!task || notifiedRef.current.has(task.id)) continue;
        if (task.status === "completed") {
          notifiedRef.current.add(task.id);
          toast.success(`${task.volume_name} downloaded and registered`);
          finished = true;
        } else if (task.status === "failed") {
          notifiedRef.current.add(task.id);
          toast.error(`Download of ${task.volume_name} failed: ${task.error || "unknown error"}`);
          finished = true;
        }
      }
      setTasks((prev) =>
        prev
          .map((t) => updates.find((u) => u?.id === t.id) || t)
          .filter((t) => t.status === "pending" || t.status === "running")
      );
      if (finished) fetchTemplates({ silent: true });
    }, 4000);
    return () => clearInterval(timer);
  }, [activeTaskIds, fetchTemplates, toast]);

  const closeRegister = () => {
    setRegisterOpen(false);
    if (searchParams.get("register")) {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          next.delete("register");
          return next;
        },
        { replace: true }
      );
    }
  };

  const handleDelete = async () => {
    if (!toDelete) return;
    try {
      await api("/api/delete-template", {
        template_id: toDelete.id,
        node_name: toDelete.node_name,
        pool_name: toDelete.pool_name,
        delete_volume: deleteVolume,
      });
      toast.success(`Template ${toDelete.name} deleted`);
      fetchTemplates({ silent: true });
    } catch (err) {
      toast.error(errorMessage(err, "Failed to delete template"));
      throw err;
    }
  };

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return templates
      .filter(
        (t) =>
          !q ||
          t.name.toLowerCase().includes(q) ||
          t.description?.toLowerCase().includes(q) ||
          t.os?.name?.toLowerCase().includes(q) ||
          t.tags?.some((tag) => tag.toLowerCase().includes(q))
      )
      .sort((a, b) => Number(isRecommendedWindowsCloudImage(b)) - Number(isRecommendedWindowsCloudImage(a)) || a.name.localeCompare(b.name));
  }, [templates, query]);

  const noNodes = !nodesLoading && nodes.length === 0;

  return (
    <>
      <PageHeader
        title="Templates"
        description="Disk images and ISOs that new instances are created from."
        onRefresh={() => fetchTemplates()}
        refreshing={refreshing}
        actions={
          <>
            <NodeSelect />
            <button className="btn-primary" onClick={() => setRegisterOpen(true)} disabled={!currentNode}>
              <Plus size={15} />
              Register template
            </button>
          </>
        }
      />

      {tasks.length > 0 && (
        <div className="card mb-4 divide-y divide-line">
          {tasks.map((task) => (
            <div key={task.id} className="flex items-center gap-3 px-4 py-3 text-sm">
              <Spinner className="text-accent" />
              <Download size={14} className="text-fg-subtle" />
              <span className="font-mono text-[13px]">{task.volume_name}</span>
              <span className="text-fg-subtle">
                → {task.node_name} / {task.pool_name}
              </span>
              <Badge tone="info" className="ml-auto capitalize">
                {task.status === "pending" ? "Queued" : "Downloading"}
              </Badge>
            </div>
          ))}
        </div>
      )}

      {noNodes ? (
        <div className="card">
          <EmptyState
            icon={<Layers size={20} />}
            title="No nodes yet"
            description="Add a node before registering templates."
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
            <select className="input sm:w-56" value={poolFilter} onChange={(e) => setPoolFilter(e.target.value)} aria-label="Storage pool">
              <option value={ALL_POOLS}>All storage pools</option>
              {pools.map((p) => (
                <option key={p.name} value={p.name}>
                  {p.name}
                </option>
              ))}
            </select>
            <SearchFilter value={query} onChange={setQuery} placeholder="Search name, OS or tag" className="sm:w-80" />
          </div>

          {templates.some(isRecommendedWindowsCloudImage) && (
            <Alert tone="info" className="mb-4">
              For automated Windows instances, use the template marked <strong className="text-fg">Recommended</strong>. Windows installer ISOs
              are for manual installation.
            </Alert>
          )}

          <Table
            rows={filtered}
            rowKey={(t) => `${t.pool_name}/${t.id}`}
            loading={loading}
            loadingLabel="Loading templates…"
            empty={
              templates.length ? (
                <EmptyState title="No matching templates" />
              ) : (
                <EmptyState
                  icon={<Layers size={20} />}
                  title="No templates yet"
                  description="Download an Ubuntu, Debian or Rocky cloud image, or register a volume you already have."
                  action={
                    <button className="btn-primary" onClick={() => setRegisterOpen(true)}>
                      <Plus size={15} />
                      Register template
                    </button>
                  }
                />
              )
            }
            columns={[
              {
                key: "name",
                header: "Template",
                render: (t) => (
                  <div className="flex items-start gap-2.5">
                    <div className="mt-0.5 flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-md bg-subtle text-fg-subtle">
                      <Layers size={15} />
                    </div>
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-1.5">
                        <span className="font-medium">{t.name}</span>
                        {isRecommendedWindowsCloudImage(t) && <Badge tone="success">Recommended</Badge>}
                      </div>
                      {t.description && <div className="max-w-xs truncate text-xs text-fg-muted">{t.description}</div>}
                      {!!t.tags?.length && (
                        <div className="mt-0.5 max-w-xs truncate text-xs text-fg-subtle" title={t.tags.join(", ")}>
                          {t.tags.map((tag) => `#${tag}`).join(" ")}
                        </div>
                      )}
                    </div>
                  </div>
                ),
              },
              {
                key: "os",
                header: "OS",
                render: (t) =>
                  t.os?.name ? (
                    <span className="whitespace-nowrap">
                      {t.os.name} {t.os.version}
                      {t.os.arch && <span className="ml-1 text-xs text-fg-subtle">{t.os.arch}</span>}
                    </span>
                  ) : (
                    <span className="text-fg-subtle">—</span>
                  ),
              },
              {
                key: "location",
                header: "Location",
                render: (t) => (
                  <div className="min-w-0">
                    <div className="text-fg-muted">{t.pool_name}</div>
                    <div className="max-w-[180px] truncate font-mono text-xs text-fg-subtle" title={t.volume_name}>
                      {t.volume_name}
                    </div>
                  </div>
                ),
              },
              {
                key: "size",
                header: "Size",
                render: (t) => (
                  <span className="whitespace-nowrap">
                    {t.size_gb} GB <span className="text-xs uppercase text-fg-subtle">{t.format}</span>
                  </span>
                ),
              },
              {
                key: "features",
                header: "Features",
                render: (t) => (
                  <div className="flex gap-1">
                    {t.features?.cloud_init && <Badge tone="accent">cloud-init</Badge>}
                    {t.features?.virtio && <Badge>virtio</Badge>}
                    {t.features?.qemu_guest_agent && <Badge>qga</Badge>}
                    {!t.features?.cloud_init && !t.features?.virtio && !t.features?.qemu_guest_agent && <span className="text-fg-subtle">—</span>}
                  </div>
                ),
              },
              {
                key: "actions",
                header: <span className="sr-only">Actions</span>,
                align: "right",
                render: (t) => (
                  <DropdownMenu
                    items={[
                      {
                        label: "Delete template",
                        icon: <Trash2 size={14} />,
                        danger: true,
                        onClick: () => {
                          setDeleteVolume(false);
                          setToDelete(t);
                        },
                      },
                    ]}
                  />
                ),
              },
            ]}
          />
        </>
      )}

      {registerOpen && currentNode && (
        <RegisterTemplateModal
          nodes={nodes}
          defaultNode={currentNode}
          defaultPool={poolFilter || undefined}
          onClose={closeRegister}
          onRegistered={({ task }) => {
            closeRegister();
            if (task) setTasks((prev) => [...prev.filter((t) => t.id !== task.id), task]);
            fetchTemplates({ silent: true });
          }}
        />
      )}

      <ConfirmDialog
        isOpen={Boolean(toDelete)}
        onClose={() => setToDelete(null)}
        onConfirm={handleDelete}
        title="Delete template?"
        message={
          <>
            <strong className="text-fg">{toDelete?.name}</strong> will no longer be available for new instances.
          </>
        }
        confirmText="Delete template"
      >
        <Checkbox
          checked={deleteVolume}
          onChange={setDeleteVolume}
          label={`Also delete the volume ${toDelete?.volume_name || ""}`}
          description="Instances created from this template use it as their backing disk and will break if it is deleted."
        />
      </ConfirmDialog>
    </>
  );
}
