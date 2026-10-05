import { AlertTriangle, Info } from "lucide-react";
import { useState } from "react";
import type { ReactNode } from "react";
import Modal from "./Modal";
import { Spinner } from "./ui";

interface ConfirmDialogProps {
  isOpen: boolean;
  onClose: () => void;
  /** 支持异步操作：执行期间按钮显示加载状态，成功后自动关闭；抛出异常时保持打开 */
  onConfirm: () => void | Promise<void>;
  title: string;
  message: ReactNode;
  confirmText?: string;
  cancelText?: string;
  variant?: "danger" | "warning" | "info";
  children?: ReactNode;
  confirmDisabled?: boolean;
}

export default function ConfirmDialog({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmText = "Confirm",
  cancelText = "Cancel",
  variant = "danger",
  children,
  confirmDisabled,
}: ConfirmDialogProps) {
  const [busy, setBusy] = useState(false);

  const handleConfirm = async () => {
    setBusy(true);
    try {
      await onConfirm();
      onClose();
    } catch {
      // 由调用方负责提示错误，这里保持对话框打开
    } finally {
      setBusy(false);
    }
  };

  const iconBox =
    variant === "danger"
      ? "bg-danger-soft text-danger"
      : variant === "warning"
        ? "bg-warning-soft text-warning"
        : "bg-accent-soft text-accent";
  const confirmClass = variant === "danger" ? "btn-danger" : "btn-primary";

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={title}
      size="sm"
      dismissible={!busy}
      footer={
        <>
          <button type="button" className="btn-secondary" onClick={onClose} disabled={busy}>
            {cancelText}
          </button>
          <button
            type="button"
            className={confirmClass}
            onClick={handleConfirm}
            disabled={busy || confirmDisabled}
            autoFocus
          >
            {busy && <Spinner size={14} className="text-current" />}
            {confirmText}
          </button>
        </>
      }
    >
      <div className="flex gap-3.5">
        <div className={`flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-full ${iconBox}`}>
          {variant === "info" ? <Info size={18} /> : <AlertTriangle size={18} />}
        </div>
        <div className="min-w-0 flex-1 pt-1.5 text-sm text-fg-muted">
          <div>{message}</div>
          {children && <div className="mt-4">{children}</div>}
        </div>
      </div>
    </Modal>
  );
}
