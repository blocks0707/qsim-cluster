"use client";

import { useEffect, useState } from "react";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from "recharts";
import { getJobResult } from "@/lib/api";
import { RefreshCw, CheckCircle, XCircle } from "lucide-react";
import type { JobResult as JobResultType } from "@/types";

const BAR_COLORS = [
  "#8b5cf6",
  "#6366f1",
  "#3b82f6",
  "#06b6d4",
  "#a78bfa",
  "#818cf8",
  "#60a5fa",
  "#22d3ee",
];

export function JobResult({ jobId, shots }: { jobId: string; shots: number }) {
  const [result, setResult] = useState<JobResultType | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    getJobResult(jobId)
      .then((data) => {
        setResult(data);
        setError(null);
      })
      .catch((e) => setError(e instanceof Error ? e.message : "Failed to load result"))
      .finally(() => setLoading(false));
  }, [jobId]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <RefreshCw className="w-6 h-6 animate-spin text-gray-400" />
      </div>
    );
  }

  if (error || !result) {
    return (
      <div className="rounded-lg bg-red-500/10 border border-red-500/30 p-6 text-red-400">
        {error ?? "결과를 불러올 수 없습니다"}
      </div>
    );
  }

  const chartData = Object.entries(result.counts)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([state, count]) => ({ state, count }));

  const totalCounts = Object.values(result.counts).reduce((a, b) => a + b, 0);
  const meta = result.metadata ?? {};

  return (
    <div className="space-y-6">
      {/* Summary */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <MetaCard label="Shots" value={shots.toLocaleString()} />
        <MetaCard label="총 측정" value={totalCounts.toLocaleString()} />
        <MetaCard
          label="성공"
          value={
            <span className="flex items-center gap-1">
              {totalCounts === shots ? (
                <><CheckCircle className="w-4 h-4 text-green-400" /> Yes</>
              ) : (
                <><XCircle className="w-4 h-4 text-yellow-400" /> Partial</>
              )}
            </span>
          }
        />
        <MetaCard label="상태 수" value={chartData.length.toString()} />
      </div>

      {/* Metadata */}
      {Object.keys(meta).length > 0 && (
        <div className="rounded-xl border border-gray-700 bg-gray-800/50 p-4">
          <h4 className="text-sm font-semibold text-gray-400 mb-2">메타데이터</h4>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            {Object.entries(meta).map(([k, v]) => (
              <div key={k}>
                <p className="text-xs text-gray-500">{k}</p>
                <p className="text-sm font-medium">{String(v)}</p>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Histogram */}
      <div className="rounded-xl border border-gray-700 bg-gray-800 p-6">
        <h4 className="text-lg font-semibold mb-4">Measurement 히스토그램</h4>
        <ResponsiveContainer width="100%" height={300}>
          <BarChart data={chartData} margin={{ top: 5, right: 20, bottom: 5, left: 20 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
            <XAxis dataKey="state" stroke="#9ca3af" fontSize={12} fontFamily="monospace" />
            <YAxis stroke="#9ca3af" fontSize={12} />
            <Tooltip
              contentStyle={{
                backgroundColor: "#1f2937",
                border: "1px solid #374151",
                borderRadius: "8px",
                color: "#f3f4f6",
              }}
            />
            <Bar dataKey="count" radius={[4, 4, 0, 0]}>
              {chartData.map((_, i) => (
                <Cell key={i} fill={BAR_COLORS[i % BAR_COLORS.length]} />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

function MetaCard({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-gray-700 bg-gray-800/50 p-3">
      <p className="text-xs text-gray-400 mb-1">{label}</p>
      <div className="text-lg font-semibold">{value}</div>
    </div>
  );
}
