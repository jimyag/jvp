import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import type { ReactNode } from "react";
import { useSearchParams } from "react-router-dom";
import { api, errorMessage } from "./api";
import type { Node } from "./types";

const STORAGE_KEY = "jvp.node";

interface NodeContextValue {
  nodes: Node[];
  loading: boolean;
  error: string | null;
  /** 全局选中的节点，跨页面保持一致并持久化到 localStorage */
  currentNode: string;
  setCurrentNode: (name: string) => void;
  refresh: () => Promise<void>;
}

const NodeContext = createContext<NodeContextValue | undefined>(undefined);

function readStoredNode(): string {
  try {
    return localStorage.getItem(STORAGE_KEY) || "";
  } catch {
    return "";
  }
}

export function NodeProvider({ children }: { children: ReactNode }) {
  const [nodes, setNodes] = useState<Node[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [currentNode, setCurrentNodeState] = useState<string>(readStoredNode);

  const setCurrentNode = useCallback((name: string) => {
    setCurrentNodeState(name);
    try {
      localStorage.setItem(STORAGE_KEY, name);
    } catch {
      // ignore
    }
  }, []);

  const refresh = useCallback(async () => {
    try {
      const data = await api<{ nodes: Node[] }>("/api/list-nodes");
      const list = data.nodes || [];
      setNodes(list);
      setError(null);
      setCurrentNodeState((prev) => {
        if (prev && list.some((n) => n.name === prev)) return prev;
        const preferred = list.find((n) => n.state === "online") || list[0];
        return preferred?.name || "";
      });
    } catch (err) {
      setError(errorMessage(err, "Failed to load nodes"));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const value = useMemo(
    () => ({ nodes, loading, error, currentNode, setCurrentNode, refresh }),
    [nodes, loading, error, currentNode, setCurrentNode, refresh]
  );

  return <NodeContext.Provider value={value}>{children}</NodeContext.Provider>;
}

// eslint-disable-next-line react-refresh/only-export-components
export function useNodes() {
  const ctx = useContext(NodeContext);
  if (!ctx) throw new Error("useNodes must be used within a NodeProvider");
  return ctx;
}

/**
 * 用于按节点划分的页面：
 * - 首次进入时优先采用 URL 中的 ?node= 参数（便于分享链接）
 * - 之后将全局选中的节点同步回 URL
 */
// eslint-disable-next-line react-refresh/only-export-components
export function useScopedNode() {
  const ctx = useNodes();
  const { loading, nodes, currentNode, setCurrentNode } = ctx;
  const [params, setParams] = useSearchParams();
  const urlNode = params.get("node");
  const appliedRef = useRef(false);

  useEffect(() => {
    if (loading) return;
    if (!appliedRef.current) {
      appliedRef.current = true;
      if (urlNode && urlNode !== currentNode && nodes.some((n) => n.name === urlNode)) {
        setCurrentNode(urlNode);
        return;
      }
    }
    if (currentNode && urlNode !== currentNode) {
      setParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          next.set("node", currentNode);
          return next;
        },
        { replace: true }
      );
    }
  }, [loading, nodes, currentNode, urlNode, setCurrentNode, setParams]);

  return ctx;
}
