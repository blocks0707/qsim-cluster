"use client";

import { useEffect, useState, useCallback } from "react";
import { getMetrics, listJobs, getNodes } from "@/lib/api";
import type { ClusterMetrics, Job, Node } from "@/types";
import { PerformanceSummary } from "@/components/metrics/PerformanceSummary";
import { JobThroughput } from "@/components/metrics/JobThroughput";
import { ComplexityDistribution } from "@/components/metrics/ComplexityDistribution";
import { QubitDistribution } from "@/components/metrics/QubitDistribution";
import { ClusterResources } from "@/components/metrics/ClusterResources";

export default function MetricsPage() {
  const [metrics, setMetrics] = useState<ClusterMetrics | null>(null);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchData = useCallback(async () => {
    try {
      const [m, j, n] = await Promise.allSettled([getMetrics(), listJobs(), getNodes()]);
      if (m.status === "fulfilled") setMetrics(m.value);
      if (j.status === "fulfilled") setJobs(j.value);
      if (n.status === "fulfilled") setNodes(n.value);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
    const id = setInterval(fetchData, 60_000);
    return () => clearInterval(id);
  }, [fetchData]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">메트릭 & 트렌드</h2>
        <span className="text-xs text-gray-500">자동 새로고침 60초</span>
      </div>

      {/* 상단: 성능 요약 */}
      <PerformanceSummary jobs={jobs} />

      {/* 중단: Job 처리량 */}
      <JobThroughput jobs={jobs} />

      {/* 하단: 분포 차트 2열 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <ComplexityDistribution jobs={jobs} />
        <QubitDistribution jobs={jobs} />
      </div>

      {/* 최하단: 클러스터 리소스 */}
      <ClusterResources metrics={metrics} nodes={nodes} />
    </div>
  );
}
