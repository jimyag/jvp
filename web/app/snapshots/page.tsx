"use client";

import DashboardLayout from "@/components/DashboardLayout";
import Header from "@/components/Header";
import { Camera } from "lucide-react";

export default function SnapshotsPage() {
  return (
    <DashboardLayout>
      <div className="space-y-6">
        <Header
          title="Snapshots (Deprecated)"
          description="EBS Snapshots feature has been removed"
        />

        <div className="bg-white rounded-lg border border-gray-200 p-12">
          <div className="text-center max-w-2xl mx-auto">
            <Camera className="w-16 h-16 text-gray-400 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-4">
              EBS Snapshots Feature Removed
            </h3>
            <div className="text-left text-gray-600 space-y-4">
              <p>
                JVP no longer supports EBS snapshots (volume-level snapshots).
              </p>
              <p>
                <strong>Please use VM Templates instead:</strong>
              </p>
              <ul className="list-disc pl-6 space-y-2">
                <li>VM Templates use libvirt domain snapshots (full VM state)</li>
                <li>Navigate to <a href="/templates" className="text-blue-600 hover:underline">Templates</a> page to view VMs with snapshots</li>
                <li>Create snapshots using: <code className="bg-gray-100 px-2 py-1 rounded">virsh snapshot-create-as &lt;vm-name&gt; &lt;snapshot-name&gt;</code></li>
              </ul>
              <p className="mt-6">
                <a href="/templates" className="btn btn-primary">
                  Go to Templates →
                </a>
              </p>
            </div>
          </div>
        </div>
      </div>
    </DashboardLayout>
  );
}
