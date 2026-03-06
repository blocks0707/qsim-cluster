"use client";

import { useState } from "react";
import { X } from "lucide-react";
import { createJupyter } from "@/lib/api";
import { useToast } from "@/components/Toast";

interface Props {
  open: boolean;
  onClose: () => void;
  onCreated: () => void;
}

export function JupyterCreateForm({ open, onClose, onCreated }: Props) {
  const { showToast } = useToast();
  const [loading, setLoading] = useState(false);
  const [form, setForm] = useState({
    name: "",
    image: "jupyter/scipy-notebook:latest",
    cpu: "1",
    memory: "2Gi",
    storage: "5Gi",
    timeout: "3600",
    packages: "",
  });

  if (!open) return null;

  const set = (k: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm((f) => ({ ...f, [k]: e.target.value }));

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.name.trim()) return;
    setLoading(true);
    try {
      const pkgs = form.packages
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean);
      await createJupyter({
        name: form.name.trim(),
        image: form.image,
        cpu: form.cpu,
        memory: form.memory,
        storage: form.storage,
        timeout: Number(form.timeout) || 3600,
        packages: pkgs.length > 0 ? pkgs : undefined,
      });
      showToast("success", `세션 "${form.name}" 생성 요청 완료`);
      onCreated();
      onClose();
      setForm((f) => ({ ...f, name: "", packages: "" }));
    } catch (err) {
      showToast("error", err instanceof Error ? err.message : "생성 실패");
    } finally {
      setLoading(false);
    }
  };

  const fields: { label: string; key: keyof typeof form; placeholder?: string; hint?: string }[] = [
    { label: "세션 이름", key: "name", placeholder: "my-notebook" },
    { label: "Image", key: "image" },
    { label: "CPU", key: "cpu", placeholder: "1" },
    { label: "Memory", key: "memory", placeholder: "2Gi" },
    { label: "Storage", key: "storage", placeholder: "5Gi" },
    { label: "Timeout (초)", key: "timeout", placeholder: "3600" },
    { label: "추가 패키지 (콤마 구분)", key: "packages", placeholder: "numpy, pandas", hint: "qiskit, qiskit-aer는 자동 포함" },
  ];

  return (
    <div className="fixed inset-0 z-40 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
      <div className="relative bg-gray-800 border border-gray-700 rounded-xl shadow-2xl w-full max-w-lg p-6 z-50">
        <div className="flex items-center justify-between mb-6">
          <h3 className="text-lg font-semibold">새 Jupyter 세션</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-white">
            <X className="w-5 h-5" />
          </button>
        </div>
        <form onSubmit={submit} className="space-y-4">
          {fields.map((f) => (
            <div key={f.key}>
              <label className="block text-sm text-gray-300 mb-1">{f.label}</label>
              <input
                value={form[f.key]}
                onChange={set(f.key)}
                placeholder={f.placeholder}
                className="w-full bg-gray-900 border border-gray-600 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-blue-500"
              />
              {f.hint && <p className="text-xs text-gray-500 mt-1">{f.hint}</p>}
            </div>
          ))}
          <button
            type="submit"
            disabled={loading || !form.name.trim()}
            className="w-full py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 rounded-lg text-sm font-medium transition-colors"
          >
            {loading ? "생성 중..." : "세션 생성"}
          </button>
        </form>
      </div>
    </div>
  );
}
