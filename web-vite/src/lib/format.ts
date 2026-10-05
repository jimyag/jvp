// 通用格式化函数

const GiB = 1024 * 1024 * 1024;

export function formatBytes(bytes?: number | null, digits = 1): string {
  if (!bytes || bytes <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB", "PB"];
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const value = bytes / Math.pow(1024, i);
  return `${value.toFixed(i === 0 ? 0 : digits).replace(/\.0+$/, "")} ${units[i]}`;
}

export function bytesToGB(bytes: number): number {
  return bytes / GiB;
}

export function formatMemoryMB(mb?: number | null): string {
  if (!mb) return "—";
  if (mb < 1024) return `${mb} MB`;
  return `${(mb / 1024).toFixed(1).replace(/\.0$/, "")} GB`;
}

export function formatDate(value?: string | null): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

export function formatRelative(value?: string | null): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  const seconds = Math.round((Date.now() - date.getTime()) / 1000);
  const abs = Math.abs(seconds);
  const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });
  if (abs < 60) return rtf.format(-seconds, "second");
  if (abs < 3600) return rtf.format(-Math.round(seconds / 60), "minute");
  if (abs < 86400) return rtf.format(-Math.round(seconds / 3600), "hour");
  if (abs < 86400 * 30) return rtf.format(-Math.round(seconds / 86400), "day");
  return date.toLocaleDateString();
}

export function percent(used: number, total: number): number {
  if (!total || total <= 0) return 0;
  return Math.min(100, Math.max(0, (used / total) * 100));
}

export function netmaskToPrefix(netmask?: string): number | null {
  if (!netmask) return null;
  const parts = netmask.split(".").map(Number);
  if (parts.length !== 4 || parts.some((p) => Number.isNaN(p) || p < 0 || p > 255)) return null;
  return parts.reduce((bits, part) => bits + part.toString(2).split("1").length - 1, 0);
}

export function shortId(id?: string, length = 12): string {
  if (!id) return "";
  return id.length > length ? `${id.slice(0, length)}…` : id;
}
