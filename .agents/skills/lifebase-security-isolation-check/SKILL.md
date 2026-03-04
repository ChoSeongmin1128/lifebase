---
name: lifebase-security-isolation-check
description: Check security and tenant isolation requirements in LifeBase. Use when modifying auth, data access, sharing, sync, file handling, logging, or any flow that could leak sensitive or cross-user data.
---

# LifeBase Security Isolation Check

- Verify user-scoped filtering on all read/write paths.
- Confirm no sensitive values are hardcoded or logged.
- Validate file/storage security policies (UUID naming, MIME verification).
- Ensure sharing/token expiration and rate-limiting rules are intact.
- Require platform parity checks for Web/Desktop/Mobile where security behavior can diverge.
