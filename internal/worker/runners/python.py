import json, sys, time, signal, importlib.util, traceback, io

SOURCE_PATH = '/workspace/solution.py'
test_cases = json.loads(open('/workspace/tests.json').read())
time_limit_sec = %d / 1000.0
time_limit_ms = %d
results = []

_real_stdout = sys.stdout

def _compare(actual_str, expected_str):
    try:
        return json.loads(actual_str) == json.loads(expected_str)
    except (ValueError, TypeError):
        return actual_str == expected_str.strip()

_spec = importlib.util.spec_from_file_location('solution', SOURCE_PATH)
_solution = importlib.util.module_from_spec(_spec)
sys.stdout = io.StringIO()
try:
    _spec.loader.exec_module(_solution)
except Exception:
    sys.stdout = _real_stdout
    err = traceback.format_exc()[:500]
    for tc in test_cases:
        results.append({'id': tc['id'], 'verdict': 'CompilationError', 'time_ms': 0, 'actual': '', 'stderr': err, 'stdout': ''})
    print(json.dumps(results))
    sys.exit(0)
sys.stdout = _real_stdout

class _Timeout(Exception):
    pass

def _on_alarm(signum, frame):
    raise _Timeout()

signal.signal(signal.SIGALRM, _on_alarm)

for tc in test_cases:
    _capture = io.StringIO()
    sys.stdout = _capture
    t0 = time.perf_counter()
    signal.setitimer(signal.ITIMER_REAL, time_limit_sec)
    try:
        data = json.loads(tc['input'])
        actual_val = _solution._run_test(data)
        signal.setitimer(signal.ITIMER_REAL, 0)
        elapsed_ms = int((time.perf_counter() - t0) * 1000)
        actual_str = json.dumps(actual_val)
        verdict = 'Accepted' if _compare(actual_str, tc['expected']) else 'WrongAnswer'
        user_out = _capture.getvalue()[:2000]
        results.append({'id': tc['id'], 'verdict': verdict, 'time_ms': elapsed_ms, 'actual': actual_str, 'stdout': user_out})
    except _Timeout:
        signal.setitimer(signal.ITIMER_REAL, 0)
        user_out = _capture.getvalue()[:2000]
        results.append({'id': tc['id'], 'verdict': 'TimeLimitExceeded', 'time_ms': time_limit_ms, 'actual': '', 'stdout': user_out})
    except Exception:
        signal.setitimer(signal.ITIMER_REAL, 0)
        elapsed_ms = int((time.perf_counter() - t0) * 1000)
        user_out = _capture.getvalue()[:2000]
        tb = traceback.format_exc()[:500]
        results.append({'id': tc['id'], 'verdict': 'RuntimeError', 'time_ms': elapsed_ms, 'actual': '', 'stdout': user_out, 'stderr': tb})
    finally:
        sys.stdout = _real_stdout

print(json.dumps(results))
