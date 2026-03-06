#!/usr/bin/env python3
"""
Quantum Circuit Analyzer Service

FastAPI service that analyzes Qiskit quantum circuits without execution
to estimate complexity, resource requirements, and optimal simulation methods.
"""

import os
import sys
import time
import traceback
from typing import Dict, Any, Optional

import structlog
import uvicorn
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field

from .interceptor import analyze_circuit_from_code
from .complexity import ComplexityAnalyzer

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

app = FastAPI(
    title="Quantum Circuit Analyzer",
    description="Analyzes Qiskit quantum circuits for complexity and resource estimation",
    version="0.1.0",
)

class AnalysisRequest(BaseModel):
    """Request model for circuit analysis"""
    code: str = Field(..., description="Python code containing Qiskit circuit")
    language: str = Field(default="python", description="Programming language")
    options: Optional[Dict[str, Any]] = Field(default=None, description="Analysis options")

class ComplexityResult(BaseModel):
    """Circuit complexity analysis result"""
    qubits: int = Field(..., description="Number of qubits")
    depth: int = Field(..., description="Circuit depth")
    gate_count: int = Field(..., description="Total number of gates")
    cx_count: int = Field(..., description="Number of CNOT/CX gates")
    parallelism: float = Field(..., description="Circuit parallelism factor (0.0-1.0)")
    memory_bytes: int = Field(..., description="Estimated memory requirement in bytes")
    complexity_class: str = Field(..., description="Complexity class (A/B/C/D)")
    recommended_method: str = Field(..., description="Recommended simulation method")
    estimated_cpu: int = Field(..., description="Estimated CPU cores needed")
    estimated_memory_mb: int = Field(..., description="Estimated memory in MB")
    estimated_time_sec: int = Field(..., description="Estimated execution time in seconds")
    recommended_pool: str = Field(..., description="Recommended node pool")

class AnalysisResponse(BaseModel):
    """Response model for circuit analysis"""
    success: bool = Field(..., description="Whether analysis succeeded")
    complexity: Optional[ComplexityResult] = Field(None, description="Complexity analysis result")
    error: Optional[str] = Field(None, description="Error message if analysis failed")
    analysis_time_ms: int = Field(..., description="Time taken for analysis in milliseconds")

@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {"status": "healthy", "service": "quantum-circuit-analyzer"}

@app.post("/analyze", response_model=AnalysisResponse)
async def analyze_circuit(request: AnalysisRequest):
    """
    Analyze a quantum circuit for complexity and resource requirements.
    
    This endpoint analyzes Qiskit Python code without executing the simulation,
    extracting circuit properties and estimating resource requirements.
    """
    start_time = time.time()
    
    logger.info("circuit_analysis_started", 
                language=request.language,
                code_length=len(request.code))
    
    try:
        # Validate language
        if request.language != "python":
            raise HTTPException(
                status_code=400, 
                detail=f"Unsupported language: {request.language}. Only 'python' is supported."
            )
        
        # Analyze circuit from code
        circuits = analyze_circuit_from_code(request.code)
        
        if not circuits:
            raise HTTPException(
                status_code=400,
                detail="No quantum circuits found in the provided code"
            )
        
        # For now, analyze the first circuit found
        # TODO: Handle multiple circuits
        circuit_data = circuits[0]
        
        # Perform complexity analysis
        analyzer = ComplexityAnalyzer()
        complexity = analyzer.analyze(circuit_data['circuit'], circuit_data.get('shots', 1024))
        
        # Calculate analysis time
        analysis_time_ms = int((time.time() - start_time) * 1000)
        
        result = ComplexityResult(
            qubits=complexity['qubits'],
            depth=complexity['depth'],
            gate_count=complexity['gate_count'],
            cx_count=complexity['cx_count'],
            parallelism=complexity['parallelism'],
            memory_bytes=complexity['memory_bytes'],
            complexity_class=complexity['complexity_class'],
            recommended_method=complexity['recommended_method'],
            estimated_cpu=complexity['estimated_cpu'],
            estimated_memory_mb=complexity['estimated_memory_mb'],
            estimated_time_sec=complexity['estimated_time_sec'],
            recommended_pool=complexity['recommended_pool']
        )
        
        logger.info("circuit_analysis_completed",
                   qubits=result.qubits,
                   depth=result.depth,
                   complexity_class=result.complexity_class,
                   analysis_time_ms=analysis_time_ms)
        
        return AnalysisResponse(
            success=True,
            complexity=result,
            analysis_time_ms=analysis_time_ms
        )
        
    except HTTPException:
        # Re-raise HTTP exceptions as-is
        raise
    
    except Exception as e:
        # Log detailed error information
        error_msg = f"Circuit analysis failed: {str(e)}"
        logger.error("circuit_analysis_failed",
                    error=error_msg,
                    traceback=traceback.format_exc())
        
        analysis_time_ms = int((time.time() - start_time) * 1000)
        
        return AnalysisResponse(
            success=False,
            error=error_msg,
            analysis_time_ms=analysis_time_ms
        )

if __name__ == "__main__":
    # Configuration from environment variables
    host = os.getenv("HOST", "0.0.0.0")
    port = int(os.getenv("PORT", "8081"))
    log_level = os.getenv("LOG_LEVEL", "info")
    workers = int(os.getenv("WORKERS", "1"))
    
    logger.info("starting_circuit_analyzer_service",
               host=host,
               port=port,
               log_level=log_level,
               workers=workers)
    
    uvicorn.run(
        "src.server:app",
        host=host,
        port=port,
        log_level=log_level,
        workers=workers,
        reload=os.getenv("RELOAD", "false").lower() == "true"
    )