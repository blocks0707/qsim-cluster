"""
QSim Client SDK

Provides a high-level interface for interacting with the qsim-cluster API.
"""

import time
import requests
import json
from typing import Dict, Any, Optional, List, Union
from dataclasses import dataclass

import qiskit
from qiskit import QuantumCircuit


@dataclass
class ComplexityAnalysis:
    """Circuit complexity analysis result"""
    qubits: int
    depth: int
    gate_count: int
    complexity_class: str
    recommended_method: str
    estimated_time: int
    recommended_pool: str
    estimated_memory_mb: int
    estimated_cpu: int


class QSimJob:
    """Represents a quantum simulation job"""
    
    def __init__(self, job_id: str, client: 'QSimClient'):
        self.id = job_id
        self._client = client
        self._status_cache = None
        self._last_status_check = 0
        
    @property
    def status(self) -> str:
        """Get the current job status"""
        # Cache status for 5 seconds to avoid too many API calls
        now = time.time()
        if self._status_cache is None or (now - self._last_status_check) > 5:
            try:
                response = self._client._get(f"/jobs/{self.id}")
                self._status_cache = response.get("status", "unknown")
                self._last_status_check = now
            except Exception:
                return "unknown"
        
        return self._status_cache
    
    @property
    def execution_time(self) -> Optional[int]:
        """Get execution time in seconds"""
        try:
            response = self._client._get(f"/jobs/{self.id}")
            return response.get("execution_time")
        except Exception:
            return None
    
    def wait(self, timeout: int = 300, poll_interval: int = 5) -> str:
        """
        Wait for job completion.
        
        Args:
            timeout: Maximum time to wait in seconds
            poll_interval: Time between status checks in seconds
            
        Returns:
            Final job status
        """
        start_time = time.time()
        
        while True:
            status = self.status
            
            if status in ["succeeded", "failed", "cancelled"]:
                return status
            
            if time.time() - start_time > timeout:
                raise TimeoutError(f"Job {self.id} did not complete within {timeout} seconds")
            
            time.sleep(poll_interval)
    
    def result(self):
        """
        Get the execution result.
        
        Returns:
            Qiskit Result object or dictionary with results
        """
        response = self._client._get(f"/jobs/{self.id}/result")
        return response
    
    def logs(self) -> str:
        """Get execution logs"""
        response = self._client._get(f"/jobs/{self.id}/logs")
        return response.get("logs", "")
    
    def cancel(self):
        """Cancel the job"""
        self._client._delete(f"/jobs/{self.id}")
        self._status_cache = "cancelled"
    
    def retry(self) -> 'QSimJob':
        """Retry a failed job"""
        response = self._client._post(f"/jobs/{self.id}/retry")
        return QSimJob(response["job_id"], self._client)
    
    def __str__(self):
        return f"QSimJob(id='{self.id}', status='{self.status}')"
    
    def __repr__(self):
        return self.__str__()


class QSimClient:
    """Client for interacting with the qsim-cluster API"""
    
    def __init__(self, endpoint: str, api_key: Optional[str] = None):
        """
        Initialize the QSim client.
        
        Args:
            endpoint: API server endpoint URL
            api_key: API authentication key (optional)
        """
        self.endpoint = endpoint.rstrip('/')
        self.api_key = api_key
        self.session = requests.Session()
        
        # Set default headers
        self.session.headers.update({
            'Content-Type': 'application/json',
            'User-Agent': f'qsim-client/{qiskit.__version__}'
        })
        
        if api_key:
            self.session.headers['Authorization'] = f'Bearer {api_key}'
    
    def run(self, circuit: QuantumCircuit, shots: int = 1024, 
            method: str = "automatic", **options) -> QSimJob:
        """
        Submit a quantum circuit for execution.
        
        Args:
            circuit: Qiskit QuantumCircuit to execute
            shots: Number of measurement shots
            method: Simulation method ("automatic", "statevector", "stabilizer", "mps")
            **options: Additional job options
            
        Returns:
            QSimJob instance for tracking the execution
        """
        # Convert circuit to QASM or Python code
        # For now, we'll use Python code representation
        code = self._circuit_to_code(circuit, shots, method)
        
        payload = {
            "code": code,
            "language": "python",
            "options": {
                "shots": shots,
                "method": method,
                **options
            }
        }
        
        response = self._post("/jobs", payload)
        return QSimJob(response["job_id"], self)
    
    def analyze(self, circuit: QuantumCircuit) -> ComplexityAnalysis:
        """
        Analyze circuit complexity without execution.
        
        Args:
            circuit: Qiskit QuantumCircuit to analyze
            
        Returns:
            ComplexityAnalysis with resource estimates
        """
        code = self._circuit_to_code(circuit, shots=1024, method="automatic")
        
        payload = {
            "code": code,
            "language": "python"
        }
        
        response = self._post("/analyze", payload)
        
        return ComplexityAnalysis(
            qubits=response["qubits"],
            depth=response["depth"],
            gate_count=response["gate_count"],
            complexity_class=response["complexity_class"],
            recommended_method=response["recommended_method"],
            estimated_time=response["estimated_time_sec"],
            recommended_pool=response["recommended_pool"],
            estimated_memory_mb=response["estimated_memory_mb"],
            estimated_cpu=response["estimated_cpu"]
        )
    
    def list_jobs(self, status: Optional[str] = None, limit: int = 10) -> List[Dict[str, Any]]:
        """
        List jobs for the current user.
        
        Args:
            status: Filter by job status
            limit: Maximum number of jobs to return
            
        Returns:
            List of job dictionaries
        """
        params = {"limit": limit}
        if status:
            params["status"] = status
        
        response = self._get("/jobs", params=params)
        return response["jobs"]
    
    def get_job(self, job_id: str) -> QSimJob:
        """
        Get a job by ID.
        
        Args:
            job_id: Job identifier
            
        Returns:
            QSimJob instance
        """
        return QSimJob(job_id, self)
    
    def cluster_status(self) -> Dict[str, Any]:
        """Get cluster status and resource information"""
        return self._get("/cluster/status")
    
    def cluster_nodes(self) -> List[Dict[str, Any]]:
        """Get list of cluster nodes and their status"""
        response = self._get("/cluster/nodes")
        return response["nodes"]
    
    def cluster_metrics(self) -> Dict[str, Any]:
        """Get cluster performance metrics"""
        return self._get("/cluster/metrics")
    
    def _circuit_to_code(self, circuit: QuantumCircuit, shots: int, method: str) -> str:
        """Convert a QuantumCircuit to executable Python code"""
        # Generate Python code that recreates the circuit
        # This is a simplified version - real implementation would be more robust
        
        lines = [
            "from qiskit import QuantumCircuit, QuantumRegister, ClassicalRegister",
            "from qiskit_aer import AerSimulator",
            "import numpy as np",
            "",
        ]
        
        # Create circuit declaration
        n_qubits = circuit.num_qubits
        n_clbits = circuit.num_clbits
        
        if n_clbits > 0:
            lines.append(f"qc = QuantumCircuit({n_qubits}, {n_clbits})")
        else:
            lines.append(f"qc = QuantumCircuit({n_qubits})")
        
        # Add circuit operations
        for instruction in circuit.data:
            gate = instruction.operation
            qubits = [circuit.qubits.index(q) for q in instruction.qubits]
            clbits = [circuit.clbits.index(c) for c in instruction.clbits] if instruction.clbits else []
            
            # Handle common gates
            gate_name = gate.name.lower()
            
            if gate_name == 'h':
                lines.append(f"qc.h({qubits[0]})")
            elif gate_name == 'x':
                lines.append(f"qc.x({qubits[0]})")
            elif gate_name == 'y':
                lines.append(f"qc.y({qubits[0]})")
            elif gate_name == 'z':
                lines.append(f"qc.z({qubits[0]})")
            elif gate_name == 'cx' or gate_name == 'cnot':
                lines.append(f"qc.cx({qubits[0]}, {qubits[1]})")
            elif gate_name == 'measure':
                if clbits:
                    lines.append(f"qc.measure({qubits[0]}, {clbits[0]})")
            else:
                # Generic gate handling
                if hasattr(gate, 'params') and gate.params:
                    params_str = ', '.join(str(p) for p in gate.params)
                    lines.append(f"qc.{gate_name}({params_str}, {', '.join(map(str, qubits))})")
                else:
                    lines.append(f"qc.{gate_name}({', '.join(map(str, qubits))})")
        
        # Add simulation code
        lines.extend([
            "",
            f"simulator = AerSimulator(method='{method}')",
            f"result = simulator.run(qc, shots={shots}).result()",
            "counts = result.get_counts()",
        ])
        
        return '\n'.join(lines)
    
    def _get(self, path: str, params: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Make GET request to API"""
        url = f"{self.endpoint}/api/v1{path}"
        response = self.session.get(url, params=params)
        response.raise_for_status()
        return response.json()
    
    def _post(self, path: str, data: Dict[str, Any]) -> Dict[str, Any]:
        """Make POST request to API"""
        url = f"{self.endpoint}/api/v1{path}"
        response = self.session.post(url, json=data)
        response.raise_for_status()
        return response.json()
    
    def _delete(self, path: str) -> Dict[str, Any]:
        """Make DELETE request to API"""
        url = f"{self.endpoint}/api/v1{path}"
        response = self.session.delete(url)
        response.raise_for_status()
        return response.json() if response.content else {}