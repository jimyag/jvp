import { useEffect, useMemo, useState } from "react";
import { useParams, useNavigate, useSearchParams, useLocation } from "react-router-dom";
import {
  Database,
  HardDrive,
  ArrowLeft,
  RefreshCw,
  Play,
  Square,
  Trash2,
  Maximize2,
} from "lucide-react";
import { apiPost } from "@/lib/api";
import { useToast } from "@/components/ToastContainer";
import Header from "@/components/Header";

interface Volume {
  volume_id: string;
  name: string;
  node_name: string;
  pool: string;
  path: string;
  capacity_b: number;
  size_gb: number;
  allocation_b: number;
  format: string;
}

interface StoragePool {
  name: string;
  uuid: string;
  state: string;
  type: string;
  capacity: number;
  allocation: number;
  available: number;
  path: string;
  volume_count: number;
}

interface DescribeStoragePoolResponse {
  pool: StoragePool;
}

interface ListVolumesResponse {
  volumes: Volume[];
}

export default function StoragePoolDetailPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const nodeName = searchParams.get("node") || "";
  const { poolName: encodedPoolName } = useParams<{ poolName: string }>();
  const poolName = useMemo(() => {
    const decoded = decodeURIComponent(encodedPoolName || "");
    if (decoded === "placeholder") {
      const segments = location.pathname.split("/").filter(Boolean);
      const urlPool = segments[1];
      return urlPool ? decodeURIComponent(urlPool) : decoded;
    }
    return decoded;
  }, [encodedPoolName, location.pathname]);

  const [pool, setPool] = useState<StoragePool | null>(null);
  const [volumes, setVolumes] = useState<Volume[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [loadingVolumes, setLoadingVolumes] = useState(false);

  // Delete pool modal
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deleteVolumesOnDelete, setDeleteVolumesOnDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);

  // Create volume modal
  const [showCreateVolumeModal, setShowCreateVolumeModal] = useState(false);
  const [createVolumeName, setCreateVolumeName] = useState("");
  const [createVolumeSize, setCreateVolumeSize] = useState(20);
  const [createVolumeFormat, setCreateVolumeFormat] = useState("qcow2");
  const [creatingVolume, setCreatingVolume] = useState(false);

  // Delete volume modal
  const [showDeleteVolumeModal, setShowDeleteVolumeModal] = useState(false);
  const [volumeToDelete, setVolumeToDelete] = useState<Volume | null>(null);
  const [deletingVolume, setDeletingVolume] = useState(false);

  // Resize volume modal
  const [showResizeVolumeModal, setShowResizeVolumeModal] = useState(false);
  const [volumeToResize, setVolumeToResize] = useState<Volume | null>(null);
  const [newVolumeSize, setNewVolumeSize] = useState(0);
  const [resizingVolume, setResizingVolume] = useState(false);

  const toast = useToast();

  useEffect(() => {
    if (poolName && nodeName) {
      fetchPoolDetail();
      fetchVolumes();
    }
  }, [poolName, nodeName]);

  const fetchPoolDetail = async () => {
    setRefreshing(true);
    try {
      const response = await apiPost<DescribeStoragePoolResponse>(
        "/api/describe-storage-pool",
        {
          node_name: nodeName,
          pool_name: poolName,
        }
      );
      setPool(response.pool);
    } catch (error: any) {
      console.error("Failed to fetch storage pool detail:", error);
      toast.error(error?.message || "Failed to fetch storage pool detail");
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  const fetchVolumes = async () => {
    setLoadingVolumes(true);
    try {
      const response = await apiPost<ListVolumesResponse>(
        "/api/list-volumes",
        {
          node_name: nodeName,
          pool_name: poolName,
        }
      );
      setVolumes(response.volumes || []);
    } catch (error: any) {
      console.error("Failed to fetch volumes:", error);
      toast.error(error?.message || "Failed to fetch volumes");
    } finally {
      setLoadingVolumes(false);
    }
  };

  const handleStartPool = async () => {
    try {
      await apiPost("/api/start-storage-pool", {
        node_name: nodeName,
        pool_name: poolName,
      });
      toast.success(`Storage pool ${poolName} started successfully`);
      await fetchPoolDetail();
    } catch (error: any) {
      console.error("Failed to start storage pool:", error);
      toast.error(error?.message || "Failed to start storage pool");
    }
  };

  const handleStopPool = async () => {
    if (!confirm(`Are you sure you want to stop storage pool "${poolName}"?`)) {
      return;
    }

    try {
      await apiPost("/api/stop-storage-pool", {
        node_name: nodeName,
        pool_name: poolName,
      });
      toast.success(`Storage pool ${poolName} stopped successfully`);
      await fetchPoolDetail();
    } catch (error: any) {
      console.error("Failed to stop storage pool:", error);
      toast.error(error?.message || "Failed to stop storage pool");
    }
  };

  const handleRefreshPool = async () => {
    try {
      await apiPost("/api/refresh-storage-pool", {
        node_name: nodeName,
        pool_name: poolName,
      });
      toast.success(`Storage pool ${poolName} refreshed successfully`);
      await fetchPoolDetail();
      await fetchVolumes();
    } catch (error: any) {
      console.error("Failed to refresh storage pool:", error);
      toast.error(error?.message || "Failed to refresh storage pool");
    }
  };

  const handleDeletePool = () => {
    setShowDeleteModal(true);
  };

  const confirmDeletePool = async () => {
    setDeleting(true);
    try {
      await apiPost("/api/delete-storage-pool", {
        node_name: nodeName,
        pool_name: poolName,
        delete_volumes: deleteVolumesOnDelete,
      });
      toast.success(
        deleteVolumesOnDelete
          ? `Storage pool ${poolName} and all volumes deleted successfully`
          : `Storage pool ${poolName} deleted successfully`
      );
      navigate(`/storage-pools?node=${nodeName}`);
    } catch (error: any) {
      console.error("Failed to delete storage pool:", error);
      toast.error(error?.message || "Failed to delete storage pool");
    } finally {
      setDeleting(false);
    }
  };

  // Volume operations
  const handleCreateVolume = async () => {
    setCreatingVolume(true);
    try {
      await apiPost("/api/create-volume", {
        node_name: nodeName,
        pool_name: poolName,
        name: createVolumeName || undefined,
        size_gb: createVolumeSize,
        format: createVolumeFormat,
      });
      toast.success(`Volume created successfully`);
      setShowCreateVolumeModal(false);
      setCreateVolumeName("");
      await fetchVolumes();
      await fetchPoolDetail();
    } catch (error: any) {
      console.error("Failed to create volume:", error);
      toast.error(error?.message || "Failed to create volume");
    } finally {
      setCreatingVolume(false);
    }
  };

  const handleDeleteVolumeClick = (volume: Volume) => {
    setVolumeToDelete(volume);
    setShowDeleteVolumeModal(true);
  };

  const confirmDeleteVolume = async () => {
    if (!volumeToDelete) return;
    setDeletingVolume(true);
    try {
      await apiPost("/api/delete-volume", {
        node_name: nodeName,
        pool_name: poolName,
        volume_id: volumeToDelete.volume_id,
      });
      toast.success(`Volume ${volumeToDelete.name} deleted successfully`);
      setShowDeleteVolumeModal(false);
      setVolumeToDelete(null);
      await fetchVolumes();
      await fetchPoolDetail();
    } catch (error: any) {
      console.error("Failed to delete volume:", error);
      toast.error(error?.message || "Failed to delete volume");
    } finally {
      setDeletingVolume(false);
    }
  };

  const handleResizeVolumeClick = (volume: Volume) => {
    setVolumeToResize(volume);
    setNewVolumeSize(volume.size_gb + 10);
    setShowResizeVolumeModal(true);
  };

  const confirmResizeVolume = async () => {
    if (!volumeToResize) return;
    if (newVolumeSize <= volumeToResize.size_gb) {
      toast.error("New size must be larger than current size");
      return;
    }
    setResizingVolume(true);
    try {
      await apiPost("/api/resize-volume", {
        node_name: nodeName,
        pool_name: poolName,
        volume_id: volumeToResize.volume_id,
        new_size_gb: newVolumeSize,
      });
      toast.success(
        `Volume ${volumeToResize.name} resized to ${newVolumeSize} GB successfully`
      );
      setShowResizeVolumeModal(false);
      setVolumeToResize(null);
      await fetchVolumes();
    } catch (error: any) {
      console.error("Failed to resize volume:", error);
      toast.error(error?.message || "Failed to resize volume");
    } finally {
      setResizingVolume(false);
    }
  };

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
  };

  const getStateColor = (state: string): string => {
    switch (state) {
      case "Active":
      case "Running":
        return "text-green-600 bg-green-100";
      case "Inactive":
        return "text-gray-600 bg-gray-100";
      case "Building":
        return "text-yellow-600 bg-yellow-100";
      case "Degraded":
        return "text-orange-600 bg-orange-100";
      case "Inaccessible":
        return "text-red-600 bg-red-100";
      default:
        return "text-gray-600 bg-gray-100";
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <RefreshCw className="w-8 h-8 animate-spin text-primary mx-auto mb-2" />
          <p className="text-gray-600">Loading storage pool...</p>
        </div>
      </div>
    );
  }

  if (!pool) {
    return (
      <>
        <div className="space-y-6">
          <Header
            title="Storage Pool Not Found"
            description="The requested storage pool does not exist"
          />
          <button
            onClick={() => navigate("/storage-pools")}
            className="flex items-center gap-2 text-primary hover:underline"
          >
            <ArrowLeft className="w-4 h-4" />
            Back to Storage Pools
          </button>
        </div>
      </>
    );
  }

  const usagePercent =
    pool.capacity > 0
      ? ((pool.allocation / pool.capacity) * 100).toFixed(1)
      : "0";

  return (
    <>
      <div className="space-y-6">
        {/* Header */}
        <Header
          title={
            <div className="flex items-center gap-3">
              <button
                onClick={() => navigate(`/storage-pools?node=${nodeName}`)}
                className="p-2 hover:bg-gray-100 rounded-lg"
              >
                <ArrowLeft className="w-5 h-5" />
              </button>
              <Database className="w-8 h-8 text-primary" />
              <div>
                <h1 className="text-2xl font-bold text-gray-900">
                  {pool.name}
                </h1>
                <p className="text-sm text-gray-600">{pool.path}</p>
              </div>
            </div>
          }
          description=""
          action={
            <div className="flex gap-2">
              <button
                onClick={handleRefreshPool}
                disabled={refreshing}
                className="flex items-center gap-2 px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50"
              >
                <RefreshCw
                  className={`w-4 h-4 ${refreshing ? "animate-spin" : ""}`}
                />
                Refresh
              </button>
              {pool.state.toLowerCase() === "inactive" ? (
                <button
                  onClick={handleStartPool}
                  className="flex items-center gap-2 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700"
                >
                  <Play className="w-4 h-4" />
                  Start
                </button>
              ) : (
                <button
                  onClick={handleStopPool}
                  className="flex items-center gap-2 px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700"
                >
                  <Square className="w-4 h-4" />
                  Stop
                </button>
              )}
              <button
                onClick={handleDeletePool}
                className="flex items-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700"
              >
                <Trash2 className="w-4 h-4" />
                Delete
              </button>
            </div>
          }
        />

        {/* Pool Info Card */}
        <div className="bg-white border border-gray-200 rounded-lg p-6">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            <div>
              <p className="text-sm text-gray-600 mb-1">Status</p>
              <span
                className={`px-3 py-1 text-sm font-medium rounded-full ${getStateColor(
                  pool.state
                )}`}
              >
                {pool.state}
              </span>
            </div>
            <div>
              <p className="text-sm text-gray-600 mb-1">Type</p>
              <p className="text-lg font-semibold text-gray-900">{pool.type || "N/A"}</p>
            </div>
            <div>
              <p className="text-sm text-gray-600 mb-1">Capacity</p>
              <p className="text-lg font-semibold text-gray-900">
                {formatBytes(pool.capacity)}
              </p>
            </div>
            <div>
              <p className="text-sm text-gray-600 mb-1">Volumes</p>
              <p className="text-lg font-semibold text-gray-900">
                {pool.volume_count}
              </p>
            </div>
          </div>

          <div className="mt-6">
            <div className="flex justify-between items-center mb-2">
              <p className="text-sm text-gray-600">Storage Usage</p>
              <p className="text-sm font-medium text-gray-900">
                {formatBytes(pool.allocation)} / {formatBytes(pool.capacity)} (
                {usagePercent}%)
              </p>
            </div>
            <div className="w-full bg-gray-200 rounded-full h-3">
              <div
                className="bg-primary h-3 rounded-full transition-all"
                style={{ width: `${usagePercent}%` }}
              />
            </div>
            <p className="text-sm text-gray-600 mt-2">
              {formatBytes(pool.available)} free
            </p>
          </div>
        </div>

        {/* Volumes List */}
        <div className="bg-white border border-gray-200 rounded-lg">
          <div className="p-4 border-b border-gray-200 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900">
              Volumes ({volumes.length})
            </h2>
            <button
              onClick={() => setShowCreateVolumeModal(true)}
              className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed"
              disabled={!pool || pool.state !== "Active"}
              title={
                !pool
                  ? "Loading..."
                  : pool.state !== "Active"
                  ? "Pool must be active to create volumes. Click 'Start' button first."
                  : "Create a new volume"
              }
            >
              Create Volume
            </button>
          </div>
          {loadingVolumes ? (
            <div className="p-8 text-center">
              <RefreshCw className="w-6 h-6 animate-spin text-primary mx-auto mb-2" />
              <p className="text-sm text-gray-600">Loading volumes...</p>
            </div>
          ) : volumes.length > 0 ? (
            <div className="divide-y divide-gray-200">
              {volumes.map((volume) => (
                <div
                  key={volume.volume_id}
                  className="p-4 hover:bg-gray-50 transition-colors"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3 flex-1">
                      <HardDrive className="w-5 h-5 text-gray-400" />
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <p className="font-medium text-gray-900 truncate">
                            {volume.name}
                          </p>
                          <span className="px-2 py-0.5 text-xs font-medium text-gray-600 bg-gray-100 rounded flex-shrink-0">
                            {volume.format}
                          </span>
                        </div>
                        <div className="flex items-center gap-2 mt-0.5">
                          <p className="text-xs text-gray-600">
                            ID: {volume.volume_id}
                          </p>
                          <span className="text-xs text-gray-400">•</span>
                          <p
                            className="text-xs text-gray-600 truncate"
                            title={volume.path}
                          >
                            {volume.path}
                          </p>
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-4 ml-4 flex-shrink-0">
                      <div className="text-right">
                        <p className="text-sm font-medium text-gray-900">
                          {volume.size_gb} GB
                        </p>
                        <p className="text-xs text-gray-600">
                          {formatBytes(volume.allocation_b)} used
                          {volume.capacity_b > 0 && (
                            <span className="ml-1">
                              (
                              {(
                                (volume.allocation_b / volume.capacity_b) *
                                100
                              ).toFixed(0)}
                              %)
                            </span>
                          )}
                        </p>
                      </div>
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => handleResizeVolumeClick(volume)}
                          className="p-2 text-gray-600 hover:text-primary hover:bg-gray-100 rounded transition-colors"
                          title="Resize Volume"
                        >
                          <Maximize2 className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => handleDeleteVolumeClick(volume)}
                          className="p-2 text-gray-600 hover:text-red-600 hover:bg-red-50 rounded transition-colors"
                          title="Delete Volume"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
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
      </div>

      {/* Create Volume Modal */}
      {showCreateVolumeModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h2 className="text-xl font-semibold mb-4">Create Volume</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Name (Optional)
                </label>
                <input
                  type="text"
                  value={createVolumeName}
                  onChange={(e) => setCreateVolumeName(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-primary"
                  placeholder="my-volume (leave empty to auto-generate)"
                />
                <p className="text-xs text-gray-500 mt-1">
                  If not provided, a unique ID will be generated automatically
                </p>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Size (GB)
                </label>
                <input
                  type="number"
                  value={createVolumeSize}
                  onChange={(e) => setCreateVolumeSize(Number(e.target.value))}
                  min="1"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-primary"
                  placeholder="20"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Format
                </label>
                <select
                  value={createVolumeFormat}
                  onChange={(e) => setCreateVolumeFormat(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-primary"
                >
                  <option value="qcow2">qcow2</option>
                  <option value="raw">raw</option>
                </select>
              </div>
            </div>
            <div className="flex gap-2 mt-6">
              <button
                onClick={() => setShowCreateVolumeModal(false)}
                disabled={creatingVolume}
                className="flex-1 px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                onClick={handleCreateVolume}
                disabled={creatingVolume}
                className="flex-1 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50"
              >
                {creatingVolume ? "Creating..." : "Create"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Volume Modal */}
      {showDeleteVolumeModal && volumeToDelete && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h2 className="text-xl font-semibold mb-4 text-red-600">
              Delete Volume
            </h2>
            <p className="text-gray-700 mb-4">
              Are you sure you want to delete volume{" "}
              <span className="font-semibold">{volumeToDelete.name}</span>?
            </p>
            <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-4">
              <p className="text-sm text-red-700">
                This action cannot be undone. All data in this volume will be permanently deleted.
              </p>
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => {
                  setShowDeleteVolumeModal(false);
                  setVolumeToDelete(null);
                }}
                disabled={deletingVolume}
                className="flex-1 px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                onClick={confirmDeleteVolume}
                disabled={deletingVolume}
                className="flex-1 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50"
              >
                {deletingVolume ? "Deleting..." : "Delete"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Resize Volume Modal */}
      {showResizeVolumeModal && volumeToResize && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h2 className="text-xl font-semibold mb-4">Resize Volume</h2>
            <p className="text-gray-700 mb-4">
              Resize volume <span className="font-semibold">{volumeToResize.name}</span>
            </p>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Current Size
                </label>
                <p className="text-lg font-semibold text-gray-900">
                  {volumeToResize.size_gb} GB
                </p>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  New Size (GB)
                </label>
                <input
                  type="number"
                  value={newVolumeSize}
                  onChange={(e) => setNewVolumeSize(Number(e.target.value))}
                  min={volumeToResize.size_gb + 1}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-primary"
                />
                <p className="text-xs text-gray-500 mt-1">
                  Must be larger than current size ({volumeToResize.size_gb} GB)
                </p>
              </div>
            </div>
            <div className="flex gap-2 mt-6">
              <button
                onClick={() => {
                  setShowResizeVolumeModal(false);
                  setVolumeToResize(null);
                }}
                disabled={resizingVolume}
                className="flex-1 px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                onClick={confirmResizeVolume}
                disabled={resizingVolume}
                className="flex-1 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50"
              >
                {resizingVolume ? "Resizing..." : "Resize"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Pool Modal */}
      {showDeleteModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h2 className="text-xl font-semibold mb-4 text-red-600">
              Delete Storage Pool
            </h2>
            <p className="text-gray-700 mb-4">
              Are you sure you want to delete storage pool{" "}
              <span className="font-semibold">{poolName}</span>?
            </p>
            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 mb-4">
              <label className="flex items-start gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  checked={deleteVolumesOnDelete}
                  onChange={(e) => setDeleteVolumesOnDelete(e.target.checked)}
                  className="mt-1"
                />
                <div>
                  <p className="font-medium text-yellow-800">
                    Delete all volumes and directory
                  </p>
                  <p className="text-sm text-yellow-700 mt-1">
                    This will permanently delete all volumes in the pool and the
                    pool directory. This action cannot be undone.
                  </p>
                </div>
              </label>
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => {
                  setShowDeleteModal(false);
                  setDeleteVolumesOnDelete(false);
                }}
                disabled={deleting}
                className="flex-1 px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                onClick={confirmDeletePool}
                disabled={deleting}
                className="flex-1 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50"
              >
                {deleting ? "Deleting..." : "Delete"}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

