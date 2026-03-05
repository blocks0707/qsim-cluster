# API 레퍼런스

## 개요

qsim-cluster API Server는 Go + Gin 프레임워크 기반의 REST API를 제공합니다. 기본 포트는 `8080`이며, 모든 API 엔드포인트는 `/api/v1` 프리픽스를 사용합니다.

---

## 미들웨어

| 미들웨어 | 설명 |
|----------|------|
| `Logger` | 구조화 로깅 (zap) — 요청 method, path, status, latency 기록 |
| `Recovery` | panic 복구 — 500 응답 반환 |
| `CORS` | Cross-Origin 요청 허용 — `Access-Control-Allow-Origin: *` |
| `Auth` | JWT 인증 (현재 placeholder — `user_id: "user-123"` 고정) |

> **참고**: Auth 미들웨어는 아직 구현되지 않았으며, 모든 요청에 `user_id: "user-123"`이 자동 설정됩니다.

---

## 엔드포인트 목록

### Health Check

#### `GET /health`

서버 상태 확인.

**Response** `200 OK`
```json
{
  "status": "ok",
  "service": "qsim-api-server"
}
```

---

### Job 관리 (`handlers/jobs.go`)

#### `POST /api/v1/jobs`

양자 시뮬레이션 Job 생성.

**Request Body**
```json
{
  "code": "from qiskit import QuantumCircuit\nqc = QuantumCircuit(2)\nqc.h(0)\nqc.cx(0,1)",
  "language": "python",
  "options": {}
}
```

| 필드 | 타입 | 필수 | 설명 |
|------|------|------|------|
| `code` | string | ✅ | 양자 회로 소스 코드 |
| `language` | string | ❌ | 기본값: `"python"` |
| `options` | object | ❌ | 추가 옵션 |

**Response** `201 Created`
```json
{
  "job_id": "uuid",
  "status": "submitted",
  "message": "Job submitted successfully",
  "analysis": {
    "qubits": 2,
    "complexity_class": "A",
    "estimated_time_sec": 5,
    "recommended_pool": "cpu"
  }
}
```

**처리 흐름**: 회로 분석 → DB 저장 → QuantumJob CR 생성 → 상태 업데이트

---

#### `GET /api/v1/jobs`

사용자의 Job 목록 조회.

**Query Parameters**

| 파라미터 | 타입 | 설명 |
|----------|------|------|
| `page` | int | 페이지 번호 (기본값: 1) |
| `limit` | int | 페이지당 항목 수 (기본값: 10, 최대: 100) |
| `status` | string | 상태 필터 (pending, running, completed, failed) |

**Response** `200 OK`
```json
{
  "jobs": [
    {
      "id": "uuid",
      "user_id": "user-123",
      "status": "completed",
      "language": "python",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:01:00Z"
    }
  ],
  "total": 42,
  "page": 1,
  "limit": 10
}
```

---

#### `GET /api/v1/jobs/:id`

특정 Job 상세 조회.

**Response** `200 OK`
```json
{
  "id": "uuid",
  "user_id": "user-123",
  "status": "completed",
  "code": "...",
  "language": "python",
  "error_message": "",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:01:00Z"
}
```

**에러**: `404` Job 미존재, `403` 권한 없음

---

#### `DELETE /api/v1/jobs/:id`

Job 취소. `completed`, `failed`, `cancelled` 상태의 Job은 취소할 수 없음.

**Response** `200 OK`
```json
{
  "message": "Job cancelled successfully"
}
```

**에러**: `400` 취소 불가능한 상태, `404` Job 미존재, `403` 권한 없음

---

#### `POST /api/v1/jobs/:id/retry`

실패/취소된 Job 재시도. 새로운 Job ID가 생성됨.

**Response** `201 Created`
```json
{
  "original_job_id": "uuid-1",
  "new_job_id": "uuid-2",
  "status": "submitted",
  "message": "Job retry submitted successfully"
}
```

**에러**: `400` 재시도 불가능한 상태 (failed, cancelled만 가능)

---

#### `GET /api/v1/jobs/:id/result`

완료된 Job의 실행 결과 조회.

**Response** `200 OK`
```json
{
  "job_id": "uuid",
  "status": "completed",
  "execution_time": 1500,
  "started_at": "...",
  "completed_at": "...",
  "assigned_node": "node-1",
  "assigned_pool": "cpu",
  "result": {
    "counts": {"00": 256, "01": 244, "10": 255, "11": 269},
    "shots": 1024,
    "success": true
  },
  "metadata": {
    "qubits": 2,
    "depth": 3,
    "gate_count": 4,
    "method": "statevector"
  }
}
```

**에러**: `400` Job 미완료

---

#### `GET /api/v1/jobs/:id/logs`

Job 실행 로그 조회.

**Query Parameters**

| 파라미터 | 타입 | 설명 |
|----------|------|------|
| `since` | string | 시간 범위 (예: `"1h"`, `"30m"`) |
| `tail` | string | 마지막 N줄 |
| `follow` | string | `"true"` — 스트리밍 (미구현) |

**Response** `200 OK`
```json
{
  "job_id": "uuid",
  "status": "running",
  "logs": "...",
  "timestamp": "2024-01-01T00:00:00Z",
  "source": "kubernetes"
}
```

---

### 회로 분석 (`handlers/analysis.go`)

#### `POST /api/v1/analyze`

양자 회로 복잡도 분석 (실행 없이).

**Request Body**
```json
{
  "code": "from qiskit import QuantumCircuit\nqc = QuantumCircuit(5)\nqc.h(0)",
  "language": "python"
}
```

**Response** `200 OK`
```json
{
  "qubits": 5,
  "depth": 1,
  "gate_count": 1,
  "cx_count": 0,
  "parallelism": 0.2,
  "memory_bytes": 512,
  "complexity_class": "A",
  "recommended_method": "statevector",
  "estimated_cpu": 1,
  "estimated_memory_mb": 512,
  "estimated_time_sec": 5,
  "recommended_pool": "cpu"
}
```

> Analyzer 서비스가 비가용 시 fallback 추정값이 반환됩니다.

---

### 클러스터 (`handlers/cluster.go`)

#### `GET /api/v1/cluster/status`

클러스터 전체 상태 조회.

**Response** `200 OK`
```json
{
  "status": "healthy",
  "version": "v0.1.0",
  "nodes": {
    "total": 3,
    "ready": 3,
    "pools": {"cpu": 2, "gpu": 1}
  },
  "jobs": {
    "total": 100,
    "pending": 5,
    "running": 10,
    "completed": 80,
    "failed": 5
  },
  "resources": {
    "cpu_usage": "25%",
    "memory_usage": "40%",
    "gpu_usage": "0%"
  }
}
```

---

#### `GET /api/v1/cluster/nodes`

클러스터 노드 목록 조회.

**Query Parameters**

| 파라미터 | 타입 | 설명 |
|----------|------|------|
| `pool` | string | 노드 풀 필터 (`cpu`, `high-cpu`, `gpu`) |
| `status` | string | 상태 필터 (`ready`, `not-ready`) |

**Response** `200 OK`
```json
{
  "nodes": [
    {
      "name": "node-1",
      "pool": "cpu",
      "status": "ready",
      "cpu_cores": 4,
      "memory_gb": 8,
      "gpu": false,
      "active_jobs": 1,
      "cpu_usage": "20%",
      "memory_usage": "35%"
    }
  ],
  "total": 1
}
```

---

#### `GET /api/v1/cluster/metrics`

클러스터 메트릭 조회.

**Response** `200 OK`
```json
{
  "cluster": {
    "total_cpu_cores": 16,
    "total_memory_gb": 32,
    "total_gpus": 1,
    "total_nodes": 3,
    "ready_nodes": 3,
    "cpu_utilization": 0.25,
    "memory_utilization": 0.40,
    "gpu_utilization": 0.0
  },
  "jobs": {
    "total_jobs": 100,
    "completed_jobs": 80,
    "failed_jobs": 5,
    "success_rate": 0.8,
    "avg_execution_time_sec": 45,
    "jobs_per_minute": 1.67
  },
  "complexity_distribution": {
    "class_a": 40,
    "class_b": 35,
    "class_c": 20,
    "class_d": 5
  },
  "pools": {"cpu": 2, "gpu": 1}
}
```

---

### WebSocket (`handlers/websocket.go`)

#### `GET /ws/jobs/:id`

Job 실시간 상태 업데이트 (WebSocket).

> ⚠️ **현재 미구현** — `501 Not Implemented` 반환

---

## 에러 코드

| HTTP 코드 | 의미 | 예시 |
|-----------|------|------|
| `400` | Bad Request | 잘못된 요청 본문, 취소/재시도 불가능한 상태 |
| `401` | Unauthorized | 인증되지 않은 사용자 |
| `403` | Forbidden | 다른 사용자의 Job 접근 |
| `404` | Not Found | 존재하지 않는 Job |
| `500` | Internal Server Error | DB 오류, K8s CR 생성 실패 |
| `501` | Not Implemented | WebSocket (미구현) |
| `503` | Service Unavailable | K8s 클러스터 연결 실패 |

에러 응답 형식:
```json
{
  "error": "에러 메시지",
  "details": "상세 정보 (선택)"
}
```
