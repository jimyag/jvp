import { useEffect, useMemo, useState } from "react";
import { Download, ExternalLink, HardDrive } from "lucide-react";
import Modal from "@/components/Modal";
import { Checkbox, Field, SegmentedControl, Spinner } from "@/components/ui";
import { useToast } from "@/components/ToastContainer";
import { api, errorMessage } from "@/lib/api";
import type { DownloadTask, Node, StoragePool, Template, Volume } from "@/lib/types";

interface Preset {
  name: string;
  category: string;
  url: string;
  fileName?: string;
  os: { name: string; version: string; arch: string };
  tags?: string;
  description?: string;
  helpUrl?: string;
  helpText?: string;
  features: { cloudInit: boolean; virtio: boolean; qga: boolean };
}

// 预设的常用镜像 / ISO
const PRESETS: Preset[] = [
  {
    name: "Ubuntu 26.04 LTS",
    category: "Linux cloud images",
    url: "https://cloud-images.ubuntu.com/resolute/current/resolute-server-cloudimg-amd64.img",
    os: { name: "Ubuntu", version: "26.04", arch: "x86_64" },
    features: { cloudInit: true, virtio: true, qga: false },
  },
  {
    name: "Ubuntu 24.04 LTS (Noble)",
    category: "Linux cloud images",
    url: "https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img",
    os: { name: "Ubuntu", version: "24.04", arch: "x86_64" },
    features: { cloudInit: true, virtio: true, qga: false },
  },
  {
    name: "Ubuntu 22.04 LTS (Jammy)",
    category: "Linux cloud images",
    url: "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img",
    os: { name: "Ubuntu", version: "22.04", arch: "x86_64" },
    features: { cloudInit: true, virtio: true, qga: false },
  },
  {
    name: "Ubuntu 20.04 LTS (Focal)",
    category: "Linux cloud images",
    url: "https://cloud-images.ubuntu.com/focal/current/focal-server-cloudimg-amd64.img",
    os: { name: "Ubuntu", version: "20.04", arch: "x86_64" },
    features: { cloudInit: true, virtio: true, qga: false },
  },
  {
    name: "Debian 13 (Trixie)",
    category: "Linux cloud images",
    url: "https://cloud.debian.org/images/cloud/trixie/latest/debian-13-generic-amd64.qcow2",
    os: { name: "Debian", version: "13", arch: "x86_64" },
    features: { cloudInit: true, virtio: true, qga: false },
  },
  {
    name: "Debian 12 (Bookworm)",
    category: "Linux cloud images",
    url: "https://cloud.debian.org/images/cloud/bookworm/latest/debian-12-generic-amd64.qcow2",
    os: { name: "Debian", version: "12", arch: "x86_64" },
    features: { cloudInit: true, virtio: true, qga: false },
  },
  {
    name: "Debian 11 (Bullseye)",
    category: "Linux cloud images",
    url: "https://cloud.debian.org/images/cloud/bullseye/latest/debian-11-generic-amd64.qcow2",
    os: { name: "Debian", version: "11", arch: "x86_64" },
    features: { cloudInit: true, virtio: true, qga: false },
  },
  {
    name: "CentOS Stream 10",
    category: "Linux cloud images",
    url: "https://cloud.centos.org/centos/10-stream/x86_64/images/CentOS-Stream-GenericCloud-10-latest.x86_64.qcow2",
    os: { name: "CentOS Stream", version: "10", arch: "x86_64" },
    features: { cloudInit: true, virtio: true, qga: false },
  },
  {
    name: "Rocky Linux 10",
    category: "Linux cloud images",
    url: "https://dl.rockylinux.org/vault/rocky/10.0/images/x86_64/Rocky-10-GenericCloud-Base.latest.x86_64.qcow2",
    os: { name: "Rocky Linux", version: "10", arch: "x86_64" },
    features: { cloudInit: true, virtio: true, qga: false },
  },
  {
    name: "Windows 11 25H2 x64 (includes Pro)",
    category: "Windows installation media",
    url: "",
    fileName: "Win11_25H2_English_x64.iso",
    os: { name: "Windows", version: "11 25H2", arch: "x86_64" },
    tags: "windows,installer,iso",
    description:
      "Microsoft Windows 11 multi-edition x64 installer ISO. Paste the temporary ISO URL generated from the official Microsoft download page.",
    helpUrl: "https://www.microsoft.com/software-download/windows11",
    helpText: "Get a download link from Microsoft",
    features: { cloudInit: false, virtio: true, qga: false },
  },
  {
    name: "VirtIO Windows Drivers (stable)",
    category: "Windows driver media",
    url: "https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/stable-virtio/virtio-win.iso",
    fileName: "virtio-win.iso",
    os: { name: "Windows VirtIO Drivers", version: "stable", arch: "x86_64" },
    tags: "windows,virtio,driver,iso",
    description: "Stable virtio-win driver ISO for Windows guests on KVM/QEMU.",
    helpUrl: "https://github.com/virtio-win/virtio-win-pkg-scripts/blob/master/README.md",
    helpText: "virtio-win download notes",
    features: { cloudInit: false, virtio: true, qga: false },
  },
];

const PRESET_CATEGORIES = Array.from(new Set(PRESETS.map((p) => p.category)));

function fileNameFromUrl(url: string) {
  try {
    const path = new URL(url).pathname;
    return decodeURIComponent(path.split("/").filter(Boolean).pop() || "");
  } catch {
    return url.split("/").pop()?.split("?")[0] || "";
  }
}

type SourceType = "url" | "volume";

interface RegisterTemplateModalProps {
  nodes: Node[];
  defaultNode: string;
  defaultPool?: string;
  onClose: () => void;
  onRegistered: (result: { template?: Template; task?: DownloadTask }) => void;
}

export default function RegisterTemplateModal({ nodes, defaultNode, defaultPool, onClose, onRegistered }: RegisterTemplateModalProps) {
  const toast = useToast();
  const [source, setSource] = useState<SourceType>("url");
  const [nodeName, setNodeName] = useState(defaultNode);
  const [poolName, setPoolName] = useState(defaultPool || "");
  const [pools, setPools] = useState<StoragePool[]>([]);
  const [volumes, setVolumes] = useState<Volume[]>([]);
  const [loadingVolumes, setLoadingVolumes] = useState(false);

  const [preset, setPreset] = useState("");
  const [url, setUrl] = useState("");
  const [volumeName, setVolumeName] = useState("");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [tags, setTags] = useState("");
  const [osName, setOsName] = useState("");
  const [osVersion, setOsVersion] = useState("");
  const [osArch, setOsArch] = useState("x86_64");
  const [cloudInit, setCloudInit] = useState(true);
  const [virtio, setVirtio] = useState(true);
  const [qga, setQga] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const selectedPreset = useMemo(() => PRESETS.find((p) => p.name === preset), [preset]);

  useEffect(() => {
    if (!nodeName) return;
    let cancelled = false;
    api<{ pools: StoragePool[] }>("/api/list-storage-pools", { node_name: nodeName })
      .then((data) => {
        if (cancelled) return;
        const list = data.pools || [];
        setPools(list);
        setPoolName((prev) => (prev && list.some((p) => p.name === prev) ? prev : list[0]?.name || ""));
      })
      .catch(() => !cancelled && setPools([]));
    return () => {
      cancelled = true;
    };
  }, [nodeName]);

  useEffect(() => {
    if (source !== "volume" || !nodeName || !poolName) return;
    let cancelled = false;
    setLoadingVolumes(true);
    api<{ volumes: Volume[] }>("/api/list-volumes", { node_name: nodeName, pool_name: poolName })
      .then((data) => !cancelled && setVolumes((data.volumes || []).sort((a, b) => a.name.localeCompare(b.name))))
      .catch(() => !cancelled && setVolumes([]))
      .finally(() => !cancelled && setLoadingVolumes(false));
    return () => {
      cancelled = true;
    };
  }, [source, nodeName, poolName]);

  const applyPreset = (value: string) => {
    setPreset(value);
    const p = PRESETS.find((item) => item.name === value);
    if (!p) return;
    setUrl(p.url);
    setVolumeName(p.fileName || fileNameFromUrl(p.url));
    setName(p.name);
    setDescription(p.description || "");
    setTags(p.tags || "");
    setOsName(p.os.name);
    setOsVersion(p.os.version);
    setOsArch(p.os.arch);
    setCloudInit(p.features.cloudInit);
    setVirtio(p.features.virtio);
    setQga(p.features.qga);
  };

  const valid =
    nodeName &&
    poolName &&
    name.trim() &&
    volumeName.trim() &&
    (source === "volume" || url.trim());

  const handleSubmit = async (e?: React.FormEvent) => {
    e?.preventDefault();
    if (!valid) return;
    setSubmitting(true);
    try {
      const result = await api<{ template?: Template; download_task?: DownloadTask }>("/api/register-template", {
        node_name: nodeName,
        pool_name: poolName,
        volume_name: volumeName.trim(),
        name: name.trim(),
        description: description.trim(),
        tags: tags
          .split(",")
          .map((t) => t.trim())
          .filter(Boolean),
        os: { name: osName.trim(), version: osVersion.trim(), arch: osArch.trim() },
        features: { cloud_init: cloudInit, virtio, qemu_guest_agent: qga },
        source: source === "url" ? { type: "url", url: url.trim() } : undefined,
      });
      if (result.download_task) {
        toast.info(`Downloading ${volumeName.trim()} — the template appears when the download finishes.`);
      } else {
        toast.success(`Template ${name.trim()} registered`);
      }
      onRegistered({ template: result.template, task: result.download_task });
    } catch (err) {
      toast.error(errorMessage(err, "Failed to register template"));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      isOpen
      onClose={onClose}
      dismissible={!submitting}
      size="lg"
      title="Register template"
      description="Templates are reusable disk images or ISOs that instances are created from."
      footer={
        <>
          <button className="btn-secondary" onClick={onClose} disabled={submitting}>
            Cancel
          </button>
          <button className="btn-primary" onClick={() => handleSubmit()} disabled={submitting || !valid}>
            {submitting && <Spinner size={14} className="text-current" />}
            {source === "url" ? "Download & register" : "Register template"}
          </button>
        </>
      }
    >
      <form className="space-y-6" onSubmit={handleSubmit}>
        <SegmentedControl
          value={source}
          onChange={(value) => {
            setSource(value);
            setVolumeName("");
          }}
          options={[
            { value: "url", label: "Download from URL", icon: <Download size={14} /> },
            { value: "volume", label: "Existing volume", icon: <HardDrive size={14} /> },
          ]}
        />

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Node" required>
            <select className="input" value={nodeName} onChange={(e) => setNodeName(e.target.value)}>
              {nodes.map((n) => (
                <option key={n.name} value={n.name}>
                  {n.name}
                </option>
              ))}
            </select>
          </Field>
          <Field label="Storage pool" required hint={source === "url" ? "The image is downloaded into this pool." : undefined}>
            <select className="input" value={poolName} onChange={(e) => setPoolName(e.target.value)} disabled={pools.length === 0}>
              {pools.length === 0 && <option value="">No storage pools</option>}
              {pools.map((p) => (
                <option key={p.name} value={p.name}>
                  {p.name}
                </option>
              ))}
            </select>
          </Field>
        </div>

        {source === "url" ? (
          <div className="space-y-4">
            <Field label="Image" hint="Choose a known image to fill in the details, or enter any URL below.">
              <select className="input" value={preset} onChange={(e) => applyPreset(e.target.value)}>
                <option value="">Custom URL</option>
                {PRESET_CATEGORIES.map((category) => (
                  <optgroup key={category} label={category}>
                    {PRESETS.filter((p) => p.category === category).map((p) => (
                      <option key={p.name} value={p.name}>
                        {p.name}
                      </option>
                    ))}
                  </optgroup>
                ))}
              </select>
            </Field>
            <Field
              label="Download URL"
              required
              hint={
                selectedPreset?.helpUrl ? (
                  <a href={selectedPreset.helpUrl} target="_blank" rel="noreferrer" className="link inline-flex items-center gap-1">
                    {selectedPreset.helpText || "Download page"}
                    <ExternalLink size={11} />
                  </a>
                ) : undefined
              }
            >
              <input
                className="input font-mono text-[13px]"
                value={url}
                onChange={(e) => {
                  setUrl(e.target.value);
                  if (!volumeName || volumeName === fileNameFromUrl(url)) setVolumeName(fileNameFromUrl(e.target.value));
                }}
                placeholder="https://cloud-images.example.com/image.qcow2"
              />
            </Field>
            <Field label="Save as" required hint="File name of the new volume in the pool.">
              <input className="input font-mono text-[13px]" value={volumeName} onChange={(e) => setVolumeName(e.target.value)} placeholder="ubuntu-24.04.qcow2" />
            </Field>
          </div>
        ) : (
          <Field label="Volume" required hint="An existing disk image or ISO in the selected pool.">
            <select className="input" value={volumeName} onChange={(e) => setVolumeName(e.target.value)} disabled={loadingVolumes}>
              <option value="">{loadingVolumes ? "Loading volumes…" : volumes.length ? "Select a volume" : "No volumes in this pool"}</option>
              {volumes.map((v) => (
                <option key={v.volume_id} value={v.name}>
                  {v.name} ({v.size_gb} GB, {v.format})
                </option>
              ))}
            </select>
          </Field>
        )}

        <div className="space-y-4 border-t border-line pt-5">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field label="Template name" required>
              <input className="input" value={name} onChange={(e) => setName(e.target.value)} placeholder="Ubuntu 24.04" />
            </Field>
            <Field label="Tags" hint="Comma separated.">
              <input className="input" value={tags} onChange={(e) => setTags(e.target.value)} placeholder="linux, lts" />
            </Field>
          </div>
          <Field label="Description">
            <textarea className="input" rows={2} value={description} onChange={(e) => setDescription(e.target.value)} />
          </Field>
          <div className="grid grid-cols-3 gap-4">
            <Field label="OS">
              <input className="input" value={osName} onChange={(e) => setOsName(e.target.value)} placeholder="Ubuntu" />
            </Field>
            <Field label="Version">
              <input className="input" value={osVersion} onChange={(e) => setOsVersion(e.target.value)} placeholder="24.04" />
            </Field>
            <Field label="Architecture">
              <input className="input" value={osArch} onChange={(e) => setOsArch(e.target.value)} />
            </Field>
          </div>
          <div>
            <div className="label">Guest features</div>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
              <Checkbox checked={cloudInit} onChange={setCloudInit} label="Cloud-init" description="or Cloudbase-Init" />
              <Checkbox checked={virtio} onChange={setVirtio} label="VirtIO drivers" />
              <Checkbox checked={qga} onChange={setQga} label="QEMU guest agent" />
            </div>
          </div>
        </div>
      </form>
    </Modal>
  );
}
