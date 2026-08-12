import { Navigate, isRouteErrorResponse, useRouteError } from 'react-router-dom';
import type { RouteObject } from 'react-router-dom';
import DashboardLayout from './components/DashboardLayout';

// Pages
import InstancesPage from './pages/instances/InstancesPage';
import InstanceDetailPage from './pages/instances/InstanceDetailPage';
import InstanceConsolePage from './pages/instances/InstanceConsolePage';
import NodesPage from './pages/nodes/NodesPage';
import NodeDetailPage from './pages/nodes/NodeDetailPage';
import StoragePoolsPage from './pages/storage-pools/StoragePoolsPage';
import StoragePoolDetailPage from './pages/storage-pools/StoragePoolDetailPage';
import TemplatesPage from './pages/templates/TemplatesPage';
import SnapshotsPage from './pages/snapshots/SnapshotsPage';
import KeypairsPage from './pages/keypairs/KeypairsPage';
import NetworksPage from './pages/networks/NetworksPage';

function RouteErrorPage() {
  const error = useRouteError();
  const message = isRouteErrorResponse(error)
    ? error.statusText
    : error instanceof Error
      ? error.message
      : 'The page failed to load.';

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-6">
      <div className="bg-white border border-gray-200 rounded-lg shadow-sm max-w-md w-full p-6 text-center">
        <h1 className="text-xl font-semibold text-gray-900 mb-2">Application Update Required</h1>
        <p className="text-sm text-gray-600 mb-5">{message}</p>
        <button
          className="btn-primary"
          onClick={() => window.location.reload()}
        >
          Reload Page
        </button>
      </div>
    </div>
  );
}

const routes: RouteObject[] = [
  {
    element: <DashboardLayout />,
    errorElement: <RouteErrorPage />,
    children: [
      { path: '/', element: <Navigate to="/instances" replace /> },
      { path: '/instances', element: <InstancesPage /> },
      { path: '/instances/:nodeName/:id', element: <InstanceDetailPage /> },
      { path: '/instances/:nodeName/:id/console', element: <InstanceConsolePage /> },
      { path: '/nodes', element: <NodesPage /> },
      { path: '/nodes/:name', element: <NodeDetailPage /> },
      { path: '/storage-pools', element: <StoragePoolsPage /> },
      { path: '/storage-pools/:poolName', element: <StoragePoolDetailPage /> },
      { path: '/templates', element: <TemplatesPage /> },
      { path: '/snapshots', element: <SnapshotsPage /> },
      { path: '/keypairs', element: <KeypairsPage /> },
      { path: '/networks', element: <NetworksPage /> },
    ]
  },
  { path: '*', element: <Navigate to="/instances" replace /> }
];

export default routes;
