---
name: lifebase-db-migration-safety
description: Create and validate safe database migrations in LifeBase. Use when adding or modifying goose migrations, indexes, constraints-by-code policies, rollback safety checks, and migration verification steps.
---

# LifeBase DB Migration Safety

- Create ordered goose migration files with explicit Up/Down paths.
- Validate naming, index coverage, and timestamp/soft-delete conventions.
- Confirm rollback feasibility before merge.
- Run migration status checks and verify critical query paths after applying changes.
- Block merge if rollback or index strategy is unclear.
