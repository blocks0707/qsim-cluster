"use client";

import type { JobStatus } from "@/types";

const statusConfig: Record<
  JobStatus | "analyzing",
  { label: string; bg: string; text: string; dot: string }
> = {
  pending: {
    label: "Pending",
    bg: "bg-yellow-500/10",
    text: "text-yellow-500",
    dot: "bg-yellow-500",
  },
  running: {
    label: "Running",
    bg: "bg-blue-500/10",
    text: "text-blue-500",
    dot: "bg-blue-500",
  },
  completed: {
    label: "Completed",
    bg: "bg-green-500/10",
    text: "text-green-500",
    dot: "bg-green-500",
  },
  failed: {
    label: "Failed",
    bg: "bg-red-500/10",
    text: "text-red-500",
    dot: "bg-red-500",
  },
  cancelled: {
    label: "Cancelled",
    bg: "bg-gray-500/10",
    text: "text-gray-400",
    dot: "bg-gray-400",
  },
  analyzing: {
    label: "Analyzing",
    bg: "bg-purple-500/10",
    text: "text-purple-500",
    dot: "bg-purple-500",
  },
};

interface StatusBadgeProps {
  status: JobStatus | "analyzing";
}

export function StatusBadge({ status }: StatusBadgeProps) {
  const config = statusConfig[status] ?? statusConfig.pending;

  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium ${config.bg} ${config.text}`}
    >
      <span
        className={`w-1.5 h-1.5 rounded-full ${config.dot} ${
          status === "running" ? "animate-pulse" : ""
        }`}
      />
      {config.label}
    </span>
  );
}
