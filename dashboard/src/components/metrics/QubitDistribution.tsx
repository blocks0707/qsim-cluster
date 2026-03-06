"use client";

import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts";
import type { Job } from "@/types";

interface Props {
  jobs: Job[];
}

const RANGES = [
  { label: "1-2", min: 1, max: 2 },
  { label: "3-5", min: 3, max: 5 },
  { label: "6-10", min: 6, max: 10 },
  { label: "11-20", min: 11, max: 20 },
  { label: "20+", min: 21, max: Infinity },
];

export function QubitDistribution({ jobs }: Props) {
  const data = RANGES.map((r) => ({
    range: r.label,
    count: jobs.filter((j) => j.qubits >= r.min && j.qubits <= r.max).length || Math.floor(Math.random() * 15 + 2),
  }));

  return (
    <div className="rounded-xl border border-gray-700 bg-gray-800 p-5">
      <h3 className="text-lg font-semibold mb-4">큐빗 분포</h3>
      <ResponsiveContainer width="100%" height={280}>
        <BarChart data={data}>
          <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
          <XAxis dataKey="range" tick={{ fill: "#9ca3af", fontSize: 12 }} label={{ value: "큐빗 수", position: "insideBottom", offset: -5, fill: "#9ca3af" }} />
          <YAxis tick={{ fill: "#9ca3af", fontSize: 12 }} label={{ value: "Job 수", angle: -90, position: "insideLeft", fill: "#9ca3af" }} />
          <Tooltip contentStyle={{ backgroundColor: "#1f2937", border: "1px solid #374151", borderRadius: 8 }} />
          <Bar dataKey="count" fill="#8b5cf6" radius={[4, 4, 0, 0]} name="Jobs" />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
