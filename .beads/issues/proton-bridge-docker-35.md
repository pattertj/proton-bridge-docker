---
type: task
title: "SEC-MEDIUM: CDN dependencies lack SRI hashes"
status: open
priority: high
labels: [security]
created: 2026-02-27
---

# CDN dependencies lack Subresource Integrity hashes

**Severity:** Medium

**Location:** `webui/frontend/index.html:11-13`

**Description:** React, ReactDOM, and Babel are loaded from `cdnjs.cloudflare.com` at runtime. No Subresource Integrity (SRI) hashes are specified.

**Impact:** If cdnjs is compromised or a CDN cache is poisoned, malicious JavaScript could be injected, stealing Proton credentials directly from the login form.

**Recommendation:** Either add SRI hashes:

```html
<script src="https://cdnjs.cloudflare.com/ajax/libs/react/18.2.0/umd/react.production.min.js"
  integrity="sha384-..." crossorigin="anonymous"></script>
```

Or better: download the JS files, place them in `webui/frontend/vendor/`, and serve them via `embed.FS` to eliminate the CDN dependency entirely. This also allows the UI to work in air-gapped environments.
