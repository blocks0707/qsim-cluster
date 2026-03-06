"use client";

import type { Job } from "@/types";

interface Props {
  jobs: Job[];
}

export function PerformanceSummary({ jobs }: Props) {
  const total = jobs.length;
  const completed = jobs.filter((j) => j.status === "completed").length;
  const failed = jobs.filter((j) => j.status === "failed").length;
  const successRate = total > 0 ? ((completed / (completed + failed)) * 100) : 0;

  // Average execution time from completed jobs
  const completedJobs = jobs.filter((j) => j.status === "completed" && j.startedAt && j.completedAt);
  const avgTime =
    completedJobs.length > 0
      ? completedJobs.reduce((sum, j) => {
          const start = new Date(j.startedAt!).getTime();
          const end = new Date(j.completedAt!).getTime();
          return sum + (end - start) / 1000;
        }, 0) / completedJobs.length
      : 0;

  // Jobs per minute (based on time span of all jobs)
  const timestamps = jobs.map((j) => new Date(j.createdAt).getTime()).sort();
  const spanMin = timestamps.length > 1 ? (timestamps[timestamps.length - 1] - timestamps[0]) / 60000 : 1;
  const jobsPerMin = total / Math.max(spanMin, 1);

  const cards = [
    { label: "총 Job 수", value: total.toLocaleString(), sub: `${completed} completed` },
    { label: "성공률", value: `${successRate.toFixed(1)}%`, sub: `${failed} failed` },
    { label: "평균 실행시간", value: `${avgTime.toFixed(1)}s`, sub: `${completedJobs.length} jobs 기준` },
    { label: "Jobs / 분", value: jobsPerMin.toFixed(2), sub: "throughput" },
  ];

  return (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
      {cards.map((c) => (
        <div key={c.label} className="rounded-xl border border-gray-700 bg-gray-800 p-5 flex flex-col items-center text-center">
          <p className="text-3xl font-bold text-white">{c.value}</p>
          <p className="text-sm font-medium text-gray-300 mt-1">{c.label}</p>
          <p className="text-xs text-gray-500 mt-0.5">{c.sub}</p>
        </div>
      ))}
    </div>
  );
}
