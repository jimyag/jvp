import { Outlet } from "react-router-dom";
import { NodeProvider } from "@/lib/nodes";
import Sidebar from "./Sidebar";

export default function DashboardLayout() {
  return (
    <NodeProvider>
      <div className="min-h-full">
        <Sidebar />
        <main className="lg:pl-60">
          <div className="mx-auto w-full max-w-[1440px] px-4 py-6 sm:px-6 lg:px-8 lg:py-8">
            <Outlet />
          </div>
        </main>
      </div>
    </NodeProvider>
  );
}
