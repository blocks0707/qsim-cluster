"""Unit tests for ComplexityAnalyzer"""

import pytest
from qiskit import QuantumCircuit

from src.complexity import ComplexityAnalyzer


@pytest.fixture
def analyzer():
    return ComplexityAnalyzer()


class TestComplexityAnalyzer:
    """Tests for ComplexityAnalyzer.analyze()"""

    def test_simple_bell_state(self, analyzer):
        """Class A circuit: 2-qubit Bell state"""
        qc = QuantumCircuit(2)
        qc.h(0)
        qc.cx(0, 1)

        result = analyzer.analyze(qc, shots=1024)

        assert result["qubits"] == 2
        assert result["complexity_class"] == "A"
        assert result["estimated_memory_mb"] >= 512  # minimum
        assert result["recommended_pool"] == "cpu"

    def test_medium_circuit(self, analyzer):
        """Class B circuit: 10-qubit GHZ"""
        qc = QuantumCircuit(10)
        qc.h(0)
        for i in range(9):
            qc.cx(i, i + 1)

        result = analyzer.analyze(qc, shots=1024)

        assert result["qubits"] == 10
        assert result["complexity_class"] == "B"
        assert result["cx_count"] >= 9

    def test_clifford_detection(self, analyzer):
        """Clifford-only circuit should be detected"""
        qc = QuantumCircuit(3)
        qc.h(0)
        qc.cx(0, 1)
        qc.cx(1, 2)
        qc.x(2)
        qc.z(0)

        result = analyzer.analyze(qc, shots=1024)

        assert result["is_clifford"] is True
        assert result["recommended_method"] == "stabilizer"

    def test_non_clifford_detection(self, analyzer):
        """Circuit with rotation gates is not Clifford"""
        qc = QuantumCircuit(2)
        qc.h(0)
        qc.rx(0.5, 1)
        qc.cx(0, 1)

        result = analyzer.analyze(qc, shots=1024)

        assert result["is_clifford"] is False

    def test_empty_circuit(self, analyzer):
        """Empty circuit should not crash"""
        qc = QuantumCircuit(1)
        result = analyzer.analyze(qc, shots=1024)

        assert result["qubits"] == 1
        assert result["complexity_class"] == "A"

    def test_result_keys(self, analyzer):
        """All expected keys should be present"""
        qc = QuantumCircuit(2)
        qc.h(0)
        qc.cx(0, 1)
        qc.measure_all()

        result = analyzer.analyze(qc, shots=1024)

        expected_keys = {
            "qubits", "depth", "original_depth", "gate_count",
            "original_gate_count", "cx_count", "parallelism",
            "memory_bytes", "complexity_class", "is_clifford",
            "recommended_method", "estimated_cpu", "estimated_memory_mb",
            "estimated_time_sec", "recommended_pool", "gate_breakdown",
        }
        assert expected_keys.issubset(result.keys())

    def test_memory_estimate_scales_with_qubits(self, analyzer):
        """Memory should scale exponentially for non-Clifford circuits"""
        qc_small = QuantumCircuit(5)
        qc_small.rx(0.5, 0)

        qc_large = QuantumCircuit(10)
        qc_large.rx(0.5, 0)

        r_small = analyzer.analyze(qc_small)
        r_large = analyzer.analyze(qc_large)

        assert r_large["memory_bytes"] > r_small["memory_bytes"]


class TestClassifyComplexity:
    """Tests for _classify_complexity"""

    def test_class_a(self):
        a = ComplexityAnalyzer()
        assert a._classify_complexity(3, 5, 10) == "A"

    def test_class_b(self):
        a = ComplexityAnalyzer()
        assert a._classify_complexity(10, 30, 100) == "B"

    def test_class_c(self):
        a = ComplexityAnalyzer()
        assert a._classify_complexity(20, 100, 500) == "C"

    def test_class_d(self):
        a = ComplexityAnalyzer()
        assert a._classify_complexity(30, 300, 2000) == "D"


class TestRecommendMethod:
    """Tests for _recommend_method"""

    def test_clifford_uses_stabilizer(self):
        a = ComplexityAnalyzer()
        assert a._recommend_method(10, True, 5, 20) == "stabilizer"

    def test_small_circuit_uses_statevector(self):
        a = ComplexityAnalyzer()
        assert a._recommend_method(10, False, 5, 20) == "statevector"

    def test_medium_low_entanglement_uses_mps(self):
        a = ComplexityAnalyzer()
        # cx_ratio = 5/100 = 0.05 < 0.3 → mps
        assert a._recommend_method(20, False, 5, 100) == "mps"

    def test_medium_high_entanglement_uses_statevector(self):
        a = ComplexityAnalyzer()
        # cx_ratio = 50/100 = 0.5 > 0.3 → statevector
        assert a._recommend_method(20, False, 50, 100) == "statevector"
