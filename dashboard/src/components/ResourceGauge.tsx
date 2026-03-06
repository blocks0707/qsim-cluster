"use client";

import { PieChart, Pie, Cell, ResponsiveContainer } from "recharts";

interface ResourceGaugeProps {
  label: string;
  value: number; // 0-100
  color: string;
}

function GaugeChart({ value, color }: { value: number; color: string }) {
  const data = [
    { value },
    { value: 100 - value },
  ];
  return (
    <ResponsiveContainer width="100%" height={120}>
      <PieChart>
        <Pie
          data={data}
          startAngle={180}
          endAngle={0}
          innerRadius="70%"
          outerRadius="90%"
          dataKey="value"
          stroke="none"
        >
          <Cell fill={color} />
          <Cell fill="#374151" />
        </Pie>
      </PieChart>
    </ResponsiveContainer>
  );
}

function SingleGauge({ label, value, color }: ResourceGaugeProps) {
  return (
    <div className="flex flex-col items-center">
      <GaugeChart value={value} color={color} />
      <p className="text-2xl font-bold -mt-10">{value}%</p>
      <p className="text-sm text-gray-400 mt-1">{label}</p>
    </div>
  );
}

interface ResourceGaugesProps {
  cpu: number;
  memory: number;
  gpu: number;
}

export function ResourceGauges({ cpu, memory, gpu }: ResourceGaugesProps) {
  return (
    <div className="rounded-xl border border-gray-700 bg-gray-800 p-5">
      <h3 className="text-lg font-semibold mb-4">리소스 사용률</h3>
      <div className="grid grid-cols-3 gap-4">
        <SingleGauge label="CPU" value={cpu} color="#3b82f6" />
        <SingleGauge label="Memory" value={memory} color="#8b5cf6" />
        <SingleGauge label="GPU" value={gpu} color="#10b981" />
      </div>
    </div>
  );
}
