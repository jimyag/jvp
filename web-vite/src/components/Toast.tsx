import { useEffect } from "react";
import { CheckCircle, XCircle, AlertCircle, Info, X } from "lucide-react";

export type ToastType = "success" | "error" | "warning" | "info";

interface ToastProps {
  type: ToastType;
  message: string;
  onClose: () => void;
  duration?: number;
}

export default function Toast({
  type,
  message,
  onClose,
  duration = 5000,
}: ToastProps) {
  useEffect(() => {
    if (duration > 0) {
      const timer = setTimeout(() => {
        onClose();
      }, duration);
      return () => clearTimeout(timer);
    }
  }, [duration, onClose]);

  const getTypeStyles = () => {
    switch (type) {
      case "success":
        return {
          bg: "bg-green-50 border-green-200",
          icon: <CheckCircle className="text-green-600" size={20} />,
          text: "text-green-800",
        };
      case "error":
        return {
          bg: "bg-red-50 border-red-200",
          icon: <XCircle className="text-red-600" size={20} />,
          text: "text-red-800",
        };
      case "warning":
        return {
          bg: "bg-orange-50 border-orange-200",
          icon: <AlertCircle className="text-orange-600" size={20} />,
          text: "text-orange-800",
        };
      case "info":
        return {
          bg: "bg-blue-50 border-blue-200",
          icon: <Info className="text-blue-600" size={20} />,
          text: "text-blue-800",
        };
    }
  };

  const styles = getTypeStyles();

  return (
    <div
      className={`flex items-start gap-3 p-4 rounded-lg border ${styles.bg} shadow-lg animate-slideIn`}
    >
      <div className="flex-shrink-0">{styles.icon}</div>
      <p className={`flex-1 text-sm font-medium ${styles.text}`}>{message}</p>
      <button
        onClick={onClose}
        className={`flex-shrink-0 ${styles.text} hover:opacity-70 transition-opacity`}
      >
        <X size={16} />
      </button>
    </div>
  );
}

