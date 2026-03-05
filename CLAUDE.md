# CLAUDE.md - Development Guidelines

Claude Code configuration and development conventions for qsim-cluster.

## 🎯 Project Overview

Kubernetes-based quantum circuit simulation cluster with intelligent scheduling. This is a complex distributed system with Go, Python, and K8s components.

## 📋 Development Conventions

### Go Code Standards

- **Layout**: Follow [Go Standard Project Layout](https://github.com/golang-standards/project-layout)
- **Linting**: Use `golangci-lint` with strict configuration
- **Testing**: Unit tests required (minimum 80% coverage)
- **Logging**: Use structured logging (logrus/zap)
- **Error Handling**: Always wrap errors with context
- **Dependencies**: Use Go modules, pin versions

```go
// Example error handling
if err != nil {
    return fmt.Errorf("failed to create quantum job: %w", err)
}

// Example logging
log.WithFields(log.Fields{
    "job_id": jobID,
    "complexity_class": "B",
}).Info("quantum job scheduled successfully")
```

### Python Code Standards

- **Formatter**: Use `ruff` for formatting and linting
- **Type Hints**: Required for all functions and classes
- **Testing**: pytest with minimum 80% coverage
- **Dependencies**: Use `requirements.txt` with pinned versions
- **Async**: Use async/await for I/O operations in FastAPI

```python
# Example type hints
from typing import Optional, List, Dict, Any

async def analyze_circuit(circuit: QuantumCircuit) -> ComplexityResult:
    """Analyze quantum circuit complexity."""
    pass

# Example class definition
@dataclass
class ComplexityResult:
    qubits: int
    depth: int
    gate_count: int
    complexity_class: str
    estimated_memory_mb: int
```

### Kubernetes Standards

- **CRDs**: Use controller-gen for code generation
- **RBAC**: Minimal required permissions only
- **Resources**: Always set resource limits and requests
- **Labels**: Consistent labeling for observability
- **Namespace**: Use dedicated namespaces for separation

## 🔄 Git Workflow

### Branching Strategy

- `main`: Production-ready code (protected)
- `feat/*`: New features
- `fix/*`: Bug fixes
- `docs/*`: Documentation changes
- `refactor/*`: Code refactoring

### Commit Message Convention

Use [Conventional Commits](https://conventionalcommits.org/):

```
feat(api): add circuit complexity analyzer
fix(operator): resolve scheduling race condition
docs(readme): update installation instructions
test(analyzer): add unit tests for complexity estimation
refactor(scheduler): improve node scoring algorithm
```

### Pull Request Process

1. **Branch**: Create feature branch from `main`
2. **Code**: Implement feature with tests
3. **Lint**: Run linters locally before push
4. **PR**: Create PR with descriptive title and description
5. **CI**: All CI checks must pass (lint, test, security)
6. **Review**: At least one approval required
7. **Merge**: Squash merge to `main`

## 🛠️ Local Development

### Prerequisites

```bash
# Required tools
go version          # 1.21+
python --version    # 3.11+
docker --version
minikube version
kubectl version
gh --version
```

### Development Environment

```bash
# Start minikube
minikube start --memory=8192 --cpus=4

# Install development dependencies
make dev-deps

# Set up pre-commit hooks
make setup-hooks

# Run local development stack
make dev-up
```

### Testing Strategy

#### Unit Tests
- Go: `go test ./...`
- Python: `pytest`
- Coverage minimum: 80%

#### Integration Tests
- K8s integration tests in `test/` directory
- Use KIND cluster for CI
- Test CRD lifecycle, operator reconciliation

#### E2E Tests
- Full workflow: job submission → scheduling → execution → result
- Test different complexity classes
- Performance benchmarks

### Code Review Checklist

#### Go Code
- [ ] Error handling with proper context
- [ ] Structured logging
- [ ] Unit tests with good coverage
- [ ] No hardcoded values
- [ ] Resource management (close files, contexts)
- [ ] Thread safety for concurrent code

#### Python Code
- [ ] Type hints for all functions
- [ ] Async/await for I/O operations
- [ ] Proper exception handling
- [ ] Unit tests with mocks
- [ ] Input validation

#### Kubernetes
- [ ] Resource limits defined
- [ ] RBAC permissions minimal
- [ ] Labels and annotations consistent
- [ ] Health checks defined
- [ ] Observability (logs, metrics)

#### Architecture
- [ ] Follows separation of concerns
- [ ] Error handling at boundaries
- [ ] Backwards compatible changes
- [ ] Performance impact considered
- [ ] Security implications reviewed

## 📊 Monitoring & Observability

### Logging Standards

```go
// Structured logging in Go
logger.WithFields(logrus.Fields{
    "component": "scheduler",
    "job_id": job.ID,
    "node": selectedNode,
    "complexity_class": job.ComplexityClass,
}).Info("job scheduled successfully")
```

```python
# Structured logging in Python
import structlog
logger = structlog.get_logger()
logger.info("circuit analyzed", 
           qubits=circuit.num_qubits,
           depth=circuit.depth(),
           complexity_class="B")
```

### Metrics

- Use Prometheus metrics for observability
- Track job queue depth, execution times, resource utilization
- Custom metrics for quantum-specific data (circuit complexity distribution)

### Health Checks

- Kubernetes readiness/liveness probes
- Dependency health checks (database, Redis)
- Circuit analyzer service health

## 🔒 Security Guidelines

### Code Security
- No secrets in code (use K8s secrets)
- Input validation and sanitization
- Container security (non-root user, minimal base images)
- Network policies for pod-to-pod communication

### RBAC
- Service accounts with minimal permissions
- Role-based access for different user types
- OAuth 2.0 integration for user authentication

## 🚀 CI/CD Pipeline

### GitHub Actions Workflow

```yaml
# .github/workflows/ci.yml
on: [push, pull_request]

jobs:
  go-test:
    # Go linting, testing, building
  
  python-test:
    # Python linting, testing, type checking
  
  k8s-test:
    # KIND cluster, CRD installation, operator testing
  
  security:
    # Container scanning, dependency checking
  
  build:
    # Docker image building and pushing
```

### Deployment
- ArgoCD for GitOps deployment
- Staging environment for integration testing
- Rolling updates with health checks

## 🎯 Performance Targets

### API Server
- Response time < 100ms for job submission
- Support 1000+ concurrent jobs
- Graceful degradation under load

### Operator
- Reconciliation < 5 seconds for job state changes
- Handle 100+ simultaneous jobs
- Efficient resource usage

### Circuit Analyzer
- Analysis time < 10 seconds for complex circuits
- Memory usage < 1GB per analysis
- Stateless and horizontally scalable

## 📚 Key Learning Resources

- [Kubernetes Operators](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- [Qiskit Aer Documentation](https://qiskit.org/ecosystem/aer/)
- [Go Best Practices](https://github.com/golang/go/wiki/CodeReviewComments)
- [Python Type Hints](https://docs.python.org/3/library/typing.html)

---

## 🎪 Claude Agent Instructions

When working on this project:

1. **Always read the architecture document** in `/docs/ARCHITECTURE.md` first
2. **Follow the exact directory structure** defined in the project layout
3. **Use conventional commits** for all changes
4. **Create PRs** instead of pushing directly to main
5. **Test thoroughly** before submitting PR
6. **Consider performance implications** for distributed systems
7. **Security first** - validate inputs, use secure defaults
8. **Document everything** - code, APIs, deployment procedures

### Typical Development Tasks

```bash
# Create new feature
git checkout -b feat/node-scoring-algorithm
# ... implement feature ...
git add . && git commit -m "feat(scheduler): implement node scoring algorithm"
gh pr create --title "feat: add intelligent node scoring for quantum job scheduling"

# Fix bug
git checkout -b fix/memory-leak-analyzer
# ... fix issue ...
git commit -m "fix(analyzer): resolve memory leak in circuit parsing"
gh pr create --title "fix: resolve memory leak in circuit analyzer service"

# Add tests
git checkout -b test/scheduler-unit-tests
# ... add tests ...
git commit -m "test(scheduler): add comprehensive unit tests for node scoring"
```

This is a complex, distributed system. Take time to understand the architecture before making changes. When in doubt, ask for clarification rather than guessing.