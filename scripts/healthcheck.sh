#!/usr/bin/env bash
# ────────────────────────────────────────────────────────────
# SynergyFlow — Health check script
# ────────────────────────────────────────────────────────────
# Verifies that key endpoints respond successfully.
# Designed to run on the EC2 instance (or locally if the
# Docker Compose stack is running on localhost).
# ────────────────────────────────────────────────────────────
set -euo pipefail

BASE="${HEALTHCHECK_BASE:-http://localhost}"
FAILED=0

check() {
  local label="$1" url="$2"

  STATUS=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "$url" 2>/dev/null || true)
  if [ "$STATUS" = "000" ]; then
    echo "FAIL  $label  — $url (connection refused / timeout)"
    FAILED=1
  elif [ "$STATUS" -lt 200 ] || [ "$STATUS" -ge 400 ]; then
    echo "FAIL  $label  — $url returned HTTP $STATUS"
    FAILED=1
  else
    echo "OK    $label  — $url returned HTTP $STATUS"
  fi
}

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo " SynergyFlow Health Check  ($(date -u '+%Y-%m-%dT%H:%M:%SZ'))"
echo " Base URL: $BASE"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

check "Frontend root"    "$BASE/"
check "Backend health"   "$BASE/health"
check "Backend ready"    "$BASE/ready"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [ "$FAILED" -ne 0 ]; then
  echo "Result: SOME CHECKS FAILED"
  exit 1
fi

echo "Result: ALL CHECKS PASSED"
exit 0
