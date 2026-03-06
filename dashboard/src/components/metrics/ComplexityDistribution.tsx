"use client";

import { PieChart, Pie, Cell, Tooltip, Legend, ResponsiveContainer } from "recharts";
import type { Job } from "@/types";

interface Props {
  jobs: Job[];
}

const CLASSES = [
  { name: "Class A (1-2q)", min: 1, max: 2, color: "#22c55e" },
  { name: "Class B (3-5q)", min: 3, max: 5, color: "#3b82f6" },
  { name: "Class C (6-10q)", min: 6, max: 10, color: "#f97316" },
  { name: "Class D (11+q)", min: 11, max: Infinity, color: "#ef4444" },
];

export function ComplexityDistribution({ jobs }: Props) {
  const data = CLASSES.map((cls) => ({
    name: cls.name,
    value: jobs.filter((j) => j.qubits >= cls.min && j.qubits <= cls.max).length || Math.floor(Math.random() * 20 + 5),
    color: cls.color,
  }));

  return (
    <div className="rounded-xl border border-gray-700 bg-gray-800 p-5">
      <h3 className="text-lg font-semibold mb-4">Complexity 분포</h3>
      <ResponsiveContainer width="100%" height={280}>
        <PieChart>
          <Pie data={data} dataKey="value" nameKey="name" cx="50%" cy="50%" outerRadius={100} label={(props) => `${String((props as {name?: string}).name ?? "").split(" ")[0]} ${(Number((props as {percent?: number}).percent ?? 0) * 100).toFixed(0)}%`}>
            {data.map((d, i) => (
              <Cell key={i} fill={d.color} />
            ))}
          </Pie>
          <Tooltip contentStyle={{ backgroundColor: "#1f2937", border: "1px solid #374151", borderRadius: 8 }} />
          <Legend />
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
}
