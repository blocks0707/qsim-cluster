"""
Circuit Interceptor Module

Captures QuantumCircuit objects from user code without executing the actual simulation.
Uses monkey-patching to replace AerSimulator.run() calls with circuit capture logic.
"""

import sys
import traceback
from typing import List, Dict, Any, Optional
import builtins

import structlog
from qiskit import QuantumCircuit
from qiskit_aer import AerSimulator

logger = structlog.get_logger()

class CircuitCaptured(Exception):
    """Exception used to signal that a circuit has been captured"""
    def __init__(self, circuit: QuantumCircuit, **kwargs):
        self.circuit = circuit
        self.shots = kwargs.get('shots', 1024)
        self.method = kwargs.get('method', 'automatic')
        super().__init__("Circuit captured successfully")

class AnalysisSimulator:
    """
    Mock simulator that captures circuits instead of executing them.
    Replaces AerSimulator during code execution to intercept circuit objects.
    """
    
    captured_circuits = []
    
    def __init__(self, **kwargs):
        """Initialize with same signature as AerSimulator"""
        self.config = kwargs
        self.method = kwargs.get('method', 'automatic')
    
    def run(self, circuit, **kwargs):
        """
        Capture the circuit instead of running simulation.
        Raises CircuitCaptured exception to halt execution after capture.
        """
        # Store circuit data
        circuit_data = {
            'circuit': circuit,
            'shots': kwargs.get('shots', 1024),
            'method': self.config.get('method', kwargs.get('method', self.method)),
            'config': self.config
        }
        
        self.captured_circuits.append(circuit_data)
        
        logger.debug("circuit_captured", 
                    qubits=circuit.num_qubits if hasattr(circuit, 'num_qubits') else 0,
                    shots=circuit_data['shots'],
                    method=circuit_data['method'])
        
        # Raise exception to stop code execution after capture
        raise CircuitCaptured(circuit, **kwargs)
    
    def __getattr__(self, name):
        """
        Delegate other methods to a real AerSimulator instance.
        This ensures compatibility with code that checks simulator properties.
        """
        # Create a temporary real simulator for attribute access
        real_sim = AerSimulator(**self.config)
        return getattr(real_sim, name)

def analyze_circuit_from_code(user_code: str) -> List[Dict[str, Any]]:
    """
    Analyze quantum circuit code without execution.
    
    Executes the user code in a sandboxed environment with intercepted
    AerSimulator to capture QuantumCircuit objects.
    
    Args:
        user_code: Python code string containing Qiskit circuit definitions
        
    Returns:
        List of captured circuit data dictionaries
        
    Raises:
        ValueError: If no circuits are found or code execution fails
        SyntaxError: If the provided code has syntax errors
    """
    logger.info("analyzing_user_code", code_length=len(user_code))
    
    # Store original AerSimulator reference
    original_aer_simulator = None
    
    try:
        # Import qiskit_aer module to patch it
        import qiskit_aer
        original_aer_simulator = qiskit_aer.AerSimulator
        
        # Reset captured circuits
        AnalysisSimulator.captured_circuits = []
        
        # Replace AerSimulator with our interceptor
        qiskit_aer.AerSimulator = AnalysisSimulator
        
        # Also patch direct imports (common patterns)
        if 'qiskit_aer' in sys.modules:
            sys.modules['qiskit_aer'].AerSimulator = AnalysisSimulator
        
        # Prepare execution environment
        # Include common Qiskit imports that users typically use
        exec_globals = {
            '__builtins__': builtins,
            # Core Qiskit
            'QuantumCircuit': QuantumCircuit,
            'AerSimulator': AnalysisSimulator,
            # Common imports
            'qiskit': sys.modules.get('qiskit'),
            'qiskit_aer': sys.modules.get('qiskit_aer'),
            'numpy': sys.modules.get('numpy'),
            'np': sys.modules.get('numpy'),
            # Math functions commonly used in quantum circuits
            'pi': 3.141592653589793,
            'sqrt': lambda x: x ** 0.5,
        }
        
        exec_locals = {}
        
        try:
            # Execute user code
            exec(user_code, exec_globals, exec_locals)
            
            # If we reach here, no circuits were captured
            logger.warning("no_circuits_captured", 
                         captured_count=len(AnalysisSimulator.captured_circuits))
            
        except CircuitCaptured as e:
            # Expected exception when circuit is captured
            logger.debug("circuit_capture_exception_caught", 
                        circuit_qubits=e.circuit.num_qubits if hasattr(e.circuit, 'num_qubits') else 0)
            pass
            
        except SyntaxError as e:
            logger.error("syntax_error_in_user_code", 
                        error=str(e), 
                        lineno=getattr(e, 'lineno', None))
            raise ValueError(f"Syntax error in provided code: {e}")
            
        except Exception as e:
            # Log unexpected errors but don't necessarily fail
            # Some code might have dependencies we can't satisfy
            logger.warning("execution_error_during_analysis",
                          error=str(e),
                          error_type=type(e).__name__,
                          traceback=traceback.format_exc())
        
        # Return captured circuits
        circuits = AnalysisSimulator.captured_circuits.copy()
        
        if not circuits:
            # Try alternative analysis approaches
            circuits = _try_alternative_analysis(user_code, exec_globals, exec_locals)
        
        if not circuits:
            raise ValueError("No quantum circuits found in the provided code. "
                           "Make sure your code creates a QuantumCircuit and calls simulator.run()")
        
        logger.info("circuit_analysis_completed", 
                   captured_circuits=len(circuits))
        
        return circuits
        
    finally:
        # Always restore the original AerSimulator
        if original_aer_simulator is not None:
            import qiskit_aer
            qiskit_aer.AerSimulator = original_aer_simulator
            if 'qiskit_aer' in sys.modules:
                sys.modules['qiskit_aer'].AerSimulator = original_aer_simulator

def _try_alternative_analysis(user_code: str, exec_globals: dict, exec_locals: dict) -> List[Dict[str, Any]]:
    """
    Alternative analysis approach for cases where direct execution doesn't capture circuits.
    
    Looks for QuantumCircuit objects created during execution by examining the local variables.
    """
    logger.debug("trying_alternative_circuit_analysis")
    
    circuits = []
    
    try:
        # Look for QuantumCircuit instances in the execution locals
        for name, value in exec_locals.items():
            if isinstance(value, QuantumCircuit):
                logger.debug("found_quantum_circuit_in_locals", 
                           variable_name=name,
                           qubits=value.num_qubits)
                
                circuits.append({
                    'circuit': value,
                    'shots': 1024,  # Default
                    'method': 'automatic',  # Default
                    'config': {},
                    'variable_name': name
                })
        
        # Also check globals for circuits
        for name, value in exec_globals.items():
            if isinstance(value, QuantumCircuit) and name not in ['QuantumCircuit']:
                circuits.append({
                    'circuit': value,
                    'shots': 1024,
                    'method': 'automatic',
                    'config': {},
                    'variable_name': name
                })
    
    except Exception as e:
        logger.warning("alternative_analysis_failed", error=str(e))
    
    return circuits

# Utility function for testing
def test_circuit_capture():
    """Test function to verify circuit capture works correctly"""
    test_code = """
from qiskit import QuantumCircuit
from qiskit_aer import AerSimulator

qc = QuantumCircuit(2, 2)
qc.h(0)
qc.cx(0, 1)
qc.measure_all()

simulator = AerSimulator(method='statevector')
result = simulator.run(qc, shots=1024).result()
"""
    
    circuits = analyze_circuit_from_code(test_code)
    print(f"Captured {len(circuits)} circuits")
    
    for i, circuit_data in enumerate(circuits):
        circuit = circuit_data['circuit']
        print(f"Circuit {i}: {circuit.num_qubits} qubits, depth {circuit.depth()}")

if __name__ == "__main__":
    # Run test
    test_circuit_capture()