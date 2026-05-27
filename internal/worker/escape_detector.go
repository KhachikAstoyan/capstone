package worker

import "strings"

// escapePatterns are substrings found in container stderr that indicate
// the user's code attempted to escape the sandbox at runtime.
// These fire when code slips through AI pre-validation and actually tries
// forbidden operations (network, filesystem writes, sensitive path reads).
var escapePatterns = []string{
	// Network escape attempts
	"network is unreachable",
	"connection refused",
	"connection timed out",
	"no route to host",
	"name or service not known",
	"socket operation on non-socket",
	"ENETUNREACH",
	"ECONNREFUSED",
	"java.net.ConnectException",
	"java.net.NoRouteToHostException",
	"java.net.UnknownHostException",

	// Filesystem / privilege escape attempts
	"read-only file system",
	"operation not permitted",
	"permission denied",

	// Sensitive path probing
	"/etc/passwd",
	"/etc/shadow",
	"/etc/hosts",
	"/proc/self",
	"/proc/1/",
	"/dev/mem",
	"/dev/kmem",

	// Privilege escalation
	"setuid",
	"setgid",
	"execve: permission denied",
}

// detectEscapeAttempt returns the first matched pattern found in stderr,
// or an empty string if none match.
func detectEscapeAttempt(stderr string) string {
	lower := strings.ToLower(stderr)
	for _, p := range escapePatterns {
		if strings.Contains(lower, strings.ToLower(p)) {
			return p
		}
	}
	return ""
}
