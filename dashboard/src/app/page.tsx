"use client";

import { useEffect, useState, useCallback } from "react";
import { getClusterStatus, getMetrics, listJobs, getNodes } from "@/lib/api";
import type { ClusterStatus, ClusterMetrics, Job, Node } from "@/types";
import { ClusterStatusCards } from "@/components/StatusCard";
import { ResourceGauges } from "@/components/ResourceGauge";
import { JobStats } from "@/components/JobStats";
import { RecentJobs } from "@/components/RecentJobs";
import { RefreshCw, AlertTriangle } from "lucide-react";

// Mock / fallback data
const MOCK_STATUS: ClusterStatus = {
  name: "qsim-dev",
  status: "healthy",
  totalNodes: 4,
  activeNodes: 3,
  totalQubits: 128,
  runningJobs: 2,
  pendingJobs: 5,
};

const MOCK_JOBS: Job[] = [
  { id: "j1", name: "bell-state-exp", status: "running", backend: "qsim", shots: 1024, qubits: 2, createdAt: new Date(Date.now() - 300000).toISOString() },
  { id: "j2", name: "grover-search", status: "completed", backend: "qsim", shots: 2048, qubits: 8, createdAt: new Date(Date.now() - 3600000).toISOString() },
  { id: "j3", name: "vqe-h2", status: "pending", backend: "qsim-gpu", shots: 4096, qubits: 4, createdAt: new Date(Date.now() - 7200000).toISOString() },
  { id: "j4", name: "qft-benchmark", status: "failed", backend: "qsim", shots: 512, qubits: 16, createdAt: new Date(Date.now() - 86400000).toISOString() },
  { id: "j5", name: "shor-15", status: "completed", backend: "qsim-gpu", shots: 8192, qubits: 12, createdAt: new Date(Date.now() - 172800000).toISOString() },
];

const MOCK_NODES: Node[] = [
  { id: "n1", name: "node-0", status: "ready", pool: "cpu", qubits: 32, backend: "qsim", activeJobs: 1, labels: {}, metrics: { cpuUsage: 45, memoryUsage: 62, gpuUsage: 30, uptime: 86400 } },
  { id: "n2", name: "node-1", status: "busy", pool: "gpu", qubits: 32, backend: "qsim", activeJobs: 3, labels: {}, metrics: { cpuUsage: 78, memoryUsage: 71, gpuUsage: 85, uptime: 86400 } },
  { id: "n3", name: "node-2", status: "ready", pool: "gpu", qubits: 64, backend: "qsim-gpu", activeJobs: 0, labels: {}, metrics: { cpuUsage: 22, memoryUsage: 40, gpuUsage: 15, uptime: 43200 } },
  { id: "n4", name: "node-3", status: "offline", pool: "cpu", qubits: 32, backend: "qsim", activeJobs: 0, labels: {}, metrics: { cpuUsage: 0, memoryUsage: 0, gpuUsage: 0, uptime: 0 } },
];

export default function OverviewPage() {
  const [status, setStatus] = useState<ClusterStatus | null>(null);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [usingMock, setUsingMock] = useState(false);

  const fetchData = useCallback(async () => {
    try {
      const [s, j, n] = await Promise.all([getClusterStatus(), listJobs(), getNodes()]);
      setStatus(s);
      setJobs(j);
      setNodes(n);
      setUsingMock(false);
      setError(null);
    } catch {
      // Fallback to mock data
      setStatus(MOCK_STATUS);
      setJobs(MOCK_JOBS);
      setNodes(MOCK_NODES);
      setUsingMock(true);
      setError(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 30000);
    return () => clearInterval(interval);
  }, [fetchData]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <RefreshCw className="w-8 h-8 animate-spin text-gray-400" />
      </div>
    );
  }

  if (!status) {
    return (
      <div className="flex flex-col items-center justify-center h-64 gap-3">
        <AlertTriangle className="w-10 h-10 text-red-500" />
        <p className="text-red-400">클러스터 상태를 불러올 수 없습니다.</p>
        <button onClick={fetchData} className="text-sm text-blue-400 hover:text-blue-300">
          다시 시도
        </button>
      </div>
    );
  }

  // Compute resource averages from nodes
  const activeNodes = nodes.filter((n) => n.metrics && n.status !== "offline");
  const avgCpu = activeNodes.length > 0
    ? Math.round(activeNodes.reduce((s, n) => s + (n.metrics?.cpuUsage ?? 0), 0) / activeNodes.length)
    : 0;
  const avgMem = activeNodes.length > 0
    ? Math.round(activeNodes.reduce((s, n) => s + (n.metrics?.memoryUsage ?? 0), 0) / activeNodes.length)
    : 0;
  const avgGpu = activeNodes.length > 0
    ? Math.round(activeNodes.reduce((s, n) => s + (n.metrics?.gpuUsage ?? 0), 0) / activeNodes.length)
    : 0;

  // Job stats
  const jobCounts = { pending: 0, running: 0, completed: 0, failed: 0 };
  jobs.forEach((j) => {
    if (j.status in jobCounts) jobCounts[j.status as keyof typeof jobCounts]++;
  });

  const totalFinished = jobCounts.completed + jobCounts.failed;
  const successRate = totalFinished > 0 ? Math.round((jobCounts.completed / totalFinished) * 100) : 100;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">Overview</h2>
        <div className="flex items-center gap-3">
          {usingMock && (
            <span className="text-xs bg-yellow-500/20 text-yellow-400 px-2 py-1 rounded">Mock 데이터</span>
          )}
          <button
            onClick={fetchData}
            className="p-2 rounded-lg hover:bg-gray-800 text-gray-400 hover:text-white transition"
            title="새로고침"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
        </div>
      </div>

      <ClusterStatusCards
        clusterStatus={status.status}
        activeNodes={status.activeNodes}
        totalNodes={status.totalNodes}
        runningJobs={status.runningJobs}
        successRate={successRate}
      />

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <ResourceGauges cpu={avgCpu} memory={avgMem} gpu={avgGpu} />
        <JobStats {...jobCounts} />
      </div>

      <RecentJobs jobs={jobs} />
    </div>
  );
}
