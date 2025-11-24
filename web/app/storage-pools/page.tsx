"use client";

import { useEffect, useState } from "react";
import { Database, HardDrive, ChevronDown, ChevronRight, RefreshCw } from "lucide-react";
import { apiPost, handleApiError } from "@/lib/api";
import DashboardLayout from "@/components/DashboardLayout";

interface Volume {
  id: string;
  name: string;
  pool: string;
  path: string;
  capacity_b: number;
  allocation_b: number;
  format: string;
  volumeType: string; // disk, template, iso
}

interface StoragePool {
  name: string;
  uuid?: string;
  state: string;
  capacity_b: number;
  allocation_b: number;
  available_b: number;
  path: string;
  type?: string;
  volumeCount?: number;
  volumes?: Volume[];
}

interface ListStoragePoolsResponse {
  pools: StoragePool[];
}

export default function StoragePoolsPage() {
  const [pools, setPools] = useState<StoragePool[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedPools, setExpandedPools] = useState<Set<string>>(new Set());

  const fetchPools = async () => {
    try {
      setLoading(true);
      setError(null);
      // 直接获取包含卷列表的完整数据，参考 Flint 的实现
      const response = await apiPost<ListStoragePoolsResponse>(
        "/api/storage/pools/list",
        { includeVolumes: true }
      );
      setPools(response.pools || []);
    } catch (err) {
      const errorMessage = await handleApiError(err);
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };


  const togglePoolExpansion = (poolName: string) => {
    setExpandedPools((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(poolName)) {
        newSet.delete(poolName);
      } else {
        newSet.add(poolName);
        // 数据已经在首次加载时获取，不需要再次请求
      }
      return newSet;
    });
  };

  useEffect(() => {
    fetchPools();
  }, []);

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
  };

  const getStateColor = (state: string): string => {
    switch (state.toLowerCase()) {
      case "active":
      case "running":
        return "text-green-600 bg-green-100";
      case "inactive":
        return "text-gray-600 bg-gray-100";
      case "building":
        return "text-yellow-600 bg-yellow-100";
      case "degraded":
        return "text-orange-600 bg-orange-100";
      case "inaccessible":
        return "text-red-600 bg-red-100";
      default:
        return "text-gray-600 bg-gray-100";
    }
  };

  const getVolumeTypeColor = (volumeType: string): string => {
    switch (volumeType.toLowerCase()) {
      case "template":
        return "text-blue-600 bg-blue-100";
      case "iso":
        return "text-orange-600 bg-orange-100";
      case "disk":
      default:
        return "text-purple-600 bg-purple-100";
    }
  };

  const getVolumeTypeLabel = (volumeType: string): string => {
    switch (volumeType.toLowerCase()) {
      case "template":
        return "Template";
      case "iso":
        return "ISO";
      case "disk":
      default:
        return "Disk";
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <RefreshCw className="w-8 h-8 animate-spin text-primary mx-auto mb-2" />
          <p className="text-gray-600">Loading storage pools...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4">
        <p className="text-red-800">Error: {error}</p>
        <button
          onClick={fetchPools}
          className="mt-2 px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700"
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <DashboardLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Storage Pools</h1>
            <p className="text-gray-600 mt-1">
              Manage libvirt storage pools and volumes
            </p>
          </div>
          <button
            onClick={fetchPools}
            className="flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90"
          >
            <RefreshCw className="w-4 h-4" />
            Refresh
          </button>
        </div>

      {/* Storage Pools List */}
      {pools.length === 0 ? (
        <div className="text-center py-12 bg-gray-50 rounded-lg">
          <Database className="w-12 h-12 text-gray-400 mx-auto mb-4" />
          <p className="text-gray-600">No storage pools found</p>
        </div>
      ) : (
        <div className="space-y-4">
          {pools.map((pool) => {
            const isExpanded = expandedPools.has(pool.name);
            const usagePercent = pool.capacity_b > 0
              ? ((pool.allocation_b / pool.capacity_b) * 100).toFixed(1)
              : "0";

            return (
              <div
                key={pool.name}
                className="bg-white border border-gray-200 rounded-lg overflow-hidden"
              >
                {/* Pool Header */}
                <div
                  className="p-4 cursor-pointer hover:bg-gray-50 transition-colors"
                  onClick={() => togglePoolExpansion(pool.name)}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3 flex-1">
                      <button className="text-gray-600">
                        {isExpanded ? (
                          <ChevronDown className="w-5 h-5" />
                        ) : (
                          <ChevronRight className="w-5 h-5" />
                        )}
                      </button>
                      <Database className="w-8 h-8 text-primary" />
                      <div className="flex-1">
                        <div className="flex items-center gap-3">
                          <h3 className="text-lg font-semibold text-gray-900">
                            {pool.name}
                          </h3>
                          <span
                            className={`px-2 py-1 text-xs font-medium rounded-full ${getStateColor(
                              pool.state
                            )}`}
                          >
                            {pool.state}
                          </span>
                          {pool.type && (
                            <span className="px-2 py-1 text-xs font-medium text-blue-600 bg-blue-100 rounded-full">
                              {pool.type}
                            </span>
                          )}
                        </div>
                        <p className="text-sm text-gray-600 mt-1">{pool.path}</p>
                      </div>
                    </div>
                    <div className="text-right">
                      <div className="text-sm font-medium text-gray-900">
                        {formatBytes(pool.allocation_b)} / {formatBytes(pool.capacity_b)}
                      </div>
                      <div className="text-xs text-gray-600">
                        {usagePercent}% used · {formatBytes(pool.available_b)} free
                      </div>
                      <div className="text-xs text-gray-500 mt-0.5">
                        {pool.volumeCount || 0} volumes
                      </div>
                    </div>
                  </div>

                  {/* Progress Bar */}
                  <div className="mt-3 w-full bg-gray-200 rounded-full h-2">
                    <div
                      className="bg-primary h-2 rounded-full transition-all"
                      style={{ width: `${usagePercent}%` }}
                    />
                  </div>
                </div>

                {/* Volumes List (Expanded) */}
                {isExpanded && (
                  <div className="border-t border-gray-200 bg-gray-50">
                    {pool.volumes && pool.volumes.length > 0 ? (
                      <div className="divide-y divide-gray-200">
                        {pool.volumes.map((volume) => (
                          <div
                            key={volume.id}
                            className="p-4 hover:bg-gray-100 transition-colors"
                          >
                            <div className="flex items-center justify-between">
                              <div className="flex items-center gap-3 flex-1">
                                <HardDrive className="w-5 h-5 text-gray-400" />
                                <div className="flex-1 min-w-0">
                                  <div className="flex items-center gap-2">
                                    <p className="font-medium text-gray-900 truncate">
                                      {volume.name}
                                    </p>
                                    {volume.volumeType && (
                                      <span className={`px-2 py-0.5 text-xs font-medium rounded flex-shrink-0 ${getVolumeTypeColor(volume.volumeType)}`}>
                                        {getVolumeTypeLabel(volume.volumeType)}
                                      </span>
                                    )}
                                    <span className="px-2 py-0.5 text-xs font-medium text-gray-600 bg-gray-100 rounded flex-shrink-0">
                                      {volume.format}
                                    </span>
                                  </div>
                                  <p className="text-xs text-gray-600 mt-0.5 truncate" title={volume.path}>
                                    {volume.path}
                                  </p>
                                </div>
                              </div>
                              <div className="text-right ml-4 flex-shrink-0">
                                <p className="text-sm font-medium text-gray-900">
                                  {formatBytes(volume.capacity_b)}
                                </p>
                                <p className="text-xs text-gray-600">
                                  {formatBytes(volume.allocation_b)} used
                                  {volume.capacity_b > 0 && (
                                    <span className="ml-1">
                                      ({((volume.allocation_b / volume.capacity_b) * 100).toFixed(0)}%)
                                    </span>
                                  )}
                                </p>
                              </div>
                            </div>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <div className="p-8 text-center">
                        <HardDrive className="w-8 h-8 text-gray-400 mx-auto mb-2" />
                        <p className="text-sm text-gray-600">No volumes in this pool</p>
                      </div>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
      </div>
    </DashboardLayout>
  );
}
