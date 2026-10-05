import { useCallback, useState } from "react";
import ConfirmDialog from "@/components/ConfirmDialog";
import { Checkbox } from "@/components/ui";
import { useToast } from "@/components/ToastContainer";
import { api, errorMessage } from "@/lib/api";
import type { Instance } from "@/lib/types";

export type InstanceAction = "start" | "stop" | "reboot" | "terminate";

export const actionProgressLabel: Record<InstanceAction, string> = {
  start: "Starting",
  stop: "Stopping",
  reboot: "Rebooting",
  terminate: "Terminating",
};

interface Options {
  /** 操作成功后回调（列表刷新、详情页跳转等） */
  onChanged?: (instance: Instance, action: InstanceAction) => void;
}

/** 实例电源 / 删除操作，包含确认弹窗，列表页和详情页共用 */
export function useInstanceActions({ onChanged }: Options = {}) {
  const toast = useToast();
  const [busy, setBusy] = useState<Record<string, InstanceAction>>({});
  const [pending, setPending] = useState<{ instance: Instance; action: InstanceAction } | null>(null);
  const [deleteVolumes, setDeleteVolumes] = useState(false);

  const execute = useCallback(
    async (instance: Instance, action: InstanceAction, extra: Record<string, unknown> = {}) => {
      setBusy((prev) => ({ ...prev, [instance.id]: action }));
      try {
        await api(`/api/${action}-instances`, {
          node_name: instance.node_name,
          instance_ids: [instance.id],
          ...extra,
        });
        const label = instance.name || instance.id;
        toast.success(action === "terminate" ? `Instance ${label} terminated` : `${actionProgressLabel[action]} ${label}…`);
        onChanged?.(instance, action);
      } catch (err) {
        toast.error(errorMessage(err, `Failed to ${action} instance`));
        throw err;
      } finally {
        setBusy((prev) => {
          const next = { ...prev };
          delete next[instance.id];
          return next;
        });
      }
    },
    [toast, onChanged]
  );

  const request = useCallback(
    (instance: Instance, action: InstanceAction) => {
      if (action === "start") {
        execute(instance, action).catch(() => undefined);
        return;
      }
      setDeleteVolumes(false);
      setPending({ instance, action });
    },
    [execute]
  );

  const label = pending ? pending.instance.name || pending.instance.id : "";
  const dialogCopy: Record<Exclude<InstanceAction, "start">, { title: string; message: string; confirm: string }> = {
    stop: {
      title: "Stop instance?",
      message: `${label} will be shut down. Unsaved data inside the guest may be lost.`,
      confirm: "Stop",
    },
    reboot: {
      title: "Reboot instance?",
      message: `${label} will be restarted. Active connections will be interrupted.`,
      confirm: "Reboot",
    },
    terminate: {
      title: "Terminate instance?",
      message: `${label} will be permanently deleted. This cannot be undone.`,
      confirm: "Terminate",
    },
  };
  const copy = pending && pending.action !== "start" ? dialogCopy[pending.action] : null;

  const dialog = (
    <ConfirmDialog
      isOpen={Boolean(pending && copy)}
      onClose={() => setPending(null)}
      onConfirm={() =>
        pending
          ? execute(pending.instance, pending.action, pending.action === "terminate" ? { delete_volumes: deleteVolumes } : {})
          : undefined
      }
      title={copy?.title || ""}
      message={copy?.message || ""}
      confirmText={copy?.confirm}
      variant={pending?.action === "terminate" ? "danger" : "warning"}
    >
      {pending?.action === "terminate" && (
        <Checkbox
          checked={deleteVolumes}
          onChange={setDeleteVolumes}
          label="Also delete attached disks"
          description="Removes the instance's volumes from the storage pool."
        />
      )}
    </ConfirmDialog>
  );

  return { request, busy, dialog };
}
