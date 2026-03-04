---
name: lifebase-google-sync-reliability
description: Enforce reliable Google Calendar and Google Tasks synchronization for LifeBase. Use when implementing or reviewing sync workers, retry/backoff behavior, account isolation, conflict handling, or outbox processing.
---

# LifeBase Google Sync Reliability

- Keep Calendar and Tasks sync logic separated by domain.
- Apply account-level isolation for failures and retries.
- Use conservative retry/backoff with clear status transitions.
- Enforce conflict handling with deterministic user-facing resolution flow.
- Verify sync state persistence (`syncToken`, `updatedMin`, outbox status) and reauth handling.
