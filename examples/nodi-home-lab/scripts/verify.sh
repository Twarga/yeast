#!/usr/bin/env bash
set -euo pipefail

# Host-side verification script for the Yeast Nodi Home Lab.
# Run this after `yeast up` has finished provisioning all VMs.

echo "=========================================="
echo "  Yeast Nodi Home Lab Verification"
echo "=========================================="

FAIL=0

check() {
  local name="$1"
  shift
  echo -n "  [$name] ... "
  if "$@" > /dev/null 2>&1; then
    echo "OK"
  else
    echo "FAIL"
    FAIL=1
  fi
}

echo ""
echo "1. Gateway Landing Page"
check "HTTP 200" yeast exec gateway -- curl -sf http://localhost
check "Contains 'Yeast'" grep -q "Yeast" <<< "$(yeast exec gateway -- curl -sf http://localhost)"

echo ""
echo "2. Storage Services"
check "Nodi UI reachable" yeast exec storage -- curl -sf http://localhost:7319

echo ""
echo "3. Alpha (Dev Workstation)"
check "NFS shared mounted" yeast exec alpha -- test -d /mnt/nfs/shared
check "NFS backup mounted" yeast exec alpha -- test -d /mnt/nfs/backup
check "Shared file exists" yeast exec alpha -- test -f /mnt/nfs/shared/readme.txt
check "File content OK" yeast exec alpha -- grep -q "Hello from alpha" /mnt/nfs/shared/readme.txt

echo ""
echo "4. Beta (Backup Workstation)"
check "SMB backup mounted" yeast exec beta -- test -d /mnt/smb/backup
check "Workspace synced" yeast exec beta -- test -d /mnt/smb/backup/beta-workspace
check "Backup data exists" yeast exec beta -- test -f /mnt/smb/backup/beta-workspace/data.txt

echo ""
echo "5. Cross-VM Reachability"
check "alpha pings storage" yeast exec alpha -- ping -c 1 -W 3 192.168.2.50
check "beta pings storage" yeast exec beta -- ping -c 1 -W 3 192.168.2.50
check "alpha pings beta" yeast exec alpha -- ping -c 1 -W 3 192.168.2.12

echo ""
echo "=========================================="
if [ "$FAIL" -eq 0 ]; then
  echo "  All checks passed."
else
  echo "  Some checks failed. Review output above."
  exit 1
fi
echo "=========================================="
