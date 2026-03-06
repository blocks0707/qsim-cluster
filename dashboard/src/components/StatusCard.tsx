"use client";

import { Activity, Server, Briefcase, CheckCircle, type LucideIcon } from "lucide-react";

interface StatusCardProps {
  icon: LucideIcon;
  label: string;
  value: string | number;
  status?: "green" | "yellow" | "red";
}

const statusColors = {
  green: "text-green-500 bg-green-500/10 border-green-500/30",
  yellow: "text-yellow-500 bg-yellow-500/10 border-yellow-500/30",
  red: "text-red-500 bg-red-500/10 border-red-500/30",
};

export function StatusCard({ icon: Icon, label, value, status = "green" }: StatusCardProps) {
  const colors = statusColors[status];
  return (
    <div className={`rounded-xl border bg-gray-800 p-5 ${colors.split(" ").slice(2).join(" ")} border`}>
      <div className="flex items-center gap-3 mb-2">
        <div className={`p-2 rounded-lg ${colors.split(" ").slice(0, 2).join(" ")}`}>
          <Icon className="w-5 h-5" />
        </div>
        <span className="text-sm text-gray-400">{label}</span>
      </div>
      <p className="text-2xl font-bold">{value}</p>
    </div>
  );
}

interface ClusterStatusCardsProps {
  clusterStatus: string;
  activeNodes: number;
  totalNodes: number;
  runningJobs: number;
  successRate: number;
}

export function ClusterStatusCards({
  clusterStatus,
  activeNodes,
  totalNodes,
  runningJobs,
  successRate,
}: ClusterStatusCardsProps) {
  const clusterColor = clusterStatus === "healthy" ? "green" : clusterStatus === "degraded" ? "yellow" : "red";
  const nodeColor = activeNodes === totalNodes ? "green" : activeNodes > 0 ? "yellow" : "red";
  const successColor = successRate >= 90 ? "green" : successRate >= 70 ? "yellow" : "red";

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      <StatusCard icon={Activity} label="클러스터 상태" value={clusterStatus} status={clusterColor} />
      <StatusCard icon={Server} label="노드 현황" value={`${activeNodes} / ${totalNodes}`} status={nodeColor} />
      <StatusCard icon={Briefcase} label="활성 Job" value={runningJobs} status="green" />
      <StatusCard icon={CheckCircle} label="성공률" value={`${successRate}%`} status={successColor} />
    </div>
  );
}
