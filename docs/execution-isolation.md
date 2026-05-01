# Execution Isolation

## What "execution identity" means

Every process on Linux runs as an OS user identified by a UID and GID. "Execution identity" refers to which UID/GID a submitted program runs as inside the sandbox. If two submissions run as the same user, one could theoretically read the other's temporary files if cleanup fails or if they overlap in time.

## What is already in place

Each submission runs inside its own isolated Docker container or Firecracker microVM. The following restrictions are applied to every container:

| Control | Setting | Effect |
|---------|---------|--------|
| Network | `NetworkMode: none` | No outbound or inbound network access |
| Filesystem | `ReadonlyRootfs: true` | Container root is read-only |
| `/tmp` | tmpfs mount | Writable scratch space, isolated per container |
| Capabilities | `CapDrop: ALL` | All Linux capabilities removed |
| Privilege escalation | `no-new-privileges` | `setuid`/`setgid` binaries have no effect |
| Memory | Per-problem limit | Enforced by cgroups |
| PIDs | Language-dependent limit (64–256) | Prevents fork bombs |
| User | `65534:65534` (nobody) | Process runs as non-root inside the container |

For Firecracker, each VM is restored from a language snapshot and run under a dedicated jailer UID (default 900) on the host.

## The `nobody` user change

Previously Docker containers had no `User` set, so submitted code ran as root inside the container. Although all capabilities were dropped, root-inside-container is still a larger attack surface for container escape exploits.

Setting `User: "65534:65534"` (the `nobody` user on Linux) means:
- Submitted code cannot write to most of the container filesystem
- File permission checks apply normally — no root bypass
- Any container escape attempt starts from a low-privilege identity

This is configured in `internal/worker/docker_executor.go`.

## What is NOT done: per-submission UID isolation

True per-submission UID isolation would assign a different, unique UID to each container so that even if two containers' `/tmp` directories somehow became accessible to each other on the host, the OS would prevent cross-user reads.

This requires:
- Allocating a UID range per worker host in `/etc/subuid` and `/etc/subgid`
- Using Docker user namespace remapping (`--userns-remap`) per container, or allocating UIDs dynamically at job start
- Tracking which UIDs are in use to avoid collisions

This complexity is not justified by the current threat model because:
1. Each container already has its own isolated tmpfs — there is no shared filesystem path
2. Containers are removed immediately after execution (`ContainerRemove` with `Force: true`)
3. The `nobody` user combined with dropped capabilities already prevents most privilege-based escapes

## gVisor (stronger syscall isolation)

The Docker executor supports the gVisor runtime (`runsc`), which interposes on all syscalls from the container in user space. This provides an additional layer of isolation beyond Linux namespaces and capabilities.

Enable it by setting:

```
WORKER_DOCKER_RUNTIME=runsc
```

gVisor trades some performance for significantly stronger syscall-level isolation and is recommended for higher-security deployments.
