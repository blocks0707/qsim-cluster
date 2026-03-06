"use client";

import type { Node } from "@/types";
import { ProgressBar } from "./ProgressBar";

const poolColors: Record<string, string> = {
  cpu: "bg-blue-600 text-blue-100",
  gpu: "bg-purple-600 text-purple-100",
  "high-memory": "bg-orange-600 text-orange-100",
};

function isReady(status: Node["status"]) {
  return status === "ready" || status === "busy";
}

export function NodeCard({ node }: { node: Node }) {
  const ready = isReady(node.status);

  return (
    <div className="bg-gray-800 rounded-xl p-5 border border-gray-700 hover:border-blue-500 transition-colors">
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <span
            className={`w-2.5 h-2.5 rounded-full ${ready ? "bg-green-400" : "bg-red-400"}`}
          />
          <h3 className="font-semibold text-sm truncate">{node.name}</h3>
        </div>
        <span
          className={`text-[10px] font-medium px-2 py-0.5 rounded-full ${poolColors[node.pool] ?? "bg-gray-600 text-gray-200"}`}
        >
          {node.pool}
        </span>
      </div>

      {/* Status */}
      <p className={`text-xs mb-3 ${ready ? "text-green-400" : "text-red-400"}`}>
        {ready ? "Ready" : "NotReady"}
      </p>

      {/* Resource bars */}
      {node.metrics && (
        <div className="space-y-2 mb-3">
          <ProgressBar label="CPU" value={node.metrics.cpuUsage} />
          <ProgressBar label="Memory" value={node.metrics.memoryUsage} />
        </div>
      )}

      {/* GPU info */}
      {node.gpu && (
        <div className="mb-3">
          <ProgressBar
            label={`GPU (${node.gpu.type})`}
            value={node.gpu.usage}
          />
        </div>
      )}

      {/* Active jobs */}
      <div className="flex items-center justify-between text-xs text-gray-400 mb-3">
        <span>Active Jobs</span>
        <span className="text-gray-200 font-medium">{node.activeJobs}</span>
      </div>

      {/* Labels */}
      {Object.keys(node.labels).length > 0 && (
        <div className="flex flex-wrap gap-1">
          {Object.entries(node.labels).map(([k, v]) => (
            <span
              key={k}
              className="text-[10px] bg-gray-700 text-gray-300 px-1.5 py-0.5 rounded"
            >
              {k}={v}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}
