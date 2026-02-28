#!/bin/bash
# Tests for entrypoint security hardening

PASS=0
FAIL=0

assert_contains() {
    local file="$1" pattern="$2" desc="$3"
    if grep -qE "$pattern" "$file"; then
        echo "PASS: $desc"
        PASS=$((PASS + 1))
    else
        echo "FAIL: $desc"
        FAIL=$((FAIL + 1))
    fi
}

ENTRYPOINT="docker/entrypoint.sh"

# #29: Verify restrictive permissions are set
assert_contains "$ENTRYPOINT" "chmod 700 /root$" "Sets /root to 700"
assert_contains "$ENTRYPOINT" "chmod 700 /root/\.gnupg" "Sets .gnupg to 700"
assert_contains "$ENTRYPOINT" "chmod 700 /root/\.password-store" "Sets .password-store to 700"

# #36: Verify port validation
assert_contains "$ENTRYPOINT" 'must be numeric' "Validates ports are numeric"
assert_contains "$ENTRYPOINT" 'SMTP_PORT' "Validates SMTP_PORT"
assert_contains "$ENTRYPOINT" 'IMAP_PORT' "Validates IMAP_PORT"
assert_contains "$ENTRYPOINT" 'WEBUI_PORT' "Validates WEBUI_PORT"

echo ""
echo "Results: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ] || exit 1
