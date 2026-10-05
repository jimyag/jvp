import { Suspense, lazy } from "react";
import type { RouteObject } from "react-router-dom";
import { LoadingState } from "./components/ui";
import DashboardLayout from "./components/DashboardLayout";
import { NotFoundPage, RouteErrorPage } from "./pages/ErrorPages";

import OverviewPage from "./pages/overview/OverviewPage";
import InstancesPage from "./pages/instances/InstancesPage";
import InstanceDetailPage from "./pages/instances/InstanceDetailPage";
import NodesPage from "./pages/nodes/NodesPage";
import NodeDetailPage from "./pages/nodes/NodeDetailPage";
import StoragePoolsPage from "./pages/storage-pools/StoragePoolsPage";
import StoragePoolDetailPage from "./pages/storage-pools/StoragePoolDetailPage";
import TemplatesPage from "./pages/templates/TemplatesPage";
import SnapshotsPage from "./pages/snapshots/SnapshotsPage";
import KeypairsPage from "./pages/keypairs/KeypairsPage";
import NetworksPage from "./pages/networks/NetworksPage";

// 控制台依赖 xterm，按需加载以减小首屏体积
const InstanceConsolePage = lazy(() => import("./pages/instances/InstanceConsolePage"));

const routes: RouteObject[] = [
  {
    element: <DashboardLayout />,
    errorElement: <RouteErrorPage />,
    children: [
      { path: "/", element: <OverviewPage /> },
      { path: "/instances", element: <InstancesPage /> },
      { path: "/instances/:nodeName/:id", element: <InstanceDetailPage /> },
      {
        path: "/instances/:nodeName/:id/console",
        element: (
          <Suspense fallback={<LoadingState label="Loading console…" />}>
            <InstanceConsolePage />
          </Suspense>
        ),
      },
      { path: "/nodes", element: <NodesPage /> },
      { path: "/nodes/:name", element: <NodeDetailPage /> },
      { path: "/storage-pools", element: <StoragePoolsPage /> },
      { path: "/storage-pools/:poolName", element: <StoragePoolDetailPage /> },
      { path: "/templates", element: <TemplatesPage /> },
      { path: "/snapshots", element: <SnapshotsPage /> },
      { path: "/keypairs", element: <KeypairsPage /> },
      { path: "/networks", element: <NetworksPage /> },
      { path: "*", element: <NotFoundPage /> },
    ],
  },
];

export default routes;
