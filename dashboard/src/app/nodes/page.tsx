"use client";

import { useEffect, useState, useCallback } from "react";
import type { Node, NodePool } from "@/types";
import { getNodes } from "@/lib/api";
import { NodeCard } from "@/components/NodeCard";

const POOL_TABS: { label: string; value: NodePool | "all" }[] = [
  { label: "All", value: "all" },
  { label: "CPU", value: "cpu" },
  { label: "GPU", value: "gpu" },
  { label: "High-Memory", value: "high-memory" },
];

type StatusFilter = "all" | "ready" | "notready";
const STATUS_TABS: { label: string; value: StatusFilter }[] = [
  { label: "All", value: "all" },
  { label: "Ready", value: "ready" },
  { label: "NotReady", value: "notready" },
];

function isReady(status: Node["status"]) {
  return status === "ready" || status === "busy";
}

export default function NodesPage() {
  const [nodes, setNodes] = useState<Node[]>([]);
  const [loading, setLoading] = useState(true);
  const [poolFilter, setPoolFilter] = useState<NodePool | "all">("all");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");

  const fetchNodes = useCallback(async () => {
    try {
      const data = await getNodes();
      setNodes(data);
    } catch {
      // silently retry on next interval
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchNodes();
    const id = setInterval(fetchNodes, 30_000);
    return () => clearInterval(id);
  }, [fetchNodes]);

  const filtered = nodes.filter((n) => {
    if (poolFilter !== "all" && n.pool !== poolFilter) return false;
    if (statusFilter === "ready" && !isReady(n.status)) return false;
    if (statusFilter === "notready" && isReady(n.status)) return false;
    return true;
  });

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold">Nodes</h2>
        <span className="text-xs text-gray-500">Auto-refresh 30s</span>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-4 mb-6">
        {/* Pool filter */}
        <div className="flex gap-1 bg-gray-800 rounded-lg p-1">
          {POOL_TABS.map((t) => (
            <button
              key={t.value}
              onClick={() => setPoolFilter(t.value)}
              className={`px-3 py-1 text-xs rounded-md transition-colors ${
                poolFilter === t.value
                  ? "bg-blue-600 text-white"
                  : "text-gray-400 hover:text-gray-200"
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>

        {/* Status filter */}
        <div className="flex gap-1 bg-gray-800 rounded-lg p-1">
          {STATUS_TABS.map((t) => (
            <button
              key={t.value}
              onClick={() => setStatusFilter(t.value)}
              className={`px-3 py-1 text-xs rounded-md transition-colors ${
                statusFilter === t.value
                  ? "bg-blue-600 text-white"
                  : "text-gray-400 hover:text-gray-200"
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>
      </div>

      {/* Grid */}
      {loading ? (
        <p className="text-gray-500">Loading nodes…</p>
      ) : filtered.length === 0 ? (
        <p className="text-gray-500">No nodes match the current filters.</p>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filtered.map((node) => (
            <NodeCard key={node.id} node={node} />
          ))}
        </div>
      )}
    </div>
  );
}
