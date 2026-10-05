import { X } from "lucide-react";
import { useEffect, useId, useRef } from "react";
import type { ReactNode } from "react";
import { createPortal } from "react-dom";

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: ReactNode;
  description?: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
  size?: "sm" | "md" | "lg" | "xl";
  /** 防止误操作：处理中时禁止通过遮罩 / Esc 关闭 */
  dismissible?: boolean;
  bodyClassName?: string;
}

const sizeClasses = {
  sm: "max-w-md",
  md: "max-w-lg",
  lg: "max-w-2xl",
  xl: "max-w-5xl",
};

// 打开中的弹窗栈：Esc 只关闭最上层的弹窗
const modalStack: symbol[] = [];

// 记录最近两次获得焦点的元素：弹窗内 autoFocus 会先于 effect 执行，
// 此时需要回退到更早的那个元素作为关闭后恢复焦点的目标
let previousFocus: HTMLElement | null = null;
let currentFocus: HTMLElement | null = null;
if (typeof document !== "undefined") {
  document.addEventListener("focusin", (e) => {
    previousFocus = currentFocus;
    currentFocus = e.target as HTMLElement;
  });
}

const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]):not([type="hidden"]), select:not([disabled]), textarea:not([disabled]), iframe, [tabindex]:not([tabindex="-1"])';

function focusableIn(root: HTMLElement): HTMLElement[] {
  return Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE)).filter((el) => el.offsetParent !== null || el === document.activeElement);
}

export default function Modal({
  isOpen,
  onClose,
  title,
  description,
  children,
  footer,
  size = "md",
  dismissible = true,
  bodyClassName = "px-6 py-5",
}: ModalProps) {
  const titleId = useId();
  const panelRef = useRef<HTMLDivElement>(null);
  const onCloseRef = useRef(onClose);
  const dismissibleRef = useRef(dismissible);

  useEffect(() => {
    onCloseRef.current = onClose;
    dismissibleRef.current = dismissible;
  });

  useEffect(() => {
    if (!isOpen) return;
    const id = Symbol("modal");
    modalStack.push(id);
    document.body.style.overflow = "hidden";

    const panel = panelRef.current;
    const active = document.activeElement as HTMLElement | null;
    const restoreTarget = panel && active && panel.contains(active) ? previousFocus : active;
    if (panel && !panel.contains(document.activeElement)) {
      (focusableIn(panel)[0] || panel).focus();
    }

    const handleKey = (e: KeyboardEvent) => {
      if (modalStack[modalStack.length - 1] !== id) return;
      if (e.key === "Escape" && dismissibleRef.current) {
        e.stopPropagation();
        onCloseRef.current();
        return;
      }
      // 焦点限制在弹窗内
      if (e.key === "Tab" && panel) {
        const items = focusableIn(panel);
        if (items.length === 0) {
          e.preventDefault();
          panel.focus();
          return;
        }
        const first = items[0];
        const last = items[items.length - 1];
        const current = document.activeElement;
        if (e.shiftKey && (current === first || !panel.contains(current))) {
          e.preventDefault();
          last.focus();
        } else if (!e.shiftKey && (current === last || !panel.contains(current))) {
          e.preventDefault();
          first.focus();
        }
      }
    };
    window.addEventListener("keydown", handleKey);

    return () => {
      window.removeEventListener("keydown", handleKey);
      modalStack.splice(modalStack.indexOf(id), 1);
      if (modalStack.length === 0) {
        document.body.style.overflow = "";
      }
      if (restoreTarget?.isConnected) restoreTarget.focus();
    };
  }, [isOpen]);

  if (!isOpen) return null;

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-end justify-center sm:items-center sm:p-6" role="dialog" aria-modal="true" aria-labelledby={titleId}>
      <div
        className="fixed inset-0 animate-fade-in bg-black/40 backdrop-blur-[2px]"
        onClick={() => dismissible && onClose()}
      />
      <div
        ref={panelRef}
        tabIndex={-1}
        className={`relative flex max-h-[92vh] outline-none w-full flex-col overflow-hidden rounded-t-xl border border-line bg-surface shadow-pop animate-pop-in sm:rounded-xl ${sizeClasses[size]}`}
      >
        <div className="flex items-start justify-between gap-4 border-b border-line px-6 py-4">
          <div className="min-w-0">
            <h2 id={titleId} className="text-base font-semibold text-fg">{title}</h2>
            {description && <p className="mt-0.5 text-sm text-fg-muted">{description}</p>}
          </div>
          <button
            type="button"
            onClick={onClose}
            disabled={!dismissible}
            className="btn-icon -mr-2 -mt-1"
            aria-label="Close"
          >
            <X size={18} />
          </button>
        </div>

        <div className={`min-h-0 flex-1 overflow-y-auto ${bodyClassName}`}>{children}</div>

        {footer && (
          <div className="flex items-center justify-end gap-2 border-t border-line bg-canvas/60 px-6 py-3.5">
            {footer}
          </div>
        )}
      </div>
    </div>,
    document.body
  );
}
