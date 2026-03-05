#!/bin/bash
set -e

echo "🚀 Setting up qsim-cluster development environment on minikube"

# Check if minikube is installed
if ! command -v minikube &> /dev/null; then
    echo "❌ minikube is not installed. Please install it first:"
    echo "   brew install minikube"
    exit 1
fi

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl is not installed. Please install it first:"
    echo "   brew install kubernetes-cli"
    exit 1
fi

# Start minikube with sufficient resources
echo "🔧 Starting minikube cluster..."
minikube start \
    --memory=8192 \
    --cpus=4 \
    --disk-size=20GB \
    --kubernetes-version=v1.28.0 \
    --driver=docker

# Enable required addons
echo "🔌 Enabling minikube addons..."
minikube addons enable ingress
minikube addons enable metrics-server

# Create namespaces
echo "📦 Creating namespaces..."
kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: quantum-system
  labels:
    name: quantum-system
---
apiVersion: v1
kind: Namespace
metadata:
  name: quantum-jobs
  labels:
    name: quantum-jobs
---
apiVersion: v1
kind: Namespace
metadata:
  name: quantum-monitor
  labels:
    name: quantum-monitor
EOF

# Create PostgreSQL deployment
echo "🗄️  Deploying PostgreSQL..."
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
  namespace: quantum-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15
        ports:
        - containerPort: 5432
        env:
        - name: POSTGRES_DB
          value: qsim
        - name: POSTGRES_USER
          value: qsim
        - name: POSTGRES_PASSWORD
          value: password
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
      volumes:
      - name: postgres-storage
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: quantum-system
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
    targetPort: 5432
  type: ClusterIP
EOF

# Create Redis deployment
echo "🔥 Deploying Redis..."
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
  namespace: quantum-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:7
        ports:
        - containerPort: 6379
        command: ["redis-server"]
        args: ["--appendonly", "yes"]
        volumeMounts:
        - name: redis-storage
          mountPath: /data
      volumes:
      - name: redis-storage
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: redis
  namespace: quantum-system
spec:
  selector:
    app: redis
  ports:
  - port: 6379
    targetPort: 6379
  type: ClusterIP
EOF

# Wait for databases to be ready
echo "⏳ Waiting for databases to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment/postgres -n quantum-system
kubectl wait --for=condition=available --timeout=300s deployment/redis -n quantum-system

# Create node profiles for the minikube nodes
echo "🖥️  Creating sample node profiles..."
kubectl apply -f - <<EOF
apiVersion: quantum.blocksq.io/v1alpha1
kind: QuantumNodeProfile
metadata:
  name: minikube
  namespace: quantum-system
spec:
  pool: cpu
  cpu:
    cores: 4
    architecture: x86_64
  memory:
    totalGB: 8
  gpu:
    available: false
  simulatorConfig:
    maxConcurrentJobs: 2
    supportedMethods:
      - statevector
      - stabilizer
      - mps
status:
  ready: true
  currentLoad:
    cpuUsagePercent: 25.0
    memoryUsagePercent: 30.0
    activeJobs: 0
EOF

echo "✅ Minikube setup completed!"
echo ""
echo "📋 Next steps:"
echo "1. Build and push container images:"
echo "   make docker-build"
echo ""
echo "2. Deploy the qsim-cluster components:"
echo "   make deploy-dev"
echo ""
echo "3. Access the cluster:"
echo "   kubectl get pods -n quantum-system"
echo ""
echo "🔗 Useful commands:"
echo "   minikube dashboard  # Open Kubernetes dashboard"
echo "   minikube service list  # List all services"
echo "   kubectl logs -f deployment/api-server -n quantum-system  # View API server logs"