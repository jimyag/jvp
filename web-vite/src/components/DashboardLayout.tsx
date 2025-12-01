import { Outlet } from "react-router-dom";
import Sidebar from "./Sidebar";

export default function DashboardLayout() {
  return (
    <div className="flex min-h-screen">
      <Sidebar />
      <main className="flex-1 lg:ml-0 overflow-x-hidden">
        <div className="w-full px-4 py-4 lg:px-6 lg:py-6">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
