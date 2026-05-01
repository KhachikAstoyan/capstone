const fs = require('fs');

const SOURCE_PATH = '/workspace/solution.js';
const testCases = JSON.parse(fs.readFileSync('/workspace/tests.json', 'utf8'));
const timeLimitMs = %d;
const tleTimeMs = %d;
const results = [];

const _origWrite = process.stdout.write.bind(process.stdout);
let _captureBuffer = null;

process.stdout.write = function(chunk, encoding, callback) {
    const str = typeof chunk === 'string' ? chunk : chunk.toString(encoding || 'utf8');
    if (_captureBuffer !== null) {
        _captureBuffer += str;
        if (typeof encoding === 'function') encoding();
        else if (typeof callback === 'function') callback();
        return true;
    }
    return _origWrite(chunk, encoding, callback);
};

function compare(actualStr, expectedStr) {
    try {
        return JSON.stringify(JSON.parse(actualStr)) === JSON.stringify(JSON.parse(expectedStr));
    } catch (_) {
        return actualStr === expectedStr.trim();
    }
}

let sol;
try {
    _captureBuffer = '';
    sol = require(SOURCE_PATH);
} catch (e) {
    _captureBuffer = null;
    const err = String((e && e.stack) || e).slice(0, 500);
    for (const tc of testCases) {
        results.push({ id: tc.id, verdict: 'CompilationError', time_ms: 0, actual: '', stderr: err, stdout: '' });
    }
    _origWrite(JSON.stringify(results) + '\n');
    process.exit(0);
}
_captureBuffer = null;

for (const tc of testCases) {
    _captureBuffer = '';
    const t0 = Date.now();
    try {
        const data = JSON.parse(tc.input);
        const actualVal = sol._runTest(data);
        const elapsedMs = Date.now() - t0;
        const actualStr = JSON.stringify(actualVal);
        const stdout = _captureBuffer.slice(0, 2000);
        _captureBuffer = null;
        if (elapsedMs > timeLimitMs) {
            results.push({ id: tc.id, verdict: 'TimeLimitExceeded', time_ms: tleTimeMs, actual: actualStr, stdout });
        } else {
            const verdict = compare(actualStr, tc.expected) ? 'Accepted' : 'WrongAnswer';
            results.push({ id: tc.id, verdict, time_ms: elapsedMs, actual: actualStr, stdout });
        }
    } catch (e) {
        const elapsedMs = Date.now() - t0;
        const err = String((e && e.stack) || e).slice(0, 500);
        const stdout = (_captureBuffer || '').slice(0, 2000);
        _captureBuffer = null;
        results.push({ id: tc.id, verdict: 'RuntimeError', time_ms: elapsedMs, actual: '', stderr: err, stdout });
    }
}

_captureBuffer = null;
_origWrite(JSON.stringify(results) + '\n');
