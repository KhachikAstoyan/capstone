import json, subprocess, sys, time, os, shutil

test_cases = json.loads(open('/workspace/tests.json').read())
time_limit_sec = %d / 1000.0
results = []

# /workspace is read-only, so copy source into the writable tmpfs before compiling.
os.makedirs('/tmp/build', exist_ok=True)
shutil.copy('/workspace/Main.java', '/tmp/build/Main.java')

compile_proc = subprocess.run(
    ['javac', '-cp', '/opt/jars/*', '-d', '/tmp/build', '/tmp/build/Main.java'],
    capture_output=True,
    text=True,
)
if compile_proc.returncode != 0:
    err = compile_proc.stderr[:500]
    for tc in test_cases:
        results.append({'id': tc['id'], 'verdict': 'CompilationError', 'time_ms': 0, 'actual': '', 'stderr': err, 'stdout': ''})
    print(json.dumps(results))
    sys.exit(0)

for tc in test_cases:
    t0 = time.perf_counter()
    try:
        proc = subprocess.run(
            ['java', '-cp', '/tmp/build:/opt/jars/*', 'Main'],
            input=tc['input'],
            capture_output=True,
            text=True,
            timeout=time_limit_sec,
        )
        elapsed_ms = int((time.perf_counter() - t0) * 1000)
        user_stdout = proc.stdout[:2000] if proc.stdout else ''
        expected = tc['expected'].strip()
        if proc.returncode != 0:
            verdict = 'RuntimeError'
            actual = ''
            stderr_out = proc.stderr[:500] if proc.stderr else ''
        else:
            stderr_out = ''
            try:
                with open('/tmp/capstone_result') as _rf:
                    actual = _rf.read().strip()
            except Exception:
                actual = ''
            try:
                import json as _j
                verdict = 'Accepted' if _j.loads(actual) == _j.loads(expected) else 'WrongAnswer'
            except (ValueError, TypeError):
                verdict = 'Accepted' if actual == expected else 'WrongAnswer'
        results.append({
            'id': tc['id'],
            'verdict': verdict,
            'time_ms': elapsed_ms,
            'actual': actual,
            'stderr': stderr_out,
            'stdout': user_stdout,
        })
    except subprocess.TimeoutExpired:
        results.append({'id': tc['id'], 'verdict': 'TimeLimitExceeded', 'time_ms': %d, 'actual': '', 'stdout': ''})
    except Exception as e:
        results.append({'id': tc['id'], 'verdict': 'RuntimeError', 'time_ms': 0, 'stderr': str(e), 'stdout': str(e)})

print(json.dumps(results))
