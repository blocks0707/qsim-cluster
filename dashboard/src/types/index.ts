// Cluster
export interface ClusterStatus {
  name: string;
  status: "healthy" | "degraded" | "error";
  totalNodes: number;
  activeNodes: number;
  totalQubits: number;
  runningJobs: number;
  pendingJobs: number;
}

export interface Node {
  id: string;
  name: string;
  status: "ready" | "busy" | "offline" | "error";
  qubits: number;
  backend: string;
  currentJob?: string;
  metrics?: NodeMetrics;
}

export interface NodeMetrics {
  cpuUsage: number;
  memoryUsage: number;
  gpuUsage?: number;
  uptime: number;
}

// Jobs
export type JobStatus =
  | "pending"
  | "running"
  | "completed"
  | "failed"
  | "cancelled";

export interface Job {
  id: string;
  name: string;
  status: JobStatus;
  circuit?: string;
  backend: string;
  shots: number;
  qubits: number;
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
  nodeId?: string;
  error?: string;
}

export interface JobResult {
  jobId: string;
  counts: Record<string, number>;
  statevector?: number[];
  metadata?: Record<string, unknown>;
}

export interface JobLog {
  timestamp: string;
  level: "info" | "warn" | "error";
  message: string;
}

export interface CreateJobRequest {
  name: string;
  circuit: string;
  backend: string;
  shots: number;
  priority?: number;
}

// Jupyter
export interface JupyterSession {
  name: string;
  status: "running" | "stopped" | "error";
  url?: string;
  createdAt: string;
  nodeId?: string;
}

export interface CreateJupyterRequest {
  name: string;
  nodeId?: string;
}

// Metrics
export interface MetricPoint {
  timestamp: string;
  value: number;
}

export interface ClusterMetrics {
  jobThroughput: MetricPoint[];
  queueDepth: MetricPoint[];
  avgExecutionTime: MetricPoint[];
  errorRate: MetricPoint[];
  nodeUtilization: MetricPoint[];
}

// Circuit Analysis
export interface CircuitAnalysis {
  qubits: number;
  depth: number;
  gates: Record<string, number>;
  estimatedTime: number;
  recommendedBackend: string;
}

export interface AnalyzeCircuitRequest {
  circuit: string;
  backend?: string;
}
