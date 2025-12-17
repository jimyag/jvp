import { Navigate } from 'react-router-dom';
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

const routes: RouteObject[] = [
  {
    element: <DashboardLayout />,
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

