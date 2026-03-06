"use client";

import { useEffect, useState, useRef } from "react";
import { getJobLogs } from "@/lib/api";
import { RefreshCw, Terminal } from "lucide-react";
import type { JobLog } from "@/types";

const LEVEL_COLORS: Record<string, string> = {
  info: "text-green-400",
  warn: "text-yellow-400",
  error: "text-red-400",
};

export function LogViewer({ jobId }: { jobId: string }) {
  const [logs, setLogs] = useState<JobLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setLoading(true);
    getJobLogs(jobId)
      .then((data) => {
        setLogs(data);
        setError(null);
      })
      .catch((e) => setError(e instanceof Error ? e.message : "Failed to load logs"))
      .finally(() => setLoading(false));
  }, [jobId]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [logs]);

  const refresh = () => {
    getJobLogs(jobId)
      .then(setLogs)
      .catch(() => {});
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <RefreshCw className="w-6 h-6 animate-spin text-gray-400" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg bg-red-500/10 border border-red-500/30 p-6 text-red-400">
        {error}
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <h4 className="text-lg font-semibold flex items-center gap-2">
          <Terminal className="w-5 h-5 text-green-400" />
          로그
        </h4>
        <button
          onClick={refresh}
          className="flex items-center gap-1 text-sm text-gray-400 hover:text-white transition-colors"
        >
          <RefreshCw className="w-4 h-4" />
          새로고침
        </button>
      </div>
      <div className="rounded-xl border border-gray-700 bg-black p-4 max-h-96 overflow-y-auto font-mono text-sm">
        {logs.length === 0 ? (
          <p className="text-gray-500">로그가 없습니다</p>
        ) : (
          logs.map((log, i) => (
            <div key={i} className="flex gap-3 py-0.5 hover:bg-gray-900/50">
              <span className="text-gray-500 shrink-0 text-xs leading-5">
                {new Date(log.timestamp).toLocaleTimeString("ko-KR", {
                  hour12: false,
                  hour: "2-digit",
                  minute: "2-digit",
                  second: "2-digit",
                })}
              </span>
              <span
                className={`shrink-0 uppercase text-xs leading-5 w-12 ${
                  LEVEL_COLORS[log.level] ?? "text-gray-400"
                }`}
              >
                {log.level}
              </span>
              <span className="text-green-400">{log.message}</span>
            </div>
          ))
        )}
        <div ref={bottomRef} />
      </div>
    </div>
  );
}
