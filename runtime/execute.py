#!/usr/bin/env python3
"""
Quantum Circuit Runtime Executor

Executes quantum circuits in a sandboxed container environment.
Reads circuit code from mounted volume, executes it, and saves results.
"""

import os
import sys
import json
import time
import traceback
from pathlib import Path
from typing import Dict, Any, Optional

import structlog

# Configure structured logging
structlog.configure(
    processors=[
        structlog.stdlib.filter_by_level,
        structlog.stdlib.add_logger_name,
        structlog.stdlib.add_log_level,
        structlog.stdlib.PositionalArgumentsFormatter(),
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.StackInfoRenderer(),
        structlog.processors.format_exc_info,
        structlog.processors.UnicodeDecoder(),
        structlog.processors.JSONRenderer()
    ],
    context_class=dict,
    logger_factory=structlog.stdlib.LoggerFactory(),
    wrapper_class=structlog.stdlib.BoundLogger,
    cache_logger_on_first_use=True,
)

logger = structlog.get_logger()

class QuantumExecutor:
    """Handles execution of quantum circuits in a sandboxed environment"""
    
    def __init__(self):
        self.code_path = Path("/code")
        self.results_path = Path("/results")
        self.max_execution_time = int(os.getenv("MAX_EXECUTION_TIME", "300"))
        self.job_id = os.getenv("JOB_ID", "unknown")
        
        logger.info("quantum_executor_initialized",
                   job_id=self.job_id,
                   max_execution_time=self.max_execution_time,
                   code_path=str(self.code_path),
                   results_path=str(self.results_path))
    
    def execute(self) -> Dict[str, Any]:
        """
        Execute the quantum circuit code and return results.
        
        Returns:
            Dictionary containing execution results and metadata
        """
        start_time = time.time()
        
        try:
            # Read circuit code
            code_file = self.code_path / "circuit.py"
            if not code_file.exists():
                raise FileNotFoundError(f"Circuit code not found at {code_file}")
            
            code = code_file.read_text()
            logger.info("circuit_code_loaded",
                       code_length=len(code),
                       code_file=str(code_file))
            
            # Prepare execution environment
            exec_globals = self._prepare_execution_environment()
            exec_locals = {}
            
            # Execute the circuit code
            logger.info("starting_circuit_execution")
            
            try:
                exec(code, exec_globals, exec_locals)
            except Exception as e:
                logger.error("circuit_execution_failed",
                           error=str(e),
                           error_type=type(e).__name__,
                           traceback=traceback.format_exc())
                raise
            
            # Extract results
            results = self._extract_results(exec_locals)
            
            execution_time = time.time() - start_time
            logger.info("circuit_execution_completed",
                       execution_time=execution_time,
                       result_keys=list(results.keys()))
            
            # Prepare final result
            final_result = {
                "job_id": self.job_id,
                "status": "succeeded",
                "execution_time_sec": execution_time,
                "results": results,
                "timestamp": time.time()
            }
            
            return final_result
            
        except Exception as e:
            execution_time = time.time() - start_time
            error_result = {
                "job_id": self.job_id,
                "status": "failed",
                "execution_time_sec": execution_time,
                "error": str(e),
                "error_type": type(e).__name__,
                "traceback": traceback.format_exc(),
                "timestamp": time.time()
            }
            
            logger.error("quantum_execution_failed",
                        execution_time=execution_time,
                        error=str(e))
            
            return error_result
    
    def _prepare_execution_environment(self) -> Dict[str, Any]:
        """Prepare the execution environment with necessary imports"""
        # Import commonly used quantum computing libraries
        import qiskit
        from qiskit import QuantumCircuit, QuantumRegister, ClassicalRegister
        from qiskit_aer import AerSimulator
        from qiskit.circuit.library import *
        import numpy as np
        
        return {
            '__builtins__': __builtins__,
            # Core Qiskit
            'qiskit': qiskit,
            'QuantumCircuit': QuantumCircuit,
            'QuantumRegister': QuantumRegister,
            'ClassicalRegister': ClassicalRegister,
            'AerSimulator': AerSimulator,
            # Circuit library
            'HGate': HGate,
            'XGate': XGate,
            'YGate': YGate,
            'ZGate': ZGate,
            'CXGate': CXGate,
            'CZGate': CZGate,
            # Numpy
            'numpy': np,
            'np': np,
            # Math
            'pi': np.pi,
            'sqrt': np.sqrt,
            'sin': np.sin,
            'cos': np.cos,
            'exp': np.exp
        }
    
    def _extract_results(self, exec_locals: Dict[str, Any]) -> Dict[str, Any]:
        """
        Extract quantum execution results from local variables.
        
        Looks for common result patterns like 'result', 'counts', 'statevector', etc.
        """
        results = {}
        
        # Look for common result variable names
        result_keys = ['result', 'results', 'counts', 'statevector', 'measurement_results']
        
        for key in result_keys:
            if key in exec_locals:
                value = exec_locals[key]
                try:
                    # Convert Qiskit results to JSON-serializable format
                    results[key] = self._serialize_qiskit_result(value)
                    logger.debug("extracted_result", key=key, type=type(value).__name__)
                except Exception as e:
                    logger.warning("failed_to_serialize_result", 
                                 key=key, 
                                 error=str(e))
        
        # If no standard results found, look for any variables ending with 'result'
        if not results:
            for key, value in exec_locals.items():
                if key.endswith('result') or key.endswith('counts'):
                    try:
                        results[key] = self._serialize_qiskit_result(value)
                    except Exception as e:
                        logger.warning("failed_to_serialize_variable",
                                     variable=key,
                                     error=str(e))
        
        # If still no results, include basic execution info
        if not results:
            results = {"message": "Execution completed but no results captured"}
            logger.warning("no_results_extracted", 
                          available_variables=list(exec_locals.keys()))
        
        return results
    
    def _serialize_qiskit_result(self, obj) -> Any:
        """Convert Qiskit result objects to JSON-serializable format"""
        if hasattr(obj, 'get_counts'):
            # Qiskit Result object
            try:
                return {
                    'counts': obj.get_counts(),
                    'success': obj.success,
                    'job_id': getattr(obj, 'job_id', None),
                    'backend_name': getattr(obj, 'backend_name', None)
                }
            except Exception:
                pass
        
        if isinstance(obj, dict):
            # Already a dictionary (e.g., counts)
            return obj
        
        if hasattr(obj, '__dict__'):
            # Try to extract attributes
            try:
                return {
                    'type': type(obj).__name__,
                    'str_repr': str(obj)
                }
            except Exception:
                pass
        
        # Fallback: convert to string
        return str(obj)
    
    def save_results(self, results: Dict[str, Any]) -> None:
        """Save execution results to the results directory"""
        try:
            self.results_path.mkdir(parents=True, exist_ok=True)
            
            # Save main results
            result_file = self.results_path / "result.json"
            with open(result_file, 'w') as f:
                json.dump(results, f, indent=2, default=str)
            
            # Save metadata
            metadata_file = self.results_path / "metadata.json"
            metadata = {
                "job_id": self.job_id,
                "execution_time": results.get("execution_time_sec"),
                "status": results.get("status"),
                "timestamp": results.get("timestamp"),
                "result_files": ["result.json"]
            }
            
            with open(metadata_file, 'w') as f:
                json.dump(metadata, f, indent=2)
            
            logger.info("results_saved",
                       result_file=str(result_file),
                       metadata_file=str(metadata_file))
            
        except Exception as e:
            logger.error("failed_to_save_results", error=str(e))
            raise

def main():
    """Main entry point for the quantum executor"""
    logger.info("quantum_runtime_starting")
    
    executor = QuantumExecutor()
    
    try:
        # Execute the quantum circuit
        results = executor.execute()
        
        # Save results
        executor.save_results(results)
        
        # Print summary for container logs
        print(f"✓ Execution completed: {results['status']}")
        print(f"✓ Execution time: {results['execution_time_sec']:.2f}s")
        
        # Exit with appropriate code
        if results["status"] == "succeeded":
            sys.exit(0)
        else:
            sys.exit(1)
            
    except Exception as e:
        logger.error("quantum_runtime_failed", error=str(e))
        print(f"✗ Execution failed: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()