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

  if (res.status === 401) {
    throw new Error("Unauthorized");
  }

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(`API error ${res.status}: ${body}`);
  }

  return res.json() as Promise<T>;
}

// Cluster
export async function getClusterStatus(): Promise<ClusterStatus> {
  // API returns: { status, nodes: { ready, total, pools }, jobs: { total, running, pending, completed, failed }, resources: { cpu_usage, gpu_usage, memory_usage }, version }
  const raw = await request<Record<string, unknown>>("/api/v1/cluster/status");
  const nodes = (raw.nodes ?? {}) as Record<string, unknown>;
  const jobs = (raw.jobs ?? {}) as Record<string, unknown>;
  return {
    name: (raw.version as string) ?? "qsim-cluster",
    status: (raw.status as ClusterStatus["status"]) ?? "healthy",
    totalNodes: (nodes.total as number) ?? 0,
    activeNodes: (nodes.ready as number) ?? 0,
    totalQubits: 0,
    runningJobs: (jobs.running as number) ?? 0,
    pendingJobs: (jobs.pending as number) ?? 0,
  };
}

export async function getNodes(): Promise<Node[]> {
  // API returns: { nodes: [...], total }
  const raw = await request<Record<string, unknown>>("/api/v1/cluster/nodes");
  const nodes = Array.isArray(raw.nodes) ? raw.nodes : Array.isArray(raw) ? raw : [];
  return nodes.map((n: Record<string, unknown>) => ({
    id: (n.name as string) ?? "",
    name: (n.name as string) ?? "",
    status: (n.status as Node["status"]) ?? "ready",
    pool: (n.pool as string) ?? "cpu",
    qubits: 0,
    backend: "qsim",
    activeJobs: (n.active_jobs as number) ?? 0,
    labels: (n.labels as Record<string, string>) ?? {},
    metrics: {
      cpuUsage: parsePercent(n.cpu_usage),
      memoryUsage: parsePercent(n.memory_usage),
      gpuUsage: 0,
      uptime: 0,
    },
  } as Node));
}

function parsePercent(v: unknown): number {
  if (typeof v === "number") return v;
  if (typeof v === "string") return parseInt(v.replace("%", ""), 10) || 0;
  return 0;
}

export async function getMetrics(): Promise<ClusterMetrics> {
  // API returns: { cluster: {...}, jobs: {...}, ... }
  // We return empty metric arrays since the API doesn't provide time-series data
  await request<Record<string, unknown>>("/api/v1/cluster/metrics");
  return {
    jobThroughput: [],
    queueDepth: [],
    avgExecutionTime: [],
    errorRate: [],
    nodeUtilization: [],
  };
}

// Jobs
export async function listJobs(): Promise<Job[]> {
  // API returns: { jobs: [...], total, page, limit }
  const raw = await request<Record<string, unknown>>("/api/v1/jobs");
  const jobs = Array.isArray(raw.jobs) ? raw.jobs : Array.isArray(raw) ? raw : [];
  return jobs.map((j: Record<string, unknown>) => ({
    id: (j.id as string) ?? "",
    name: (j.name as string) ?? (j.id as string)?.slice(0, 8) ?? "",
    status: (j.status as Job["status"]) ?? "pending",
    backend: (j.backend as string) ?? (j.language as string) ?? "qsim",
    shots: (j.shots as number) ?? 0,
    qubits: (j.qubits as number) ?? 0,
    createdAt: (j.created_at as string) ?? (j.createdAt as string) ?? "",
    startedAt: (j.started_at as string) ?? undefined,
    completedAt: (j.completed_at as string) ?? (j.updated_at as string) ?? undefined,
    error: (j.error as string) ?? undefined,
  } as Job));
}

export const getJob = (id: string) =>
  request<Job>(`/api/v1/jobs/${id}`);

export const createJob = (data: CreateJobRequest) =>
  request<Job>("/api/v1/jobs", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const cancelJob = (id: string) =>
  request<void>(`/api/v1/jobs/${id}`, { method: "DELETE" });

export const retryJob = (id: string) =>
  request<Job>(`/api/v1/jobs/${id}/retry`, { method: "POST" });

export const getJobResult = (id: string) =>
  request<JobResult>(`/api/v1/jobs/${id}/result`);

export const getJobLogs = (id: string) =>
  request<JobLog[]>(`/api/v1/jobs/${id}/logs`);

// Jupyter
export async function listJupyter(): Promise<JupyterSession[]> {
  const raw = await request<Record<string, unknown>>("/api/v1/jupyter");
  const sessions = Array.isArray(raw.sessions) ? raw.sessions : Array.isArray(raw) ? raw : [];
  return sessions as JupyterSession[];
}

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
