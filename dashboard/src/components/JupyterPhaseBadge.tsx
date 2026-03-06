"use client";

import type { JupyterPhase } from "@/types";

const phaseConfig: Record<JupyterPhase, { bg: string; text: string; dot: string }> = {
  Pending:      { bg: "bg-yellow-500/10", text: "text-yellow-500", dot: "bg-yellow-500" },
  Provisioning: { bg: "bg-blue-500/10",   text: "text-blue-500",  dot: "bg-blue-500" },
  Running:      { bg: "bg-green-500/10",  text: "text-green-500", dot: "bg-green-500" },
  Stopping:     { bg: "bg-orange-500/10", text: "text-orange-500", dot: "bg-orange-500" },
  Stopped:      { bg: "bg-gray-500/10",   text: "text-gray-400",  dot: "bg-gray-400" },
  Failed:       { bg: "bg-red-500/10",    text: "text-red-500",   dot: "bg-red-500" },
};

export function JupyterPhaseBadge({ phase }: { phase: JupyterPhase }) {
  const c = phaseConfig[phase] ?? phaseConfig.Pending;
  const pulse = phase === "Running" || phase === "Provisioning";

  return (
    <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium ${c.bg} ${c.text}`}>
      <span className={`w-1.5 h-1.5 rounded-full ${c.dot} ${pulse ? "animate-pulse" : ""}`} />
      {phase}
    </span>
  );
}
