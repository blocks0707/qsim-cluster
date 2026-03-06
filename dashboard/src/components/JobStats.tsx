"use client";

import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from "recharts";

interface JobStatsProps {
  pending: number;
  running: number;
  completed: number;
  failed: number;
}

const COLORS: Record<string, string> = {
  Pending: "#eab308",
  Running: "#3b82f6",
  Completed: "#22c55e",
  Failed: "#ef4444",
};

export function JobStats({ pending, running, completed, failed }: JobStatsProps) {
  const data = [
    { name: "Pending", value: pending },
    { name: "Running", value: running },
    { name: "Completed", value: completed },
    { name: "Failed", value: failed },
  ].filter((d) => d.value > 0);

  const total = pending + running + completed + failed;

  return (
    <div className="rounded-xl border border-gray-700 bg-gray-800 p-5">
      <h3 className="text-lg font-semibold mb-4">Job 통계</h3>
      <div className="flex items-center gap-6">
        <div className="w-40 h-40 flex-shrink-0">
          <ResponsiveContainer width="100%" height="100%">
            <PieChart>
              <Pie
                data={data.length > 0 ? data : [{ name: "None", value: 1 }]}
                innerRadius="60%"
                outerRadius="85%"
                dataKey="value"
                stroke="none"
              >
                {(data.length > 0 ? data : [{ name: "None", value: 1 }]).map((entry) => (
                  <Cell key={entry.name} fill={COLORS[entry.name] ?? "#374151"} />
                ))}
              </Pie>
              <Tooltip
                contentStyle={{ backgroundColor: "#1f2937", border: "none", borderRadius: 8 }}
                itemStyle={{ color: "#e5e7eb" }}
              />
            </PieChart>
          </ResponsiveContainer>
        </div>
        <div className="space-y-3 text-sm">
          {[
            { label: "Pending", value: pending, color: "bg-yellow-500" },
            { label: "Running", value: running, color: "bg-blue-500" },
            { label: "Completed", value: completed, color: "bg-green-500" },
            { label: "Failed", value: failed, color: "bg-red-500" },
          ].map((item) => (
            <div key={item.label} className="flex items-center gap-2">
              <span className={`w-3 h-3 rounded-full ${item.color}`} />
              <span className="text-gray-400 w-20">{item.label}</span>
              <span className="font-semibold">{item.value}</span>
            </div>
          ))}
          <div className="pt-2 border-t border-gray-700 text-gray-400">
            Total: <span className="text-white font-semibold">{total}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
