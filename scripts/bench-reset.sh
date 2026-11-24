#!/bin/sh
# bench-reset.sh — measure real KVM live-snapshot timings on this host.
#
# Boots a paused QEMU with a small scratch qcow2, then measures:
#   - migrate to file: (RAM save throughput — the live-snapshot core)
#   - migrate-incoming resume (restore core)
#   - reported migration capabilities (mapped-ram, multifd)
#
# Usage: ./scripts/bench-reset.sh [memory-mb]   (default 512)
# Safe: everything runs in a mktemp dir; QEMU never touches a real disk/OS.

set -eu

MEM_MB="${1:-512}"
QEMU="${QEMU:-qemu-system-x86_64}"
QIMG="${QEMU_IMG:-qemu-img}"
PY="${PYTHON:-python3}"

echo "=== yeast reset spike ==="
$QEMU --version | head -1
$QIMG --version | head -1
if [ -c /dev/kvm ]; then echo "kvm: /dev/kvm present"; else echo "kvm: MISSING (results reflect TCG)"; fi
echo "memory: ${MEM_MB} MiB"
echo

WORK="$(mktemp -d /tmp/yeast-bench-XXXXXX)"
QPID="" ; QPID2=""
trap '[ -n "$QPID" ] && kill "$QPID" 2>/dev/null; [ -n "$QPID2" ] && kill "$QPID2" 2>/dev/null; rm -rf "$WORK"' EXIT
SOCK="$WORK/qmp.sock"
DISK="$WORK/disk.qcow2"
MEMFILE="$WORK/memory.bin"

$QIMG create -f qcow2 "$DISK" 1G >/dev/null

ACCEL="tcg"
if [ -c /dev/kvm ] && [ -w /dev/kvm ]; then ACCEL="kvm"; fi

# QMP helper: qmp.py <socket> <cmd-json> [more-json...]
# Prints one JSON object per response line (skip async events).
cat > "$WORK/qmp.py" <<'PYEOF'
import json, socket, sys

sock_path = sys.argv[1]
cmds = sys.argv[2:]

s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
s.settimeout(10)
s.connect(sock_path)
f = s.makefile("rw")
f.readline()  # greeting

def send(cmd):
    f.write(cmd + "\n"); f.flush()
    while True:
        line = f.readline()
        if not line:
            return None
        obj = json.loads(line)
        if "event" in obj:
            continue
        return obj

send('{"execute":"qmp_capabilities"}')
for c in cmds:
    resp = send(c)
    print(json.dumps(resp))
PYEOF

wait_sock() {
    i=0
    while [ ! -S "$1" ]; do
        i=$((i + 1)); [ "$i" -gt 100 ] && { echo "FAIL: $1 never appeared"; exit 1; }
        sleep 0.05
    done
}

qmp()  { "$PY" "$WORK/qmp.py" "$SOCK"  "$@" ; }
qmp2() { "$PY" "$WORK/qmp.py" "$SOCK2" "$@" ; }

# --- VM 1: source -------------------------------------------------------------
"$QEMU" \
    -machine "accel=$ACCEL" -cpu max -m "$MEM_MB" \
    -display none -nodefaults -S \
    -qmp "unix:$SOCK,server,nowait" \
    -drive "file=$DISK,if=virtio,format=qcow2" &
QPID=$!
wait_sock "$SOCK"

echo "--- migration capabilities (interesting subset) ---"
qmp '{"execute":"query-migrate-capabilities"}' | "$PY" -c '
import json,sys
r=json.loads(sys.stdin.read())["return"]
interesting={"mapped-ram","multifd","events","postcopy-ram","compress","dirty-bitmaps"}
for c in r:
    if c["capability"] in interesting:
        print("  {}: {}".format(c["capability"], c["state"]))
'

# Enable mapped-ram + multifd on BOTH source and (later) destination.
SETCAPS='{"execute":"migrate-set-capabilities","arguments":{"capabilities":[{"capability":"mapped-ram","state":true},{"capability":"multifd","state":true},{"capability":"events","state":true}]}}'
qmp "$SETCAPS" >/dev/null

# Warm guest, then freeze.
qmp '{"execute":"cont"}' >/dev/null
sleep 0.3
qmp '{"execute":"stop"}' >/dev/null

echo "--- ram save: migrate file: ---"
START=$(date +%s%N)
qmp "{\"execute\":\"migrate\",\"arguments\":{\"uri\":\"file:$MEMFILE\"}}" >/dev/null
STATUS=timeout
i=0
while [ "$i" -lt 1200 ]; do
    OUT=$(qmp '{"execute":"query-migrate"}')
    case "$OUT" in
        *'"status": "completed"'*) STATUS=completed; break ;;
        *'"status": "failed"'*|*'"status": "cancelled"'*) STATUS=failed; break ;;
    esac
    i=$((i + 1)); sleep 0.05
done
END=$(date +%s%N)
SAVE_MS=$(( (END - START) / 1000000 ))
SIZE=$(stat -c %s "$MEMFILE" 2>/dev/null || echo 0)
echo "  save: ${SAVE_MS} ms  ($(( SIZE / 1024 / 1024 )) MiB, status=$STATUS)"

# --- VM 2: destination --------------------------------------------------------
SOCK2="$WORK/qmp2.sock"
"$QEMU" \
    -machine "accel=$ACCEL" -cpu max -m "$MEM_MB" \
    -display none -nodefaults -incoming defer \
    -qmp "unix:$SOCK2,server,nowait" \
    -drive "file=$DISK,if=virtio,format=qcow2" &
QPID2=$!
wait_sock "$SOCK2"
qmp2 "$SETCAPS" >/dev/null

echo "--- ram restore: migrate-incoming file: ---"
START=$(date +%s%N)
qmp2 "{\"execute\":\"migrate-incoming\",\"arguments\":{\"uri\":\"file:$MEMFILE\"}}" >/dev/null
RSTATUS=timeout
i=0
while [ "$i" -lt 1200 ]; do
    ST=$(qmp2 '{"execute":"query-status"}')
    case "$ST" in
        *'"status": "running"'*|*'"status": "paused"'*) RSTATUS=restored; break ;;
    esac
    OUT=$(qmp2 '{"execute":"query-migrate"}')
    case "$OUT" in
        *'"status": "failed"'*) RSTATUS=failed; break ;;
    esac
    i=$((i + 1)); sleep 0.05
done
END=$(date +%s%N)
LOAD_MS=$(( (END - START) / 1000000 ))
echo "  restore: ${LOAD_MS} ms  (guest status=$RSTATUS)"

# --- internal snapshot-save support probe -------------------------------------
echo "--- snapshot-save job probe ---"
PROBE=$(qmp '{"execute":"query-commands"}' | "$PY" -c '
import json,sys
names={c["name"] for c in json.loads(sys.stdin.read())["return"]}
for n in ("snapshot-save","snapshot-load","blockdev-snapshot-sync","migrate-incoming"):
    print("  {}: {}".format(n, "yes" if n in names else "no"))
')
echo "$PROBE"

kill "$QPID" "$QPID2" 2>/dev/null || true
QPID=""; QPID2=""
echo
echo "summary: save=${SAVE_MS}ms restore=${LOAD_MS}ms mem=${MEM_MB}MiB accel=$ACCEL"
