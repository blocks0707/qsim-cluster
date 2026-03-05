"""
Circuit Complexity Analyzer

Analyzes quantum circuits to estimate computational complexity,
resource requirements, and optimal simulation methods.
"""

from typing import Dict, Any, List
import math

import structlog
from qiskit import QuantumCircuit, transpile
from qiskit.circuit.library import standard_gates

logger = structlog.get_logger()

class ComplexityAnalyzer:
    """Analyzes quantum circuit complexity and estimates resource requirements"""
    
    # Clifford gates that can be efficiently simulated
    CLIFFORD_GATES = {
        'h', 'x', 'y', 'z', 's', 'sdg', 't', 'tdg', 'cx', 'cy', 'cz', 
        'swap', 'id', 'barrier', 'measure'
    }
    
    def __init__(self):
        """Initialize the complexity analyzer"""
        pass
    
    def analyze(self, circuit: QuantumCircuit, shots: int = 1024) -> Dict[str, Any]:
        """
        Perform comprehensive complexity analysis of a quantum circuit.
        
        Args:
            circuit: The quantum circuit to analyze
            shots: Number of shots for execution
            
        Returns:
            Dictionary containing complexity metrics and resource estimates
        """
        logger.debug("starting_complexity_analysis",
                    qubits=circuit.num_qubits,
                    shots=shots)
        
        # Basic circuit properties
        n_qubits = circuit.num_qubits
        original_depth = circuit.depth()
        original_gate_count = circuit.size()
        gate_ops = circuit.count_ops()
        
        # Transpile for more accurate analysis
        try:
            transpiled = transpile(circuit, basis_gates=['u1', 'u2', 'u3', 'cx'])
            transpiled_depth = transpiled.depth()
            transpiled_gate_count = transpiled.size()
            transpiled_ops = transpiled.count_ops()
            cx_count = transpiled_ops.get('cx', 0)
        except Exception as e:
            logger.warning("transpilation_failed", error=str(e))
            # Fallback to original circuit
            transpiled_depth = original_depth
            transpiled_gate_count = original_gate_count
            cx_count = gate_ops.get('cx', gate_ops.get('cnot', 0))
        
        # Calculate parallelism factor
        parallelism = self._calculate_parallelism(circuit)
        
        # Determine if circuit is Clifford-only
        is_clifford = self._is_clifford_circuit(gate_ops)
        
        # Estimate memory requirements
        memory_bytes = self._estimate_memory(n_qubits, is_clifford)
        
        # Classify complexity
        complexity_class = self._classify_complexity(n_qubits, transpiled_depth, transpiled_gate_count)
        
        # Recommend simulation method
        recommended_method = self._recommend_method(n_qubits, is_clifford, cx_count, transpiled_gate_count)
        
        # Estimate computational resources
        estimated_cpu = self._estimate_cpu_cores(complexity_class, n_qubits, cx_count)
        estimated_memory_mb = self._estimate_memory_mb(memory_bytes, complexity_class)
        estimated_time_sec = self._estimate_execution_time(complexity_class, n_qubits, transpiled_depth, shots)
        
        # Recommend node pool
        recommended_pool = self._recommend_node_pool(complexity_class, recommended_method)
        
        result = {
            'qubits': n_qubits,
            'depth': transpiled_depth,
            'original_depth': original_depth,
            'gate_count': transpiled_gate_count,
            'original_gate_count': original_gate_count,
            'cx_count': cx_count,
            'parallelism': round(parallelism, 3),
            'memory_bytes': memory_bytes,
            'complexity_class': complexity_class,
            'is_clifford': is_clifford,
            'recommended_method': recommended_method,
            'estimated_cpu': estimated_cpu,
            'estimated_memory_mb': estimated_memory_mb,
            'estimated_time_sec': estimated_time_sec,
            'recommended_pool': recommended_pool,
            'gate_breakdown': gate_ops
        }
        
        logger.info("complexity_analysis_completed",
                   qubits=n_qubits,
                   complexity_class=complexity_class,
                   recommended_method=recommended_method,
                   estimated_memory_mb=estimated_memory_mb)
        
        return result
    
    def _calculate_parallelism(self, circuit: QuantumCircuit) -> float:
        """
        Calculate circuit parallelism factor.
        Higher values indicate more gates can execute in parallel.
        """
        if circuit.depth() == 0:
            return 0.0
        
        gate_count = circuit.size()
        depth = circuit.depth()
        
        # Parallelism = total gates / depth
        # Perfect parallelism = 1.0, sequential = gate_count/depth (low)
        parallelism = gate_count / depth if depth > 0 else 0.0
        
        # Normalize to reasonable range (0.0 - 1.0)
        # High parallelism circuits have many gates that can run simultaneously
        max_parallelism = circuit.num_qubits  # Theoretical max
        normalized = min(parallelism / max_parallelism, 1.0) if max_parallelism > 0 else 0.0
        
        return normalized
    
    def _is_clifford_circuit(self, gate_ops: Dict[str, int]) -> bool:
        """Check if the circuit contains only Clifford gates"""
        for gate_name in gate_ops.keys():
            gate_lower = gate_name.lower()
            if gate_lower not in self.CLIFFORD_GATES:
                # Check for parameterized gates that might be Clifford
                if not self._is_clifford_gate_name(gate_lower):
                    return False
        return True
    
    def _is_clifford_gate_name(self, gate_name: str) -> bool:
        """Check if a gate name represents a Clifford operation"""
        # Common Clifford gate patterns
        clifford_patterns = [
            'pauli', 'cnot', 'ccx', 'mcx', 'controlled',
            'reset', 'initialize'
        ]
        
        for pattern in clifford_patterns:
            if pattern in gate_name:
                return True
        
        return False
    
    def _estimate_memory(self, n_qubits: int, is_clifford: bool) -> int:
        """Estimate memory requirements in bytes"""
        if is_clifford:
            # Clifford circuits can be simulated efficiently
            # Stabilizer tableaux: O(n^2) memory
            return n_qubits * n_qubits * 8  # Rough estimate
        else:
            # Statevector simulation: 2^n complex amplitudes
            # Each complex128 = 16 bytes
            return (2 ** n_qubits) * 16
    
    def _classify_complexity(self, n_qubits: int, depth: int, gate_count: int) -> str:
        """Classify circuit complexity into classes A, B, C, D"""
        if n_qubits <= 5 and depth <= 10 and gate_count <= 20:
            return 'A'  # Light
        elif n_qubits <= 15 and depth <= 50 and gate_count <= 200:
            return 'B'  # Medium
        elif n_qubits <= 25 and depth <= 200 and gate_count <= 1000:
            return 'C'  # Heavy
        else:
            return 'D'  # Extreme
    
    def _recommend_method(self, n_qubits: int, is_clifford: bool, 
                         cx_count: int, gate_count: int) -> str:
        """Recommend optimal simulation method"""
        if is_clifford:
            return 'stabilizer'
        elif n_qubits <= 15:
            return 'statevector'
        elif n_qubits <= 30:
            # Check for low entanglement (heuristic: low CX/gate ratio)
            cx_ratio = cx_count / gate_count if gate_count > 0 else 0
            if cx_ratio < 0.3:
                return 'mps'  # Matrix Product State for low entanglement
            else:
                return 'statevector'
        else:
            # Large circuits need GPU or specialized methods
            return 'statevector'
    
    def _estimate_cpu_cores(self, complexity_class: str, n_qubits: int, cx_count: int) -> int:
        """Estimate required CPU cores"""
        base_cores = {
            'A': 1,
            'B': 2,
            'C': 8,
            'D': 64
        }
        
        cores = base_cores.get(complexity_class, 1)
        
        # Adjust based on circuit properties
        if complexity_class == 'B':
            cores += max(0, cx_count // 50)
        elif complexity_class == 'C':
            cores += max(0, cx_count // 20)
        
        return min(cores, 256)  # Cap at reasonable limit
    
    def _estimate_memory_mb(self, memory_bytes: int, complexity_class: str) -> int:
        """Estimate memory requirements in MB with overhead"""
        base_mb = memory_bytes // (1024 * 1024)
        
        # Add overhead based on complexity
        overhead_multiplier = {
            'A': 2.0,
            'B': 2.5,
            'C': 3.0,
            'D': 4.0
        }
        
        multiplier = overhead_multiplier.get(complexity_class, 3.0)
        total_mb = int(base_mb * multiplier)
        
        # Minimum memory requirement
        return max(total_mb, 512)
    
    def _estimate_execution_time(self, complexity_class: str, n_qubits: int, 
                               depth: int, shots: int) -> int:
        """Estimate execution time in seconds"""
        base_time = {
            'A': 5,
            'B': 15,
            'C': 60,
            'D': 300
        }
        
        time_sec = base_time.get(complexity_class, 60)
        
        # Adjust for circuit size
        if complexity_class == 'C' or complexity_class == 'D':
            # Exponential scaling for large circuits
            time_sec *= (1.2 ** (n_qubits - 15)) if n_qubits > 15 else 1
        
        # Adjust for depth (linear impact)
        time_sec *= (1 + depth / 100)
        
        # Adjust for shots (linear scaling)
        time_sec *= (shots / 1024)
        
        return max(int(time_sec), 1)
    
    def _recommend_node_pool(self, complexity_class: str, method: str) -> str:
        """Recommend appropriate node pool"""
        if complexity_class == 'A':
            return 'cpu'
        elif complexity_class == 'B':
            return 'cpu'
        elif complexity_class == 'C':
            return 'high-cpu'
        else:  # complexity_class == 'D'
            return 'gpu'