"use client";

interface ProgressBarProps {
  label: string;
  value: number; // 0-100
  className?: string;
}

export function ProgressBar({ label, value, className = "" }: ProgressBarProps) {
  const clamped = Math.min(100, Math.max(0, value));
  const color =
    clamped > 80
      ? "bg-red-500"
      : clamped > 50
        ? "bg-yellow-500"
        : "bg-green-500";

  return (
    <div className={className}>
      <div className="flex justify-between text-xs text-gray-400 mb-1">
        <span>{label}</span>
        <span>{clamped.toFixed(1)}%</span>
      </div>
      <div className="w-full h-2 bg-gray-700 rounded-full overflow-hidden">
        <div
          className={`h-full rounded-full transition-all duration-500 ${color}`}
          style={{ width: `${clamped}%` }}
        />
      </div>
    </div>
  );
}
