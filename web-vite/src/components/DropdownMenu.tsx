import { MoreHorizontal } from "lucide-react";
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import type { ReactNode } from "react";
import { createPortal } from "react-dom";

export interface MenuItem {
  label: string;
  icon?: ReactNode;
  onClick?: () => void;
  danger?: boolean;
  disabled?: boolean;
  divider?: boolean;
}

interface DropdownMenuProps {
  items: MenuItem[];
  trigger?: ReactNode;
  label?: string;
  triggerClassName?: string;
}

const MENU_WIDTH = 200;

export default function DropdownMenu({ items, trigger, label = "More actions", triggerClassName = "btn-icon" }: DropdownMenuProps) {
  const [open, setOpen] = useState(false);
  const [position, setPosition] = useState<{ top: number; left: number } | null>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  const close = useCallback(() => setOpen(false), []);

  useLayoutEffect(() => {
    if (!open || !triggerRef.current) return;
    const rect = triggerRef.current.getBoundingClientRect();
    const menuHeight = menuRef.current?.offsetHeight ?? 0;
    const spaceBelow = window.innerHeight - rect.bottom;
    const top = spaceBelow < menuHeight + 12 ? rect.top - menuHeight - 4 : rect.bottom + 4;
    const left = Math.max(8, Math.min(rect.right - MENU_WIDTH, window.innerWidth - MENU_WIDTH - 8));
    setPosition({ top, left });
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const handlePointer = (e: MouseEvent) => {
      const target = e.target as Node;
      if (menuRef.current?.contains(target) || triggerRef.current?.contains(target)) return;
      close();
    };
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") close();
    };
    document.addEventListener("mousedown", handlePointer);
    document.addEventListener("keydown", handleKey);
    window.addEventListener("scroll", close, true);
    window.addEventListener("resize", close);
    return () => {
      document.removeEventListener("mousedown", handlePointer);
      document.removeEventListener("keydown", handleKey);
      window.removeEventListener("scroll", close, true);
      window.removeEventListener("resize", close);
    };
  }, [open, close]);

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        aria-label={label}
        aria-haspopup="menu"
        aria-expanded={open}
        title={label}
        className={triggerClassName}
        onClick={(e) => {
          e.stopPropagation();
          setPosition(null);
          setOpen((v) => !v);
        }}
      >
        {trigger ?? <MoreHorizontal size={16} />}
      </button>
      {open &&
        createPortal(
          <div
            ref={menuRef}
            role="menu"
            onClick={(e) => e.stopPropagation()}
            style={{ top: position?.top ?? -9999, left: position?.left ?? -9999, width: MENU_WIDTH }}
            className="fixed z-[60] animate-pop-in rounded-lg border border-line bg-surface p-1 shadow-pop"
          >
            {items.map((item, index) =>
              item.divider ? (
                <div key={`divider-${index}`} className="my-1 h-px bg-line" />
              ) : (
                <button
                  key={item.label}
                  type="button"
                  role="menuitem"
                  disabled={item.disabled}
                  onClick={() => {
                    close();
                    item.onClick?.();
                  }}
                  className={`flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left text-sm transition-colors disabled:cursor-not-allowed disabled:opacity-40 ${
                    item.danger ? "text-danger hover:bg-danger-soft" : "text-fg hover:bg-subtle"
                  }`}
                >
                  {item.icon && <span className={item.danger ? "" : "text-fg-subtle"}>{item.icon}</span>}
                  {item.label}
                </button>
              )
            )}
          </div>,
          document.body
        )}
    </>
  );
}
