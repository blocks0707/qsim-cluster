import type {
  ClusterStatus,
  Node,
  ClusterMetrics,
  Job,
  JobResult,
  JobLog,
  CreateJobRequest,
  JupyterSession,
  CreateJupyterRequest,
  CircuitAnalysis,
  AnalyzeCircuitRequest,
} from "@/types";

function getBaseUrl(): string {
  if (typeof window !== "undefined") {
    return localStorage.getItem("qsim_api_url") || process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
  }
  return process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
}

function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("qsim_token");
}

async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${getBaseUrl()}${path}`, {
    ...options,
    headers,
  });

  if (res.status === 401 && typeof window !== "undefined") {
    localStorage.removeItem("qsim_api_url");
    localStorage.removeItem("qsim_token");
    window.location.href = "/login";
    throw new Error("Unauthorized");
  }

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(`API error ${res.status}: ${body}`);
  }

  return res.json() as Promise<T>;
}

// Cluster
export const getClusterStatus = () =>
  request<ClusterStatus>("/api/v1/cluster/status");

export const getNodes = () =>
  request<Node[]>("/api/v1/nodes");

export const getMetrics = () =>
  request<ClusterMetrics>("/api/v1/metrics");

// Jobs
export const listJobs = () =>
  request<Job[]>("/api/v1/jobs");

export const getJob = (id: string) =>
  request<Job>(`/api/v1/jobs/${id}`);

export const createJob = (data: CreateJobRequest) =>
  request<Job>("/api/v1/jobs", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const cancelJob = (id: string) =>
  request<void>(`/api/v1/jobs/${id}/cancel`, { method: "POST" });

export const retryJob = (id: string) =>
  request<Job>(`/api/v1/jobs/${id}/retry`, { method: "POST" });

export const getJobResult = (id: string) =>
  request<JobResult>(`/api/v1/jobs/${id}/result`);

export const getJobLogs = (id: string) =>
  request<JobLog[]>(`/api/v1/jobs/${id}/logs`);

// Jupyter
export const listJupyter = () =>
  request<JupyterSession[]>("/api/v1/jupyter");

export const createJupyter = (data: CreateJupyterRequest) =>
  request<JupyterSession>("/api/v1/jupyter", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const deleteJupyter = (name: string) =>
  request<void>(`/api/v1/jupyter/${name}`, { method: "DELETE" });

// Circuit Analysis
export const analyzeCircuit = (data: AnalyzeCircuitRequest) =>
  request<CircuitAnalysis>("/api/v1/analyze", {
    method: "POST",
    body: JSON.stringify(data),
  });
