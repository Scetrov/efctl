#!/usr/bin/env bash
# Run Go/CLI tests in a disposable Bubblewrap filesystem, never the host checkout.
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
command -v bwrap >/dev/null || { echo 'Bubblewrap is required for isolated tests.' >&2; exit 1; }
mkdir -p "$root/tmp/isolated-go-cache"
scratch="$(mktemp -d "$root/tmp/isolated-tests.XXXXXXXX")"
cleanup() {
  status=$?
  if [ "$status" -eq 0 ]; then
    # Go makes extracted module directories read-only. Change only directories
    # in our own scratch cache (find does not follow symlinks), then remove it.
    if [ -d "$scratch/modules" ]; then
      find "$scratch/modules" -type d -exec chmod u+w {} +
    fi
    rm -rf -- "$scratch"
  else
    echo "Isolated test artifacts retained: $scratch" >&2
  fi
}
trap cleanup EXIT
mkdir -p "$scratch"/{src,home,tmp,cache,go}
# Include working changes without copying Git metadata, ignored caches, managed
# checkouts or credentials. Tests must not inherit a parent repository identity.
(cd "$root" && git ls-files --cached --others --exclude-standard -z | tar --null -T - -cf -) | tar -xf - -C "$scratch/src"
modcache="$(go env GOMODCACHE)"
args=(--die-with-parent --new-session --unshare-pid --unshare-ipc --unshare-uts --clearenv)
if [ "${1:-}" != --pre-commit ] && [ "${1:-}" != --modules ]; then args+=(--unshare-net); fi
# Build a minimal read-only userspace. Do not expose host homes, Git metadata,
# runtime sockets, /tmp, or container storage. All writable data is under ./tmp.
for path in /usr /bin /lib /lib64; do
  if [ -e "$path" ]; then args+=(--ro-bind "$path" "$path"); fi
done
args+=(--dir /etc)
for path in /etc/passwd /etc/group /etc/os-release /etc/ld.so.cache /etc/localtime; do
  if [ -e "$path" ]; then args+=(--ro-bind "$path" "$path"); fi
done
args+=(--proc /proc --dev /dev --bind "$scratch" /workspace --bind "$root/tmp/isolated-go-cache" /gocache --ro-bind "$modcache" /gomod
  --chdir /workspace/src
  --setenv PATH /usr/local/go/bin:/usr/local/bin:/usr/bin:/bin
  --setenv HOME /workspace/home --setenv TMPDIR /workspace/tmp
  --setenv GOPATH /workspace/go --setenv GOMODCACHE /gomod --setenv GOCACHE /gocache
  --setenv GOPROXY off --setenv GOTOOLCHAIN local --setenv GOFLAGS -buildvcs=false
  --setenv GIT_CONFIG_NOSYSTEM 1 --setenv GIT_CEILING_DIRECTORIES /workspace)
if [ "${1:-}" = --pre-commit ] || [ "${1:-}" = --modules ]; then
  for path in /etc/resolv.conf /etc/ssl/certs; do
    if [ -e "$path" ]; then args+=(--ro-bind "$path" "$path"); fi
  done
fi
if [ "${1:-}" = --modules ]; then
  # Module auditing can require test-only downloads not present in the host
  # cache. Use a fresh writable cache in scratch, with authenticated downloads.
  mkdir -p "$scratch/modules"
  args+=(--bind "$scratch/modules" /module-cache --setenv GOMODCACHE /module-cache
    --setenv GOPROXY https://proxy.golang.org --setenv GOSUMDB sum.golang.org
    --setenv GOTELEMETRY off)
  shift
  if [ "$#" -eq 0 ]; then set -- sh -c 'go mod download && go mod tidy -diff && go mod verify'; fi
fi
if [ "${1:-}" = --pre-commit ]; then
  # Quality scanners need network access, but never host credentials or Git
  # state. Copy only the configured cached hook environments, not old patches.
  precommit="$(readlink -f "$(command -v pre-commit)")"
  pythonenv="$(dirname "$(dirname "$precommit")")"
  hookcache="${PRE_COMMIT_HOME:-$HOME/.cache/pre-commit}"
  mkdir -p "$scratch/hooks"
  python3 - "$hookcache" "$scratch/hooks" <<'PY'
import os, shutil, sqlite3, sys
source, target = sys.argv[1:]
shutil.copy2(os.path.join(source, 'db.db'), os.path.join(target, 'db.db'))
connection = sqlite3.connect('file:' + os.path.join(source, 'db.db') + '?mode=ro', uri=True)
required = {('local:github.com/fzipp/gocyclo/cmd/gocyclo@latest', '1'), ('https://github.com/gitleaks/gitleaks', 'v8.21.2')}
found = set()
for repo, ref, path in connection.execute('select repo,ref,path from repos'):
    if (repo, ref) in required:
        shutil.copytree(path, os.path.join(target, os.path.basename(path)), symlinks=True)
        found.add((repo, ref))
if found != required:
    raise SystemExit('Configured hook environments are not cached; refusing implicit installation.')
PY
  args+=(--ro-bind "$pythonenv" "$pythonenv" --bind "$scratch/hooks" "$hookcache"
    --ro-bind "$(go env GOPATH)/bin" /toolbin
    --setenv PRE_COMMIT_HOME "$hookcache"
    --setenv PATH /toolbin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin)
  if [ -n "${EFCTL_ISOLATED_TOOLBIN:-}" ]; then
    updated_tools="$(readlink -f "$EFCTL_ISOLATED_TOOLBIN")"
    case "$updated_tools" in
      "$root/tmp/"*) ;;
      *) echo 'Updated tool directory must resolve under repository ./tmp.' >&2; exit 1 ;;
    esac
    [ -d "$updated_tools" ] || { echo 'Updated tool directory is missing.' >&2; exit 1; }
    args+=(--ro-bind "$updated_tools" /updated-tools
      --setenv PATH /updated-tools:/toolbin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin)
  fi
  set -- sh -c 'git init --quiet && git add --all && "$1" run --all-files' sh "$precommit"
fi
if [ "$#" -eq 0 ]; then set -- go test ./... -count=1 -timeout 5m; fi
bwrap "${args[@]}" -- "$@"
