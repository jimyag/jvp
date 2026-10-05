import { useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { Check, ChevronLeft, ChevronRight, Disc3, HardDrive, Layers, Sparkles } from "lucide-react";
import Modal from "@/components/Modal";
import { Alert, Badge, Checkbox, DescriptionList, Field, SegmentedControl, Spinner } from "@/components/ui";
import { useToast } from "@/components/ToastContainer";
import { api, errorMessage } from "@/lib/api";
import { formatMemoryMB } from "@/lib/format";
import {
  isDriverISO,
  isLinuxTemplate,
  isRecommendedWindowsCloudImage,
  isWindowsCloudImage,
  isWindowsInstallISO,
} from "@/lib/templates";
import type { Instance, KeyPair, Node, NetworkSources, StoragePool, Template } from "@/lib/types";

type OSType = "linux" | "windows";
type WindowsMode = "cloud_image" | "install";
type KeyMethod = "existing" | "paste";
type StepId = "source" | "resources" | "access" | "review";

interface CreateInstanceModalProps {
  nodes: Node[];
  defaultNode: string;
  onClose: () => void;
  onCreated: (instance: Instance | undefined, nodeName: string) => void;
}

const TIMEZONES = [
  "UTC",
  "Asia/Shanghai",
  "Asia/Tokyo",
  "Asia/Singapore",
  "Asia/Hong_Kong",
  "Europe/London",
  "Europe/Berlin",
  "America/New_York",
  "America/Los_Angeles",
];

const CPU_PRESETS = [1, 2, 4, 8, 16];
const MEMORY_PRESETS_GB = [1, 2, 4, 8, 16, 32];

function splitLines(value: string) {
  return value
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);
}

function PresetChips<T extends number>({
  values,
  current,
  format,
  min,
  onSelect,
}: {
  values: T[];
  current: number;
  format: (v: T) => string;
  min?: number;
  onSelect: (v: T) => void;
}) {
  return (
    <div className="mb-2 flex flex-wrap gap-1.5">
      {values.map((value) => {
        const disabled = min !== undefined && value < min;
        return (
          <button
            key={value}
            type="button"
            disabled={disabled}
            onClick={() => onSelect(value)}
            className={`h-7 rounded-md border px-2.5 text-xs font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-40 ${
              current === value
                ? "border-accent bg-accent-soft text-accent"
                : "border-line bg-surface text-fg-muted hover:border-line-strong hover:text-fg"
            }`}
          >
            {format(value)}
          </button>
        );
      })}
    </div>
  );
}

function OptionCard({
  selected,
  onClick,
  icon,
  title,
  description,
  badges,
}: {
  selected: boolean;
  onClick: () => void;
  icon?: ReactNode;
  title: ReactNode;
  description?: ReactNode;
  badges?: ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex w-full items-start gap-3 rounded-lg border px-3.5 py-3 text-left transition-colors ${
        selected ? "border-accent bg-accent-soft/60 ring-1 ring-accent" : "border-line bg-surface hover:border-line-strong hover:bg-subtle/50"
      }`}
    >
      {icon && <div className={`mt-0.5 flex-shrink-0 ${selected ? "text-accent" : "text-fg-subtle"}`}>{icon}</div>}
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-sm font-medium text-fg">{title}</span>
          {badges}
        </div>
        {description && <div className="mt-0.5 text-xs text-fg-muted">{description}</div>}
      </div>
      <div
        className={`mt-0.5 flex h-4 w-4 flex-shrink-0 items-center justify-center rounded-full border ${
          selected ? "border-accent bg-accent text-accent-fg" : "border-line-strong"
        }`}
      >
        {selected && <Check size={10} strokeWidth={3} />}
      </div>
    </button>
  );
}

function SectionTitle({ children, hint }: { children: ReactNode; hint?: ReactNode }) {
  return (
    <div className="mb-3">
      <h3 className="text-sm font-semibold text-fg">{children}</h3>
      {hint && <p className="mt-0.5 text-xs text-fg-muted">{hint}</p>}
    </div>
  );
}

export default function CreateInstanceModal({ nodes, defaultNode, onClose, onCreated }: CreateInstanceModalProps) {
  const toast = useToast();

  // Source
  const [name, setName] = useState("");
  const [osType, setOsType] = useState<OSType>("linux");
  const [windowsMode, setWindowsMode] = useState<WindowsMode>("cloud_image");
  const [nodeName, setNodeName] = useState(defaultNode || nodes[0]?.name || "");
  const [poolName, setPoolName] = useState("");
  const [templateId, setTemplateId] = useState("");
  const [driverIsoId, setDriverIsoId] = useState("");

  // Resources
  const [vcpus, setVcpus] = useState(2);
  const [memoryMB, setMemoryMB] = useState(2048);
  const [sizeGB, setSizeGB] = useState(20);
  const [network, setNetwork] = useState("");

  // Access
  const [hostname, setHostname] = useState("");
  const [timezone, setTimezone] = useState("Asia/Shanghai");
  const [keyMethod, setKeyMethod] = useState<KeyMethod>("existing");
  const [keypairId, setKeypairId] = useState("");
  const [publicKey, setPublicKey] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [groups, setGroups] = useState("sudo");
  const [shell, setShell] = useState("/bin/bash");
  const [sudo, setSudo] = useState("ALL=(ALL) NOPASSWD:ALL");
  const [disableRoot, setDisableRoot] = useState(false);
  const [packages, setPackages] = useState("");
  const [runCmd, setRunCmd] = useState("");

  // Remote data
  const [pools, setPools] = useState<StoragePool[]>([]);
  const [templates, setTemplates] = useState<Template[]>([]);
  const [keypairs, setKeypairs] = useState<KeyPair[]>([]);
  const [sources, setSources] = useState<NetworkSources | null>(null);
  const [loadingPools, setLoadingPools] = useState(false);
  const [loadingTemplates, setLoadingTemplates] = useState(false);

  const [stepIndex, setStepIndex] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [importedKey, setImportedKey] = useState<{ publicKey: string; id: string } | null>(null);

  const isWindows = osType === "windows";
  const isWindowsCloud = isWindows && windowsMode === "cloud_image";
  const usesCloudInit = !isWindows || isWindowsCloud;

  const steps = useMemo<{ id: StepId; label: string }[]>(() => {
    const list: { id: StepId; label: string }[] = [
      { id: "source", label: "Source" },
      { id: "resources", label: "Resources" },
    ];
    if (usesCloudInit) list.push({ id: "access", label: isWindows ? "Initialization" : "Access" });
    list.push({ id: "review", label: "Review" });
    return list;
  }, [usesCloudInit, isWindows]);

  const step = steps[Math.min(stepIndex, steps.length - 1)].id;
  const isLastStep = stepIndex >= steps.length - 1;

  // 加载密钥对
  useEffect(() => {
    api<{ keypairs: KeyPair[] }>("/api/describe-keypairs")
      .then((data) => setKeypairs(data.keypairs || []))
      .catch(() => setKeypairs([]));
  }, []);

  // 节点变化：加载存储池和网络
  useEffect(() => {
    if (!nodeName) return;
    let cancelled = false;
    setLoadingPools(true);
    setPools([]);
    setPoolName("");
    setSources(null);

    api<{ pools: StoragePool[] }>("/api/list-storage-pools", { node_name: nodeName })
      .then((data) => {
        if (cancelled) return;
        const list = data.pools || [];
        setPools(list);
        const active = list.find((p) => p.state.toLowerCase() === "active") || list[0];
        setPoolName(active?.name || "");
      })
      .catch((err) => !cancelled && toast.error(errorMessage(err, "Failed to load storage pools")))
      .finally(() => !cancelled && setLoadingPools(false));

    api<{ sources: NetworkSources }>("/api/list-network-sources", { node_name: nodeName })
      .then((data) => {
        if (cancelled) return;
        const src = data.sources || null;
        setSources(src);
        const activeNet = src?.libvirt_networks?.find((n) => n.state === "active");
        const upBridge = src?.host_bridges?.find((b) => b.state === "up") || src?.host_bridges?.[0];
        if (activeNet) setNetwork(`network:${activeNet.name}`);
        else if (upBridge) setNetwork(`bridge:${upBridge.name}`);
        else setNetwork("");
      })
      .catch(() => !cancelled && setNetwork(""));

    return () => {
      cancelled = true;
    };
  }, [nodeName, toast]);

  // 存储池变化：加载模板
  useEffect(() => {
    if (!nodeName || !poolName) {
      setTemplates([]);
      return;
    }
    let cancelled = false;
    setLoadingTemplates(true);
    api<{ templates: Template[] }>("/api/list-templates", { node_name: nodeName, pool_name: poolName })
      .then((data) => !cancelled && setTemplates(data.templates || []))
      .catch(() => !cancelled && setTemplates([]))
      .finally(() => !cancelled && setLoadingTemplates(false));
    return () => {
      cancelled = true;
    };
  }, [nodeName, poolName]);

  const windowsCloudImages = useMemo(
    () =>
      templates
        .filter(isWindowsCloudImage)
        .sort((a, b) => Number(isRecommendedWindowsCloudImage(b)) - Number(isRecommendedWindowsCloudImage(a))),
    [templates]
  );
  const windowsISOs = useMemo(() => templates.filter(isWindowsInstallISO), [templates]);
  const linuxTemplates = useMemo(() => templates.filter(isLinuxTemplate), [templates]);
  const driverISOs = useMemo(() => templates.filter(isDriverISO), [templates]);

  const visibleTemplates = isWindows ? (isWindowsCloud ? windowsCloudImages : windowsISOs) : linuxTemplates;
  const selectedTemplate = templates.find((t) => t.id === templateId);

  // 模板列表或模式变化时，选择一个合理的默认模板
  // 只在模板列表本身变化时触发，因此用户手动选择 "Blank disk" 不会被覆盖
  useEffect(() => {
    setTemplateId((current) =>
      current && visibleTemplates.some((t) => t.id === current) ? current : visibleTemplates[0]?.id || ""
    );
  }, [visibleTemplates]);

  useEffect(() => {
    if (driverIsoId && !driverISOs.some((t) => t.id === driverIsoId)) setDriverIsoId("");
  }, [driverISOs, driverIsoId]);

  const minVcpus = isWindows ? 2 : 1;
  const minMemoryMB = isWindows ? 4096 : 512;
  const templateDiskGB = !isWindows || isWindowsCloud ? selectedTemplate?.size_gb || 0 : 0;
  const minDiskGB = Math.max(isWindows ? 64 : 1, Math.ceil(templateDiskGB));

  // 资源下限随系统类型 / 模板变化自动调整
  useEffect(() => {
    setVcpus((v) => Math.max(v, minVcpus));
    setMemoryMB((m) => Math.max(m, minMemoryMB));
    setSizeGB((s) => Math.max(s, minDiskGB));
  }, [minVcpus, minMemoryMB, minDiskGB]);

  const switchOS = (next: OSType) => {
    setOsType(next);
    setGroups(next === "windows" ? "Administrators" : "sudo");
    if (next === "linux") {
      setSizeGB((s) => (s === 64 ? 20 : s));
    }
    setStepIndex(0);
  };

  const sourceValid = Boolean(nodeName && poolName && (!isWindows || templateId));
  const resourcesValid = vcpus >= minVcpus && memoryMB >= minMemoryMB && sizeGB >= minDiskGB && Boolean(network);
  const stepValid = step === "source" ? sourceValid : step === "resources" ? resourcesValid : true;

  const [networkType, networkSource] = network ? network.split(":") : ["", ""];
  const keypair = keypairs.find((k) => k.id === keypairId);
  const hasSSHKey = keyMethod === "existing" ? Boolean(keypairId) : Boolean(publicKey.trim());

  const handleSubmit = async () => {
    if (!sourceValid || !resourcesValid) return;
    setSubmitting(true);
    try {
      let keypairIds: string[] = [];
      if (usesCloudInit) {
        if (keyMethod === "existing" && keypairId) {
          keypairIds = [keypairId];
        } else if (keyMethod === "paste" && publicKey.trim()) {
          // 创建失败重试时复用已导入的密钥，避免重复导入同一个公钥
          const trimmed = publicKey.trim();
          let id = importedKey?.publicKey === trimmed ? importedKey.id : "";
          if (!id) {
            const imported = await api<{ keypair: KeyPair }>("/api/import-keypair", {
              name: `${name.trim() || "instance"}-${Date.now()}`,
              public_key: trimmed,
            });
            id = imported.keypair?.id || "";
            if (id) setImportedKey({ publicKey: trimmed, id });
          }
          if (id) keypairIds = [id];
        }
      }

      const structured: Record<string, unknown> = {};
      if (usesCloudInit) {
        if (hostname.trim()) structured.hostname = hostname.trim();
        if (timezone) structured.timezone = timezone;
        if (username.trim()) {
          structured.users = [
            {
              name: username.trim(),
              plain_text_passwd: password || undefined,
              groups: groups || undefined,
              sudo: isWindows ? undefined : sudo || undefined,
              shell: isWindows ? undefined : shell,
            },
          ];
        }
        const commands = splitLines(runCmd);
        if (commands.length) structured.run_cmd = commands;
        if (!isWindows) {
          structured.disable_root = disableRoot;
          const pkgs = splitLines(packages);
          if (pkgs.length) structured.packages = pkgs;
        }
      }

      const body = {
        node_name: nodeName,
        pool_name: poolName,
        name: name.trim() || undefined,
        template_id: templateId || undefined,
        size_gb: sizeGB,
        memory_mb: memoryMB,
        vcpus,
        network_type: networkType,
        network_source: networkSource,
        os_type: osType,
        windows_boot_mode: isWindows ? windowsMode : undefined,
        driver_iso_template_id: isWindows && !isWindowsCloud ? driverIsoId || undefined : undefined,
        keypair_ids: keypairIds.length ? keypairIds : undefined,
        user_data: usesCloudInit && Object.keys(structured).length ? { structured_user_data: structured } : undefined,
      };

      const result = await api<{ instance?: Instance }>("/api/run-instances", body);
      const label = result.instance?.name || name.trim();
      toast.success(label ? `Instance ${label} is being created` : "Instance is being created");
      onCreated(result.instance, nodeName);
    } catch (err) {
      toast.error(errorMessage(err, "Failed to create instance"));
    } finally {
      setSubmitting(false);
    }
  };

  const networkLabel = (() => {
    if (!network) return "—";
    return networkType === "network" ? `${networkSource} (libvirt network)` : `${networkSource} (host bridge)`;
  })();

  const renderSource = () => (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Field label="Instance name" hint="Leave empty to generate one automatically.">
          <input
            className="input"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="web-01"
            autoFocus
          />
        </Field>
        <Field label="Operating system">
          <SegmentedControl
            value={osType}
            onChange={switchOS}
            options={[
              { value: "linux", label: "Linux" },
              { value: "windows", label: "Windows" },
            ]}
          />
        </Field>
      </div>

      {isWindows && (
        <div>
          <SectionTitle>Provisioning</SectionTitle>
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
            <OptionCard
              selected={windowsMode === "cloud_image"}
              onClick={() => setWindowsMode("cloud_image")}
              icon={<Sparkles size={16} />}
              title="Prepared cloud image"
              badges={<Badge tone="success">Recommended</Badge>}
              description="Clones a sysprepped disk and applies hostname, user and commands with Cloudbase-Init."
            />
            <OptionCard
              selected={windowsMode === "install"}
              onClick={() => setWindowsMode("install")}
              icon={<Disc3 size={16} />}
              title="Install from ISO"
              description="Boots the Windows installer on a blank disk. You finish setup in the console."
            />
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Field label="Node" required>
          <select className="input" value={nodeName} onChange={(e) => setNodeName(e.target.value)}>
            {nodes.map((node) => (
              <option key={node.name} value={node.name} disabled={node.state !== "online"}>
                {node.name}
                {node.state !== "online" ? ` (${node.state})` : ""}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Storage pool" required hint="The system disk is created in this pool.">
          <select
            className="input"
            value={poolName}
            onChange={(e) => setPoolName(e.target.value)}
            disabled={loadingPools || pools.length === 0}
          >
            {pools.length === 0 && <option value="">{loadingPools ? "Loading…" : "No storage pools"}</option>}
            {pools.map((pool) => (
              <option key={pool.name} value={pool.name}>
                {pool.name}
                {pool.state.toLowerCase() !== "active" ? ` (${pool.state.toLowerCase()})` : ""}
              </option>
            ))}
          </select>
        </Field>
      </div>

      <div>
        <SectionTitle
          hint={
            isWindows
              ? isWindowsCloud
                ? "Windows images need Cloudbase-Init and VirtIO drivers."
                : "Choose the Windows installer ISO to boot from."
              : "The instance disk is cloned from the selected template."
          }
        >
          {isWindows ? (isWindowsCloud ? "Windows image" : "Installer ISO") : "Template"}
        </SectionTitle>

        {loadingTemplates ? (
          <div className="flex items-center gap-2 py-6 text-sm text-fg-subtle">
            <Spinner /> Loading templates…
          </div>
        ) : (
          <div className="max-h-72 space-y-2 overflow-y-auto pr-1">
            {!isWindows && (
              <OptionCard
                selected={!templateId}
                onClick={() => setTemplateId("")}
                icon={<HardDrive size={16} />}
                title="Blank disk"
                description="Create an empty disk. Useful when installing an OS manually."
              />
            )}
            {visibleTemplates.map((template) => (
              <OptionCard
                key={template.id}
                selected={templateId === template.id}
                onClick={() => setTemplateId(template.id)}
                icon={<Layers size={16} />}
                title={template.name}
                badges={
                  <>
                    {isRecommendedWindowsCloudImage(template) && <Badge tone="success">Recommended</Badge>}
                    {template.features?.cloud_init && !isWindows && <Badge tone="accent">cloud-init</Badge>}
                  </>
                }
                description={
                  <>
                    {[template.os?.name, template.os?.version, template.os?.arch].filter(Boolean).join(" ") || template.volume_name}
                    <span className="text-fg-subtle">
                      {" "}
                      · {template.size_gb} GB · {template.format}
                    </span>
                  </>
                }
              />
            ))}
            {poolName && visibleTemplates.length === 0 && (
              <Alert
                tone="warning"
                title={isWindows ? "No matching Windows media in this pool" : "No templates in this pool"}
              >
                {isWindows
                  ? isWindowsCloud
                    ? "Register a Windows disk image with Cloud-init and VirtIO enabled first."
                    : "Register a Windows installer ISO as a template first."
                  : "Pick another pool, continue with a blank disk, or "}
                {" "}
                <Link to={`/templates?node=${encodeURIComponent(nodeName)}&register=1`} className="link" onClick={onClose}>
                  register a template
                </Link>
                .
              </Alert>
            )}
          </div>
        )}

        {isWindows && !isWindowsCloud && (
          <Field
            label="VirtIO driver ISO"
            className="mt-4"
            hint="Attach drivers so the installer can see VirtIO disks and network cards."
          >
            <select className="input" value={driverIsoId} onChange={(e) => setDriverIsoId(e.target.value)}>
              <option value="">No driver ISO</option>
              {driverISOs.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </select>
          </Field>
        )}
      </div>
    </div>
  );

  const renderResources = () => (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
        <Field label="vCPUs" required hint={isWindows ? "Windows requires at least 2 vCPUs." : undefined}>
          <PresetChips values={CPU_PRESETS} current={vcpus} min={minVcpus} format={(v) => `${v}`} onSelect={setVcpus} />
          <input
            type="number"
            className="input"
            min={minVcpus}
            max={128}
            value={vcpus}
            onChange={(e) => setVcpus(Number(e.target.value))}
          />
        </Field>
        <Field label="Memory" required hint={`${memoryMB} MB${isWindows ? " · Windows requires at least 4 GB." : ""}`}>
          <PresetChips
            values={MEMORY_PRESETS_GB}
            current={memoryMB / 1024}
            min={minMemoryMB / 1024}
            format={(v) => `${v} GB`}
            onSelect={(gb) => setMemoryMB(gb * 1024)}
          />
          <div className="relative">
            <input
              type="number"
              className="input pr-12"
              min={minMemoryMB / 1024}
              step={0.5}
              value={memoryMB / 1024}
              onChange={(e) => setMemoryMB(Math.round(Number(e.target.value) * 1024))}
            />
            <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-xs text-fg-subtle">GB</span>
          </div>
        </Field>
      </div>

      <Field
        label="Disk size"
        required
        hint={
          templateDiskGB
            ? `Must be at least ${minDiskGB} GB (template size).`
            : isWindows
              ? "Windows requires at least 64 GB."
              : undefined
        }
      >
        <div className="relative sm:max-w-[calc(50%-0.75rem)]">
          <input
            type="number"
            className="input pr-12"
            min={minDiskGB}
            value={sizeGB}
            onChange={(e) => setSizeGB(Number(e.target.value))}
          />
          <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-xs text-fg-subtle">GB</span>
        </div>
      </Field>

      <Field
        label="Network"
        required
        hint={
          networkType === "network"
            ? "Libvirt network: NAT with DHCP managed by libvirt."
            : networkType === "bridge"
              ? "Host bridge: the instance joins the host's physical network."
              : undefined
        }
      >
        <select className="input" value={network} onChange={(e) => setNetwork(e.target.value)}>
          <option value="">Select a network</option>
          {!!sources?.libvirt_networks?.length && (
            <optgroup label="Libvirt networks">
              {sources.libvirt_networks.map((net) => (
                <option key={`network:${net.name}`} value={`network:${net.name}`} disabled={net.state !== "active"}>
                  {net.name} · {net.mode}
                  {net.ip_address ? ` · ${net.ip_address}` : ""}
                  {net.state !== "active" ? " (inactive)" : ""}
                </option>
              ))}
            </optgroup>
          )}
          {!!sources?.host_bridges?.length && (
            <optgroup label="Host bridges">
              {sources.host_bridges.map((br) => (
                <option key={`bridge:${br.name}`} value={`bridge:${br.name}`} disabled={br.state !== "up"}>
                  {br.name}
                  {br.ips?.length ? ` · ${br.ips[0]}` : ""}
                  {br.state !== "up" ? " (down)" : ""}
                </option>
              ))}
            </optgroup>
          )}
        </select>
        {sources && !sources.libvirt_networks?.length && !sources.host_bridges?.length && (
          <Alert tone="warning" className="mt-3">
            This node has no networks.{" "}
            <Link to={`/networks?node=${encodeURIComponent(nodeName)}`} className="link" onClick={onClose}>
              Create a network
            </Link>{" "}
            first.
          </Alert>
        )}
      </Field>
    </div>
  );

  const renderAccess = () => (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Field label="Hostname">
          <input className="input" value={hostname} onChange={(e) => setHostname(e.target.value)} placeholder={name || "my-server"} />
        </Field>
        <Field label="Timezone">
          <select className="input" value={timezone} onChange={(e) => setTimezone(e.target.value)}>
            {TIMEZONES.map((tz) => (
              <option key={tz} value={tz}>
                {tz}
              </option>
            ))}
          </select>
        </Field>
      </div>

      <div>
        <SectionTitle hint="The public key is added to the default user's authorized keys.">SSH key</SectionTitle>
        <SegmentedControl
          size="sm"
          className="mb-3"
          value={keyMethod}
          onChange={setKeyMethod}
          options={[
            { value: "existing", label: "Saved key pair" },
            { value: "paste", label: "Paste public key" },
          ]}
        />
        {keyMethod === "existing" ? (
          keypairs.length === 0 ? (
            <p className="text-sm text-fg-muted">
              No saved key pairs.{" "}
              <button type="button" className="link" onClick={() => setKeyMethod("paste")}>
                Paste a public key
              </button>{" "}
              or{" "}
              <Link to="/keypairs" className="link" onClick={onClose}>
                create a key pair
              </Link>
              .
            </p>
          ) : (
            <select className="input" value={keypairId} onChange={(e) => setKeypairId(e.target.value)}>
              <option value="">No SSH key</option>
              {keypairs.map((kp) => (
                <option key={kp.id} value={kp.id}>
                  {kp.name}
                </option>
              ))}
            </select>
          )
        ) : (
          <div className="space-y-2">
            <textarea
              className="input font-mono text-xs"
              rows={3}
              value={publicKey}
              onChange={(e) => setPublicKey(e.target.value)}
              placeholder="ssh-ed25519 AAAAC3Nza… user@host"
            />
            <label className="inline-flex cursor-pointer items-center gap-1.5 text-xs text-fg-muted hover:text-fg">
              <input
                type="file"
                className="hidden"
                accept=".pub,.pem,.key,text/plain"
                onChange={(e) => {
                  const file = e.target.files?.[0];
                  if (!file) return;
                  const reader = new FileReader();
                  reader.onload = () => setPublicKey(String(reader.result || "").trim());
                  reader.readAsText(file);
                }}
              />
              <span className="link">Load from file…</span>
              <span className="text-fg-subtle">The key will be saved to Key Pairs.</span>
            </label>
          </div>
        )}
      </div>

      <div>
        <SectionTitle hint="Optional. Leave the username empty to keep the image's default user.">
          {isWindows ? "Administrator account" : "Default user"}
        </SectionTitle>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Username">
            <input
              className="input"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder={isWindows ? "admin" : "ubuntu"}
              autoComplete="off"
            />
          </Field>
          <Field label="Password">
            <input
              type="password"
              className="input"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="new-password"
              disabled={!username.trim()}
            />
          </Field>
          <Field label="Groups">
            <input className="input" value={groups} onChange={(e) => setGroups(e.target.value)} disabled={!username.trim()} />
          </Field>
          {!isWindows && (
            <Field label="Shell">
              <select className="input" value={shell} onChange={(e) => setShell(e.target.value)} disabled={!username.trim()}>
                <option value="/bin/bash">/bin/bash</option>
                <option value="/bin/sh">/bin/sh</option>
                <option value="/bin/zsh">/bin/zsh</option>
              </select>
            </Field>
          )}
          {!isWindows && (
            <Field label="Sudo rule" className="sm:col-span-2">
              <input
                className="input font-mono text-xs"
                value={sudo}
                onChange={(e) => setSudo(e.target.value)}
                disabled={!username.trim()}
              />
            </Field>
          )}
        </div>
        {!isWindows && (
          <div className="mt-4">
            <Checkbox checked={disableRoot} onChange={setDisableRoot} label="Disable root login" />
          </div>
        )}
      </div>

      <details className="group rounded-lg border border-line" open={Boolean(packages || runCmd)}>
        <summary className="flex cursor-pointer list-none items-center justify-between px-4 py-3 text-sm font-medium text-fg">
          <span>
            Advanced initialization
            <span className="ml-2 text-xs font-normal text-fg-subtle">
              {isWindows ? "First-boot commands" : "Packages and first-boot commands"}
            </span>
          </span>
          <ChevronRight size={15} className="text-fg-subtle transition-transform group-open:rotate-90" />
        </summary>
        <div className="space-y-4 border-t border-line px-4 py-4">
          {!isWindows && (
            <Field label="Packages" hint="One package per line.">
              <textarea
                className="input font-mono text-xs"
                rows={3}
                value={packages}
                onChange={(e) => setPackages(e.target.value)}
                placeholder={"nginx\ngit\ncurl"}
              />
            </Field>
          )}
          <Field label="Run commands" hint={isWindows ? "One cmd.exe command per line." : "One shell command per line, run once on first boot."}>
            <textarea
              className="input font-mono text-xs"
              rows={4}
              value={runCmd}
              onChange={(e) => setRunCmd(e.target.value)}
              placeholder={
                isWindows
                  ? 'powershell.exe -NoProfile -Command "Enable-PSRemoting -Force"'
                  : "systemctl enable --now nginx"
              }
            />
          </Field>
        </div>
      </details>
    </div>
  );

  const renderReview = () => {
    const template = selectedTemplate;
    const driverIso = driverISOs.find((t) => t.id === driverIsoId);
    return (
      <div className="space-y-6">
        <div>
          <SectionTitle>Summary</SectionTitle>
          <div className="rounded-lg border border-line p-4">
            <DescriptionList
              items={[
                { label: "Name", value: name.trim() || <span className="text-fg-subtle">Auto-generated</span> },
                {
                  label: "Operating system",
                  value: isWindows ? `Windows · ${isWindowsCloud ? "cloud image" : "ISO install"}` : "Linux",
                },
                { label: "Node", value: nodeName },
                { label: "Storage pool", value: poolName },
                { label: isWindows && !isWindowsCloud ? "Installer ISO" : "Template", value: template?.name || "Blank disk" },
                ...(isWindows && !isWindowsCloud ? [{ label: "Driver ISO", value: driverIso?.name || "None" }] : []),
                { label: "Compute", value: `${vcpus} vCPU · ${formatMemoryMB(memoryMB)}` },
                { label: "Disk", value: `${sizeGB} GB` },
                { label: "Network", value: networkLabel },
                ...(usesCloudInit
                  ? [
                      {
                        label: "SSH key",
                        value: keyMethod === "existing" ? keypair?.name || "None" : publicKey.trim() ? "Pasted public key" : "None",
                      },
                      { label: "User", value: username.trim() || "Image default" },
                      { label: "Hostname", value: hostname.trim() || "Default" },
                      { label: "Timezone", value: timezone },
                    ]
                  : []),
              ]}
            />
          </div>
        </div>
        {usesCloudInit && !hasSSHKey && !password && (
          <Alert tone="warning" title="No login credentials configured">
            Without an SSH key or password you can only sign in if the image has built-in credentials. You can still use the
            console.
          </Alert>
        )}
        {isWindows && !isWindowsCloud && (
          <Alert tone="info">
            The installer starts on first boot. Open the instance console to complete Windows setup
            {driverIso ? " and load the VirtIO drivers from the attached ISO" : ""}.
          </Alert>
        )}
      </div>
    );
  };

  return (
    <Modal
      isOpen
      onClose={onClose}
      dismissible={!submitting}
      size="xl"
      title="Create instance"
      bodyClassName="p-0"
      footer={
        <div className="flex w-full items-center justify-between">
          <span className="text-xs text-fg-subtle">
            Step {stepIndex + 1} of {steps.length}
          </span>
          <div className="flex gap-2">
            {stepIndex > 0 ? (
              <button type="button" className="btn-secondary" onClick={() => setStepIndex((i) => i - 1)} disabled={submitting}>
                <ChevronLeft size={15} />
                Back
              </button>
            ) : (
              <button type="button" className="btn-secondary" onClick={onClose}>
                Cancel
              </button>
            )}
            {isLastStep ? (
              <button type="button" className="btn-primary" onClick={handleSubmit} disabled={submitting || !sourceValid || !resourcesValid}>
                {submitting && <Spinner size={14} className="text-current" />}
                Create instance
              </button>
            ) : (
              <button type="button" className="btn-primary" onClick={() => setStepIndex((i) => i + 1)} disabled={!stepValid}>
                Continue
                <ChevronRight size={15} />
              </button>
            )}
          </div>
        </div>
      }
    >
      <div className="flex flex-col md:h-[min(600px,calc(92vh-8.5rem))] md:flex-row">
        <ol className="flex gap-1 overflow-x-auto border-b border-line bg-canvas/60 px-4 py-3 md:w-52 md:flex-shrink-0 md:flex-col md:border-b-0 md:border-r md:px-3 md:py-5">
          {steps.map((s, index) => {
            const done = index < stepIndex;
            const active = index === stepIndex;
            // 只允许回到已完成的步骤，或在前序步骤有效时前进
            const reachable = index <= stepIndex || (index === stepIndex + 1 && stepValid) || (sourceValid && resourcesValid);
            return (
              <li key={s.id}>
                <button
                  type="button"
                  disabled={!reachable}
                  onClick={() => setStepIndex(index)}
                  className={`flex w-full items-center gap-2.5 whitespace-nowrap rounded-md px-2.5 py-2 text-left text-sm transition-colors disabled:cursor-not-allowed ${
                    active ? "bg-surface font-medium text-fg shadow-card ring-1 ring-line" : "text-fg-muted hover:text-fg disabled:opacity-50"
                  }`}
                >
                  <span
                    className={`flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-full text-[11px] font-semibold ${
                      done ? "bg-accent text-accent-fg" : active ? "bg-accent-soft text-accent ring-1 ring-accent" : "bg-subtle text-fg-subtle"
                    }`}
                  >
                    {done ? <Check size={11} strokeWidth={3} /> : index + 1}
                  </span>
                  {s.label}
                </button>
              </li>
            );
          })}
        </ol>
        <div className="min-w-0 flex-1 px-6 py-5 md:overflow-y-auto">
          <form
            onSubmit={(e) => {
              e.preventDefault();
              if (!isLastStep && stepValid) setStepIndex((i) => i + 1);
            }}
          >
            {step === "source" && renderSource()}
            {step === "resources" && renderResources()}
            {step === "access" && renderAccess()}
            {step === "review" && renderReview()}
            <button type="submit" className="hidden" aria-hidden tabIndex={-1} />
          </form>
        </div>
      </div>
    </Modal>
  );
}
