"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { getJob, cancelJob, retryJob } from "@/lib/api";
import { StatusBadge } from "@/components/StatusBadge";
import { CodeViewer } from "@/components/CodeViewer";
import { JobResult } from "@/components/JobResult";
import { LogViewer } from "@/components/LogViewer";
import { useToast } from "@/components/Toast";
import {
  ArrowLeft,
  RefreshCw,
  Clock,
  Cpu,
  AlertTriangle,
  Server,
  XCircle,
  RotateCcw,
} from "lucide-react";
import type { Job, JobStatus } from "@/types";

const CANCELABLE: JobStatus[] = ["pending", "running"];
const RETRYABLE: JobStatus[] = ["failed", "cancelled"];

export default function JobDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = params.id as string;
  const { showToast } = useToast();

  const [job, setJob] = useState<Job | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<"result" | "logs">("logs");
  const [showCancelConfirm, setShowCancelConfirm] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);

  useEffect(() => {
    if (!id) return;
    setLoading(true);
    getJob(id)
      .then((data) => {
        setJob(data);
        setError(null);
        setActiveTab(data.status === "completed" ? "result" : "logs");
      })
      .catch((e) =>
        setError(e instanceof Error ? e.message : "Failed to fetch job")
      )
      .finally(() => setLoading(false));
  }, [id]);

  const handleCancel = async () => {
    setShowCancelConfirm(false);
    setActionLoading(true);
    try {
      await cancelJob(id);
      showToast("success", "Job이 취소되었습니다");
      const updated = await getJob(id);
      setJob(updated);
    } catch (e) {
      showToast("error", e instanceof Error ? e.message : "취소 실패");
    } finally {
      setActionLoading(false);
    }
  };

  const handleRetry = async () => {
    setActionLoading(true);
    try {
      const newJob = await retryJob(id);
      showToast("success", "Job이 재시도되었습니다");
      router.push(`/jobs/${newJob.id}`);
    } catch (e) {
      showToast("error", e instanceof Error ? e.message : "재시도 실패");
    } finally {
      setActionLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <RefreshCw className="w-6 h-6 animate-spin text-gray-400" />
      </div>
    );
  }

  if (error || !job) {
    return (
      <div className="space-y-4">
        <button
          onClick={() => router.push("/jobs")}
          className="flex items-center gap-2 text-gray-400 hover:text-white transition-colors"
        >
          <ArrowLeft className="w-4 h-4" />
          Job 목록
        </button>
        <div className="rounded-lg bg-red-500/10 border border-red-500/30 p-6 text-red-400">
          {error ?? "Job을 찾을 수 없습니다"}
        </div>
      </div>
    );
  }

  const formatTime = (iso?: string) =>
    iso ? new Date(iso).toLocaleString("ko-KR") : "—";

  const canCancel = CANCELABLE.includes(job.status);
  const canRetry = RETRYABLE.includes(job.status);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <button
          onClick={() => router.push("/jobs")}
          className="p-2 rounded-lg hover:bg-gray-800 transition-colors"
        >
          <ArrowLeft className="w-5 h-5 text-gray-400" />
        </button>
        <div className="flex-1">
          <h2 className="text-2xl font-bold">{job.name}</h2>
          <p className="text-sm text-gray-400 font-mono">{job.id}</p>
        </div>
        <StatusBadge status={job.status} />
      </div>

      {/* Action Buttons */}
      <div className="flex items-center gap-3">
        {canCancel && (
          <button
            onClick={() => setShowCancelConfirm(true)}
            disabled={actionLoading}
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-red-500/10 border border-red-500/30 text-red-400 hover:bg-red-500/20 transition-colors disabled:opacity-50"
          >
            <XCircle className="w-4 h-4" />
            취소
          </button>
        )}
        {canRetry && (
          <button
            onClick={handleRetry}
            disabled={actionLoading}
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-blue-500/10 border border-blue-500/30 text-blue-400 hover:bg-blue-500/20 transition-colors disabled:opacity-50"
          >
            <RotateCcw className="w-4 h-4" />
            재시도
          </button>
        )}
      </div>

      {/* Cancel Confirmation Modal */}
      {showCancelConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="rounded-xl border border-gray-700 bg-gray-800 p-6 max-w-sm w-full mx-4 space-y-4">
            <h3 className="text-lg font-semibold">Job 취소 확인</h3>
            <p className="text-sm text-gray-400">
              &quot;{job.name}&quot; Job을 취소하시겠습니까? 이 작업은 되돌릴 수 없습니다.
            </p>
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setShowCancelConfirm(false)}
                className="px-4 py-2 rounded-lg text-sm text-gray-400 hover:text-white transition-colors"
              >
                닫기
              </button>
              <button
                onClick={handleCancel}
                className="px-4 py-2 rounded-lg bg-red-600 text-white text-sm hover:bg-red-700 transition-colors"
              >
                취소 확인
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Meta Info */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <InfoCard icon={Clock} label="생성일" value={formatTime(job.createdAt)} />
        <InfoCard icon={Clock} label="시작" value={formatTime(job.startedAt)} />
        <InfoCard icon={Clock} label="완료" value={formatTime(job.completedAt)} />
        <InfoCard icon={Server} label="노드" value={job.nodeId ?? "미할당"} />
      </div>

      {/* Circuit Info */}
      <div className="rounded-xl border border-gray-700 bg-gray-800 p-6">
        <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <Cpu className="w-5 h-5 text-blue-400" />
          회로 정보
        </h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <Stat label="큐빗" value={job.qubits} />
          <Stat label="백엔드" value={job.backend} />
          <Stat label="Shots" value={job.shots} />
          <Stat label="상태" value={job.status} />
        </div>
      </div>

      {/* Error Message */}
      {job.status === "failed" && job.error && (
        <div className="rounded-xl border border-red-500/30 bg-red-500/5 p-6">
          <h3 className="text-lg font-semibold mb-2 flex items-center gap-2 text-red-400">
            <AlertTriangle className="w-5 h-5" />
            에러
          </h3>
          <pre className="text-sm text-red-300 whitespace-pre-wrap font-mono">
            {job.error}
          </pre>
        </div>
      )}

      {/* Circuit Code */}
      {job.circuit && (
        <div>
          <h3 className="text-lg font-semibold mb-3">회로 코드</h3>
          <CodeViewer code={job.circuit} language="python" />
        </div>
      )}

      {/* Tabs: Result | Logs */}
      <div>
        <div className="flex border-b border-gray-700 mb-4">
          {job.status === "completed" && (
            <button
              onClick={() => setActiveTab("result")}
              className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                activeTab === "result"
                  ? "border-purple-500 text-purple-400"
                  : "border-transparent text-gray-400 hover:text-white"
              }`}
            >
              결과
            </button>
          )}
          <button
            onClick={() => setActiveTab("logs")}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === "logs"
                ? "border-green-500 text-green-400"
                : "border-transparent text-gray-400 hover:text-white"
            }`}
          >
            로그
          </button>
        </div>

        {activeTab === "result" && job.status === "completed" && (
          <JobResult jobId={job.id} shots={job.shots} />
        )}
        {activeTab === "logs" && <LogViewer jobId={job.id} />}
      </div>
    </div>
  );
}

function InfoCard({
  icon: Icon,
  label,
  value,
}: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  value: string;
}) {
  return (
    <div className="rounded-lg border border-gray-700 bg-gray-800/50 p-4">
      <div className="flex items-center gap-2 mb-1">
        <Icon className="w-4 h-4 text-gray-400" />
        <span className="text-xs text-gray-400">{label}</span>
      </div>
      <p className="text-sm font-medium truncate">{value}</p>
    </div>
  );
}

function Stat({
  label,
  value,
}: {
  label: string;
  value: string | number;
}) {
  return (
    <div>
      <p className="text-xs text-gray-400 mb-1">{label}</p>
      <p className="text-lg font-semibold">{value}</p>
    </div>
  );
}
