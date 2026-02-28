#!/bin/bash
# Verify no CDN dependencies in frontend

PASS=0
FAIL=0

HTML="webui/frontend/index.html"

# Should NOT contain external CDN URLs
if grep -q "cdnjs.cloudflare.com\|cdn.jsdelivr.net\|unpkg.com" "$HTML"; then
    echo "FAIL: index.html still references external CDN"
    FAIL=$((FAIL + 1))
else
    echo "PASS: No external CDN references in index.html"
    PASS=$((PASS + 1))
fi

# Should reference local vendor files
for f in "vendor/react.production.min.js" "vendor/react-dom.production.min.js" "vendor/babel.min.js"; do
    if grep -q "$f" "$HTML"; then
        echo "PASS: References local $f"
        PASS=$((PASS + 1))
    else
        echo "FAIL: Missing reference to $f"
        FAIL=$((FAIL + 1))
    fi
done

# Vendor files should exist
for f in webui/frontend/vendor/react.production.min.js webui/frontend/vendor/react-dom.production.min.js webui/frontend/vendor/babel.min.js; do
    if [ -f "$f" ]; then
        echo "PASS: $f exists"
        PASS=$((PASS + 1))
    else
        echo "FAIL: $f missing"
        FAIL=$((FAIL + 1))
    fi
done

echo ""
echo "Results: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ] || exit 1
