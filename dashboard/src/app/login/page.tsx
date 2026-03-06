"use client";

import { useState } from "react";
import { useAuth } from "@/lib/auth";
import { Cpu, Loader2, CheckCircle, XCircle } from "lucide-react";

export default function LoginPage() {
  const { login } = useAuth();
  const [apiUrl, setApiUrl] = useState("http://localhost:8080");
  const [token, setToken] = useState("");
  const [testing, setTesting] = useState(false);
  const [error, setError] = useState("");

  const handleTest = async () => {
    setTesting(true);
    setError("");
    try {
      const res = await fetch(`${apiUrl.replace(/\/+$/, "")}/health`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      login(apiUrl.replace(/\/+$/, ""), token);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Connection failed");
    } finally {
      setTesting(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-950 via-indigo-950/50 to-gray-950 relative overflow-hidden">
      {/* Background decorations */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute -top-40 -right-40 w-80 h-80 bg-blue-500/10 rounded-full blur-3xl" />
        <div className="absolute -bottom-40 -left-40 w-80 h-80 bg-purple-500/10 rounded-full blur-3xl" />
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] bg-indigo-500/5 rounded-full blur-3xl" />
      </div>

      <div className="relative z-10 w-full max-w-md mx-4">
        <div className="bg-gray-900/80 backdrop-blur-xl border border-gray-800 rounded-2xl p-8 shadow-2xl">
          {/* Logo */}
          <div className="text-center mb-8">
            <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-blue-600/20 mb-4">
              <Cpu className="w-8 h-8 text-blue-400" />
            </div>
            <h1 className="text-2xl font-bold text-white">QSim Cluster</h1>
            <p className="text-sm text-gray-400 mt-1">Connect to your quantum simulation cluster</p>
          </div>

          {/* Form */}
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-1.5">API Endpoint</label>
              <input
                type="url"
                value={apiUrl}
                onChange={(e) => setApiUrl(e.target.value)}
                placeholder="http://localhost:8080"
                className="w-full px-4 py-2.5 bg-gray-800 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-1.5">Bearer Token</label>
              <input
                type="password"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder="Enter your API token"
                className="w-full px-4 py-2.5 bg-gray-800 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition"
              />
            </div>

            {error && (
              <div className="flex items-center gap-2 text-red-400 text-sm bg-red-400/10 px-3 py-2 rounded-lg">
                <XCircle className="w-4 h-4 flex-shrink-0" />
                {error}
              </div>
            )}

            <button
              onClick={handleTest}
              disabled={testing || !apiUrl || !token}
              className="w-full py-2.5 px-4 bg-blue-600 hover:bg-blue-500 disabled:bg-gray-700 disabled:text-gray-500 text-white font-medium rounded-lg transition flex items-center justify-center gap-2"
            >
              {testing ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  Testing connection...
                </>
              ) : (
                <>
                  <CheckCircle className="w-4 h-4" />
                  Connect
                </>
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
