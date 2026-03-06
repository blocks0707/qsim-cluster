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

export type NodePool = "cpu" | "gpu" | "high-memory";

export interface Node {
  id: string;
  name: string;
  status: "ready" | "busy" | "offline" | "error";
  pool: NodePool;
  qubits: number;
  backend: string;
  currentJob?: string;
  activeJobs: number;
  labels: Record<string, string>;
  metrics?: NodeMetrics;
  gpu?: GpuInfo;
}

export interface GpuInfo {
  type: string;
  usage: number;
  memoryTotal: number;
  memoryUsed: number;
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
export type JupyterPhase =
  | "Pending"
  | "Provisioning"
  | "Running"
  | "Stopping"
  | "Stopped"
  | "Failed";

export interface JupyterSession {
  name: string;
  phase: JupyterPhase;
  url?: string;
  token?: string;
  createdAt: string;
  nodeId?: string;
  image?: string;
  cpu?: string;
  memory?: string;
  storage?: string;
}

export interface CreateJupyterRequest {
  name: string;
  image?: string;
  cpu?: string;
  memory?: string;
  storage?: string;
  timeout?: number;
  packages?: string[];
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
