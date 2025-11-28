"use client";

import { RefreshCw } from "lucide-react";

interface HeaderProps {
  title: React.ReactNode;
  description?: string;
  action?: React.ReactNode;
  onRefresh?: () => void;
}

export default function Header({ title, description, action, onRefresh }: HeaderProps) {
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
              className="btn-secondary"
              title="Refresh"
            >
              <RefreshCw size={16} />
            </button>
          )}
          {action}
        </div>
      </div>
    </div>
  );
}
