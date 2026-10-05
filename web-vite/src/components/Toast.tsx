import { useEffect, useRef } from "react";
import { AlertTriangle, CheckCircle2, Info, X, XCircle } from "lucide-react";

export type ToastType = "success" | "error" | "warning" | "info";

interface ToastProps {
  type: ToastType;
  message: string;
  onClose: () => void;
  duration?: number;
}

const styles = {
  success: { icon: CheckCircle2, color: "text-success" },
  error: { icon: XCircle, color: "text-danger" },
  warning: { icon: AlertTriangle, color: "text-warning" },
  info: { icon: Info, color: "text-info" },
};

export default function Toast({ type, message, onClose, duration }: ToastProps) {
  // 错误信息停留更久，方便阅读
  const timeout = duration ?? (type === "error" ? 8000 : 4000);

  const onCloseRef = useRef(onClose);
  useEffect(() => {
    onCloseRef.current = onClose;
  });

  useEffect(() => {
    if (timeout <= 0) return;
    const timer = setTimeout(() => onCloseRef.current(), timeout);
    return () => clearTimeout(timer);
  }, [timeout]);

  const { icon: Icon, color } = styles[type];

  return (
    <div
      role={type === "error" ? "alert" : "status"}
      className="pointer-events-auto flex w-full items-start gap-3 rounded-lg border border-line bg-surface px-3.5 py-3 shadow-pop animate-slide-up"
    >
      <Icon size={18} className={`mt-px flex-shrink-0 ${color}`} />
      <p className="min-w-0 flex-1 break-words text-sm text-fg">{message}</p>
      <button
        type="button"
        onClick={onClose}
        className="-mr-1 flex-shrink-0 rounded p-0.5 text-fg-subtle transition-colors hover:bg-subtle hover:text-fg"
        aria-label="Dismiss"
      >
        <X size={14} />
      </button>
    </div>
  );
}
