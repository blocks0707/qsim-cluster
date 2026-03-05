FROM python:3.11-slim

WORKDIR /app

# Install system dependencies for Qiskit Aer compilation
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    cmake \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements first for better caching
COPY requirements.txt .

# Install Python dependencies
# Install in separate steps for better layer caching
RUN pip install --no-cache-dir --upgrade pip setuptools wheel
RUN pip install --no-cache-dir numpy
RUN pip install --no-cache-dir qiskit qiskit-aer
RUN pip install --no-cache-dir -r requirements.txt

# Copy source code
COPY . .

# Default command to run the quantum execution script
CMD ["python", "execute.py"]