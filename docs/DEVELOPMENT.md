# 개발 가이드

## 개발 환경 셋업

### 필수 도구 설치

```bash
# Go 1.21+
brew install go

# Python 3.11+
brew install python@3.11

# Docker
# Docker Desktop 설치: https://www.docker.com/products/docker-desktop

# Kubernetes 도구
brew install minikube kubernetes-cli

# GitHub CLI
brew install gh

# 개발 도구
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
pip install ruff pytest
```

### 의존성 설치

```bash
make install-deps
```

이 명령은 다음을 수행합니다:
- `api-server/` 및 `operator/`의 Go 모듈 정리 (`go mod tidy`)
- `analyzer/` 및 `runtime/`의 Python 패키지 설치
- golangci-lint, ruff, pytest 설치

---

## 코드 구조

```
qsim-cluster/
├── api-server/                    # REST API 서버 (Go)
│   ├── cmd/server/main.go         # 엔트리포인트
│   ├── internal/
│   │   ├── api/
│   │   │   ├── router.go          # Gin 라우터
│   │   │   ├── handlers/          # HTTP 핸들러 (jobs, cluster, analysis, websocket)
│   │   │   └── middleware/        # Logger, Recovery, CORS, Auth
│   │   ├── analyzer/client.go     # Analyzer 서비스 클라이언트
│   │   ├── k8s/client.go          # Kubernetes 클라이언트 래퍼
│   │   └── store/                 # PostgreSQL (JobStore) + Redis (CacheStore)
│   ├── Dockerfile
│   ├── go.mod / go.sum
│
├── operator/                      # K8s Operator (Go, controller-runtime)
│   ├── api/v1alpha1/              # CRD 타입 정의
│   │   ├── quantumjob_types.go
│   │   ├── quantumnodeprofile_types.go
│   │   └── zz_generated.deepcopy.go
│   ├── cmd/manager/main.go        # Operator 엔트리포인트
│   ├── internal/
│   │   ├── controller/            # Reconciler (QuantumJob, NodeProfile)
│   │   ├── scheduler/             # Predicates + Scorer
│   │   └── runtime/               # PodBuilder (Pod/ConfigMap 생성)
│   ├── config/crd/                # CRD YAML 매니페스트
│   ├── go.mod / go.sum
│
├── analyzer/                      # 회로 분석 서비스 (Python, FastAPI)
│   ├── src/
│   │   ├── server.py              # FastAPI 서버
│   │   ├── complexity.py          # ComplexityAnalyzer
│   │   └── interceptor.py         # AerSimulator 인터셉터
│   ├── requirements.txt
│   ├── Dockerfile
│
├── runtime/                       # 양자 회로 실행기 (Python)
│   ├── execute.py                 # QuantumExecutor
│   ├── requirements.txt
│   ├── Dockerfile
│
├── sdk/                           # Python SDK
│   ├── qsim/
│   │   ├── __init__.py
│   │   └── client.py              # QSimClient, QSimJob
│   └── setup.py
│
├── deploy/
│   └── minikube/setup.sh          # minikube 개발 환경 셋업 스크립트
│
├── .github/workflows/ci.yml      # GitHub Actions CI
├── docker-compose.yml             # 로컬 개발 환경
├── Makefile                       # 빌드/테스트/배포 자동화
├── CLAUDE.md                      # 개발 컨벤션
└── README.md
```

---

## 각 컴포넌트 로컬 빌드/실행

### API Server

```bash
cd api-server

# 빌드
go build -o bin/server ./cmd/server/main.go

# 실행 (PostgreSQL, Redis 필요)
./bin/server \
  --port 8080 \
  --postgres-url "postgres://qsim:qsim123@localhost:5432/qsim?sslmode=disable" \
  --redis-url "localhost:6379" \
  --analyzer-url "http://localhost:8000" \
  --log-level debug
```

환경 변수로도 설정 가능 (`QSIM_` 프리픽스):
```bash
export QSIM_PORT=8080
export QSIM_POSTGRES_URL="postgres://..."
```

### Operator

```bash
cd operator

# 빌드
go build -o bin/operator ./cmd/manager/main.go

# 실행 (kubeconfig 필요)
./bin/operator \
  --metrics-bind-address :8080 \
  --health-probe-bind-address :8081
```

> CRD가 먼저 클러스터에 설치되어 있어야 합니다.

### Analyzer

```bash
cd analyzer

# 의존성 설치
pip install -r requirements.txt

# 실행
python -m src.server
# 또는
uvicorn src.server:app --host 0.0.0.0 --port 8000 --reload
```

### Runtime

Runtime은 직접 실행하지 않고, Operator가 생성한 Pod 내부에서 실행됩니다. 로컬 테스트:

```bash
cd runtime
pip install -r requirements.txt

# 테스트용 코드 파일 준비
mkdir -p /tmp/code /tmp/results
echo 'from qiskit import QuantumCircuit; qc = QuantumCircuit(2); print("test")' > /tmp/code/circuit.py

# 환경 변수 설정 후 실행
JOB_ID=test-job python execute.py
```

### SDK

```bash
cd sdk
pip install -e ".[dev]"

# 사용 예시
python -c "
from qsim import QSimClient
client = QSimClient('http://localhost:8080')
print(client.cluster_status())
"
```

---

## 테스트

### Go 테스트

```bash
# 전체 테스트
cd api-server && go test -v ./...
cd operator && go test -v ./...

# 특정 패키지
cd operator && go test -v ./internal/scheduler/...

# 커버리지
cd operator && go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**Operator 테스트 구조:**
- `internal/controller/suite_test.go` — Ginkgo 테스트 스위트 (envtest 환경)
- `internal/controller/quantumjob_controller_test.go` — Job 생명주기, 검증, 이벤트, 조건 테스트
- `internal/scheduler/scorer_test.go` — ResourceFit, LoadBalance, PoolMatch, ScoreNodes, GetBestNode 테스트

### Python 테스트

```bash
# Analyzer
cd analyzer && python -m pytest tests/ -v

# Runtime
cd runtime && python -m pytest tests/ -v

# SDK
cd sdk && python -m pytest tests/ -v
```

### 전체 테스트

```bash
make test
```

### CI 파이프라인

GitHub Actions (`.github/workflows/ci.yml`)에서 자동 실행:

1. **go-lint-and-test**: golangci-lint → go test (api-server, operator)
2. **python-lint-and-test**: ruff check → pytest (analyzer, runtime)
3. **docker-build**: Docker 이미지 빌드 검증 + docker-compose config

---

## 코드 포매팅 & 린트

```bash
# 포매팅
make format

# 린트
make lint
```

개별 실행:
```bash
# Go
cd api-server && go fmt ./... && golangci-lint run ./...
cd operator && go fmt ./... && golangci-lint run ./...

# Python
cd analyzer && ruff format src/ && ruff check src/
cd runtime && ruff format *.py && ruff check *.py
```

---

## PR 작성 가이드

### 브랜치 전략

| 접두사 | 용도 |
|--------|------|
| `feat/*` | 새 기능 |
| `fix/*` | 버그 수정 |
| `docs/*` | 문서 변경 |
| `refactor/*` | 코드 리팩토링 |
| `test/*` | 테스트 추가 |

### Commit Message

[Conventional Commits](https://conventionalcommits.org/) 형식:

```
feat(api): add circuit complexity analyzer
fix(operator): resolve scheduling race condition
docs(readme): update installation instructions
test(scheduler): add unit tests for node scoring
refactor(scheduler): improve node scoring algorithm
```

### PR 프로세스

1. `main`에서 feature 브랜치 생성
2. 기능 구현 + 테스트 작성
3. 로컬 린트/테스트 통과 확인 (`make lint && make test`)
4. PR 생성: `gh pr create --title "feat: description"`
5. CI 전체 통과 확인
6. 코드 리뷰 1인 이상 승인
7. Squash merge to `main`

### 코드 리뷰 체크리스트

**Go 코드:**
- [ ] 에러 핸들링에 context 포함 (`fmt.Errorf("...: %w", err)`)
- [ ] 구조화 로깅 사용 (zap)
- [ ] 단위 테스트 80%+ 커버리지
- [ ] 하드코딩 값 없음
- [ ] 리소스 관리 (close, context 등)

**Python 코드:**
- [ ] 모든 함수에 type hints
- [ ] I/O 작업에 async/await (FastAPI)
- [ ] 적절한 예외 처리
- [ ] pytest 테스트 포함

**Kubernetes:**
- [ ] Resource limits/requests 정의
- [ ] RBAC 최소 권한
- [ ] 일관된 라벨/어노테이션
- [ ] Health check 정의

> 상세한 코드 스타일 및 아키텍처 가이드는 [CLAUDE.md](/CLAUDE.md)를 참조하세요.
