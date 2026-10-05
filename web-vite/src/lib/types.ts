// 与后端 entity 对应的前端类型

export interface Node {
  name: string;
  uuid: string;
  uri: string;
  type: string;
  state: string;
  created_at?: string;
  updated_at?: string;
}

export interface InstanceDisk {
  target?: string;
  path?: string;
  format?: string;
  device?: string;
  capacity_b?: number;
  allocation_b?: number;
}

export interface InstanceInterface {
  name: string;
  type: string;
  source: string;
  mac: string;
  ips?: string[];
}

export interface Instance {
  id: string;
  name: string;
  state: string;
  node_name: string;
  template_id?: string;
  memory_mb: number;
  vcpus: number;
  created_at?: string;
  started_at?: string;
  domain_uuid?: string;
  domain_name?: string;
  ip_address?: string;
  autostart?: boolean;
  interfaces?: InstanceInterface[];
  disks?: InstanceDisk[];
}

export interface StoragePool {
  name: string;
  uuid: string;
  state: string;
  type: string;
  capacity: number;
  allocation: number;
  available: number;
  path: string;
  volume_count: number;
}

export interface Volume {
  volume_id: string;
  name: string;
  node_name: string;
  pool: string;
  path: string;
  capacity_b: number;
  size_gb: number;
  allocation_b: number;
  format: string;
}

export interface Template {
  id: string;
  name: string;
  description?: string;
  node_name: string;
  pool_name: string;
  volume_name?: string;
  size_gb: number;
  format: string;
  created_at?: string;
  tags?: string[];
  os?: { name?: string; version?: string; arch?: string };
  features?: { cloud_init?: boolean; virtio?: boolean; qemu_guest_agent?: boolean };
}

export interface DownloadTask {
  id: string;
  node_name: string;
  pool_name: string;
  volume_name: string;
  status: string;
  error?: string;
}

export interface KeyPair {
  id: string;
  name: string;
  algorithm?: string;
  fingerprint: string;
  public_key: string;
  created_at?: string;
}

export interface SnapshotDisk {
  target?: string;
  path?: string;
  format?: string;
}

export interface Snapshot {
  id: string;
  name: string;
  vm_name: string;
  node_name: string;
  created_at?: string;
  state?: string;
  description?: string;
  parent?: string;
  memory?: boolean;
  disk_only?: boolean;
  disks?: SnapshotDisk[];
}

export interface LibvirtNetwork {
  name: string;
  uuid: string;
  node_name?: string;
  type?: string;
  mode: string;
  bridge?: string;
  state: string;
  autostart?: boolean;
  persistent?: boolean;
  ip_address: string;
  netmask: string;
  dhcp_start: string;
  dhcp_end: string;
}

export interface HostBridge {
  name: string;
  state: string;
  mac?: string;
  ips: string[];
  interfaces?: string[];
  stp?: boolean;
  mtu?: number;
}

export interface NetworkSources {
  libvirt_networks: LibvirtNetwork[];
  host_bridges: HostBridge[];
}
