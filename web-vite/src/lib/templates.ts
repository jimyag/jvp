// 模板分类辅助函数（实例创建和模板页面共用）
import type { Template } from "./types";

function templateText(template: Template) {
  return `${template.name} ${template.volume_name || ""} ${template.os?.name || ""} ${template.os?.version || ""} ${template.format || ""}`.toLowerCase();
}

export function isISO(template: Template) {
  const tags = template.tags || [];
  return Boolean(template.volume_name?.toLowerCase().endsWith(".iso") || tags.includes("iso") || tags.includes("installer"));
}

export function isDriverISO(template: Template) {
  const text = templateText(template);
  return text.includes("virtio") || text.includes("driver") || text.includes("guest-tools") || text.includes("guest tools");
}

export function isWindowsInstallISO(template: Template) {
  const text = templateText(template);
  return isISO(template) && !isDriverISO(template) && (text.includes("windows") || text.includes("winserver") || text.includes("server 20"));
}

export function isWindowsCloudImage(template: Template) {
  const text = templateText(template);
  const tags = template.tags || [];
  const isWindows = text.includes("windows") || text.includes("win10") || text.includes("win11") || tags.includes("windows");
  return (
    isWindows &&
    !isISO(template) &&
    !isDriverISO(template) &&
    template.features?.cloud_init === true &&
    template.features?.virtio === true
  );
}

export function isRecommendedWindowsCloudImage(template: Template) {
  const tags = template.tags || [];
  return isWindowsCloudImage(template) && template.features?.qemu_guest_agent === true && tags.includes("vnc-clipboard");
}

/** Linux 实例可选的模板：排除 Windows 镜像、安装 ISO 和驱动 ISO */
export function isLinuxTemplate(template: Template) {
  return !isWindowsCloudImage(template) && !isWindowsInstallISO(template) && !isDriverISO(template);
}
