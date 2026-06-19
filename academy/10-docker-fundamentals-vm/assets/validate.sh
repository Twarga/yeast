#!/usr/bin/env bash
# Lab 10 validation — run from the lab folder: bash assets/validate.sh

set -euo pipefail

SSH_PORT=2216
SSH_USER=ubuntu
SSH_HOST=127.0.0.1
PASS=0
FAIL=0

check_ssh() {
  local label="$1"
  local cmd="$2"
  local expected="$3"

  actual=$(ssh -p "$SSH_PORT" \
    -o StrictHostKeyChecking=no \
    -o ConnectTimeout=5 \
    -o BatchMode=yes \
    "${SSH_USER}@${SSH_HOST}" "$cmd" 2>/dev/null || echo "CONNECTION_FAILED")

  if echo "$actual" | grep -q "$expected"; then
    echo "  PASS  $label"
    PASS=$((PASS + 1))
  else
    echo "  FAIL  $label"
    echo "        expected: $expected"
    echo "        got:      $actual"
    FAIL=$((FAIL + 1))
  fi
}

echo ""
echo "=== Lab 10: Docker Fundamentals ==="
echo ""

echo "Docker"
check_ssh "docker installed"          "docker --version"                              "Docker"
check_ssh "docker daemon active"      "sudo systemctl is-active docker"               "active"
check_ssh "ubuntu in docker group"    "groups"                                        "docker"

echo ""
echo "Containers"
check_ssh "nginx container running"   "docker ps --format '{{.Names}}'"               "webserver"
check_ssh "nginx responds"            "curl -s -o /dev/null -w '%{http_code}' http://localhost:8080" "200"

echo ""
echo "Volumes"
check_ssh "named volume exists"       "docker volume ls"                              "webdata"

echo ""
echo "Basics"
check_ssh "can pull and run"          "docker run --rm hello-world 2>&1"              "Hello from Docker"

echo ""
echo "Results: ${PASS} passed, ${FAIL} failed"
echo ""

if [ "$FAIL" -gt 0 ]; then
  echo "Some checks failed. Re-read lab.md."
  exit 1
fi

echo "All checks passed. Lab 10 complete."
