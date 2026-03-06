"use client";

import Link from "next/link";
import type { Job, JobStatus } from "@/types";

const statusBadge: Record<JobStatus, string> = {
  pending: "bg-yellow-500/20 text-yellow-400",
  running: "bg-blue-500/20 text-blue-400",
  completed: "bg-green-500/20 text-green-400",
  failed: "bg-red-500/20 text-red-400",
  cancelled: "bg-gray-500/20 text-gray-400",
};

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "방금";
  if (mins < 60) return `${mins}분 전`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}시간 전`;
  return `${Math.floor(hours / 24)}일 전`;
}

interface RecentJobsProps {
  jobs: Job[];
}

export function RecentJobs({ jobs }: RecentJobsProps) {
  const recent = jobs.slice(0, 5);

  return (
    <div className="rounded-xl border border-gray-700 bg-gray-800 p-5">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold">최근 Job</h3>
        <Link href="/jobs" className="text-sm text-blue-400 hover:text-blue-300">
          전체 보기 →
        </Link>
      </div>
      {recent.length === 0 ? (
        <p className="text-gray-500 text-sm">Job이 없습니다.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-gray-400 border-b border-gray-700">
                <th className="text-left py-2 font-medium">이름</th>
                <th className="text-left py-2 font-medium">상태</th>
                <th className="text-right py-2 font-medium">큐빗</th>
                <th className="text-right py-2 font-medium">시간</th>
              </tr>
            </thead>
            <tbody>
              {recent.map((job) => (
                <tr key={job.id} className="border-b border-gray-700/50 hover:bg-gray-700/30">
                  <td className="py-2.5 font-medium truncate max-w-[160px]">{job.name}</td>
                  <td className="py-2.5">
                    <span className={`px-2 py-0.5 rounded-full text-xs ${statusBadge[job.status]}`}>
                      {job.status}
                    </span>
                  </td>
                  <td className="py-2.5 text-right text-gray-400">{job.qubits}</td>
                  <td className="py-2.5 text-right text-gray-400">{timeAgo(job.createdAt)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
