# Qiskit Simulator Cluster (qsim-cluster)

Kubernetes-based quantum circuit simulation cluster with intelligent scheduling and resource optimization.

## 🚀 Features

- **Smart Scheduling**: Circuit complexity analysis-based resource allocation
- **Multi-Method Support**: Statevector, Stabilizer, MPS simulation methods
- **K8s Native**: Custom Resource Definitions (CRDs) and Operator pattern
- **Scalable**: Auto-scaling based on job complexity and cluster resources
- **Observable**: Real-time job status and cluster metrics monitoring

## 📋 Architecture

```
Client (Jupyter/SDK) → API Server → K8s Operator → Simulation Pods
                           ↓
                    Circuit Analyzer (complexity estimation)
                           ↓
                    Smart Scheduler (resource matching)
```

## 🏗️ Components

- **API Server** (Go): REST API, job management, circuit analysis
- **Operator** (Go): K8s controller for QuantumJob and QuantumNodeProfile CRDs
- **Circuit Analyzer** (Python): Qiskit circuit complexity estimation
- **Runtime** (Python): Sandboxed execution environment for quantum circuits
- **SDK** (Python): Client library for job submission and management

## 🛠️ Development

### Prerequisites

- Go 1.21+
- Python 3.11+
- Kubernetes cluster (minikube for development)
- Docker

### Quick Start

```bash
# Clone the repository
git clone https://github.com/mungch0120/qsim-cluster.git
cd qsim-cluster

# Set up development environment
make dev-setup

# Deploy to minikube
make dev-deploy

# Run tests
make test
```

### Development Workflow

1. Create feature branch: `git checkout -b feat/your-feature`
2. Make changes and commit with conventional commits
3. Create PR: `gh pr create --title "feat: description"`
4. CI runs automatically (lint, test, integration tests)
5. Review and merge

## 📊 Complexity Classes

| Class | Qubits | Depth | Gate Count | Resources | Node Pool |
|-------|--------|-------|------------|-----------|-----------|
| A (Light) | 1-5 | 1-10 | 1-20 | 1 CPU, 1GB | cpu |
| B (Medium) | 6-15 | 11-50 | 21-200 | 2-4 CPU, 4-8GB | cpu/high-cpu |
| C (Heavy) | 16-25 | 51-200 | 201-1000 | 8-64 CPU, 16-64GB | high-cpu |
| D (Extreme) | 26+ | 200+ | 1000+ | 64+ CPU or GPU | gpu |

## 📚 Documentation

- [Architecture Design](./docs/ARCHITECTURE.md)
- [API Reference](./docs/API.md)
- [Deployment Guide](./docs/DEPLOYMENT.md)
- [Development Guide](./docs/DEVELOPMENT.md)

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes (use conventional commits)
4. Push to the branch
5. Open a Pull Request

---

**Status**: 🚧 Under Development (Phase 1 - Core Infrastructure)