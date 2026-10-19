#!/usr/bin/env bash
# Offline doctor integration checks using a dedicated Podman store in Bubblewrap.
set -euo pipefail
if [ "${1:-}" = --inside ]; then
  scratch="$2"; modcache="$3"; image="$4"; engine_binary="$5"
  args=(--die-with-parent --new-session --unshare-pid --unshare-ipc --unshare-uts --unshare-net --clearenv)
  for path in /usr /bin /lib /lib64 /sys; do
    if [ -e "$path" ]; then args+=(--ro-bind "$path" "$path"); fi
  done
  args+=(--dir /etc)
  for path in /etc/passwd /etc/group /etc/subuid /etc/subgid /etc/os-release /etc/ld.so.cache /etc/localtime /etc/containers/policy.json /etc/containers/seccomp.json; do
    if [ -e "$path" ]; then args+=(--ro-bind "$path" "$path"); fi
  done
  for key in _CONTAINERS_USERNS_CONFIGURED _CONTAINERS_ROOTLESS_UID _CONTAINERS_ROOTLESS_GID; do
    if [ -n "${!key:-}" ]; then args+=(--setenv "$key" "${!key}"); fi
  done
  args+=(--ro-bind "$engine_binary" "$engine_binary" --proc /proc --dev /dev --bind "$scratch" "$scratch" --bind "$scratch/tmp" /var/tmp --ro-bind "$modcache" /gomod
    --chdir "$scratch/src"
    --setenv PATH "$scratch/bin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin"
    --setenv HOME "$scratch/home" --setenv XDG_RUNTIME_DIR "$scratch/runtime" --setenv TMPDIR "$scratch/tmp"
    --setenv GOPATH "$scratch/go" --setenv GOMODCACHE /gomod --setenv GOCACHE "$scratch/cache"
    --setenv GOPROXY off --setenv GOTOOLCHAIN local --setenv GOFLAGS -buildvcs=false
    --setenv GIT_CONFIG_NOSYSTEM 1 --setenv GIT_CEILING_DIRECTORIES "$scratch"
    --setenv EFCTL_DOCTOR_ISOLATED_STORE "$scratch" --setenv EFCTL_DOCTOR_BASE_ID "$image"
    --setenv EFCTL_DOCTOR_ENGINE_BINARY "$engine_binary")
  exec bwrap "${args[@]}" -- sh -c 'podman load --input "$EFCTL_DOCTOR_ISOLATED_STORE/base.oci" && go test -tags integration ./pkg/doctor -run "^TestDoctorRuntimeIntegration$" -v -count=1 -timeout 3m'
fi
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
command -v bwrap >/dev/null
engine_binary="$(command -v podman)"
image="${1:?Usage: tools/test-doctor-runtime-isolated.sh <existing-local-image-id>}"
[[ "$image" =~ ^(sha256:)?[a-fA-F0-9]{64}$ ]] || { echo 'Require an immutable already-local image ID.' >&2; exit 1; }
# Inspection and export never pull or start the host image. No host runtime
# sockets/storage are exposed to the tests; all runtime mutations use this store.
podman image inspect "$image" >/dev/null
mkdir -p "$root/tmp"
scratch="$(mktemp -d "$root/tmp/doctor-runtime.XXXXXXXX")"
mkdir -p "$scratch"/{src,home,tmp,cache,go,bin,storage,run,libpod,runtime}
chmod 700 "$scratch/home" "$scratch/runtime"
cleanup() {
  status=$?
  # This newly created store has never contained user/managed containers.
  # Image layers are created inside podman unshare, so only that namespace can
  # remove files the host user does not own. Never touch host storage.
  HOME="$scratch/home" XDG_RUNTIME_DIR="$scratch/runtime" podman --root "$scratch/storage" --runroot "$scratch/run" --tmpdir "$scratch/libpod" --storage-driver vfs rm --force --all >/dev/null 2>&1 || true
  if [ "$status" -ne 0 ]; then
    echo "Isolated runtime artifacts retained: $scratch" >&2
    return
  fi
  podman unshare rm -rf -- "$scratch"
}
trap cleanup EXIT
(cd "$root" && git ls-files --cached --others --exclude-standard -z | tar --null -T - -cf -) | tar -xf - -C "$scratch/src"
cat > "$scratch/bin/podman" <<'WRAPPER'
#!/bin/sh
printf '%s\n' "$*" >> "$EFCTL_DOCTOR_ISOLATED_STORE/commands.log"
exec "$EFCTL_DOCTOR_ENGINE_BINARY" --root "$EFCTL_DOCTOR_ISOLATED_STORE/storage" --runroot "$EFCTL_DOCTOR_ISOLATED_STORE/run" --tmpdir "$EFCTL_DOCTOR_ISOLATED_STORE/libpod" --storage-driver vfs "$@"
WRAPPER
chmod 700 "$scratch/bin/podman"
ln -s podman "$scratch/bin/docker"
podman save --format oci-archive --output "$scratch/base.oci" "$image"
modcache="$(go env GOMODCACHE)"
HOME="$scratch/home" XDG_RUNTIME_DIR="$scratch/runtime" TMPDIR="$scratch/tmp" \
  podman --root "$scratch/storage" --runroot "$scratch/run" --tmpdir "$scratch/libpod" --storage-driver vfs \
  unshare bash "$root/tools/test-doctor-runtime-isolated.sh" --inside "$scratch" "$modcache" "$image" "$engine_binary"
