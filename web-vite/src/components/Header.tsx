import { RefreshCw } from "lucide-react";

interface HeaderProps {
  title: React.ReactNode;
  description?: string;
  action?: React.ReactNode;
  onRefresh?: () => void | Promise<void>;
  refreshLoading?: boolean;
}

export default function Header({ title, description, action, onRefresh, refreshLoading = false }: HeaderProps) {
  return (
    <div className="mb-8">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-3xl font-bold text-primary mb-2">{title}</h1>
          {description && (
            <p className="text-gray-600">{description}</p>
          )}
        </div>
        <div className="flex gap-3">
          {onRefresh && (
            <button
              onClick={onRefresh}
              disabled={refreshLoading}
              className="btn-secondary disabled:opacity-50 disabled:cursor-not-allowed"
              title="Refresh"
            >
              <RefreshCw size={16} className={refreshLoading ? "animate-spin" : ""} />
            </button>
          )}
          {action}
        </div>
      </div>
    </div>
  );
}
