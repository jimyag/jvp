import { ChevronRight, RefreshCw } from "lucide-react";
import type { ReactNode } from "react";
import { Link } from "react-router-dom";

export interface Crumb {
  label: string;
  to?: string;
}

interface PageHeaderProps {
  title: ReactNode;
  description?: ReactNode;
  breadcrumbs?: Crumb[];
  meta?: ReactNode;
  actions?: ReactNode;
  icon?: ReactNode;
  onRefresh?: () => void;
  refreshing?: boolean;
}

export default function PageHeader({
  title,
  description,
  breadcrumbs,
  meta,
  actions,
  icon,
  onRefresh,
  refreshing,
}: PageHeaderProps) {
  return (
    <div className="mb-6">
      {breadcrumbs && breadcrumbs.length > 0 && (
        <nav className="mb-3 flex items-center gap-1 text-xs text-fg-subtle" aria-label="Breadcrumb">
          {breadcrumbs.map((crumb, index) => (
            <span key={`${crumb.label}-${index}`} className="flex min-w-0 items-center gap-1">
              {index > 0 && <ChevronRight size={12} className="flex-shrink-0" />}
              {crumb.to ? (
                <Link to={crumb.to} className="truncate hover:text-fg">
                  {crumb.label}
                </Link>
              ) : (
                <span className="truncate text-fg-muted">{crumb.label}</span>
              )}
            </span>
          ))}
        </nav>
      )}
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div className="flex min-w-0 items-start gap-3">
          {icon && (
            <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg border border-line bg-surface text-fg-muted shadow-card">
              {icon}
            </div>
          )}
          <div className="min-w-0">
            <h1 className="truncate text-xl font-semibold tracking-tight text-fg">{title}</h1>
            {description && <p className="mt-1 text-sm text-fg-muted">{description}</p>}
            {meta && <div className="mt-2 flex flex-wrap items-center gap-2">{meta}</div>}
          </div>
        </div>
        {(actions || onRefresh) && (
          <div className="flex flex-shrink-0 flex-wrap items-center gap-2">
            {onRefresh && (
              <button
                type="button"
                onClick={onRefresh}
                disabled={refreshing}
                className="btn-secondary w-9 px-0"
                title="Refresh"
                aria-label="Refresh"
              >
                <RefreshCw size={15} className={refreshing ? "animate-spin" : ""} />
              </button>
            )}
            {actions}
          </div>
        )}
      </div>
    </div>
  );
}
