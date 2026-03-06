"use client";

import { PieChart, Pie, Cell, ResponsiveContainer } from "recharts";
import type { ClusterMetrics } from "@/types";
import type { Node } from "@/types";

interface Props {
  metrics: ClusterMetrics | null;
  nodes: Node[];
}

function Gauge({ label, value, color }: { label: string; value: number; color: string }) {
  const data = [{ value }, { value: 100 - value }];
  return (
    <div className="flex flex-col items-center">
      <ResponsiveContainer width="100%" height={120}>
        <PieChart>
          <Pie data={data} startAngle={180} endAngle={0} innerRadius="70%" outerRadius="90%" dataKey="value" stroke="none">
            <Cell fill={color} />
            <Cell fill="#374151" />
          </Pie>
        </PieChart>
      </ResponsiveContainer>
      <p className="text-2xl font-bold -mt-10">{value.toFixed(0)}%</p>
      <p className="text-sm text-gray-400 mt-1">{label}</p>
    </div>
  );
}

export function ClusterResources({ metrics, nodes }: Props) {
  const readyNodes = nodes.filter((n) => n.status !== "offline");
  const avgCpu = readyNodes.length > 0 ? readyNodes.reduce((s, n) => s + (n.metrics?.cpuUsage ?? 0), 0) / readyNodes.length : 0;
  const avgMem = readyNodes.length > 0 ? readyNodes.reduce((s, n) => s + (n.metrics?.memoryUsage ?? 0), 0) / readyNodes.length : 0;
  const gpuNodes = readyNodes.filter((n) => n.metrics?.gpuUsage != null);
  const avgGpu = gpuNodes.length > 0 ? gpuNodes.reduce((s, n) => s + (n.metrics?.gpuUsage ?? 0), 0) / gpuNodes.length : 0;

  const totalCores = nodes.length * 8; // estimate
  const totalMemGB = nodes.length * 32; // estimate

  return (
    <div className="rounded-xl border border-gray-700 bg-gray-800 p-5">
      <h3 className="text-lg font-semibold mb-2">클러스터 리소스</h3>
      <div className="flex gap-6 text-sm text-gray-400 mb-4">
        <span>노드: {nodes.length}</span>
        <span>코어: ~{totalCores}</span>
        <span>메모리: ~{totalMemGB} GB</span>
      </div>
      <div className="grid grid-cols-3 gap-4">
        <Gauge label="CPU" value={avgCpu} color="#3b82f6" />
        <Gauge label="Memory" value={avgMem} color="#8b5cf6" />
        <Gauge label="GPU" value={avgGpu} color="#10b981" />
      </div>
    </div>
  );
}
