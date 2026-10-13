"""Tiny test runner for machines without pytest. Prefer `pytest -q tests` when available."""
import contextlib, importlib, inspect, pathlib, sys, types

sys.path.insert(0, str(pathlib.Path(__file__).parent))
if "pytest" not in sys.modules:
    try:
        import pytest  # noqa: F401
    except ImportError:
        class _Raises:
            def __init__(self, exc): self.exc = exc
            def __enter__(self): return self
            def __exit__(self, t, v, tb):
                assert t is not None and issubclass(t, self.exc), f"expected {self.exc.__name__}"
                return True
        stub = types.ModuleType("pytest"); stub.raises = _Raises; sys.modules["pytest"] = stub

failed = 0
for path in sorted(pathlib.Path("tests").glob("test_*.py")):
    mod = importlib.import_module(f"tests.{path.stem}")
    for name, fn in inspect.getmembers(mod, inspect.isfunction):
        if name.startswith("test_"):
            try:
                fn(); print("ok  ", name)
            except Exception as e:
                failed += 1; print("FAIL", name, repr(e))
sys.exit(1 if failed else 0)
