import { Server } from "lucide-react";
import { useNodes } from "@/lib/nodes";

/** 全局节点选择器：切换后所有按节点划分的页面同步生效 */
export default function NodeSelect({ className = "" }: { className?: string }) {
  const { nodes, currentNode, setCurrentNode, loading } = useNodes();
  const current = nodes.find((n) => n.name === currentNode);
  const dot =
    current?.state === "online" ? "bg-success" : current?.state === "maintenance" ? "bg-warning" : "bg-danger";

  return (
    <label className={`relative inline-flex items-center ${className}`} title="Current node">
      <span className="pointer-events-none absolute left-2.5 flex items-center gap-1.5 text-fg-subtle">
        <Server size={14} />
        {current && <span className={`h-1.5 w-1.5 rounded-full ${dot}`} />}
      </span>
      <span className="sr-only">Node</span>
      <select
        className="input w-auto min-w-[10rem] pl-[2.6rem] font-medium"
        value={currentNode}
        disabled={loading || nodes.length === 0}
        onChange={(e) => setCurrentNode(e.target.value)}
      >
        {nodes.length === 0 && <option value="">{loading ? "Loading nodes…" : "No nodes"}</option>}
        {nodes.map((node) => (
          <option key={node.name} value={node.name}>
            {node.name}
            {node.state !== "online" ? ` (${node.state})` : ""}
          </option>
        ))}
      </select>
    </label>
  );
}
