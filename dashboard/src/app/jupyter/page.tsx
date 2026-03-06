"use client";

import { useEffect, useState, useCallback } from "react";
import { Plus, ExternalLink, Trash2 } from "lucide-react";
import { listJupyter, deleteJupyter } from "@/lib/api";
import { useToast } from "@/components/Toast";
import { JupyterPhaseBadge } from "@/components/JupyterPhaseBadge";
import { JupyterCreateForm } from "@/components/JupyterCreateForm";
import type { JupyterSession } from "@/types";

export default function JupyterPage() {
  const { showToast } = useToast();
  const [sessions, setSessions] = useState<JupyterSession[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [deleting, setDeleting] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const data = await listJupyter();
      setSessions(data ?? []);
    } catch {
      /* silent */
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
    const iv = setInterval(refresh, 15_000);
    return () => clearInterval(iv);
  }, [refresh]);

  const handleDelete = async (name: string) => {
    if (!confirm(`"${name}" 세션을 삭제하시겠습니까?`)) return;
    setDeleting(name);
    try {
      await deleteJupyter(name);
      showToast("success", `"${name}" 삭제 완료`);
      refresh();
    } catch (err) {
      showToast("error", err instanceof Error ? err.message : "삭제 실패");
    } finally {
      setDeleting(null);
    }
  };

  const openSession = (s: JupyterSession) => {
    if (!s.url) return;
    const url = s.token ? `${s.url}?token=${s.token}` : s.url;
    window.open(url, "_blank");
  };

  const fmt = (iso: string) => {
    try {
      return new Date(iso).toLocaleString("ko-KR", { dateStyle: "short", timeStyle: "short" });
    } catch {
      return iso;
    }
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold">Jupyter Sessions</h2>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition-colors"
        >
          <Plus className="w-4 h-4" />
          새 세션
        </button>
      </div>

      {loading ? (
        <div className="text-gray-400 text-center py-20">로딩 중...</div>
      ) : sessions.length === 0 ? (
        <div className="text-gray-400 text-center py-20">
          <p className="text-lg mb-2">세션이 없습니다</p>
          <p className="text-sm">새 Jupyter 세션을 생성하세요.</p>
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {sessions.map((s) => (
            <div
              key={s.name}
              className={`bg-gray-800 rounded-xl border border-gray-700 p-5 transition-colors ${
                s.phase === "Running" ? "border-l-4 border-l-green-500" : ""
              }`}
            >
              <div className="flex items-start justify-between mb-3">
                <div>
                  <h3 className="font-semibold text-white">{s.name}</h3>
                  <p className="text-xs text-gray-500 mt-1">{fmt(s.createdAt)}</p>
                </div>
                <JupyterPhaseBadge phase={s.phase} />
              </div>

              {s.image && (
                <p className="text-xs text-gray-500 mb-3 truncate" title={s.image}>
                  {s.image}
                </p>
              )}

              <div className="flex items-center gap-2 mt-4">
                {s.phase === "Running" && s.url && (
                  <button
                    onClick={() => openSession(s)}
                    className="flex items-center gap-1.5 px-3 py-1.5 bg-green-600 hover:bg-green-700 rounded-lg text-xs font-medium transition-colors"
                  >
                    <ExternalLink className="w-3.5 h-3.5" />
                    열기
                  </button>
                )}
                <button
                  onClick={() => handleDelete(s.name)}
                  disabled={deleting === s.name}
                  className="flex items-center gap-1.5 px-3 py-1.5 bg-red-600/20 hover:bg-red-600/40 text-red-400 rounded-lg text-xs font-medium transition-colors disabled:opacity-50 ml-auto"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                  {deleting === s.name ? "삭제 중..." : "삭제"}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      <JupyterCreateForm
        open={showCreate}
        onClose={() => setShowCreate(false)}
        onCreated={refresh}
      />
    </div>
  );
}
