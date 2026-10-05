import { useState } from "react";
import type { ReactNode } from "react";
import { AlertTriangle, Check, CheckCircle2, Copy, Info, Loader2, XCircle } from "lucide-react";

type Tone = "neutral" | "accent" | "success" | "warning" | "danger" | "info";

const toneClasses: Record<Tone, string> = {
  neutral: "bg-subtle text-fg-muted ring-line",
  accent: "bg-accent-soft text-accent ring-accent/20",
  success: "bg-success-soft text-success ring-success/20",
  warning: "bg-warning-soft text-warning ring-warning/20",
  danger: "bg-danger-soft text-danger ring-danger/20",
  info: "bg-info-soft text-info ring-info/20",
};

const dotClasses: Record<Tone, string> = {
  neutral: "bg-fg-subtle",
  accent: "bg-accent",
  success: "bg-success",
  warning: "bg-warning",
  danger: "bg-danger",
  info: "bg-info",
};

export function Badge({
  tone = "neutral",
  dot = false,
  pulse = false,
  className = "",
  children,
}: {
  tone?: Tone;
  dot?: boolean;
  pulse?: boolean;
  className?: string;
  children: ReactNode;
}) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 whitespace-nowrap rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${toneClasses[tone]} ${className}`}
    >
      {dot && (
        <span className="relative flex h-1.5 w-1.5">
          {pulse && <span className={`absolute inline-flex h-full w-full animate-ping rounded-full opacity-60 ${dotClasses[tone]}`} />}
          <span className={`relative inline-flex h-1.5 w-1.5 rounded-full ${dotClasses[tone]}`} />
        </span>
      )}
      {children}
    </span>
  );
}

const statusTones: Record<string, Tone> = {
  running: "success",
  active: "success",
  online: "success",
  up: "success",
  completed: "success",
  stopped: "neutral",
  shutoff: "neutral",
  inactive: "neutral",
  down: "neutral",
  pending: "warning",
  building: "warning",
  paused: "warning",
  maintenance: "warning",
  suspended: "warning",
  degraded: "warning",
  error: "danger",
  failed: "danger",
  crashed: "danger",
  offline: "danger",
  inaccessible: "danger",
};

export function StatusBadge({ status, label }: { status?: string; label?: string }) {
  const key = (status || "unknown").toLowerCase();
  const tone = statusTones[key] ?? "neutral";
  const text = label || status || "unknown";
  return (
    <Badge tone={tone} dot pulse={key === "pending" || key === "building"} className="capitalize">
      {text.toLowerCase()}
    </Badge>
  );
}

export function Spinner({ size = 16, className = "" }: { size?: number; className?: string }) {
  return <Loader2 size={size} className={`animate-spin text-fg-subtle ${className}`} />;
}

export function LoadingState({ label = "Loading…", className = "" }: { label?: string; className?: string }) {
  return (
    <div className={`flex flex-col items-center justify-center gap-3 py-16 text-fg-subtle ${className}`}>
      <Spinner size={22} />
      <p className="text-sm">{label}</p>
    </div>
  );
}

export function EmptyState({
  icon,
  title,
  description,
  action,
  className = "",
}: {
  icon?: ReactNode;
  title: string;
  description?: ReactNode;
  action?: ReactNode;
  className?: string;
}) {
  return (
    <div className={`flex flex-col items-center justify-center px-6 py-14 text-center ${className}`}>
      {icon && (
        <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-full bg-subtle text-fg-subtle">
          {icon}
        </div>
      )}
      <h3 className="text-sm font-semibold text-fg">{title}</h3>
      {description && <p className="mt-1 max-w-sm text-sm text-fg-muted">{description}</p>}
      {action && <div className="mt-5">{action}</div>}
    </div>
  );
}

const alertStyles = {
  info: { box: "border-info/20 bg-info-soft text-info", icon: Info },
  success: { box: "border-success/20 bg-success-soft text-success", icon: CheckCircle2 },
  warning: { box: "border-warning/25 bg-warning-soft text-warning", icon: AlertTriangle },
  danger: { box: "border-danger/20 bg-danger-soft text-danger", icon: XCircle },
};

export function Alert({
  tone = "info",
  title,
  children,
  action,
  className = "",
}: {
  tone?: keyof typeof alertStyles;
  title?: ReactNode;
  children?: ReactNode;
  action?: ReactNode;
  className?: string;
}) {
  const style = alertStyles[tone];
  const Icon = style.icon;
  return (
    <div className={`flex gap-3 rounded-lg border px-3.5 py-3 text-sm ${style.box} ${className}`}>
      <Icon size={16} className="mt-0.5 flex-shrink-0" />
      <div className="min-w-0 flex-1">
        {title && <p className="font-medium">{title}</p>}
        {children && <div className={`text-fg-muted ${title ? "mt-0.5" : ""}`}>{children}</div>}
      </div>
      {action && <div className="flex-shrink-0">{action}</div>}
    </div>
  );
}

export function Card({
  title,
  description,
  actions,
  children,
  className = "",
  bodyClassName = "p-5",
}: {
  title?: ReactNode;
  description?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
  bodyClassName?: string;
}) {
  return (
    <section className={`card ${className}`}>
      {(title || actions) && (
        <header className="flex items-center justify-between gap-4 border-b border-line px-5 py-3.5">
          <div className="min-w-0">
            {title && <h2 className="text-sm font-semibold text-fg">{title}</h2>}
            {description && <p className="mt-0.5 text-xs text-fg-muted">{description}</p>}
          </div>
          {actions && <div className="flex flex-shrink-0 items-center gap-2">{actions}</div>}
        </header>
      )}
      <div className={bodyClassName}>{children}</div>
    </section>
  );
}

export function StatCard({
  label,
  value,
  hint,
  icon,
  children,
}: {
  label: string;
  value: ReactNode;
  hint?: ReactNode;
  icon?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <div className="card p-4">
      <div className="flex items-center justify-between text-fg-muted">
        <span className="text-xs font-medium">{label}</span>
        {icon && <span className="text-fg-subtle">{icon}</span>}
      </div>
      <div className="mt-2 text-2xl font-semibold tracking-tight text-fg">{value}</div>
      {hint && <div className="mt-1 text-xs text-fg-muted">{hint}</div>}
      {children && <div className="mt-3">{children}</div>}
    </div>
  );
}

export function Field({
  label,
  hint,
  required,
  htmlFor,
  className = "",
  children,
}: {
  label?: ReactNode;
  hint?: ReactNode;
  required?: boolean;
  htmlFor?: string;
  className?: string;
  children: ReactNode;
}) {
  return (
    <div className={className}>
      {label && (
        <label htmlFor={htmlFor} className="label">
          {label}
          {required && <span className="ml-0.5 text-danger">*</span>}
        </label>
      )}
      {children}
      {hint && <div className="hint">{hint}</div>}
    </div>
  );
}

export interface SegmentOption<T extends string> {
  value: T;
  label: ReactNode;
  icon?: ReactNode;
  disabled?: boolean;
}

export function SegmentedControl<T extends string>({
  value,
  onChange,
  options,
  size = "md",
  className = "",
}: {
  value: T;
  onChange: (value: T) => void;
  options: SegmentOption<T>[];
  size?: "sm" | "md";
  className?: string;
}) {
  return (
    <div className={`inline-flex rounded-md bg-subtle p-0.5 ring-1 ring-inset ring-line ${className}`} role="radiogroup">
      {options.map((option) => {
        const active = option.value === value;
        return (
          <button
            key={option.value}
            type="button"
            role="radio"
            aria-checked={active}
            disabled={option.disabled}
            onClick={() => onChange(option.value)}
            className={`inline-flex items-center gap-1.5 rounded-[5px] font-medium transition-all disabled:cursor-not-allowed disabled:opacity-40 ${
              size === "sm" ? "h-6 px-2.5 text-xs" : "h-8 px-3 text-sm"
            } ${active ? "bg-surface text-fg shadow-card ring-1 ring-line" : "text-fg-muted hover:text-fg"}`}
          >
            {option.icon}
            {option.label}
          </button>
        );
      })}
    </div>
  );
}

export interface TabItem<T extends string> {
  id: T;
  label: ReactNode;
  count?: number;
  icon?: ReactNode;
}

export function Tabs<T extends string>({
  tabs,
  value,
  onChange,
  className = "",
}: {
  tabs: TabItem<T>[];
  value: T;
  onChange: (id: T) => void;
  className?: string;
}) {
  return (
    <div className={`border-b border-line ${className}`}>
      <nav className="-mb-px flex gap-5 overflow-x-auto" role="tablist">
        {tabs.map((tab) => {
          const active = tab.id === value;
          return (
            <button
              key={tab.id}
              type="button"
              role="tab"
              aria-selected={active}
              onClick={() => onChange(tab.id)}
              className={`flex items-center gap-1.5 whitespace-nowrap border-b-2 pb-2.5 pt-1 text-sm font-medium transition-colors ${
                active ? "border-accent text-fg" : "border-transparent text-fg-muted hover:border-line-strong hover:text-fg"
              }`}
            >
              {tab.icon}
              {tab.label}
              {tab.count !== undefined && (
                <span className={`rounded-full px-1.5 text-[11px] tabular-nums ${active ? "bg-accent-soft text-accent" : "bg-subtle text-fg-subtle"}`}>
                  {tab.count}
                </span>
              )}
            </button>
          );
        })}
      </nav>
    </div>
  );
}

export function Switch({
  checked,
  onChange,
  label,
  description,
  disabled,
}: {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label?: ReactNode;
  description?: ReactNode;
  disabled?: boolean;
}) {
  return (
    <label className={`flex items-start justify-between gap-4 ${disabled ? "cursor-not-allowed opacity-60" : "cursor-pointer"}`}>
      {(label || description) && (
        <span className="min-w-0">
          {label && <span className="block text-sm font-medium text-fg">{label}</span>}
          {description && <span className="mt-0.5 block text-xs text-fg-muted">{description}</span>}
        </span>
      )}
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        disabled={disabled}
        onClick={() => onChange(!checked)}
        className={`relative mt-0.5 inline-flex h-5 w-9 flex-shrink-0 items-center rounded-full transition-colors ${
          checked ? "bg-accent" : "bg-line-strong"
        }`}
      >
        <span
          className={`inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform ${
            checked ? "translate-x-[18px]" : "translate-x-0.5"
          }`}
        />
      </button>
    </label>
  );
}

export function Checkbox({
  checked,
  onChange,
  label,
  description,
  disabled,
}: {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label: ReactNode;
  description?: ReactNode;
  disabled?: boolean;
}) {
  return (
    <label className={`flex items-start gap-2.5 ${disabled ? "cursor-not-allowed opacity-60" : "cursor-pointer"}`}>
      <input
        type="checkbox"
        className="mt-0.5"
        checked={checked}
        disabled={disabled}
        onChange={(e) => onChange(e.target.checked)}
      />
      <span className="min-w-0">
        <span className="block text-sm text-fg">{label}</span>
        {description && <span className="mt-0.5 block text-xs text-fg-muted">{description}</span>}
      </span>
    </label>
  );
}

export function CopyButton({ text, label, className = "" }: { text: string; label?: string; className?: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(text);
      } else {
        const el = document.createElement("textarea");
        el.value = text;
        document.body.appendChild(el);
        el.select();
        document.execCommand("copy");
        document.body.removeChild(el);
      }
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // 复制失败时静默处理
    }
  };

  if (label) {
    return (
      <button type="button" onClick={handleCopy} className={`btn-secondary btn-sm ${className}`}>
        {copied ? <Check size={13} className="text-success" /> : <Copy size={13} />}
        {copied ? "Copied" : label}
      </button>
    );
  }

  return (
    <button
      type="button"
      onClick={handleCopy}
      title={copied ? "Copied" : "Copy"}
      className={`inline-flex h-6 w-6 flex-shrink-0 items-center justify-center rounded text-fg-subtle transition-colors hover:bg-subtle hover:text-fg ${className}`}
    >
      {copied ? <Check size={13} className="text-success" /> : <Copy size={13} />}
    </button>
  );
}

export function UsageBar({ value, className = "", size = "md" }: { value: number; className?: string; size?: "sm" | "md" }) {
  const clamped = Math.min(100, Math.max(0, value || 0));
  const color = clamped >= 90 ? "bg-danger" : clamped >= 75 ? "bg-warning" : "bg-accent";
  return (
    <div className={`w-full overflow-hidden rounded-full bg-subtle ${size === "sm" ? "h-1.5" : "h-2"} ${className}`}>
      <div className={`h-full rounded-full transition-all duration-500 ${color}`} style={{ width: `${clamped}%` }} />
    </div>
  );
}

export interface DescriptionItem {
  label: ReactNode;
  value: ReactNode;
  mono?: boolean;
  copy?: string;
  span?: boolean;
}

export function DescriptionList({ items, columns = 2 }: { items: DescriptionItem[]; columns?: 1 | 2 | 3 }) {
  const cols = columns === 1 ? "" : columns === 2 ? "sm:grid-cols-2" : "sm:grid-cols-2 lg:grid-cols-3";
  return (
    <dl className={`grid grid-cols-1 gap-x-8 gap-y-4 ${cols}`}>
      {items.map((item, index) => (
        <div key={index} className={`min-w-0 ${item.span ? "sm:col-span-full" : ""}`}>
          <dt className="text-xs font-medium text-fg-subtle">{item.label}</dt>
          <dd className={`mt-1 flex min-w-0 items-center gap-1 text-sm text-fg ${item.mono ? "font-mono text-[13px]" : ""}`}>
            <span className="min-w-0 break-all">{item.value === undefined || item.value === null || item.value === "" ? "—" : item.value}</span>
            {item.copy && <CopyButton text={item.copy} />}
          </dd>
        </div>
      ))}
    </dl>
  );
}
