import { useEffect, useRef, useState } from "react";
import { Copy } from "lucide-react";
import Modal from "@/components/Modal";
import { Alert, Checkbox, Field, Spinner, Switch } from "@/components/ui";
import { useToast } from "@/components/ToastContainer";
import { api, errorMessage } from "@/lib/api";
import type { Instance, StoragePool } from "@/lib/types";

export function CreateSnapshotModal({
  nodeName,
  instance,
  onClose,
  onCreated,
}: {
  nodeName: string;
  instance: Pick<Instance, "id" | "name" | "state">;
  onClose: () => void;
  onCreated: () => void;
}) {
  const toast = useToast();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [withMemory, setWithMemory] = useState(false);
  const [saving, setSaving] = useState(false);

  const handleSubmit = async (e?: React.FormEvent) => {
    e?.preventDefault();
    setSaving(true);
    try {
      await api("/api/create-snapshot", {
        node_name: nodeName,
        vm_name: instance.id,
        snapshot_name: name.trim() || undefined,
        description: description.trim() || undefined,
        with_memory: withMemory,
      });
      toast.success("Snapshot created");
      onCreated();
    } catch (err) {
      toast.error(errorMessage(err, "Failed to create snapshot"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      isOpen
      onClose={onClose}
      dismissible={!saving}
      title="Create snapshot"
      description={`Capture the current disk state of ${instance.name || instance.id}.`}
      footer={
        <>
          <button className="btn-secondary" onClick={onClose} disabled={saving}>
            Cancel
          </button>
          <button className="btn-primary" onClick={() => handleSubmit()} disabled={saving}>
            {saving && <Spinner size={14} className="text-current" />}
            Create snapshot
          </button>
        </>
      }
    >
      <form className="space-y-4" onSubmit={handleSubmit}>
        <Field label="Name" hint="Leave empty to generate a name automatically.">
          <input className="input" value={name} onChange={(e) => setName(e.target.value)} placeholder="before-upgrade" autoFocus />
        </Field>
        <Field label="Description">
          <textarea
            className="input"
            rows={3}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="What changed, or why this snapshot exists"
          />
        </Field>
        <Checkbox
          checked={withMemory}
          onChange={setWithMemory}
          disabled={instance.state !== "running"}
          label="Include memory state"
          description={
            instance.state === "running"
              ? "Lets you resume exactly where the instance was. Slower and uses more space."
              : "Only available while the instance is running."
          }
        />
      </form>
    </Modal>
  );
}

/** 根据源实例的磁盘路径匹配所在存储池（克隆必须与源 VM 使用同一存储池） */
function detectPool(pools: StoragePool[], instance: Pick<Instance, "disks">): string {
  const diskPath = instance.disks?.find((d) => d.device !== "cdrom" && d.path)?.path || "";
  const match = pools
    .filter((p) => p.path && diskPath.startsWith(p.path.endsWith("/") ? p.path : `${p.path}/`))
    .sort((a, b) => b.path.length - a.path.length)[0];
  return match?.name || pools[0]?.name || "";
}

export function CloneSnapshotModal({
  nodeName,
  instance,
  snapshotName,
  onClose,
  onCloned,
}: {
  nodeName: string;
  instance: Pick<Instance, "id" | "name" | "vcpus" | "memory_mb" | "disks">;
  snapshotName: string;
  onClose: () => void;
  onCloned: (newInstance?: Instance) => void;
}) {
  const toast = useToast();
  const [pools, setPools] = useState<StoragePool[]>([]);
  const [poolName, setPoolName] = useState("");
  const [loadingPools, setLoadingPools] = useState(true);
  const [newName, setNewName] = useState("");
  const [vcpus, setVcpus] = useState(instance.vcpus || 1);
  const [memoryMB, setMemoryMB] = useState(instance.memory_mb || 1024);
  const [flatten, setFlatten] = useState(false);
  const [startAfter, setStartAfter] = useState(true);
  const [cloning, setCloning] = useState(false);
  // 只在首次打开时探测存储池，避免父组件刷新时覆盖用户的选择
  const instanceRef = useRef(instance);

  useEffect(() => {
    api<{ pools: StoragePool[] }>("/api/list-storage-pools", { node_name: nodeName })
      .then((data) => {
        const list = data.pools || [];
        setPools(list);
        setPoolName(detectPool(list, instanceRef.current));
      })
      .catch((err) => toast.error(errorMessage(err, "Failed to load storage pools")))
      .finally(() => setLoadingPools(false));
  }, [nodeName, toast]);

  const handleClone = async () => {
    setCloning(true);
    try {
      const data = await api<{ instance?: Instance }>("/api/clone-from-snapshot", {
        node_name: nodeName,
        source_vm_name: instance.id,
        snapshot_name: snapshotName,
        pool_name: poolName,
        new_vm_name: newName.trim() || undefined,
        vcpus: vcpus !== instance.vcpus ? vcpus : undefined,
        memory_mb: memoryMB !== instance.memory_mb ? memoryMB : undefined,
        flatten,
        start_after_clone: startAfter,
      });
      const label = data.instance?.name;
      toast.success(label ? `Instance ${label} cloned` : "Instance cloned");
      onCloned(data.instance);
    } catch (err) {
      toast.error(errorMessage(err, "Failed to clone instance"));
    } finally {
      setCloning(false);
    }
  };

  return (
    <Modal
      isOpen
      onClose={onClose}
      dismissible={!cloning}
      title="Clone from snapshot"
      description={
        <>
          New instance from <span className="font-medium text-fg">{snapshotName}</span> of {instance.name || instance.id}.
        </>
      }
      footer={
        <>
          <button className="btn-secondary" onClick={onClose} disabled={cloning}>
            Cancel
          </button>
          <button className="btn-primary" onClick={handleClone} disabled={cloning || !poolName}>
            {cloning ? <Spinner size={14} className="text-current" /> : <Copy size={14} />}
            Clone instance
          </button>
        </>
      }
    >
      <div className="space-y-4">
        <Field label="Instance name" hint="Leave empty to generate a name automatically.">
          <input className="input" value={newName} onChange={(e) => setNewName(e.target.value)} placeholder={`${instance.name || instance.id}-clone`} autoFocus />
        </Field>
        <div className="grid grid-cols-2 gap-4">
          <Field label="vCPUs" hint={`Source: ${instance.vcpus}`}>
            <input type="number" className="input" min={1} value={vcpus} onChange={(e) => setVcpus(Number(e.target.value))} />
          </Field>
          <Field label="Memory (MB)" hint={`Source: ${instance.memory_mb} MB`}>
            <input
              type="number"
              className="input"
              min={512}
              step={512}
              value={memoryMB}
              onChange={(e) => setMemoryMB(Number(e.target.value))}
            />
          </Field>
        </div>
        <Field label="Storage pool" hint="Clones are created in the same pool as the source disk.">
          <select className="input" value={poolName} onChange={(e) => setPoolName(e.target.value)} disabled={loadingPools}>
            {loadingPools && <option value="">Loading…</option>}
            {pools.map((p) => (
              <option key={p.name} value={p.name}>
                {p.name} — {p.path}
              </option>
            ))}
          </select>
        </Field>
        {!loadingPools && pools.length === 0 && <Alert tone="warning">No storage pools found on {nodeName}.</Alert>}
        <div className="space-y-4 rounded-lg border border-line p-4">
          <Switch
            checked={flatten}
            onChange={setFlatten}
            label="Independent disk"
            description="Flatten the snapshot chain into a standalone disk. Uses more space, but doesn't depend on the source."
          />
          <Switch
            checked={startAfter}
            onChange={setStartAfter}
            label="Start after cloning"
            description="Boot the new instance as soon as it is created."
          />
        </div>
      </div>
    </Modal>
  );
}
