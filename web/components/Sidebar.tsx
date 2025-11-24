"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Server, HardDrive, Image, Key, Menu, X, Camera, Database, Package } from "lucide-react";
import { useState } from "react";

const navigation = [
  { name: "Instances", href: "/instances", icon: Server },
  { name: "Volumes", href: "/volumes", icon: HardDrive },
  { name: "Storage Pools", href: "/storage-pools", icon: Database },
  { name: "Snapshots", href: "/snapshots", icon: Camera },
  { name: "Images", href: "/images", icon: Image },
  { name: "Templates", href: "/templates", icon: Package },
  { name: "Key Pairs", href: "/keypairs", icon: Key },
];

export default function Sidebar() {
  const pathname = usePathname();
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);

  return (
    <>
      {/* Mobile menu button */}
      <button
        onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
        className="lg:hidden fixed top-4 left-4 z-50 p-2 bg-primary text-white rounded-lg"
      >
        {isMobileMenuOpen ? <X size={24} /> : <Menu size={24} />}
      </button>

      {/* Sidebar */}
      <aside
        className={`
          fixed lg:sticky top-0 left-0 h-screen w-64 bg-primary text-white
          transition-transform duration-300 ease-in-out z-40
          ${isMobileMenuOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0"}
        `}
      >
        <div className="flex flex-col h-full">
          {/* Logo */}
          <div className="p-6 border-b border-gray-700">
            <h1 className="text-2xl font-bold tracking-tight">JVP</h1>
            <p className="text-sm text-gray-400 mt-1">Virtualization Platform</p>
          </div>

          {/* Navigation */}
          <nav className="flex-1 p-4 space-y-2">
            {navigation.map((item) => {
              const Icon = item.icon;
              const isActive = pathname?.startsWith(item.href);

              return (
                <Link
                  key={item.name}
                  href={item.href}
                  onClick={() => setIsMobileMenuOpen(false)}
                  className={`
                    flex items-center gap-3 px-4 py-3 rounded-lg font-medium
                    transition-all duration-200
                    ${
                      isActive
                        ? "bg-accent text-white"
                        : "text-gray-300 hover:bg-primary-light hover:text-white"
                    }
                  `}
                >
                  <Icon size={20} />
                  <span>{item.name}</span>
                </Link>
              );
            })}
          </nav>

          {/* Footer */}
          <div className="p-4 border-t border-gray-700">
            <p className="text-xs text-gray-400">
              © 2024 JVP Platform
            </p>
          </div>
        </div>
      </aside>

      {/* Overlay for mobile */}
      {isMobileMenuOpen && (
        <div
          className="fixed inset-0 bg-black bg-opacity-50 z-30 lg:hidden"
          onClick={() => setIsMobileMenuOpen(false)}
        />
      )}
    </>
  );
}
