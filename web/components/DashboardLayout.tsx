import Sidebar from "./Sidebar";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-h-screen">
      <Sidebar />
      <main className="flex-1 lg:ml-0 overflow-x-hidden">
        <div className="container mx-auto px-6 py-8 lg:px-12 lg:py-12">
          {children}
        </div>
      </main>
    </div>
  );
}
