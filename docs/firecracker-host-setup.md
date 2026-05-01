# Firecracker Host Setup

This guide covers everything needed to run the worker with `WORKER_EXECUTOR=firecracker` on a Linux host.

---

## Prerequisites

| Requirement | Check |
|---|---|
| Linux x86-64 host | `uname -m` → `x86_64` |
| KVM available | `ls /dev/kvm` |
| Kernel ≥ 5.10 (vsock support) | `uname -r` |
| Root or `kvm` group membership | `groups $USER` |
| `e2fsprogs` (mkfs.ext4) | `apt install e2fsprogs` |
| Docker (to build rootfs images) | `docker --version` |

---

## 1. KVM access

```bash
# Add your user to the kvm group
sudo usermod -aG kvm $USER
# Or set a permissive udev rule (CI environments)
echo 'KERNEL=="kvm", GROUP="kvm", MODE="0666"' | sudo tee /etc/udev/rules.d/99-kvm.rules
sudo udevadm trigger
```

---

## 2. Install Firecracker and Jailer

Download the latest Firecracker release from GitHub:

```bash
ARCH=x86_64
FC_VERSION=v1.10.1  # check https://github.com/firecracker-microvm/firecracker/releases
RELEASE_URL="https://github.com/firecracker-microvm/firecracker/releases/download/${FC_VERSION}"

curl -Lo /tmp/firecracker.tgz \
  "${RELEASE_URL}/firecracker-${FC_VERSION}-${ARCH}.tgz"

tar -xf /tmp/firecracker.tgz -C /tmp
sudo install -m 0755 /tmp/release-${FC_VERSION}-${ARCH}/firecracker-${FC_VERSION}-${ARCH} /usr/bin/firecracker
sudo install -m 0755 /tmp/release-${FC_VERSION}-${ARCH}/jailer-${FC_VERSION}-${ARCH}      /usr/bin/jailer
```

Verify:
```bash
firecracker --version
jailer --version
```

---

## 3. Create the jailer UID/GID

The jailer drops privileges to a dedicated user so that microVMs run with minimal permissions.

```bash
sudo groupadd -g 900 fc-worker
sudo useradd  -u 900 -g 900 -s /sbin/nologin -d /srv/jailer fc-worker
```

---

## 4. Directory structure

```bash
sudo mkdir -p /var/lib/fc/{kernels,rootfs,snapshots}
sudo mkdir -p /srv/jailer
sudo chown -R fc-worker:fc-worker /var/lib/fc /srv/jailer
```

---

## 5. Download a Firecracker-compatible kernel

Firecracker requires a bare `vmlinux` kernel (not a compressed `bzImage`). The Firecracker project publishes pre-built kernels:

```bash
FC_KERNEL_VERSION=6.1.102  # or latest from FC CI
curl -Lo /var/lib/fc/kernels/vmlinux-6.1 \
  "https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.10/x86_64/vmlinux-${FC_KERNEL_VERSION}"

# Symlink to the active kernel
sudo ln -sf /var/lib/fc/kernels/vmlinux-6.1 /var/lib/fc/vmlinux
```

---

## 6. Build rootfs images

From the repository root (requires Docker and root for `mkfs.ext4`):

```bash
# All languages at once
make build-rootfs-all FC_ROOTFS_DIR=/var/lib/fc/rootfs

# Or individually
make build-rootfs-python     FC_ROOTFS_DIR=/var/lib/fc/rootfs
make build-rootfs-javascript FC_ROOTFS_DIR=/var/lib/fc/rootfs
make build-rootfs-go         FC_ROOTFS_DIR=/var/lib/fc/rootfs
make build-rootfs-java       FC_ROOTFS_DIR=/var/lib/fc/rootfs
```

Each command produces `/var/lib/fc/rootfs/{language}.ext4` (~256–512 MiB per language).

---

## 7. cgroup v2

Firecracker's jailer uses cgroups for resource limits. Verify cgroup v2 is active:

```bash
stat -f -c %T /sys/fs/cgroup  # should print "cgroup2fs"
```

If using cgroup v1, add `systemd.unified_cgroup_hierarchy=1` to the kernel command line and reboot.

---

## 8. Environment variables

Set these in the worker's environment (or `.env` file):

| Variable | Default | Description |
|---|---|---|
| `WORKER_EXECUTOR` | `docker` | Set to `firecracker` to enable |
| `WORKER_FC_BIN` | `/usr/bin/firecracker` | Path to firecracker binary |
| `WORKER_JAILER_BIN` | `/usr/bin/jailer` | Path to jailer binary |
| `WORKER_FC_KERNEL` | `/var/lib/fc/vmlinux` | Path to vmlinux kernel |
| `WORKER_FC_ROOTFS_DIR` | `/var/lib/fc/rootfs` | Directory with `{lang}.ext4` files |
| `WORKER_FC_SNAPSHOTS_DIR` | `/var/lib/fc/snapshots` | Where snapshot pairs are stored |
| `WORKER_FC_CHROOT_BASE` | `/srv/jailer` | Jailer chroot base directory |
| `WORKER_FC_JAILER_UID` | `900` | UID for jailer privilege drop |
| `WORKER_FC_JAILER_GID` | `900` | GID for jailer privilege drop |
| `WORKER_FC_VCPU` | `1` | vCPUs per microVM |
| `WORKER_FC_MEM_MB` | `256` | Memory (MiB) per microVM |

Minimal `.env` for Firecracker mode:

```env
WORKER_CP_URL=http://localhost:9090
WORKER_EXECUTOR=firecracker
WORKER_LANGUAGES=python,javascript,go,java
WORKER_CAPACITY=4
```

---

## 9. Smoke test

```bash
# Build worker binary (Linux target)
GOOS=linux GOARCH=amd64 make build-worker

# Run with Firecracker executor
WORKER_CP_URL=http://localhost:9090 \
WORKER_EXECUTOR=firecracker \
WORKER_LANGUAGES=python \
WORKER_CAPACITY=1 \
sudo -E ./build/worker
```

At startup the worker will:
1. Boot one microVM per configured language
2. Wait for the in-VM `fc-agent` to accept vsock connections
3. Capture a full memory+state snapshot
4. Kill the boot VMs
5. Log `"firecracker executor ready"` and begin polling for jobs

---

## 10. Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `cannot open /dev/kvm` | KVM inaccessible | Add user to `kvm` group |
| `snapshot/create failed` | cgroup v1 active | Enable cgroup v2 (see §7) |
| `fc-agent not ready after 30s` | Wrong kernel or broken rootfs | Rebuild rootfs; verify vmlinux is uncompressed |
| `copy snapshot: link: cross-device` | Rootfs/snapshot dirs on different filesystems | Move `/var/lib/fc` and `/srv/jailer` to the same filesystem |
| Jailer `EPERM` errors | Worker not running as root | Run worker as root or grant `CAP_SYS_CHROOT` |
