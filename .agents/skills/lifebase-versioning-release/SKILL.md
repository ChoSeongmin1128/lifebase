---
name: lifebase-versioning-release
description: Manage LifeBase release versioning and synchronization. Use when updating version numbers, preparing release tags, reconciling package and docs version drift, or validating semver bump policy.
---

# LifeBase Versioning Release

- Determine bump type (MAJOR/MINOR/PATCH) from compatibility impact.
- Update root `package.json` version first.
- Sync version references in docs (especially `docs/700-마일스톤.md`).
- Validate build/test before tagging.
- Create tag in `vX.Y.Z` format.
- If code/doc versions drift, stop release and sync first.
